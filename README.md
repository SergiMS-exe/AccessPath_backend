# AccessPath — Backend

API REST para la plataforma AccessPath: busca y valora la accesibilidad de lugares publicos.

## Stack

| Componente     | Tecnologia                              |
|----------------|-----------------------------------------|
| Lenguaje       | Go 1.25+                                |
| HTTP           | Gin                                     |
| Base de datos  | PostgreSQL 16 (`pgx/v5`)                |
| Cache          | Redis 7 (`go-redis/v9`)                 |
| Almacenamiento | MinIO S3 (`minio-go/v7`)                |
| Auth           | JWT HS256 + bcrypt                      |
| Docs           | Swagger en `/swagger`                   |
| Contenedores   | Docker Compose                          |

## Puesta en marcha

### Requisitos

- Docker Desktop
- Go 1.25+ (para desarrollo local)
- [`air`](https://github.com/air-verse/air) para hot-reload

### Desarrollo

```bash
# 1. Copia y edita el archivo de entorno
cp .env.example .env

# 2. Levanta la infra (postgres + redis + minio)
docker compose --env-file .env up -d

# 3. Arranca la API con hot-reload
air
```

La API queda disponible en `http://localhost:8080`.
Swagger en `http://localhost:8080/swagger/index.html`.

### Produccion

```bash
# Todo en Docker, incluida la API
docker compose --profile full --env-file .env up -d
```

## Variables de entorno

| Variable          | Descripcion                                           |
|-------------------|-------------------------------------------------------|
| `PORT`            | Puerto de la API (default `8080`)                     |
| `APP_ENV`         | `development` o `production`                          |
| `DATABASE_URL`    | DSN de PostgreSQL                                     |
| `REDIS_URL`       | URL de Redis                                          |
| `JWT_SECRET`      | Secreto para firmar JWT (cambia en produccion)        |
| `MINIO_ENDPOINT`  | Host:puerto de MinIO                                  |
| `MINIO_ROOT_USER` | Credencial MinIO                                      |
| `MINIO_ROOT_PASSWORD` | Credencial MinIO                                  |
| `MINIO_BUCKET`    | Nombre del bucket                                     |
| `MINIO_USE_SSL`   | `true` / `false`                                      |

En produccion los hosts son los nombres de servicio Docker (`postgres`, `redis`, `minio`).
En desarrollo son `localhost` o la IP de la maquina.

## Auth

- `POST /api/v1/auth/register` — crea cuenta
- `POST /api/v1/auth/login` — devuelve `token` (1h) + `refresh_token` (30 dias)
- `POST /api/v1/auth/refresh` — renueva el access token con el refresh token

Las rutas protegidas requieren `Authorization: Bearer <token>`.

## Estructura

```
cmd/server/main.go        # Entry point
internal/
  app/                    # Inyeccion de dependencias
  config/                 # Variables de entorno
  models/                 # Entidades y DTOs
  repositories/           # SQL con pgx
  services/               # Logica de negocio
  handlers/               # Capa HTTP
  middleware/             # Auth, CORS, Logger, Cache
  routes/                 # Setup del router
pkg/
  database/               # Conexiones PostgreSQL y Redis
  storage/                # Cliente MinIO
  response/               # Helpers de respuesta JSON
db/
  ddl.sql                 # Esquema de base de datos
  dml.sql                 # Seed inicial
```

Ver `.claude/PROJECT.md` para documentacion tecnica detallada.
