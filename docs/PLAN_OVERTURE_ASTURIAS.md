# Plan: catálogo Overture + fallback por enlace

> AccessPath · Backend + móvil · Fecha: 2026-11-13
> Alcance inicial: **Asturias**. Después, España.

## Estado de implementación (2026-11-14)

Implementado en el backend (sin Google Places: el codigo legacy se ha
eliminado, no queda detras de un flag):

| Pieza | Archivo |
|---|---|
| Migración `000002_place_catalog` | `db/migrations/000002_place_catalog.{up,down}.sql` |
| Catálogo + búsqueda trigram | `pkg/catalog/{item,provider,postgres,matching}.go` |
| Parser y resolver de enlaces | `pkg/gmaplink/{parser,resolver}.go` |
| Endpoints nuevos | `internal/handlers/place_handler.go`, `internal/routes/routes.go` |
| Servicio | `internal/services/place_service.go` |
| ETL | `scripts/import_overture.sh` |
| Config | `internal/config/config.go`, `.env.example` |
| Tests | `tests/pkg/{catalog,gmaplink}`, `tests/internal/{services,handlers}` |

También en móvil: `PlaceApi` (`importFromCatalog`, `resolveLink`,
`createCommunity`), DTOs nuevos y el fallback completo en `AddPlaceSheet.kt`.

Pendiente de ejecutar por el usuario (requiere base de datos y SDK):

1. `make migrate-up`.
2. `DATABASE_URL=... ./scripts/import_overture.sh` (necesita `duckdb`).
3. Probar los tres sitios de Lugones con `curl`.
4. Compilar e instalar la app.

## Objetivo

Sustituir Google Places por un catálogo propio cargado con **Overture Maps
Places**, y añadir un **fallback comunitario**:

1. El usuario busca un sitio.
2. Si está en el catálogo, se importa la ficha de Overture.
3. Si no está, el usuario puede pegar un **enlace de Google Maps**.
4. El backend extrae nombre, coordenadas y `ftid` del enlace (sin scrapear la
   ficha de Google).
5. Se busca otra vez en Overture por nombre + proximidad.
6. Si aparece, se importa.
7. Si no aparece, se abre un formulario prerellenado y el usuario crea el sitio.
8. El sitio queda **pendiente de moderación** hasta que se verifique.

El resultado es que la base crece sola con Overture y se corrige con los
usuarios, sin pagar API y sin depender de Google.

---

## Estado de partida

Estado **anterior** a esta migración (histórico, ya eliminado):

- `GET /api/v1/places/search` llamaba a Google Autocomplete.
- `POST /api/v1/places/from-google` llamaba a Google Details y creaba la fila.
- `place.google_place_id` era la clave de deduplicación.

Hoy el buscador usa `geo.place_catalog` y la clave es `(source, external_id)`.
- `published` significa “tiene al menos una contribución de accesibilidad”.
- La app móvil solo usa esos dos endpoints, así que el cambio de proveedor no
  obliga a rediseñar la UI.

Prueba de cobertura ya hecha en Lugones:

| Lugar | Overture | Foursquare (snapshot 2025-02) |
|---|---|---|
| Reactiva Lugones | ✅ Av. Luis Braille, 3 | ❌ |
| Sidrería de Diego | ✅ Monte Naranco, 4 | ⚠️ dirección antigua |
| La Tayuela | ✅ Pje. del Parque, 3 | ❌ |

Conclusión: Overture cubre bien la zona y es la mejor base inicial. Foursquare
puede añadirse más adelante como segunda fuente, pero no es necesario para el
MVP.

---

## Arquitectura

```
Búsqueda
  └─> GET /places/search
        └─> geo.place_catalog (PostgreSQL + pg_trgm)
              └─> POST /places/from-catalog
                    └─> place (fila local)

Sin resultados
  └─> POST /places/resolve-link
        ├─> match en catálogo → POST /places/from-catalog
        └─> sin match → prefill
              └─> POST /places/community
                    └─> place (pending) → moderación → verified
```

