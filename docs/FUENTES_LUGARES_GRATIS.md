# Obtener lugares (nombre, dirección, coordenadas) sin pagar

> Análisis para AccessPath. Fecha: 2026-11-12.

## TL;DR

Hoy AccessPath usa Google Places (Legacy) para dos cosas:

- `GET /api/v1/places/search` → `Autocomplete` (`pkg/gmaps/client.go`).
- `POST /api/v1/places/from-google` → `Details` + insert en `place`.

La forma **más barata y sin tarjeta** de mantener el mismo producto es dejar de
depender de Google y servir el buscador desde una tabla propia de POIs cargada
con datos POI comerciales abiertos:

- **Foursquare OS Places** (109 M de POIs, Apache 2.0, actualización mensual con
  `date_refreshed` / `date_closed`). Es la fuente más “directorio de negocios”.
- **Overture Maps Places** (~81 M de POIs, CDLA-Permissive 2.0, sin registro ni
  API key, actualización mensual con `operating_status` y `sources[].update_time`).

**Overture no es OSM**: se construye a partir de proveedores comerciales (Meta,
Microsoft, BrightQuery, etc.) y no contiene datos de OpenStreetMap. Esto es
importante porque el problema que planteas —sitios que faltan o están
desactualizados— es precisamente la debilidad de OSM como directorio de
negocios, no la de Overture/Foursquare.

La recomendación es usar **Foursquare como fuente principal de negocios** y
**Overture como fuente de cobertura/deduplicación**. Se importa una vez al mes
(o solo la región que interese), se busca con `pg_trgm` en PostgreSQL y se
mantiene el mismo contrato HTTP, por lo que **la app móvil no cambia**.

Si solo se quiere “no pagar” pero se acepta tener cuenta de facturación de
Google, la opción 0 es no cambiar nada y subir el límite mensual: los SKUs que
usa el proyecto tienen cuota gratuita muy por encima de 500. Pero como has
descartado Google, esa opción queda solo como referencia.

---

## 0. Lo que hace hoy el backend

`pkg/gmaps/client.go` llama a:

```
https://maps.googleapis.com/maps/api/place/autocomplete/json
https://maps.googleapis.com/maps/api/place/details/json
```

y pide exactamente:

```
place_id, name, formatted_address, geometry, types, business_status
```

`place_service.ImportFromGoogle` comprueba duplicados por `google_place_id`,
aplica `GOOGLE_MAPS_MONTHLY_LIMIT` (500 por defecto), llama a `Details` y crea
la fila en `place`. El cliente está detrás de la interfaz `gmaps.Client`:

```go
type Client interface {
    Autocomplete(ctx, query, sessionToken string) ([]AutocompleteItem, error)
    Details(ctx, placeID, sessionToken string) (*PlaceDetails, error)
}
```

Eso es una ventaja: **se puede implementar otro proveedor sin tocar handlers,
servicios ni la app móvil**.

### Cuota real de Google desde marzo 2025

Google sustituyó el crédito de 200 USD/mes por cuotas gratuitas por SKU:

| SKU | Gratis/mes | Precio después |
|---|---:|---:|
| Autocomplete Requests (Legacy/New) | 10.000 | 2,83 USD/1k |
| Places Details (Legacy) | 5.000 | 17 USD/1k |
| Place Details Essentials (New) | 10.000 | 5 USD/1k |
| Place Details Pro (New) | 5.000 | 17 USD/1k |

El comentario del `.env.example` (“500 ≈ 8,50 USD/mes”) es correcto con el
precio antiguo, pero hoy 500 está dentro de la cuota gratuita. **Aun así, Google
exige cuenta de facturación y método de pago para habilitar Places**, aunque no
llegues a pagar. Si “sin pagar” significa “sin tarjeta”, hay que salir de Google.

Además, la licencia de Google no permite almacenar el contenido de Places de
forma permanente: el `place_id` sí se puede guardar indefinidamente, pero
nombre/dirección/etc. solo se pueden cachear de forma temporal (30 días). Para
una app que guarda lugares y valoraciones, eso es un problema de diseño, no
solo de precio.

---

## Frescura y cobertura: por qué OSM no sirve como base

