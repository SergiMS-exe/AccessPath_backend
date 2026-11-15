# Catálogo de lugares: documento técnico

> Estado: 2026-11-14 · Backend Go + PostgreSQL 16 · Sin dependencia de Google Places
> Documento de referencia del código que existe hoy. El plan original está en
> `docs/PLAN_OVERTURE_ASTURIAS.md`; esto describe lo implementado.

---

## 1. Resumen

El buscador de sitios ya no llama a ninguna API externa. Funciona así:

```
                 ┌──────────────────────────────┐
GET /search ────▶│ geo.place_catalog (Overture) │──┐
                 └──────────────────────────────┘  ├──▶ resultados
                 ┌──────────────────────────────┐  │
                 │ place (comunitarios verified)│──┘
                 └──────────────────────────────┘

POST /resolve-link ──▶ pkg/gmaplink ──▶ nombre + coords + ftid
                                         │
                                         ├─ match en catálogo (<50 m) ──▶ from-catalog
                                         └─ sin match ──▶ prefill ──▶ community (pending)
```

Tres ideas que conviene tener claras:

1. **El catálogo no es la tabla pública.** `geo.place_catalog` es una fuente de
   solo lectura que se recarga cada mes. Lo que ve el usuario sale de `place`.
2. **Moderación ≠ publicación.** `moderation_status` dice si el sitio es real;
   `published` dice si además tiene datos de accesibilidad.
3. **El enlace de Google es una pista, no una fuente.** Se parsea la URL; no se
   descarga la ficha ni se llama a la API.

---

## 2. Modelo de datos

Migración única: `db/migrations/000002_place_catalog.{up,down}.sql`.

### 2.1 `geo.place_catalog`

```sql
CREATE TABLE geo.place_catalog (
    id               TEXT NOT NULL,          -- GERS id
    source           TEXT NOT NULL,          -- 'overture'
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
    phones           TEXT[],
    websites         TEXT[],
    refreshed_at     TIMESTAMPTZ,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (source, id)
);
```

| Índice | Estado |
|---|---|
| `gin (name gin_trgm_ops)` | **Preparado, hoy no lo usa el buscador**: el SQL filtra con `strpos()`/`similarity()`, que no son indexables por GIN. Ver §8. |
| `gin (address gin_trgm_ops)` | Igual: preparado para cuando se reescriba el `WHERE`. |
| `btree (latitude, longitude)` | **Sí se usa** en el bounding box de `Nearby`. |
| `btree (operating_status)` | Preparado; el filtro actual va envuelto en `COALESCE(...)`, que impide usarlo. |

`PRIMARY KEY (source, id)` deja la puerta abierta a cargar Foursquare en la
misma tabla sin colisiones de id.

### 2.2 `geo.catalog_import`

Auditoría de cada carga: `source`, `release`, `scope`, `row_count`,
`imported_at`. No se usa en runtime. Sirve para detectar una carga vacía o
parcial (el script aborta por debajo de `MIN_ROWS`).

### 2.3 `place`

Columnas nuevas:

| Columna | Semántica |
|---|---|
| `source` | `overture` \| `community` |
| `external_id` | id en el proveedor; null en los comunitarios |
| `google_ftid` | `ftid` del enlace pegado; trazabilidad (no se usa para deduplicar) |
| `source_url` | enlace original; trazabilidad |
| `moderation_status` | `verified` \| `pending` \| `rejected` |

Índices:

```sql
CREATE UNIQUE INDEX idx_place_source_external
    ON place (source, external_id)
    WHERE external_id IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX idx_place_moderation ON place (moderation_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_place_name_trgm  ON place USING gin (name gin_trgm_ops);
```

Invariantes:

```
Mapa:      published = TRUE AND moderation_status = 'verified'
Buscador:  moderation_status = 'verified'
Detalle:   por id (el creador puede abrir su pending)
```

Eliminado en la misma migración: `place.google_place_id`, su índice y la tabla
`gmaps_api_log`. No queda código que los use.

---

## 3. Código

