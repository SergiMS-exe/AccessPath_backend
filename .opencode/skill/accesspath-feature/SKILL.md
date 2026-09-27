---
name: accesspath-feature
description: Skill de referencia para implementar funcionalidades en el backend Go de AccessPath. Aplica al modificar/crear handlers, services, repositories, modelos o endpoints. Contiene convenciones del repo, patrones canónicos, checklist OWASP API Top 10 aplicado y la lista de "lo que NO hacer".
---

# AccessPath Backend — Skill para nuevas funcionalidades

Cuando el usuario pida implementar una nueva feature, una nueva ruta, un nuevo modelo, o modificar piezas existentes, **lee esta skill primero** y aplica los patrones aquí descritos.

## Arquitectura en capas

```
HTTP request
  -> Middleware (Recovery, RequestID, Logger, CORS, [Auth JWT], [Rate limit], [Cache Redis])
  -> Handler    (parseo, validación, delegación)
  -> Service    (lógica de negocio, transacciones, sentinels de error)
  -> Repository (SQL parametrizado, sin lógica)
  -> PostgreSQL / Redis / MinIO / Google Maps
```

Inyección en `internal/app/app.go`. **Sin interfaces explícitas**; inyección directa por struct.

## Convenciones (extraídas de `CLAUDE.md`)

1. **Sin acentos** en código, comentarios, identificadores ni strings de log. Texto de UI (`apperr.UserMessage`, respuestas JSON) sí puede llevar acentos.
2. Handlers solo transforman HTTP → servicio → respuesta. **Cero lógica de negocio**.
3. Repos solo hacen SQL parametrizado. **Cero lógica de negocio**.
4. **Escaneo de filas a struct**: `pgx.CollectRows` / `pgx.CollectOneRow` con `pgx.RowToStructByName[models.X]`. Los modelos llevan tag `db:"columna"`. **Nunca** `rows.Scan(&a, &b, ...)`. Excepción: escalares sueltos (`COUNT(*)`).
5. Transacciones se abren en el service (ver `ContributionService.Create`).
6. Tras cambiar la API pública: `swag init -g cmd/server/main.go`.
7. Errores siempre vía `apperr.New(...)` o `apperr.Wrap(...)`, **nunca** `errors.New` para errores que llegan al cliente.
8. Errores de dominio del service: sentinel `var ErrFoo = errors.New(...)` + mapeo en `internal/handlers/errors.go:serviceErrorMappings`.

## Ubicacion de los tests

**Todos los tests viven en `tests/`, en una carpeta top-level con la misma ruta que el paquete productivo**. Esta es la unica ubicacion valida en el repo:

```text
internal/services/foo.go
  -> tests/internal/services/foo_test.go     (package services_test)
internal/middleware/bar.go
  -> tests/internal/middleware/bar_test.go   (package middleware_test)
pkg/apperr/baz.go
  -> tests/pkg/apperr/baz_test.go            (package apperr_test)
```

Todos los tests son **black-box** (`package X_test`) y solo acceden a la API publica del paquete. No se permite `*_internal_test.go` ni `export_test.go` adyacentes al codigo productivo: `internal/` y `pkg/` no deben contener NINGUN `*_test.go`.

### Regla para tests que necesitan simbolos internos

Go **no permite** tener un test en una carpeta distinta al codigo y que acceda a simbolos no exportados: el mecanismo `export_test.go` solo se compila con los tests de su misma carpeta. Esto esta documentado oficialmente en https://pkg.go.dev/testing.

**Regla del repo**: si un helper, tipo o sentinel merece un test dedicado en `tests/`, se exporta a la API publica. Convecciones como "minimizar la API" o "encapsular" ceden ante "tests separados". Los nombres exportados son los oficiales; los internos pasan a tests dedicados solo cuando no son parte estable del contrato.

Simbolos exportados en este repo especificamente para habilitar tests separados:

| Paquete | Simbolo | Funcion |
|---|---|---|
| `pkg/apperr` | `DefaultMessageForStatus`, `LevelForStatus`, `OpFromContext` | Helpers de log y mapeo |
| `internal/middleware` | `ItoaForLimit`, `RetryAfterSeconds` | Helpers del rate limit |
| `internal/handlers` | `MatchServiceError` | Registry de errores de dominio |
| `internal/services` | `GenerateUsernameFromEmail`, `ShortEmailFingerprint` | Generacion de username |