Tu objeción es correcta y es el criterio que debería decidir la arquitectura.
OSM es un mapa excelente, pero **no es un directorio de negocios**: la
actualización depende de voluntarios, muchos locales no están o conservan datos
de hace años, y no hay una señal fiable de “cerrado” para la mayoría de POIs.

Comparativa de frescura:

| Fuente | POIs | Actualización | Señal de cierre | ¿Sirve como base? |
|---|---:|---|---|---|
| Google Places | — | Continua | `business_status` | Sí, pero descartado |
| **Foursquare OS Places** | 109,5 M | Mensual (~10,9 M refrescados/mes) | `date_closed`, `unresolved_flags` | **Sí** |
| **Overture Places** | ~81 M | Mensual | `operating_status`, `confidence`, `sources[].update_time` | **Sí** |
| OSM | ~? | Continua pero desigual | `opening_hours` no fiable | Solo complemento |

Ninguna fuente gratuita iguala la frescura de Google en tiempo real. Pero para
AccessPath el objetivo no es replicar Google: es tener una base suficientemente
buena y **corregirla con las aportaciones de los propios usuarios**. Eso es
justamente lo que la app ya hace con accesibilidad.

Reglas prácticas:

1. Importar solo POIs con `confidence` razonable (Overture) y sin `date_closed`
   (Foursquare).
2. Excluir `operating_status = 'permanently_closed'`.
3. Ordenar resultados por `date_refreshed` / `sources[].update_time` además de
   por relevancia.
4. Cuando un usuario reporte “este sitio ya no existe”, marcarlo y no volver a
   importarlo en la siguiente carga.

---

## 1. Opción recomendada: Foursquare OS Places + Overture Maps

### Por qué

- **Gratis de verdad**: Foursquare publica el dataset bajo Apache 2.0; Overture
  es GeoParquet público en S3/Azure, sin API key.
- **Licencias limpias**: Apache 2.0 y CDLA-Permissive 2.0. Sin share-alike.
- **Datos de negocio, no solo de mapa**: nombre, dirección, coordenadas,
  categoría, teléfono, web, estado y fecha de refresco.
- **Frescura medible**: Foursquare refresca ~10-14 M de POIs al mes y tiene
  `date_refreshed`; Overture publica `operating_status` y la fecha de cada
  fuente en `sources[].update_time`.
- **Se puede filtrar por país/bbox**: no hay que descargar el planeta.
- **Encaja con el stack actual**: PostgreSQL + `pg_trgm`; no hace falta PostGIS
  (el proyecto ya evita PostGIS y usa bounding box + Haversine).
- **Overture no usa OSM**: son datos de proveedores comerciales agregados.

Aviso de calidad: Overture reconoce que su tema de places tiene duplicados,
junk y baja completitud de atributos. `confidence` filtra parte del ruido, pero
no elimina duplicados. Por eso la primera fuente debería ser Foursquare y
Overture la segunda. En Foursquare, `date_refreshed` no es perfecto (parte del
dataset tiene fechas antiguas), así que conviene filtrar por los últimos 1–2
años si se quiere máxima frescura.

### Qué trae cada registro

| Campo Overture | Uso en AccessPath |
|---|---|
| `id` | ID externo estable (GERS) |
| `names.primary` | `place.name` |
| `addresses[1].freeform` | `place.address` |
| `addresses[1].locality/region/postcode/country` | filtros y UI |
| `geometry` | lat/lng |
| `basic_category` / `taxonomy.primary` | categoría / icono |
| `confidence` | filtrar resultados dudosos (0 = no existe) |
| `operating_status` | excluir `permanently_closed` |
| `sources[].update_time` | frescura por proveedor |
| `phones`, `websites`, `socials`, `emails` | enriquecimiento futuro |

Foursquare OS Places aporta además:

| Campo Foursquare | Uso en AccessPath |
|---|---|
| `fsq_place_id` | ID externo |
| `name`, `address`, `locality`, `postcode`, `country` | nombre y dirección |
| `latitude`, `longitude` | coordenadas |
| `fsq_category_labels` | categoría |
| `date_refreshed` | ordenar por frescura |
| `date_closed` | excluir cerrados |
| `unresolved_flags` | detectar `closed`, `duplicate`, `doesnt_exist` |
| `tel`, `website`, `email`, redes | enriquecimiento |