```
pkg/catalog/
  item.go        Item (fila + JSON), SecondaryText, HaversineMeters, BoundingBox
  provider.go    Provider (interfaz), ErrNotFound
  postgres.go    postgresProvider: Search, Details, Nearby
  matching.go    BestMatch, NameScore, Normalize
pkg/gmaplink/
  parser.go      Parse: clasifica y extrae datos de la URL (sin red)
  resolver.go    Resolver: expande acortadores con allowlist y anti-SSRF
internal/services/place_service.go
                 Search, ImportFromCatalog, ResolveLink, CreateCommunity
internal/handlers/place_handler.go
                 handlers HTTP + validación
scripts/import_overture.sh
                 ETL DuckDB → Parquet → Postgres
```

### 3.1 `Provider`

```go
type Provider interface {
    Source() string
    Search(ctx, query string, limit int) ([]Item, error)
    Details(ctx, source, externalID string) (*Item, error)
    Nearby(ctx, lat, lng, radiusMeters float64, name string) ([]Item, error)
}
```

`Nearby` devuelve candidatos por caja + similitud; la decisión final (≤50 m y
score) se toma en Go para poder testearla sin base de datos.

### 3.2 Matching

- `Normalize`: minúsculas, sin acentos, sin puntuación, espacios colapsados.
- `NameScore`: 1 si son iguales; 0.85 si uno contiene al otro (≥4 caracteres);
  si no, Jaccard por palabras.
- `BestMatch`: exige `distance ≤ 50 m` y `score ≥ 0.3`; empata por score y luego
  por distancia.

No usa `pg_trgm` para decidir: así el criterio es determinista y testeable.

### 3.3 Parser de enlaces

Soporta:

```
https://maps.app.goo.gl/XXXX
https://goo.gl/maps/XXXX
https://www.google.com/maps/place/<nombre>/@<lat>,<lng>,<zoom>z/data=...
https://www.google.com/maps/place/...!3d<lat>!4d<lng>...
https://www.google.com/maps/search/?api=1&query=<texto|lat,lng>
https://maps.google.com/?q=<lat>,<lng>
```

Extrae nombre (`/place/<nombre>/`), coordenadas (`!3d/!4d`, luego `@`, luego
`q=`/`query=`/`ll=`/`destination=`) y `ftid` (`!1s0x...:0x...` o `ftid=`).

### 3.4 Resolver (seguridad)

- Solo `https` (los `http` se elevan).
- Allowlist: `maps.app.goo.gl`, `goo.gl`, `google.com`, `*.google.com`,
  `www/maps.google.<tld>`.
- Máximo 3 redirecciones, timeout 5 s, cuerpo drenado hasta 64 KB.
- `DialContext.Control` rechaza loopback, privadas, link-local, multicast y
  CGNAT (defensa en profundidad frente a DNS rebinding).
- URL máxima 2048 caracteres.

Si el enlace no es corto, no hay ninguna petición de red: se parsea la URL tal
cual.

---

## 4. API

### 4.1 `GET /places/search`

Público. Rate limit 10/s por IP.

| Param | |
|---|---|
| `q` | obligatorio |
| `lat`, `lng` | opcionales; añaden `distance_meters` |
| `limit` | defecto 10; se recorta a 1..20 |

La respuesta va envuelta en el envelope común (`{"data": [...]}`), igual que el
resto de la API. Cada elemento:

```json
{
  "place_id": "08f2ab...",          // id del proveedor o id local (string)
  "description": "Cafetería Eclipse, Calle Leopoldo Lugones, 24, Lugones",
  "main_text": "Cafetería Eclipse",
  "secondary_text": "Calle Leopoldo Lugones, 24, Lugones",
  "source": "overture",
  "external_id": "08f2ab...",
  "address": "Calle Leopoldo Lugones, 24",
  "latitude": 43.4021,
  "longitude": -5.804946,
  "category": "cafe",
  "distance_meters": 12.4
}
```

`local_place_id` solo aparece si el sitio ya existe en `place`; tanto ese campo
como `distance_meters` se omiten cuando no aplican (no se envía `null`).

Comportamiento:

1. Busca en el catálogo (10 resultados).
2. Busca en `place` con `moderation_status='verified'` (10 resultados).
3. Descarta duplicados: mismo `(source, external_id)` o nombre ≥ 0.85 a <50 m.
4. Los resultados locales llevan `local_place_id` y `place_id` = id numérico.

