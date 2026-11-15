#!/usr/bin/env bash
#
# Importa los lugares de Overture Maps a geo.place_catalog.
#
# Uso:
#   DATABASE_URL=postgres://user:pass@host:5432/accesspath_db \
#     ./scripts/import_overture.sh
#
#   SCOPE=spain DATABASE_URL=... ./scripts/import_overture.sh
#
# Requiere duckdb en el PATH (o `nix shell nixpkgs#duckdb -c ./scripts/import_overture.sh`).
#
# El script:
#   1. Lee el release *latest* de Overture via el catalogo STAC (sin fijar fecha).
#   2. Filtra por bounding box, pais y calidad.
#   3. Exporta a Parquet.
#   4. Recarga geo.place_catalog en una transaccion y registra la importacion.
#
# Overture Places usa licencia CDLA-Permissive 2.0: se puede almacenar y usar
# comercialmente, con atribucion ("Datos de lugares: (c) Overture Maps Foundation").

set -euo pipefail

SCOPE="${SCOPE:-asturias}"
SOURCE="${SOURCE:-overture}"
# RELEASE es solo una etiqueta para geo.catalog_import: el ETL lee siempre el
# release "latest" del catalogo STAC.
RELEASE="${RELEASE:-latest}"
OUT="${OUT:-./overture_${SCOPE}.parquet}"
MIN_ROWS="${MIN_ROWS:-1000}"

if ! command -v duckdb >/dev/null 2>&1; then
  echo "error: duckdb no esta en el PATH" >&2
  echo "  prueba: nix shell nixpkgs#duckdb -c $0" >&2
  exit 1
fi

if [ -z "${DATABASE_URL:-}" ]; then
  echo "error: DATABASE_URL no esta definida" >&2
  exit 1
fi

if [ "$SOURCE" != "overture" ]; then
  echo "error: SOURCE solo admite 'overture' (el ETL siempre lee de Overture)" >&2
  exit 1
fi

# Bounding boxes (min_lng max_lng min_lat max_lat). Asturias es un rectangulo
# algo mayor que el principado: el filtro real de pais lo hace addresses[].country.
case "$SCOPE" in
  asturias)
    BBOX_MIN_LNG=-7.25; BBOX_MAX_LNG=-4.40
    BBOX_MIN_LAT=42.85; BBOX_MAX_LAT=43.80
    ;;
  spain)
    BBOX_MIN_LNG=-9.35; BBOX_MAX_LNG=4.35
    BBOX_MIN_LAT=35.95; BBOX_MAX_LAT=43.85
    ;;
  *)
    echo "error: SCOPE desconocido: $SCOPE (usa asturias o spain)" >&2
    exit 1
    ;;
esac

echo "==> exportando $SCOPE desde Overture ($RELEASE)"
echo "    bbox: lng [$BBOX_MIN_LNG, $BBOX_MAX_LNG] lat [$BBOX_MIN_LAT, $BBOX_MAX_LAT]"

duckdb -c "
INSTALL spatial; LOAD spatial;
INSTALL httpfs;  LOAD httpfs;
ATTACH 'https://stac.overturemaps.org/latest.ddb' AS ov (READ_ONLY);

COPY (
  SELECT
    id,
    '$SOURCE'                                        AS source,
    names.primary                                    AS name,
    addresses[1].freeform                            AS address,
    addresses[1].locality                            AS city,
    addresses[1].region                              AS region,
    addresses[1].postcode                            AS postcode,
    addresses[1].country                             AS country,
    ST_Y(geometry)                                   AS latitude,
    ST_X(geometry)                                   AS longitude,
    basic_category                                   AS category,
    taxonomy.primary                                 AS subcategory,
    confidence,
    operating_status,
    phones,
    websites,
    list_max(list_transform(sources, s -> s.update_time)) AS refreshed_at
  FROM ov.place
  WHERE bbox.xmin BETWEEN $BBOX_MIN_LNG AND $BBOX_MAX_LNG
    AND bbox.ymin BETWEEN $BBOX_MIN_LAT AND $BBOX_MAX_LAT
    AND addresses[1].country = 'ES'
    AND names.primary IS NOT NULL
    AND confidence >= 0.5
    AND coalesce(operating_status, 'open') <> 'permanently_closed'
) TO '$OUT' (FORMAT PARQUET);
"

ROWS=$(duckdb -noheader -list -c "SELECT count(*) FROM read_parquet('$OUT');")
echo "==> $ROWS filas exportadas a $OUT"

if [ "$ROWS" -lt "$MIN_ROWS" ]; then
  echo "error: solo $ROWS filas (< MIN_ROWS=$MIN_ROWS). Abortando para no vaciar el catalogo." >&2
  exit 1
fi

echo "==> recargando geo.place_catalog"
duckdb -c "
INSTALL postgres; LOAD postgres;
ATTACH '$DATABASE_URL' AS pg (TYPE postgres);

BEGIN TRANSACTION;
DELETE FROM pg.geo.place_catalog WHERE source = '$SOURCE';
INSERT INTO pg.geo.place_catalog
  (id, source, name, address, city, region, postcode, country,
   latitude, longitude, category, subcategory, confidence, operating_status,
   phones, websites, refreshed_at)
SELECT * FROM read_parquet('$OUT');
INSERT INTO pg.geo.catalog_import (source, release, scope, row_count)
VALUES ('$SOURCE', '$RELEASE', '$SCOPE', $ROWS);
COMMIT;
"

echo "==> hecho: $ROWS lugares de $SCOPE en geo.place_catalog"
echo "    ojo: se ha reemplazado el catalogo anterior de source='$SOURCE'"
echo "    atribucion obligatoria: (c) Overture Maps Foundation, CDLA-Permissive 2.0"