### ETL: exportar solo la zona de interés

Con DuckDB (CLI o Python) se lee directamente de S3 y se filtra por `bbox` y
país. Ejemplo para España:

```sql
INSTALL spatial;
INSTALL httpfs;
LOAD spatial;
LOAD httpfs;
SET s3_region = 'us-west-2';

COPY (
  SELECT
    id,
    names.primary              AS name,
    addresses[1].freeform      AS address,
    addresses[1].locality      AS city,
    addresses[1].region        AS region,
    addresses[1].postcode      AS postcode,
    addresses[1].country       AS country,
    ST_X(geometry)             AS lng,
    ST_Y(geometry)             AS lat,
    basic_category             AS category,
    taxonomy.primary           AS subcategory,
    confidence,
    operating_status,
    list_max(list_transform(sources, s -> s.update_time)) AS source_updated_at,
    phones,
    websites
  FROM read_parquet(
    's3://overturemaps-us-west-2/release/2026-10-21.0/theme=places/type=place/*',
    hive_partitioning = 1
  )
  WHERE addresses[1].country = 'ES'
    AND confidence >= 0.5
    AND coalesce(operating_status, 'open') <> 'permanently_closed'
) TO 'spain_places.parquet' (FORMAT PARQUET);
```

Foursquare OS Places se puede consultar con DuckDB/Iceberg o descargar desde
Hugging Face. Filtro equivalente:

```sql
SELECT
    fsq_place_id, name, address, locality, region, postcode, country,
    latitude, longitude,
    fsq_category_labels AS categories,
    date_refreshed, date_closed, unresolved_flags,
    tel, website, email
FROM places
WHERE country = 'ES'
  AND date_closed IS NULL
  AND coalesce(len(unresolved_flags), 0) = 0
  AND date_refreshed >= current_date - INTERVAL '2 years'
ORDER BY date_refreshed DESC;
```

Para no fijar la fecha del release se puede usar el catálogo STAC
(`https://stac.overturemaps.org/latest.ddb`) o consultar `catalog.json`.
También se puede acotar por `bbox.xmin/xmax/ymin/ymax` (más eficiente que
filtrar por país, porque hace *predicate pushdown* sobre los ficheros).

### Tabla de catálogo

Mantener el catálogo separado de `place` (que son los sitios con datos de
accesibilidad de AccessPath):

```sql
CREATE SCHEMA IF NOT EXISTS geo;

CREATE TABLE geo.place_catalog (
    id               TEXT NOT NULL,        -- GERS id o fsq_place_id
    source           TEXT NOT NULL,        -- 'overture' | 'foursquare'
    name             TEXT NOT NULL,
    address          TEXT,
    city             TEXT,
    region           TEXT,
    postcode         TEXT,
    country          TEXT,
    latitude         DOUBLE PRECISION NOT NULL,
    longitude        DOUBLE PRECISION NOT NULL,
    category         TEXT,
    subcategory      TEXT,
    confidence       DOUBLE PRECISION,
    operating_status TEXT,
    refreshed_at     TIMESTAMPTZ,          -- update_time / date_refreshed
    phones           TEXT[],
    websites         TEXT[],
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (source, id)
);

CREATE INDEX idx_catalog_name_trgm
    ON geo.place_catalog USING gin (name gin_trgm_ops);
CREATE INDEX idx_catalog_address_trgm
    ON geo.place_catalog USING gin (address gin_trgm_ops);
CREATE INDEX idx_catalog_geo
    ON geo.place_catalog (latitude, longitude);
```

Hace falta `CREATE EXTENSION IF NOT EXISTS pg_trgm;` (viene en `postgresql-contrib`,
que la imagen oficial de Postgres ya incluye).

### Búsqueda (sustituye a Autocomplete)