Si el servicio se construye sin proveedor (solo ocurre en tests) →
`503 places.catalog_unavailable`. En la app el proveedor siempre existe.

### 4.2 `POST /places/from-catalog`

Auth. Body:

```json
{"source": "overture", "external_id": "08f2ab..."}
```

o, para un resultado que ya existe en la BD:

```json
{"place_id": 12}
```

- Idempotente por `(source, external_id)`.
- `404 not_found` si el `place_id` local no existe o la ficha no está en el
  catálogo (el `code` del envelope es `not_found` en ambos casos).
- `400 places.closed_permanently` si `operating_status = permanently_closed`.
- `400 places.unsupported_source` si el `source` no es el del proveedor.
- **No** escribe en ninguna cola ni llama a servicios externos.

### 4.3 `POST /places/resolve-link`

Auth. Rate limit 10/hora por usuario. Body `{"url": "..."}`.

Con match:

```json
{
  "matched": true,
  "catalog": { "source": "overture", "external_id": "...", "name": "...", "address": "...", "latitude": 43.4, "longitude": -5.8 }
}
```

Sin match:

```json
{
  "matched": false,
  "prefill": {
    "name": "Bar Nuevo",
    "latitude": 43.4,
    "longitude": -5.8,
    "google_ftid": "0xaa:0xbb",
    "source_url": "https://maps.app.goo.gl/..."
  }
}
```

`message` solo aparece cuando el enlace no trae nombre (enlaces de búsqueda);
en ese caso el cliente tiene que pedirlo.

Errores: `400 places.invalid_link`, `places.link_without_data`,
`places.link_too_many_redirects`, `places.link_timeout`, `places.link_error`.

### 4.4 `POST /places/community`

Auth. Rate limit 5/día por usuario. Body:

```json
{
  "name": "Bar Nuevo",
  "address": "Calle X, 1",
  "latitude": 43.4,
  "longitude": -5.8,
  "category": "bar",
  "description": null,
  "source_url": "https://maps.app.goo.gl/...",
  "google_ftid": "0xaa:0xbb"
}
```

- Valida nombre (≤255), dirección obligatoria (≤500), coordenadas, longitudes.
- Busca duplicado: caja de ±50 m y `similarity(name, $1) > 0.3` (pg_trgm), sin
  Haversine final; en la esquina de la caja puede llegar a ~70 m. Si hay
  candidato, devuelve el existente con **200**.
- Si crea: `source='community'`, `moderation_status='pending'`, **201**.

### 4.5 Moderación

No hay endpoint de admin todavía. Se hace por SQL:

```sql
UPDATE place SET moderation_status = 'verified', updated_at = NOW() WHERE id = $1;
UPDATE place SET moderation_status = 'rejected', updated_at = NOW() WHERE id = $1;
```

Un `pending` no aparece en `/places`, `/places/map`, `/places/nearby` ni en el
buscador. **Limitación**: `GET /places/:id` sí lo devuelve, y el endpoint es
público; cualquiera que conozca el id (son secuenciales) puede leerlo. No es
información sensible, pero conviene añadir control de propietario cuando exista
el panel de moderación.

---

## 5. ETL

`scripts/import_overture.sh`:

1. `ATTACH 'https://stac.overturemaps.org/latest.ddb'` (release actual, sin fijar fecha).
2. Filtra bbox + `addresses[1].country='ES'` + `names.primary IS NOT NULL`
   (la columna `name` es `NOT NULL`: un solo nulo abortaría la carga) +
   `confidence >= 0.5` + `COALESCE(operating_status,'open') <> 'permanently_closed'`
   (incluye los que no informan estado).
3. Exporta a Parquet.
4. Aborta si hay menos de `MIN_ROWS` filas.
5. `DELETE` + `INSERT` en una transacción e inserta en `catalog_import`.

```bash
DATABASE_URL=... ./scripts/import_overture.sh          # Asturias
SCOPE=spain DATABASE_URL=... ./scripts/import_overture.sh
```

Requiere `duckdb` (`nix shell nixpkgs#duckdb -c ./scripts/import_overture.sh`).