### Naming de los archivos de test

Cada archivo en `tests/` se llama igual que el archivo productivo principal del paquete, con sufijo `_test.go`. Si el paquete tiene varios archivos a testear, se dividen por responsabilidad:

```text
tests/pkg/apperr/
  apperr_test.go              # cubre helpers de app errors
  respond_test.go             # cubre Respond, RespondInternal, RequestID, OpFromContext, LevelForStatus, DefaultMessageForStatus
```

## Patrones canónicos

### Handler

```go
// @Summary      Descripcion corta
// @Description  Detalle
// @Tags         modulo
// @Produce      json
// @Param        id   path   int   true   "ID"
// @Success      200  {object}  models.X
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Router       /x/{id} [get]
func (h *XHandler) Get(c *gin.Context) {
    id, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        Respond(c, apperr.BadRequest("x.get", "x.invalid_id", "Invalid ID"))
        return
    }
    item, err := h.service.Get(c.Request.Context(), id)
    if Respond(c, err) {
        return
    }
    response.OK(c, item)
}
```

### Service con sentinel

```go
var ErrXNotFound = errors.New("x not found")

func (s *XService) Get(ctx context.Context, id int64) (*models.X, error) {
    item, err := s.repo.FindByID(ctx, id)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, ErrXNotFound
        }
        return nil, apperr.Wrap("x.get", err)
    }
    return item, nil
}
```

Mapeo del sentinel en `internal/handlers/errors.go:serviceErrorMappings`:
```go
{
    target:  services.ErrXNotFound,
    matcher: func(t error) bool { return errors.Is(t, services.ErrXNotFound) },
    build:   func(op string, _ error) *apperr.AppError { return apperr.NotFound(op, "X") },
},
```

### Repository con escaneo por nombre

```go
const xColumns = `id, code, name, created_at`

func (r *XRepository) FindByID(ctx context.Context, id int64) (*models.X, error) {
    rows, err := r.db.Query(ctx,
        `SELECT `+xColumns+` FROM x WHERE id = $1 AND deleted_at IS NULL`, id)
    if err != nil {
        return nil, err
    }
    item, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.X])
    if err != nil {
        return nil, err
    }
    return &item, nil
}
```

### Modelo

```go
type X struct {
    ID        int64      `db:"id" json:"id"`
    Code      string     `db:"code" json:"code"`
    Name      string     `db:"name" json:"name"`
    CreatedAt time.Time  `db:"created_at" json:"created_at"`
    DeletedAt *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}
```

### Test (table-driven con testify)

```go
func TestXService_Get_NotFound_Returns_ErrXNotFound(t *testing.T) {
    tests := []struct {
        name    string
        id      int64
        repoErr error
        want    error
    }{
        {"no rows", 1, pgx.ErrNoRows, ErrXNotFound},
        {"db error", 2, errors.New("boom"), errors.New("boom")},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // mock repo, ejecutar, assert
        })
    }
}
```

## Checklist de seguridad (OWASP API Top 10 aplicado al repo)

| OWASP | Verificar |
|---|---|
| API1 Broken Object Level Auth | ¿`Update`/`Delete` recibe `userID` del token y compara con `CreatedBy`? Si no → BLOQUEAR. |
| API2 Broken Authentication | ¿Ruta que muta estado tiene `middleware.Auth`? ¿JWT solo HS256? ¿Token sensible se loguea? |
| API3 Broken Property Level Auth | ¿Campos sensibles (`password_hash`, etc.) llevan `json:"-"`? |
| API4 Unrestricted Resource Consumption | ¿Rate limit en `/auth/*` y `/places/search`? ¿Body max size? |
| API5 Broken Function Level Auth | ¿Rutas admin tienen middleware extra? |
| API6 Unrestricted Access to Sensitive Flows | ¿`/auth/register` rate-limited? |
| API7 SSRF | Si introduces una llamada HTTP saliente: ¿URL validada? ¿timeout? |
| API8 Security Misconfiguration | ¿CORS restrictivo? ¿Security headers? ¿Debug logs en prod? |
| API9 Improper Inventory Management | ¿Swagger regenerado? ¿Ruta documentada? |
| API10 Unsafe Consumption of APIs | ¿Timeouts en cliente HTTP saliente? ¿Retry con backoff? |