```sql
SELECT id, source, name, address, latitude, longitude, category,
       confidence, refreshed_at,
       similarity(name, $1) AS score
FROM geo.place_catalog
WHERE (name ILIKE '%' || $1 || '%'
    OR address ILIKE '%' || $1 || '%'
    OR similarity(name, $1) > 0.25)
  AND coalesce(operating_status, 'open') <> 'permanently_closed'
ORDER BY
    (name ILIKE $1 || '%') DESC,
    score DESC,
    (source = 'foursquare') DESC,   -- prioriza la fuente mas fresca
    refreshed_at DESC NULLS LAST,
    confidence DESC
LIMIT 10;
```

- `pg_trgm` + GIN hace que `ILIKE '%texto%'` no sea un table scan.
- El `ORDER BY` prioriza coincidencia por prefijo, similitud, fuente y frescura.
- Se puede cachear en Redis por `query` normalizada (ya hay middleware de cache).

### Adaptador Go

Se implementa la misma interfaz que ya consume el servicio:

```go
package localplaces

type Client struct{ db *pgxpool.Pool }

func (c *Client) Autocomplete(ctx context.Context, query, _ string) ([]gmaps.AutocompleteItem, error) {
    // SELECT ... FROM geo.place_catalog WHERE ...
    // Devuelve PlaceID = id de Overture, MainText = name, SecondaryText = address
}

func (c *Client) Details(ctx context.Context, placeID, _ string) (*gmaps.PlaceDetails, error) {
    // SELECT ... FROM geo.place_catalog WHERE id = $1
    // Devuelve PlaceID, Name, FormattedAddress, Lat, Lng, Types, BusinessStatus=""
}
```

En `internal/app/app.go` se elige el proveedor:

```go
var placesClient gmaps.Client
switch cfg.PlacesProvider {
case "overture":
    placesClient = localplaces.New(repos.DB)
default:
    placesClient = gmaps.New(cfg.GMapsAPIKey)
}
```

Pero hay dos detalles que ajustar en `place_service.ImportFromGoogle`:

1. **Nombre del endpoint**: `/places/from-google` puede quedarse por
   compatibilidad, pero internamente debe ser “importar desde catálogo”.
2. **Cuota/log**: `gmapsLog.CountThisMonth()` y `gmapsLog.Log()` solo aplican a
   Google. Añadir un método `Billable() bool` (o `Source() string`) a la
   interfaz y saltarse el límite y el log cuando el proveedor es local.

### Cambio de esquema en `place`

Hoy la unicidad es `google_place_id TEXT UNIQUE`. Para no mentir con el nombre:

```sql
ALTER TABLE place ADD COLUMN source TEXT NOT NULL DEFAULT 'google';
ALTER TABLE place ADD COLUMN external_id TEXT;
UPDATE place SET external_id = google_place_id WHERE google_place_id IS NOT NULL;

CREATE UNIQUE INDEX idx_place_source_external
    ON place (source, external_id)
    WHERE external_id IS NOT NULL AND deleted_at IS NULL;
```

Se puede mantener `google_place_id` relleno con el ID externo para no romper la
app móvil (`PlaceDto` lo espera), y migrar la UI a `external_id` más adelante.

### Atribución

Añadir en la app (pantalla “Acerca de” o pie del buscador):

> Datos de lugares: © Foursquare, Apache 2.0; © Overture Maps Foundation,
> CDLA-Permissive 2.0.

No hay obligación de share-alike, pero la atribución es buena práctica y algunas
licencias de las fuentes subyacentes la esperan. Si se usa Overpass/OSM para
accesibilidad, añadir “© OpenStreetMap contributors (ODbL)”.

---

## 2. Foursquare OS Places (fuente principal de negocios)

- ~109,5 M de POIs, licencia **Apache 2.0**, actualización mensual.
- Es un directorio de negocios, no un mapa: nombre, dirección, localidad,
  categoría, teléfono, web, email, redes y coordenadas.
- **Frescura real**: en el release de octubre de 2026 se actualizaron ~10,9 M de
  POIs, se añadieron 87.000 y se eliminaron ~18.000. El campo `date_refreshed`
  indica la última validación fiable; `date_closed` marca cierres.
- **Señales de calidad**: `unresolved_flags` puede traer `closed`, `duplicate`,
  `doesnt_exist`, `delete`, etc. Se pueden excluir esos registros.
- Ahora se distribuye por un portal con token (catálogo Iceberg) y también en
  Hugging Face. Sigue siendo gratis, pero requiere crear cuenta.