Principio: **el enlace de Google solo se usa como pista**, no como fuente de
datos. La ficha se guarda desde Overture o desde lo que complete el usuario.

---

## 1. Base de datos

### 1.1 Migración `000002_place_catalog`

Crear:

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE SCHEMA IF NOT EXISTS geo;

CREATE TABLE geo.place_catalog (
    id               TEXT NOT NULL,        -- GERS id de Overture
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
    refreshed_at     TIMESTAMPTZ,
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

CREATE TABLE geo.catalog_import (
    id          BIGSERIAL PRIMARY KEY,
    source      TEXT NOT NULL,
    release     TEXT NOT NULL,
    scope       TEXT NOT NULL,             -- 'asturias' | 'spain'
    row_count   INTEGER NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 1.2 Cambios en `place`

No reutilizar `published` para moderación. Añadir columnas nuevas:

```sql
ALTER TABLE place ADD COLUMN source TEXT NOT NULL DEFAULT 'community';
ALTER TABLE place ADD COLUMN external_id TEXT;
ALTER TABLE place ADD COLUMN google_ftid TEXT;
ALTER TABLE place ADD COLUMN source_url TEXT;
ALTER TABLE place ADD COLUMN moderation_status TEXT NOT NULL DEFAULT 'verified';

ALTER TABLE place ADD CONSTRAINT place_moderation_status_check
    CHECK (moderation_status IN ('verified', 'pending', 'rejected'));

-- (Eliminado: la migracion ya no conserva google_place_id. Las filas
-- historicas no se reetiquetan: el proyecto no esta en produccion.)

CREATE UNIQUE INDEX idx_place_source_external
    ON place (source, external_id)
    WHERE external_id IS NOT NULL AND deleted_at IS NULL;
```

Semántica:

- `moderation_status='verified'` → el sitio es real.
- `moderation_status='pending'` → creado por un usuario, pendiente de revisión.
- `moderation_status='rejected'` → descartado.
- `published` sigue significando “tiene datos de accesibilidad”.
- Mapa: `published = TRUE AND moderation_status = 'verified'`.
- Buscador público: `moderation_status = 'verified'`.

---

## 2. ETL de Overture

### 2.1 Extraer Asturias

Script `scripts/import_overture.sh` con DuckDB:

```sql
INSTALL spatial;
INSTALL httpfs;
LOAD spatial;
LOAD httpfs;
SET s3_region = 'us-west-2';

COPY (
  SELECT
    id,
    'overture' AS source,
    names.primary              AS name,
    addresses[1].freeform      AS address,
    addresses[1].locality      AS city,
    addresses[1].region        AS region,
    addresses[1].postcode      AS postcode,
    addresses[1].country       AS country,
    ST_Y(geometry)             AS latitude,
    ST_X(geometry)             AS longitude,
    basic_category             AS category,
    taxonomy.primary           AS subcategory,
    confidence,
    operating_status,
    list_max(list_transform(sources, s -> s.update_time)) AS refreshed_at,
    phones,
    websites
  FROM read_parquet(
    's3://overturemaps-us-west-2/release/2026-10-21.0/theme=places/type=place/*',
    hive_partitioning = 1
  )
  WHERE bbox.xmin BETWEEN -7.25 AND -4.40
    AND bbox.ymin BETWEEN 42.85 AND 43.80
    AND addresses[1].country = 'ES'
    AND confidence >= 0.5
    AND coalesce(operating_status, 'open') <> 'permanently_closed'
) TO 'asturias_places.parquet' (FORMAT PARQUET);
```

Notas:

- El `bbox` de Asturias es aproximado; sirve para reducir la descarga.
- `confidence >= 0.5` es un punto de partida. Se puede subir a 0.6 si aparecen
  duplicados o ruido.
- No fijar el release a mano: leer el último de
  `https://stac.overturemaps.org/latest.ddb` o del `catalog.json`.
- Para España, cambiar el `bbox` y mantener `addresses[1].country = 'ES'`.

### 2.2 Cargar en PostgreSQL

Opción A: extensión `postgres` de DuckDB.

```sql
INSTALL postgres;
LOAD postgres;
ATTACH 'postgresql://user:pass@host:5432/accesspath_db' AS pg (TYPE postgres);

INSERT INTO pg.geo.place_catalog
  (id, source, name, address, city, region, postcode, country,
   latitude, longitude, category, subcategory, confidence, operating_status,
   phones, websites, refreshed_at)
SELECT * FROM read_parquet('asturias_places.parquet');
```

Opción B: exportar a CSV y usar `psql`:

```bash
psql "$DATABASE_URL" -c "\copy geo.place_catalog FROM 'asturias_places.csv' CSV HEADER"
```

Opción C: comando Go `cmd/import-catalog` con `pgx.CopyFrom`. Es la más
controlable si luego hay que transformar arrays (`phones`, `websites`).

Recomendación: empezar por A; si falla, C.

### 2.3 Estrategia de recarga

Para el MVP, recarga completa en una transacción:

```sql
BEGIN;
DELETE FROM geo.place_catalog WHERE source = 'overture';
INSERT INTO geo.place_catalog SELECT ...;
INSERT INTO geo.catalog_import (source, release, scope, row_count)
VALUES ('overture', '2026-10-21.0', 'asturias', 123456);
COMMIT;
```

Cuando el catálogo crezca, cargar en `geo.place_catalog_new`, renombrar tablas
y volver a crear índices.

### 2.4 Deduplicación

Overture reconoce duplicados. Para el MVP:

- Ordenar por `confidence` y `refreshed_at` en la búsqueda.
- Si aparecen duplicados evidentes, añadir un paso de limpieza:

```sql
DELETE FROM geo.place_catalog a
USING geo.place_catalog b
WHERE a.source = 'overture'
  AND b.source = 'overture'
  AND a.id <> b.id
  AND lower(a.name) = lower(b.name)
  AND abs(a.latitude - b.latitude) < 0.0005
  AND abs(a.longitude - b.longitude) < 0.0005
  AND a.confidence < b.confidence;
```

No es obligatorio en la primera versión.

### 2.5 Programación

- Job mensual (cron o GitHub Action).
- Guardar release, fecha y número de filas en `geo.catalog_import`.
- Si el número de filas cae más de un 20 % respecto al mes anterior, abortar y
  avisar: probablemente el filtro o el release están mal.

---

## 3. Backend

### 3.1 Proveedor de lugares

Nuevo paquete `pkg/catalog`:

```go
type Item struct {
    Source       string
    ExternalID   string
    Name         string
    Address      *string
    City         *string
    Latitude     float64
    Longitude    float64
    Category     *string
    Confidence   *float64
    Operating    *string
    RefreshedAt  *time.Time
}

type Provider interface {
    Source() string
    Search(ctx context.Context, query string, lat, lng *float64) ([]Item, error)
    Details(ctx context.Context, source, externalID string) (*Item, error)
}
```

Implementación `pkg/catalog/postgres.go`:

- `Search`: SQL con `pg_trgm` y filtros de estado.
- `Details`: `SELECT ... WHERE source=$1 AND id=$2`.
- Si se recibe `lat/lng`, añadir distancia Haversine para ordenar.

Config:

```env
PLACES_LINK_TIMEOUT=5s
PLACES_LINK_MAX_REDIRECTS=3
```

`pkg/gmaps`, `/places/from-google`, `place.google_place_id` y `gmaps_api_log`
se han eliminado por completo: el codigo en el refactor y las estructuras de BD
en la migracion `000002`. No queda ningun camino que llame a la API de Google.

### 3.2 Búsqueda

`internal/services/place_service.go`:

```go
type PlaceService interface {
    // ...
    Search(ctx context.Context, query string, lat, lng *float64) ([]models.CatalogItem, error)
    ImportFromCatalog(ctx context.Context, source, externalID string, userID int64) (*models.Place, error)
    ResolveLink(ctx context.Context, rawURL string) (*models.ResolvedLink, error)
    CreateCommunity(ctx context.Context, req models.CreateCommunityPlaceRequest, userID int64) (*models.Place, error)
}
```

SQL de búsqueda:

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
    refreshed_at DESC NULLS LAST,
    confidence DESC
LIMIT 10;
```

Si hay `lat/lng`, añadir:

```sql
, (6371 * acos(
    cos(radians($2)) * cos(radians(latitude)) *
    cos(radians(longitude) - radians($3)) +
    sin(radians($2)) * sin(radians(latitude))
  )) AS distance
```

y ordenar primero por distancia cuando la consulta sea muy genérica.

También hay que buscar en `place` los sitios comunitarios ya verificados que no
estén en el catálogo, y fusionar resultados sin duplicados.

### 3.3 Endpoints nuevos

```
POST /api/v1/places/from-catalog   [Auth]
POST /api/v1/places/resolve-link   [Auth]
POST /api/v1/places/community      [Auth]
```

`from-catalog`:

```json
{ "source": "overture", "external_id": "abc123" }
```

- Lee la ficha del catálogo.
- Deduplica por `(source, external_id)`.
- Crea la fila en `place` con `moderation_status='verified'`.
- No llama a Google.
- No registra llamadas externas (no hay ninguna).

`resolve-link`:

```json
{ "url": "https://maps.app.goo.gl/..." }
```

Respuesta con match:

```json
{
  "matched": true,
  "catalog": {
    "source": "overture",
    "external_id": "abc123",
    "name": "Cafetería Eclipse",
    "address": "Calle Leopoldo Lugones, 24",
    "latitude": 43.4021,
    "longitude": -5.804946
  }
}
```

Respuesta sin match:

```json
{
  "matched": false,
  "prefill": {
    "name": "Cafetería Eclipse",
    "latitude": 43.4021,
    "longitude": -5.804946,
    "google_ftid": "0xd368bf0484e32bf:0xb9556be5462221b7"
  }
}
```

`community`:

```json
{
  "name": "Cafetería Eclipse",
  "address": "Calle Leopoldo Lugones, 24",
  "latitude": 43.4021,
  "longitude": -5.804946,
  "category": "cafe",
  "source_url": "https://maps.app.goo.gl/...",
  "google_ftid": "0xd368bf0484e32bf:0xb9556be5462221b7"
}
```

- Requiere auth.
- Deduplica por nombre + distancia (< 50 m).
- Inserta con `source='community'`, `moderation_status='pending'`.
- Devuelve el lugar para que el creador lo vea.

### 3.4 Parser de enlaces

Nuevo paquete `pkg/gmaplink`:

Formatos que debe soportar:

- `https://maps.app.goo.gl/XXXX`
- `https://goo.gl/maps/XXXX`
- `https://www.google.com/maps/place/<nombre>/@<lat>,<lng>,<zoom>z/data=...`
- `https://www.google.com/maps/place/.../data=...!3d<lat>!4d<lng>...`
- `https://www.google.com/maps/search/?api=1&query=<lat>,<lng>`
- `https://maps.google.com/?q=<lat>,<lng>`

Datos a extraer:

- Nombre: segmento `/place/<nombre>/`, URL-decoded.
- Coordenadas: `!3d<lat>!4d<lng>`; si no existe, `@<lat>,<lng>` o `?q=`.
- `ftid`: `!1s0x...:0x...` o `ftid=0x...:0x...`.
- Si el link es de búsqueda, no tratarlo como nombre de lugar.

Seguridad:

- Solo HTTPS.
- Allowlist de hosts: `maps.app.goo.gl`, `goo.gl`, `www.google.com`,
  `maps.google.com`.
- Máximo 3 redirecciones.
- Timeout de 5 s.
- `io.LimitReader` de 64 KB.
- Rechazar IPs privadas/loopback/link-local (SSRF).
- Longitud máxima de URL: 2048 caracteres.

No descargar la ficha de Google. Solo seguir el redirect y leer la URL final.

### 3.5 Búsqueda por proximidad

Para el match del enlace:

```sql
SELECT ...,
  (6371 * acos(
    cos(radians($1)) * cos(radians(latitude)) *
    cos(radians(longitude) - radians($2)) +
    sin(radians($1)) * sin(radians(latitude))
  )) AS distance
FROM geo.place_catalog
WHERE latitude BETWEEN $1 - 0.001 AND $1 + 0.001
  AND longitude BETWEEN $2 - 0.001 AND $2 + 0.001
  AND similarity(name, $3) > 0.25
ORDER BY distance ASC, similarity(name, $3) DESC
LIMIT 1;
```

Criterio de match: distancia < 50 m y similitud > 0.3. Si hay empate, quedarse
con `confidence` más alto.

### 3.6 Moderación

MVP sin panel de administración:

```sql
-- Aprobar
UPDATE place
SET moderation_status = 'verified', updated_at = NOW()
WHERE id = $1;

-- Rechazar
UPDATE place
SET moderation_status = 'rejected', updated_at = NOW()
WHERE id = $1;
```

Opcional: endpoint `PUT /api/v1/admin/places/:id/verify` protegido con
`ADMIN_TOKEN`.

Reglas:

- `pending` solo lo ve su creador.
- `pending` no sale en el mapa ni en búsquedas de otros usuarios.
- `rejected` no sale en ningún sitio.
- Si otro usuario confirma el mismo sitio, se puede auto-verificar en una fase
  posterior.

### 3.7 Filtros existentes

Actualizar:

- `FindByBounds`: añadir `AND moderation_status = 'verified'`.
- `FindNearby`: añadir `AND moderation_status = 'verified'`.
- `GetAll`: decidir si el listado público filtra por `verified`. Recomendado:
  sí, y añadir un endpoint aparte para “mis sitios pendientes”.

---

## 4. Móvil

### 4.1 `AddPlaceSheet.kt`

Añadir estados:

```kotlin
var showLinkFallback by remember { mutableStateOf(false) }
var linkUrl by remember { mutableStateOf("") }
var resolved by remember { mutableStateOf<ResolvedLinkDto?>(null) }
var communityForm by remember { mutableStateOf<PlacePrefillDto?>(null) }
```

Flujo:

1. Búsqueda normal.
2. Si `suggestions.isEmpty()` y `query.length >= 3`:
   - “No encontramos ese sitio”
   - Botón “Pegar enlace de Google Maps”
   - Botón “Añadirlo manualmente”
3. Al pegar el link:
   - `PlaceApi.resolveLink(url)`
   - `matched=true` → mostrar ficha de Overture + “Usar este sitio”
   - `matched=false` → abrir formulario con nombre y coordenadas.
4. Formulario comunitario:
   - Nombre (editable)
   - Dirección (obligatoria)
   - Categoría (opcional)
   - Notas (opcional)
   - Coordenadas ocultas
5. Al enviar:
   - `PlaceApi.createCommunity(...)`
   - Mostrar “Pendiente de revisión”.
   - Añadir a “Mis sitios” si existe esa pantalla.

### 4.2 `PlaceApi.kt`

Nuevos métodos:

```kotlin
suspend fun importFromCatalog(source: String, externalId: String): Result<PlaceDto>
suspend fun resolveLink(url: String): Result<ResolvedLinkDto>
suspend fun createCommunity(req: CreateCommunityPlaceRequest): Result<PlaceDto>
```

Sustituir `importFromGoogle` por `importFromCatalog` (hecho: el endpoint
`/from-google` ya no existe).

### 4.3 `PlaceModels.kt`

Nuevos DTOs:

```kotlin
@Serializable
data class CatalogItemDto(
    val source: String,
    @SerialName("external_id") val externalId: String,
    val name: String,
    val address: String? = null,
    val latitude: Double,
    val longitude: Double,
)

@Serializable
data class ResolvedLinkDto(
    val matched: Boolean,
    val catalog: CatalogItemDto? = null,
    val prefill: PlacePrefillDto? = null,
)
```

### 4.4 Textos y atribución

- Cambiar “Busca un lugar en Google Maps” → “Busca un lugar”.
- Añadir al pie del buscador:
  “Datos de lugares: © Overture Maps Foundation, CDLA-Permissive 2.0.”
- Si se usa Nominatim para dirección inversa, añadir “© OpenStreetMap contributors”.

---

## 5. Configuración

`.env.example`:

```env
# Resolución de enlaces de Google Maps
PLACES_LINK_TIMEOUT=5s
PLACES_LINK_MAX_REDIRECTS=3
```

Las variables de Google (`GOOGLE_MAPS_API_KEY`, `GOOGLE_MAPS_MONTHLY_LIMIT`) y
el endpoint `/places/from-google` ya no existen.

---

## 6. Pruebas

### 6.1 Unitarias

- Parser de enlaces:
  - link corto
  - link completo con `!3d/!4d`
  - link con `ftid=`
  - link de búsqueda con `?q=`
  - nombre con acentos y `+`
  - host no permitido
  - URL malformada
  - redirect loop
- Búsqueda SQL:
  - “Sidrería de Diego” → primer resultado correcto
  - “Reactiva Lugones” → primer resultado correcto
  - “La Tayuela” → primer resultado correcto
  - consulta sin resultados
- Deduplicación:
  - mismo `(source, external_id)` no crea dos filas
  - mismo nombre a < 50 m no crea duplicado comunitario

### 6.2 Integración

- `GET /places/search?q=...` responde con resultados de Overture.
- `POST /places/from-catalog` crea la fila y devuelve 201.
- `POST /places/resolve-link` con link de un sitio del catálogo devuelve
  `matched=true`.
- `POST /places/resolve-link` con link de un sitio que no está devuelve
  `matched=false` y `prefill`.
- `POST /places/community` crea un `pending`.
- Un `pending` no aparece en `/places/map`.
- Tras aprobarlo, aparece en búsqueda (aunque siga `published=false` hasta tener
  contribuciones).

### 6.3 Criterios de aceptación

- Los tres sitios de Lugones se encuentran por nombre.
- Un link de Google de un sitio del catálogo importa la ficha sin llamar a
  Google.
- Un link de un sitio que no está en el catálogo permite crearlo como pendiente.
- No se hace ninguna petición a `maps.googleapis.com`.
- La búsqueda en Asturias responde en < 300 ms.
- El mapa solo muestra sitios `verified` y `published`.

---

## 7. Fases y estimación

| Fase | Trabajo | Tiempo |
|---|---|---|
| 0 | Validar cobertura en Asturias (30 sitios) | 3 h |
| 1 | Migración + ETL Asturias + índices | 2–3 días |
| 2 | `pkg/catalog` + búsqueda + `from-catalog` | 1–2 días |
| 3 | `pkg/gmaplink` + `resolve-link` + `community` | 2 días |
| 4 | UI móvil (fallback + formulario) | 2 días |
| 5 | Moderación, atribución, pruebas y ajustes | 1–2 días |

**Total MVP Asturias: 8–10 días.** Si se recorta a Lugones y la moderación se
hace por SQL, se puede quedar en 5–6 días.

---

## 8. Orden de implementación recomendado

1. Migración y ETL de Asturias.
2. Cargar el catálogo y validar con SQL:
   ```sql
   SELECT name, address FROM geo.place_catalog
   WHERE name ILIKE '%sidreria%' AND name ILIKE '%diego%';
   ```
3. `pkg/catalog` + `Search` + `from-catalog`.
4. Cambiar la app para que `AddPlaceSheet` use `/from-catalog`.
5. `pkg/gmaplink` + `resolve-link`.
6. `community` + moderación.
7. UI del fallback.
8. Pruebas y despliegue.

Así se puede parar en cualquier punto con algo funcional.

---

## 9. Riesgos y decisiones

### Riesgos

- **Cobertura**: Overture puede fallar en locales muy nuevos o pequeños. El
  fallback comunitario lo cubre, pero la experiencia depende de que el usuario
  colabore.
- **Duplicados**: Overture tiene duplicados. Se mitiga con `confidence`,
  proximidad y deduplicación.
- **Frescura**: es una foto mensual, no tiempo real. Mostrar
  “actualizado hace X” ayuda a gestionar expectativas.
- **ToS de Google**: parsear un enlace compartido por el usuario es mucho menos
  problemático que scrapear la ficha, pero no conviene almacenar contenido de
  Google ni presentarlo como propio.
- **Spam**: el fallback comunitario puede atraer basura. Moderación + rate limit.

### Decisiones pendientes

1. ¿La dirección cuando no hay match la escribe el usuario o se obtiene con
   reverse geocoding?
   - Recomendado: que la escriba el usuario en el MVP. Si se usa Nominatim,
     añadir atribución ODbL.
2. ¿Se añade Foursquare desde el principio?
   - Recomendado: no. Primero Overture; Foursquare en una fase 2.
3. ¿Moderación manual o auto-publicar?
   - Recomendado: manual al principio.
4. ¿Se mantiene `/places/from-google`?
   - No: eliminado junto con `pkg/gmaps` y `place.google_place_id` (000002).

---

## 10. Archivos que se tocarán

```
db/migrations/000002_place_catalog.up.sql
db/migrations/000002_place_catalog.down.sql

scripts/import_overture.sh
cmd/import-catalog/main.go          (opcional, si no se usa DuckDB → Postgres)

pkg/catalog/provider.go
pkg/catalog/postgres.go
pkg/gmaplink/parser.go
pkg/gmaplink/parser_test.go

internal/config/config.go
internal/app/app.go
internal/models/place.go
internal/services/place_service.go
internal/handlers/place_handler.go
internal/routes/routes.go

composeApp/src/commonMain/kotlin/.../ui/place/AddPlaceSheet.kt
composeApp/src/commonMain/kotlin/.../data/api/PlaceApi.kt
composeApp/src/commonMain/kotlin/.../data/api/PlaceModels.kt

docs/PLAN_OVERTURE_ASTURIAS.md   (este documento)
```

---

## 11. Comandos útiles

Contar sitios de Asturias en Overture:

```sql
SELECT count(*)
FROM read_parquet(
  's3://overturemaps-us-west-2/release/2026-10-21.0/theme=places/type/place/*',
  hive_partitioning = 1
)
WHERE bbox.xmin BETWEEN -7.25 AND -4.40
  AND bbox.ymin BETWEEN 42.85 AND 43.80
  AND addresses[1].country = 'ES';
```

Comprobar un sitio concreto:

```sql
SELECT names.primary, addresses[1].freeform, operating_status, confidence
FROM read_parquet(
  's3://overturemaps-us-west-2/release/2026-10-21.0/theme=places/type/place/*',
  hive_partitioning = 1
)
WHERE names.primary ILIKE '%sidreria%'
  AND names.primary ILIKE '%diego%'
  AND bbox.xmin BETWEEN -5.83 AND -5.79
  AND bbox.ymin BETWEEN 43.39 AND 43.42;
```

Probar el parser de enlaces:

```bash
curl -sIL "https://maps.app.goo.gl/43FKHoBREWRVfGwe8" | grep -i location
```

---

## 12. Definición de “hecho”

- [ ] Catálogo de Asturias cargado y contado en `geo.catalog_import`.
- [ ] `GET /places/search` devuelve resultados de Overture.
- [ ] `POST /places/from-catalog` crea lugares desde el catálogo.
- [ ] `POST /places/resolve-link` resuelve links cortos y completos.
- [ ] `POST /places/community` crea lugares pendientes.
- [ ] Moderación básica documentada.
- [ ] Mapa y búsqueda filtran `pending`.
- [ ] App móvil muestra el fallback y el formulario.
- [ ] Atribución de Overture visible.
- [ ] Tests unitarios e integración en verde.
- [ ] Ninguna llamada a la API de Google.