**Limitación conocida**: dentro de la transacción el `DELETE` deja el catálogo
vacío para esa sesión, pero los lectores siguen viendo la versión anterior hasta
el `COMMIT` (MVCC): no hay ventana de datos vacíos. Sí hay bloqueo de escritura
sobre la tabla mientras dura la carga. Con el catálogo de España habrá que pasar
al patrón tabla nueva + `RENAME`.

**Etiquetas**: `RELEASE` y `SOURCE` solo se guardan en `catalog_import`. El ETL
lee siempre el release `latest` del STAC y siempre datos de Overture; `SOURCE`
se valida para que no se etiquete por error otra fuente.

---

## 6. Configuración

```env
PLACES_LINK_TIMEOUT=5s
PLACES_LINK_MAX_REDIRECTS=3
```

No hay claves de Google, ni cuotas, ni contadores.

---

## 7. Operación

```bash
make migrate-up
curl -s "localhost:8080/api/v1/places/search?q=sidreria%20de%20diego" | jq
```

Comprobaciones útiles:

```sql
SELECT count(*) FROM geo.place_catalog;
SELECT source, scope, row_count, imported_at FROM geo.catalog_import ORDER BY id DESC LIMIT 5;
SELECT count(*) FROM place WHERE moderation_status = 'pending';
```

Rollback: `make migrate-down` (borra `geo` y las columnas nuevas, y recrea las
estructuras legacy de Google).

---

## 8. Rendimiento actual

Volumen esperado:

| Ámbito | POIs | Tamaño aprox. con índices |
|---|---|---|
| Asturias | ~30–60k | < 100 MB |
| España | ~400–800k | 1–2 GB |

Consultas del buscador: 2 queries por petición (catálogo + place), sin joins,
`LIMIT 10`, con `strpos(lower(name), ...)` y `similarity(...)` en el `WHERE`.
Con el catálogo de Asturias el rango realista es **5–30 ms** en caliente.

**Importante, y hoy es deuda técnica**: ninguna de esas dos expresiones usa los
índices GIN trigram. Son indexables `ILIKE`/`%` y `similarity` solo a través de
`%` (que compara contra `pg_trgm.similarity_threshold`, por defecto 0.3). Es
decir: con Asturias funcionan por volumen, no por índice. Al pasar a España hay
que reescribir el `WHERE` antes de culpar al motor:

```sql
WHERE name % $1 OR strpos(lower(name), lower($1)) > 0
```

y quitar el `COALESCE` del filtro de `operating_status` para que use su btree.
`ORDER BY similarity(...)` también obliga a calcularla fila a fila: conviene
que el pre-filtro sea selectivo.

Medición recomendada antes de optimizar: `EXPLAIN (ANALYZE, BUFFERS)` sobre 20
consultas reales + `vegeta`/`k6` a 10–20 rps.

---

## 9. ¿Rentable meter Elasticsearch?

**Recomendación: no, hoy.** El problema actual no es la velocidad de búsqueda,
es la cobertura del catálogo y la moderación. Añadir ES ahora suma coste fijo y
superficie operativa sin mejorar la experiencia de forma perceptible.

### 9.1 Qué aportaría de verdad

| Capacidad | Postgres + pg_trgm | Elasticsearch / OpenSearch |
|---|---|---|
| Prefijo / contiene | Sí (`strpos`, `ILIKE`, trgm) | Sí (`edge_ngram`, `match_phrase_prefix`) |
| Tolerancia a typos | Aceptable (trigramas) | Mejor (fuzzy, distancia de edición) |
| Ranking | Manual (`ORDER BY similarity, confidence`) | BM25 + boosts + function_score |
| Sinónimos / stopwords | Manual y limitado | Nativo |
| Multi-idioma | Trabajo manual | Analyzers por campo |
| Facetas / agregaciones | `GROUP BY` | Nativo |
| Geo | `BETWEEN` + Haversine; PostGIS si hace falta | `geo_distance`, `geo_bounding_box` |
| Escala de escritura/lectura | 1 nodo | Shards, réplicas |
| Coste | 0 (ya existe) | Servicio + RAM + operación |

Para nuestro caso el patrón de consulta es muy estrecho: **nombre + proximidad +
10 resultados**. No hay facetas, ni búsqueda semántica, ni multi-idioma.

### 9.2 Qué cuesta