- El dataset global son ~263 GB; conviene extraer una vez el subconjunto de
  España (o de las provincias que cubra la app) y cargarlo en Postgres.

Formato de acceso típico con DuckDB/Iceberg o Parquet en HF:

```
hf://datasets/foursquare/fsq-os-places/release/dt=2026-10-15/places/parquet/*.parquet
```

Para AccessPath lo pondría como **primera opción para negocios** (restaurantes,
tiendas, clínicas, gimnasios, etc.), y Overture como segunda fuente para cubrir
huecos y deduplicar.

### Fusión Foursquare + Overture

No hace falta elegir una sola. Estrategia sencilla:

1. Cargar Foursquare en `geo.place_catalog` con `source='foursquare'`.
2. Cargar Overture con `source='overture'`.
3. Para cada POI de Overture, buscar un candidato Foursquare por distancia
   (< 50 m) y similitud de nombre; si coincide, no duplicar y quedarse con los
   atributos de Foursquare (más frescos).
4. Si no coincide, conservar Overture.
5. En la búsqueda, priorizar Foursquare y mostrar `date_refreshed` como señal de
   confianza.

---

## 3. OpenStreetMap: solo complemento de accesibilidad

Coincido contigo: **OSM no debe ser la base del buscador**. Sirve para otras
cosas, pero no para “¿existe este negocio y está actualizado?”.

- **Datos**: OSM, licencia **ODbL** (atribución + share-alike sobre bases de
  datos derivadas). Si se mezcla con la base de AccessPath, hay que valorar
  publicar la parte derivada bajo ODbL.
- **Nominatim**: geocodificación y búsqueda por nombre/dirección. La instancia
  pública (`nominatim.openstreetmap.org`) limita a 1 req/s, prohíbe autocompletar
  y prohíbe consultas masivas. **No sirve para producción**; habría que
  autoalojarlo (PostgreSQL + PostGIS + import de un extracto de Geofabrik).
- **Photon**: pensado para búsqueda type-ahead, autoalojable, índice más ligero.
  La instancia pública es de cortesía; para producción, autoalojar.
- **Overpass API**: excelente para extraer etiquetas concretas por bounding box.
  Aquí sí tiene valor para AccessPath: `wheelchair=yes/limited/no`,
  `toilets:wheelchair`, `handrail`, etc. Se usaría **solo para prerellenar
  accesibilidad**, no para el directorio de lugares.
- **Coste**: 0 € en software, pero sí servidor y mantenimiento.

Resumen: OSM como fuente de datos de accesibilidad, nunca como fuente principal
de existencia/actualización de negocios.

---

## 4. Alternativa: APIs comerciales con free tier

Sirven si se acepta una API key (no son datos abiertos). Ojo: muchas prohíben
guardar los resultados de forma permanente, que es justo lo que hace AccessPath.

| Proveedor | Free tier | Nota |
|---|---|---|
| Geoapify | ~3.000 créditos/día | geocoding + places; revisar ToS de almacenamiento |
| LocationIQ | 5.000 req/día | basado en OSM; atribución |
| OpenCage | 2.500 req/día | geocoding; permite cachear con condiciones |
| Mapbox | 100.000 geocoding/mes | geocoding temporal, no almacenamiento permanente |
| HERE | ~1.000 req/día | requiere cuenta |
| TomTom | ~2.500 req/día | requiere cuenta |

Para un prototipo o un MVP con poco tráfico pueden valer, pero no resuelven el
problema de fondo (dependencia + licencia de almacenamiento). No las pondría
como base de AccessPath.

---

## 5. Lo que NO recomendaría: scraping de Google Maps

Librerías tipo `google-maps-scraper`, Puppeteer/Playwright contra Maps, o APIs
“no oficiales”:

- Viola los Términos de Servicio de Google.
- Requiere proxies y mantenimiento constante (HTML cambiante, captchas).
- Riesgo de bloqueo de IP/cuenta y de responsabilidad legal.
- Datos sin licencia clara para redistribuir.

No es una base sostenible para una app pública.

---

## 6. Plan de implementación sugerido

### Fase 0 — validar cobertura antes de programar (2–3 horas)

