# AccessPath Backend — Referencia del Proyecto

> Arquitectura, estado actual y convenciones. Actualizado: 2026-06-21.

---

## Que es AccessPath?

Plataforma para valorar la **accesibilidad de lugares publicos**.
Los usuarios buscan sitios y puntuan caracteristicas de accesibilidad en tres categorias:
**Fisica** (rampas, ascensores, banos), **Sensorial** (Braille, bucles), **Psiquica** (senalizacion, zonas tranquilas).

---

## Stack

| Componente        | Tecnologia                                         |
|-------------------|----------------------------------------------------|
| Lenguaje          | Go 1.25+                                           |
| Framework HTTP    | Gin                                                |
| Base de datos     | PostgreSQL 16 (`pgx/v5` + `pgxpool`)               |
| Cache             | Redis 7 (`go-redis/v9`, degradacion graceful)      |
| Almacenamiento    | MinIO S3-compatible (`minio-go/v7`)                |
| Imagenes          | `image/jpeg` stdlib (calidad 80, sin CGO)          |
| Auth              | JWT (`golang-jwt/jwt/v5`) + bcrypt                 |
| Docs              | Swagger (`swaggo/gin-swagger`) en `/swagger`       |
| Config            | `godotenv` + `.env`                                |
| Hot reload        | Air (`.air.toml`)                                  |

---

## Estructura

```
AccessPath_backend/
├── cmd/server/main.go          # Entry point: config -> DB -> Redis -> MinIO -> router -> graceful shutdown
├── internal/
│   ├── app/app.go              # Inyeccion de dependencias (BuildHandlers)
│   ├── config/config.go        # Variables de entorno
│   ├── models/                 # Entidades + DTOs
│   ├── repositories/           # SQL parametrizado con pgx
│   ├── services/               # Logica de negocio y transacciones
│   ├── handlers/               # Capa HTTP
│   ├── middleware/             # Auth JWT, CORS, Logger, Cache Redis
│   └── routes/routes.go        # Setup() del router
├── pkg/
│   ├── database/               # NewPostgresConnection, NewRedisClient
│   ├── storage/                # NewMinioClient + EnsureBucket
│   └── response/               # Helpers OK/Created/BadRequest/Unauthorized/...
├── db/
│   ├── ddl.sql                 # Esquema completo
│   └── dml.sql                 # Seed de categorias y subcategorias
├── docs/                       # Swagger generado
└── docker-compose.yml          # postgres + redis + minio; perfil `full` incluye la API
```

---

## Arquitectura en capas

```
HTTP Request
  -> Middleware (Recovery, Logger, CORS, [Auth JWT], [Cache Redis])
  -> Handler    (parseo, validacion, delegacion al servicio)
  -> Service    (logica de negocio, transacciones)
  -> Repository (SQL parametrizado, sin logica)
  -> PostgreSQL / MinIO / Redis
```

Inyeccion de dependencias en `internal/app/app.go`. Sin interfaces explicitas; inyeccion directa por struct.

Los repos escanean filas a struct con `pgx.CollectRows` / `pgx.CollectOneRow` +
`pgx.RowToStructByName`, emparejando columnas por el tag `db:"..."` del modelo (no por
orden). Las structs embebidas (`PlaceWithDistance`, `ReviewWithDetails`, etc.) se aplanan
automaticamente. Esto elimina el `rows.Scan(...)` manual y la fragilidad de orden de columnas.

---

## Modelo de datos

```
users ──< places (created_by)
      ──< collections ──< collection_places >── places
      ──< reviews ──< review_photos
                   ──< review_ratings >── subcategories ── categories
                                              ▲
                            place_rating_cache ┘  (avg_score, total_ratings)
```