- **Infra**: un nodo ES/OpenSearch con heap 1–2 GB necesita ~2–4 GB de RAM. En el
  mismo VPS que Postgres, compiten. Gestionado (Elastic Cloud) arranca en
  ~50–100 $/mes.
- **Pipeline**: el catálogo es mensual y de solo lectura, así que la
  sincronización sería una reindexación completa con alias swap. Es la parte
  sencilla.
- **Operación**: backups, upgrades, seguridad, monitorización, mapeos y
  reindexados. Es un sistema más que puede caerse a las 3 de la mañana.
- **Complejidad de código**: dos fuentes de verdad; hay que mantener el ranking
  en dos sitios y decidir qué pasa cuando ES y Postgres discrepan.
- **Licencia**: Elasticsearch usa SSPL/Elastic License 2.0; OpenSearch es
  Apache 2.0. Si el proyecto es abierto, importa.

### 9.3 Cuándo sí

Reconsiderar si se cumple **al menos dos** de estos:

1. El catálogo pasa de ~1–2 M de POIs o de varias fuentes solapadas.
2. p95 del buscador > 300–500 ms con índices en caliente.
3. Se necesitan sinónimos, tolerancia a typos fuerte o ranking por popularidad.
4. Hay facetas (categoría, accesibilidad, ciudad) o autocompletado por carácter.
5. Varios clientes/países con idiomas distintos.
6. >50 rps sostenidos.

### 9.4 Alternativas antes que ES

En orden de coste/beneficio:

1. **Postgres, pero mejor**: columna `search_name` normalizada (sin acentos) en
   el ETL + `unaccent` + `%` indexable + `tsvector` para ranking léxico. Cero
   infra nueva.
2. **PostGIS** si las consultas geo crecen (radios, polígonos, `ST_DWithin`).
   Sigue siendo el mismo Postgres.
3. **Meilisearch / Typesense**: un binario, typo tolerance y facetas nativas,
   mucha menos operación que ES. Buena opción si el problema es solo la UX de
   búsqueda.
4. **OpenSearch** solo si además se quieren logs o analítica.

### 9.5 Números para decidir

Antes de invertir, medir con el catálogo de España cargado:

```sql
\timing on
EXPLAIN (ANALYZE, BUFFERS)
SELECT id, name FROM geo.place_catalog
WHERE strpos(lower(name), lower('sidreria de diego')) > 0
   OR similarity(name, 'sidreria de diego') > 0.2
ORDER BY similarity(name, 'sidreria de diego') DESC
LIMIT 10;
```

Si el `Execution Time` se queda en decenas de ms, ES no aporta nada medible al
usuario. Si se va a cientos de ms con carga real, primero reescribir el `WHERE`
para usar el índice; solo después plantearse un motor aparte.

---

## 10. Decisiones y descartes

| Decisión | Motivo |
|---|---|
| Catálogo en Postgres, no en ES | Volumen y patrón de consulta no lo justifican |
| `pg_trgm` y no `tsvector` puro | Buscamos nombres propios; el stemming estorba |
| Match en Go | Testeable sin BD y con criterio explícito |
| `source`+`external_id` en vez de `google_place_id` | No depender de un proveedor |
| `moderation_status` separado de `published` | Son dos ejes distintos |
| `pending` invisible en mapa/búsqueda | Evitar basura y datos falsos |
| Sin endpoint de admin (aún) | Moderación por SQL es suficiente en el MVP |
| Sin caché de búsqueda | 2 queries ligeras; Redis no aporta todavía |
| Foursquare aparcado | Overture cubre Asturias; se añadirá como `source` si hace falta |

---

## 11. Tests

| Área | Dónde |
|---|---|
| Matching, Haversine, bbox | `tests/pkg/catalog` |
| Parser de enlaces | `tests/pkg/gmaplink` |
| Servicio (búsqueda, import, link, community) | `tests/internal/services` |
| Handlers y mapeo de errores | `tests/internal/handlers` |
| Fakes de repos/proveedor | `tests/internal/fakes` |

```bash
go test ./... -race -count=1
```

Lo que no hay (y conviene añadir cuando haya BD de test): tests de integración
reales contra Postgres para validar el SQL del catálogo y el ETL.