1. Elegir 20–30 negocios reales de tu ciudad (una mezcla de cadenas, locales
   pequeños, clínicas, bares, tiendas) y comprobar cuántos aparecen en
   Foursquare y en Overture.
2. Medir tres cosas: % encontrados, % con dirección correcta y % con
   `date_refreshed`/`update_time` de los últimos 12 meses.
3. Si Foursquare supera el 80–85 %, es viable como base. Si no, replantear la
   zona de lanzamiento o combinar fuentes.

Esto evita construir todo el ETL y descubrir después que la cobertura no sirve.

### Fase 1 — catálogo e importación (2–3 días)

1. Añadir migración: `pg_trgm`, `geo.place_catalog`, `place.source`,
   `place.external_id`.
2. Script `scripts/import_places.sh` (DuckDB) para Foursquare + Overture:
   - Recortar por país/bbox.
   - Excluir cerrados y flags de calidad.
   - Cargar a Postgres con la extensión `postgres` de DuckDB o exportar Parquet
     y hacer `pgx.CopyFrom` en un comando Go.
3. Job mensual (GitHub Actions o cron) que actualice el catálogo.
4. Deduplicación básica Foursquare vs Overture por distancia + similitud.

### Fase 2 — adaptador y UI (1–2 días)

1. Implementar `pkg/localplaces` con la interfaz actual (`Autocomplete` y
   `Details`).
2. Selección de proveedor por `PLACES_PROVIDER=foursquare|overture|hybrid`
   (sin opción Google).
3. Quitar la cuota/log de Google del camino local (`Billable()`).
4. Cambiar el texto de la app: “Busca un lugar en Google Maps” → “Busca un lugar”.
5. Añadir atribución de Foursquare/Overture.
6. Mostrar “actualizado hace X” usando `date_refreshed`/`refreshed_at`.

### Fase 3 — correcciones de la comunidad (opcional pero recomendable)

- Botón “este sitio ya no existe / está mal” en la ficha del lugar.
- Guardar la corrección y excluir el POI de futuras importaciones.
- Importar `wheelchair` de OSM vía Overpass solo para prerellenar accesibilidad
  (parte ODbL separada y atribuida).

---

## 7. Comparativa rápida

| Opción | Coste | ¿Tarjeta? | ¿Guardar datos? | Frescura | Esfuerzo | ¿Base? |
|---|---|---|---|---|---|---|
| **Foursquare OS Places** | **0** | **No (cuenta)** | **Sí** | **Alta (mensual)** | **Medio** | **Sí** |
| **Overture Places** | **0** | **No** | **Sí** | **Alta (mensual)** | **Medio** | **Sí** |
| OSM/Nominatim propio | Servidor | No | Sí (ODbL) | Baja/media | Alto | No |
| APIs free tier | 0 dentro de cuota | Normalmente sí | Según ToS | Alta | Bajo | Prototipo |
| Google free tier | 0 dentro de cuota | Sí | Limitado (30 días) | Muy alta | Muy bajo | Descartado |
| Scraping Google | 0 + proxies | No | Ilegal | Muy alta | Alto | No |

---

## Fuentes

- Google Maps Platform, precios y cuotas gratuitas:
  <https://mapsplatform.google.com/pricing/>
- Google Maps Platform, lista de precios por SKU:
  <https://developers.google.com/maps/billing-and-pricing/pricing>
- Overture Maps, Places:
  <https://docs.overturemaps.org/guides/places/>
- Overture Maps, esquema de Place (incluye `operating_status`):
  <https://docs.overturemaps.org/schema/reference/places/place/>
- Overture Maps, acceso con DuckDB:
  <https://docs.overturemaps.org/getting-data/duckdb/>
- Foursquare OS Places, acceso:
  <https://docs.foursquare.com/data-products/docs/access-fsq-os-places>
- Foursquare OS Places, esquema (incluye `date_refreshed`, `date_closed`):
  <https://docs.foursquare.com/data-products/docs/places-os-data-schema>
- Foursquare OS Places, release notes:
  <https://docs.foursquare.com/data-products/docs/fsq-os-places-release-notes>
- Nominatim, política de uso:
  <https://operations.osmfoundation.org/policies/nominatim/>