- Identidad dual: `id BIGSERIAL` interno + `code UUID` publico en cada entidad.
- Soft delete con `deleted_at` en users, places, reviews, photos, collections.
- Ratings materializados en `place_rating_cache`, recalculados en la misma transaccion.
- Busqueda geografica sin PostGIS: bounding box + Haversine en SQL.
- **`place.published`**: un lugar importado de Google nace `false` (sirve de cache
  anti-duplicados por `google_place_id`, pero oculto en el mapa). Pasa a `true` en su
  primera valoracion (`ReviewService.Create` -> `PlaceRepository.MarkPublishedTx`, misma tx).
  `GET /places/map` filtra `published = TRUE`. Indice parcial `idx_place_published_location`.

---

## API — Endpoints (`/api/v1`)

```
GET  /health
GET  /swagger/*any

/auth
  POST /register
  POST /login      → { token, refresh_token, user }
  POST /refresh    → { token, refresh_token }

/users
  GET  /:id
  GET  /:id/collections

/places
  GET  /            [Cache Redis]
  GET  /map         [bounding box]
  GET  /nearby      [Cache Redis]
  GET  /:id
  POST /            [Auth]
  PUT  /:id         [Auth]
  DELETE /:id       [Auth]
  GET  /:id/reviews

/reviews            [Auth en todo el grupo]
  POST /
  DELETE /:id

/collections        [Auth en todo el grupo]
  POST /
  DELETE /:id
  GET  /:id/places
  POST /:id/places/:placeId
  DELETE /:id/places/:placeId

/categories
  GET  /
  GET  /:id
  POST /                      [Auth]
  GET  /:id/subcategories
  GET  /subcategories
  POST /subcategories         [Auth]
```

---

## Auth — Flujo de tokens

- **Access token**: JWT HS256, 1 hora. Claims: `user_id` (float64 en MapClaims → cast a int64).
- **Refresh token**: JWT HS256, 30 dias. Claims: `user_id` + `type: "refresh"`.
- El middleware `Auth` rechaza tokens con `type == "refresh"`.
- `POST /auth/refresh` verifica el refresh token, emite nuevo access + refresh (rotacion).
- Tokens stateless: no hay tabla en DB, no revocables individualmente.

---

## Variables de entorno

```env
PORT=8080
APP_ENV=development            # production activa gin.ReleaseMode
DATABASE_URL=postgres://accesspath:pass@localhost:5432/accesspath_db?sslmode=disable
REDIS_URL=redis://localhost:6379
JWT_SECRET=cambia-esto-en-produccion
MINIO_ENDPOINT=localhost:9000
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=minioadmin_dev_pass
MINIO_BUCKET=accesspath
MINIO_USE_SSL=false
```

---

## Estado actual

| Area                       | Estado                                                        |
|----------------------------|---------------------------------------------------------------|
| CRUD lugares               | Implementado                                                  |
| Busqueda geografica        | Implementado (bounding box + Haversine)                       |
| Reviews + ratings + fotos  | Implementado (transaccion atomica, fotos JPEG a MinIO)        |
| Colecciones                | Implementado                                                  |
| Categorias/subcategorias   | Implementado                                                  |
| Registro usuarios          | Implementado (bcrypt)                                         |
| Login con JWT              | Implementado (access 1h + refresh 30d)                       |
| Refresh token              | Implementado (`POST /auth/refresh`, rotacion de tokens)      |
| Middleware Auth            | Implementado, rechaza refresh tokens                          |
| Cache HTTP (Redis)         | Lee; escritura pendiente de revision                          |
| Fotos publicas             | URL apunta al endpoint interno de MinIO (pendiente Caddy)    |
| `user_id` desde token      | Implementado en Auth middleware; pendiente en Review/Place    |

---

## Pendientes conocidos

1. **`created_by` en places/reviews**: hoy viene del body JSON, deberia derivarse del token.
   El middleware ya expone `c.Get("user_id")` en el contexto Gin.
2. **URLs publicas de fotos**: MinIO escucha en `localhost:9000`, no expuesto al exterior.
   Cuando se configure Caddy como proxy hay que actualizar la URL base de las fotos.
3. **Cache Redis escritura**: `SetCache` tiene un bug con `ctx nil` (ver `middleware/cache.go`).