## Checklist de calidad (pre-PR)

- [ ] `gofmt -l .` no reporta archivos.
- [ ] `go vet ./...` limpio.
- [ ] `go test ./... -race` pasa; cobertura nueva tiene ≥ 3 casos (feliz + error + límite).
- [ ] `golangci-lint run` limpio (reglas activas: `gosec`, `staticcheck`, `gocritic`, `revive`, `misspell`).
- [ ] `govulncheck ./...` sin hallazgos.
- [ ] Si cambió API pública: `swag init -g cmd/server/main.go` regenerado y commiteado.
- [ ] Si cambió schema: nueva migración en `db/migrations/NNNN_*.up.sql` + `.down.sql`.
- [ ] Si cambió config: `.env.example` actualizado.

## Decisiones arquitectónicas (no desafiar sin propuesta formal)

- **Sin interfaces** en services/repos. Inyección por struct concreto. La razón histórica es simplicidad.
- **Sin ORM**. SQL crudo con `pgx`. Optimizaciones finas (índices, partial indexes) requieren control directo.
- **JWT HS256 + `golang-jwt/jwt/v5`**. No migrar a RS256 sin discutir trade-offs.
- **Cache Redis con degradación graceful**. Si Redis cae, el middleware sigue dejando pasar.
- **Soft delete** (`deleted_at IS NULL`) en `user`, `place`, `submission`, `collection`, `review_photo`. No cambiar a hard delete sin discutir.
- **Transacciones explícitas** en services que tocan múltiples tablas. Los repos participantes exponen variantes `*Tx(ctx, tx, ...)`.

## Lo que NO hacer

1. **No añadir acentos** a comentarios, identificadores, ni logs.
2. **No usar `interface{}`** en código de negocio. Si necesitas polimorfismo, probablemente hay un mal diseño.
3. **No usar ORM** (gorm, sqlx, ent, sqlc). El proyecto es `pgx` puro.
4. **No usar `errors.New`** para errores que llegan al cliente. Usa `apperr.New` o sentinels mapeados.
5. **No concatenar strings en SQL**. Solo `$1`, `$2`, ... Para listas usa `unnest($1::bigint[])`.
6. **No usar `rows.Scan(&a, &b, ...)`**. Usa `pgx.CollectRows` + `pgx.RowToStructByName`.
7. **No loguear `Authorization`, `Cookie`, `password`, `token`** ni datos personales (email, IP si aplica GDPR).
8. **No usar `reflect`, `json.RawMessage`, `interface{}` en DTOs de respuesta**.
9. **No añadir dependencias** sin discutir primero (`go get` añade al `go.sum`, hay que justificar).
10. **No exponer secretos** en respuestas JSON ni en logs.

## Archivos clave (mapa rápido)

| Fichero | Qué hace |
|---|---|
| `cmd/server/main.go` | Entry point, carga config, conecta DB/Redis/MinIO, registra handlers, arranca Gin, graceful shutdown |
| `internal/app/app.go` | Inyección de dependencias (BuildHandlers) |
| `internal/config/config.go` | Variables de entorno (lee `.env` con godotenv) |
| `internal/routes/routes.go` | Definición de rutas y middlewares por ruta |
| `internal/middleware/auth.go` | Verificación JWT HS256 |
| `internal/middleware/cache.go` | Cache Redis con degradación graceful |
| `internal/handlers/errors.go` | Punto único de respuesta a errores (`Respond`) |
| `internal/handlers/*.go` | Capa HTTP |
| `internal/services/*.go` | Lógica de negocio + sentinels |
| `internal/repositories/*.go` | SQL puro |
| `internal/models/*.go` | Entidades y DTOs |
| `pkg/apperr/apperr.go` | `*AppError`, helpers `BadRequest`/`Unauthorized`/`Internal`/... |
| `pkg/response/response.go` | Envelope `{ data, error }` y helpers `OK`/`Created` |
| `pkg/gmaps/client.go` | Cliente Google Maps (autocomplete, details) |
| `pkg/database/postgres.go` | Pool pgx |
| `db/migrations/` | Migraciones incrementales (golang-migrate) |
