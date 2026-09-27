# CLAUDE.md — AccessPath Backend

Instrucciones para Claude Code al trabajar en este repositorio.
Para arquitectura detallada ver `.claude/PROJECT.md`.
Skill de referencia (decisiones, checklist OWASP, patrones): `.opencode/skill/accesspath-feature/SKILL.md`.

## Convenciones de codigo

- **Sin acentos ni diacriticos** en codigo, comentarios, identificadores ni strings de log.
  El texto de UI (respuestas JSON que ve el usuario final) puede tener acentos.
- **No ejecutar builds** (`go build`, `docker compose build`, `air`, etc.) salvo que se pida
  explicitamente. El usuario compila y arranca por su cuenta.
- Handlers solo transforman HTTP → servicio → respuesta. Cero logica de negocio.
- Repos solo hacen SQL parametrizado. Cero logica de negocio.
- **Escaneo de filas a struct**: usar `pgx.CollectRows` / `pgx.CollectOneRow` con
  `pgx.RowToStructByName[models.X]`. Los modelos llevan tag `db:"columna"` en cada campo.
  No escribir `rows.Scan(&a, &b, ...)` a mano (el emparejamiento es por nombre, no por orden).
  Excepcion: escalares sueltos (`COUNT(*)`) siguen con `QueryRow(...).Scan(&n)`.
- Transacciones se abren en el servicio (ver `ReviewService.Create` como referencia).
- Tras cambiar la API publica regenerar Swagger: `swag init -g cmd/server/main.go`.

## Como anadir una funcionalidad nueva

Orden recomendado (cada paso incluye la verificacion que debe pasar antes de continuar):

1. **Migracion** en `db/migrations/NNNN_<nombre>.up.sql` + `.down.sql`. Aplicar con `make migrate-up`.
2. **Modelos** en `internal/models/<recurso>.go` con tag `db:` en cada campo. Si la respuesta
   es nueva, anadir tambien un DTO con `json:` separado.
3. **Repositorio** en `internal/repositories/<recurso>_repository.go` con SQL parametrizado.
4. **Servicio** en `internal/services/<recurso>_service.go`. Sentinels con `errors.New` si la
   ruta de error es nueva (mapear en `internal/handlers/errors.go:serviceErrorMappings`).
5. **Tests del servicio** en `tests/internal/services/<recurso>_test.go` con `package services_test`.
   Tabla-driven con testify. Si el test necesita acceder a helpers que hoy son privados, se exportan
   (la regla "tests separados" prima sobre la de "minimizar la API publica").
6. **Handler** en `internal/handlers/<recurso>_handler.go`. Validar inputs con `pkg/validate`.
   Pasar `userID` desde `middleware.UserID(c)` a `Update`/`Delete` (OWASP API1).
7. **Ruta** en `internal/routes/routes.go`. Anadir `auth` middleware a mutaciones.
   Si es sensible (`/auth/*`, `/places/search`), aplicar rate limit.
8. **Tests del handler** en `tests/internal/handlers/<recurso>_test.go` con `package handlers_test`. Mismo principio que el paso 5: si necesita acceder a simbolos no exportados, se exponen.
9. **Swagger** regenerado (`swag init -g cmd/server/main.go`).
10. **Verificacion local**: `make ci` (= gofmt + vet + test + golangci-lint).
11. **CI** valida todo automaticamente al abrir PR.

## Arranque local (desarrollo)

```bash
# 1. Infra (solo si no esta corriendo)
docker compose --env-file .env up -d

# 2. API con hot-reload
air
```

Requiere Go instalado y `air` en PATH. Ver `.env.example` para variables necesarias.

## Auth — estado actual

- Access token: JWT HS256, 1 hora, claim `user_id` (int64).
- Refresh token: JWT HS256, 30 dias, claim `type: "refresh"`. Solo valido en `POST /auth/refresh`.
- El middleware `Auth` rechaza tokens con `type: "refresh"`.
- No hay tabla de refresh tokens en DB — tokens stateless, no revocables individualmente.

## Migraciones

`db/migrations/` con archivos `NNNN_nombre.up.sql` y `NNNN_nombre.down.sql`. Comandos:

```bash
make migrate-up     # aplica todas las pendientes
make migrate-down   # revierte la ultima
```

El `docker-entrypoint-initdb.d` apunta al `000001_initial_schema.up.sql` para inicializar
la BD en el primer arranque del contenedor (caso comun en CI). A partir de ahi, todas las
nuevas se aplican con `make migrate-up`.

## CI

`/.github/workflows/ci.yml` ejecuta en cada push a `main` y en cada PR:

- `gofmt -l .`
- `go vet ./...`
- `go build ./...`
- `go test ./... -race`
- `golangci-lint run` (configurado en `.golangci.yml`: vet, staticcheck, gosec, revive, etc.)
- `govulncheck ./...`
- Verifica que `docs/` este sincronizado con `swag init`.

El atajo local es `make ci`.
