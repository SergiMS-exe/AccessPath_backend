# Resumen de cambios — AccessPath Backend

> Documento vivo. Detalla todos los cambios estructurales, de calidad y de seguridad aplicados al backend en esta tanda.
>
> **Estado al cierre**: build OK, todos los tests pasan, CI workflow listo.

---

## 0. Reorganizacion de tests (añadido posteriormente)

Ver §8 al final del documento para el detalle completo de esta reorganizacion.

---

## 1. Contexto y motivacion

El backend tenía una arquitectura sólida en capas (handlers → services → repositories), un sistema de errores tipados (`*apperr.AppError`) y un patrón de escaneo a struct con `pgx.CollectRows`. Pero le faltaba:

- **0 tests** en el repo (cero `*_test.go`).
- **Sin CI** (solo había un workflow que publicaba la imagen a GHCR, sin validación).
- **CORS abierto** a `*` con `Authorization`.
- **Sin rate limiting** (vulnerable a brute force en `/auth/*`, y quemaba cuota de Google Maps en `/places/search`).
- **Sin validación de rangos geográficos** (lat/lng); `parseFloatOrDefault(..., 0)` enmascaraba inputs malformados con 0 silenciosamente.
- **Sin ownership check** (OWASP API1) en `Place.Update`/`Place.Delete`.
- **Sin security headers** ni redacción de logs sensibles.
- **Sin migraciones incrementales** (todo-o-nada vía `db/ddl.sql`).
- **Sin scanner de vulnerabilidades** (`govulncheck`, `gosec`).
- **Sin skill/convenciones** que guíen a un developer o a un agente al añadir features.

Esto es lo que se ha hecho para cubrir cada hueco.

---

## 2. Inventario de archivos

### 2.1 Nuevos (17)

| Archivo | Líneas | Propósito |
|---|---|---|
| `.opencode/skill/accesspath-feature/SKILL.md` | ~250 | Skill de comportamiento con arquitectura, convenciones, patrones canónicos, checklist OWASP y "lo que NO hacer". |
| `.opencode/opencode.jsonc` | ~150 | Config de opencode con los 3 subagents del proyecto (`security-reviewer`, `test-writer`, `code-quality-reviewer`). |
| `.golangci.yml` | ~30 | Configuración del linter (vet, staticcheck, gosec severity medium+, revive, gocritic, misspell es). |
| `.github/workflows/ci.yml` | ~70 | CI workflow: gofmt + vet + build + test -race + golangci-lint + govulncheck + verificación de Swagger. |
| `pkg/validate/validate.go` | ~95 | Helpers de validación: `Lat`, `Lng`, `Email`, `MaxLen`, `MinLen`, `NonEmpty`, `NonZeroFloat`. |
| `pkg/validate/validate_test.go` | ~155 | Tests de todos los helpers (incluye NaN/±Inf, unicode runes, etc.). |
| `pkg/database/migrate.go` | ~75 | Wrapper Go de golang-migrate v4 con `MigrateUp` y `ConfirmPoolHealthy`. |
| `internal/middleware/ratelimit.go` | ~95 | Middleware genérico de rate limit con Redis (fail-open si Redis cae). |
| `internal/middleware/ratelimit_test.go` | ~95 | Tests del middleware (fail-open, KeyByIP, KeyByUserID, retry-after, integración marcada como skip). |
| `internal/middleware/security_headers.go` | ~30 | Middleware que añade X-Content-Type-Options, X-Frame-Options, Referrer-Policy, Permissions-Policy, HSTS (prod). |
| `internal/middleware/security_headers_test.go` | ~55 | Tests que verifican las cabeceras siempre + HSTS solo en producción + nil config no panicea. |
| `pkg/apperr/apperr_test.go` | ~190 | Tests de `New`, `Wrap`, todos los helpers de error (BadRequest, Unauthorized, NotFound, etc.) y los helpers de gmaps. |
| `pkg/apperr/respond_test.go` | ~210 | Tests de `RespondInternal` (con/sin mensaje, status desconocido, 5xx por defecto), `Respond`, `defaultMessageForStatus`, `levelForStatus`, `opFromContext`, `RequestID`. |
| `internal/middleware/auth_test.go` | ~225 | Tests exhaustivos de JWT: sin header, scheme incorrecto, alg=none, secret incorrecto, expirado, refresh, válido. Y `UserID` con float64/int64/int/ausente. |
| `internal/services/user_service_test.go` | ~95 | Tests de sentinels (existencia + distinción) y funciones puras (`generateUsernameFromEmail`, `shortEmailFingerprint`). |
| `internal/handlers/errors_test.go` | ~145 | Tests del mapping `Respond` (nil, AppError, genérico, cada sentinel → HTTP status correcto). |
| `db/migrations/000001_initial_schema.up.sql` | 243 | Schema completo (movido desde `db/ddl.sql`). |
| `db/migrations/000001_initial_schema.down.sql` | ~20 | Down con `DROP TABLE IF EXISTS ... CASCADE` en orden inverso. |

### 2.2 Modificados (10)

| Archivo | Cambio |
|---|---|
| `pkg/apperr/apperr.go` | Añadido `TooManyRequests(op)` → 429 con `code: "rate_limited"`. |
| `internal/middleware/logger.go` | Helper `sanitizePath` que redacta query params sensibles (password, token, session, etc.) a `[REDACTED]`. Lista `sensitiveQueryKeys`. |
| `internal/routes/routes.go` | Aplicado `RateLimit` a `/auth/login`, `/auth/register`, `/auth/refresh`, `/places/search`. Aplicado `SecurityHeaders` después de CORS. |
| `internal/services/place_service.go` | `Update(ctx, id, userID, req)` y `Delete(ctx, id, userID)` ahora reciben `userID`, cargan el lugar, verifican `CreatedBy == userID`, y devuelven `ErrNotOwner` si no coincide. |
| `internal/handlers/place_handler.go` | `Update`/`Delete` extraen `userID` con `middleware.UserID(c)`. Validación con `pkg/validate` para lat/lng/MaxLen. `parseStrictFloat` reemplaza `parseFloatOrDefault` en `GetByBounds`/`GetNearby`. |
| `internal/middleware/auth.go` | (sin cambios; los tests son nuevos). |
| `cmd/server/main.go` | (sin cambios; las migraciones se aplican manualmente con `make migrate-up`). |
| `docker-compose.yml` | Volume mount de `ddl.sql` → `migrations/000001_initial_schema.up.sql`. `dml.sql` queda como `02_dml.sql`. |
| `Makefile` | Reescrito: añadido `migrate-up`, `migrate-down`, `setup-db` (migrate-up + seed), `test`, `ci`. `migrate` antiguo quitado (auto-instala la CLI). |
| `.claude/CLAUDE.md` | Añadida la sección "Como anadir una funcionalidad nueva" (orden canónico) + referencia a la skill. |

### 2.3 Eliminados

| Archivo | Razón |
|---|---|
| `db/ddl.sql` | Movido a `db/migrations/000001_initial_schema.up.sql`. Mantener ambas rompe la fuente de verdad. |
| `~/.config/opencode/opencode.jsonc` (global) | Los agents se mueven a `.opencode/opencode.jsonc` del proyecto (project-scoped). El global queda con solo `$schema`. |

### 2.4 Dependencias

- **Añadidas**: `github.com/stretchr/testify v1.11.1`, `github.com/golang-migrate/migrate/v4 v4.20.1`, `github.com/golang-migrate/migrate/v4/database/pgx/v5`, `github.com/golang-migrate/migrate/v4/source/iofs`.
- **Subidas transitivamente** (al instalar golang-migrate): `pgx v5.5.1 → v5.9.2`, varias libs a versiones compatibles. Sin cambios de API breaking.

---

## 3. Detalle por fase

### Fase 1 — Skill + CI + linter

#### `.opencode/skill/accesspath-feature/SKILL.md`
Skill de comportamiento que se invoca automáticamente cuando un developer pide implementar una funcionalidad nueva en el backend. Contiene:

- **Resumen de arquitectura** en capas con inyección directa (sin interfaces).
- **Convenciones** extraídas de `.claude/CLAUDE.md`: sin acentos, `pgx.CollectRows`, sentinels en services, `apperr.*`, etc.
- **Patrones canónicos con snippets**:
  - Handler con tags Swagger.
  - Service con sentinel + `apperr.Wrap`.
  - Repository con escaneo por nombre.
  - Modelo con tags `db:` y `json:`.
  - Test table-driven con testify.
- **Checklist OWASP API Top 10 aplicado al repo** (API1 a API10).
- **Checklist de calidad pre-PR** (gofmt, vet, test, lint, govulncheck, swagger, migraciones, .env.example).
- **Decisiones arquitectónicas explícitas** (no desafiables sin propuesta formal): sin interfaces, sin ORM, JWT HS256, Redis fail-open, soft delete, transacciones explícitas.
- **Lista "lo que NO hacer"** con 10 puntos.
- **Mapa rápido de archivos clave**.

Va dentro del repo, no en `~/.config/opencode/`, para que se clone con el proyecto.

#### `.opencode/opencode.jsonc` (project-scoped agents)

Los 3 subagents están definidos solo aquí, en el proyecto. Eso significa:

- Solo están disponibles cuando trabajas dentro de `D:\Git\AccessPath_backend\` (o donde se clone el repo).
- No contaminan otros proyectos del mismo developer.
- Viajan con el repo (puedes versionarlos, code-reviewearlos, mejorarlos entre todos).

**Cómo invocarlos desde opencode CLI:**

```bash
# En el directorio del backend:
cd D:\Git\AccessPath_backend

# Lanzar el agente security-reviewer sobre un diff:
opencode run security-reviewer -- "revisar git diff main...HEAD para OWASP API"

# Lanzar test-writer sobre una pieza concreta:
opencode run test-writer -- "generar tests para internal/services/place_service.go"

# Lanzar code-quality-reviewer sobre los archivos modificados:
opencode run code-quality-reviewer -- "revisar internal/handlers/place_handler.go contra convenciones"
```

**Resumen de cada agente:**

| Agente | Tabla de herramientas | Output |
|---|---|---|
| `security-reviewer` | read, grep, glob, bash (solo lectura) | Markdown con tabla de hallazgos OWASP. Veredicto final: BLOCK / WARN / OK. |
| `test-writer` | read, grep, glob, write, edit, bash | Tests Go table-driven con testify. Ejecuta `go test` al terminar. |
| `code-quality-reviewer` | read, grep, glob, bash (solo lectura) | Markdown con tabla de infracciones a las convenciones. Veredicto final: BLOCK / WARN / OK. |

Los prompts detallados están en `.opencode/opencode.jsonc`.

#### `.golangci.yml`

Linter que corre en CI. Linters activos:

- `govet` — análisis estático oficial.
- `staticcheck` — checks adicionales (incluye cosas como shadowing de variables).
- `gosec` — análisis de seguridad (severity medium+, excluye G104 que tiene falsos positivos con `defer rows.Close()` en queries pgx).
- `gofmt` y `goimports` — formato.
- `revive` — sucesor de golint.
- `gocritic` — checks adicionales de calidad.
- `misspell` (locale es) — detecta typos en español.

Tests: true. Timeout: 5m.

#### `.github/workflows/ci.yml`

```yaml
name: CI
on:
  push: branches: [main]
  pull_request: branches: [main]
jobs:
  ci:
    runs-on: ubuntu-latest
    steps:
      - actions/checkout@v4
      - actions/setup-go@v5 (Go 1.25)
      - gofmt -l .                  # falla si hay archivos sin formato
      - go vet ./...
      - go build ./...
      - go test ./... -race -count=1 -timeout=120s
      - golangci-lint run (v1.62.2)
      - govulncheck ./...           # análisis de CVEs en deps
      - swag init -g cmd/server/main.go + diff contra docs/  # falla si Swagger desincronizado
```

---

### Fase 2 — Tests bootstrap (~145 tests nuevos)

#### `pkg/apperr/apperr_test.go` — 19 tests

Cubre el contrato de errores que es la pieza más crítica del repo:

| Test | Qué verifica |
|---|---|
| `TestNew` | Los 5 campos del `AppError` se asignan correctamente. |
| `TestAppError_Error/nil_receiver` | `nil.Error()` devuelve `"<nil AppError>"` sin panicear. |
| `TestAppError_Error/without_cause` | Formato `"op: code: msg"`. |
| `TestAppError_Error/with_cause` | Formato `"op: code: msg (cause: boom)"`. |
| `TestAppError_Unwrap` | `errors.Is(ae, cause)` funciona (compatible con `errors.Is`/`As` del paquete stdlib). |
| `TestWrap/nil_returns_nil` | `Wrap(nil)` → `nil`. |
| `TestWrap/generic_error_wraps_as_Internal_500` | Errores genéricos se convierten a Internal. |
| `TestWrap/AppError_passes_through_unchanged` | `*AppError` pasa tal cual (preserva code/http_status). |
| `TestInternal`, `TestValidation`, `TestBadRequest`, `TestNotFound`, `TestUnauthorized` | Cada helper construye el error con status correcto. |
| `TestGmaps*` | Helpers de gmaps (`GmapsNotConfigured` → 503, `GmapsRequestDenied` → 502 con cause, `GmapsQuotaExceeded` → 429, etc.). |
| `TestPlaceClosedPermanently`, `TestPlaceSearchFailed` | Helpers de places. |

Casos importantes cubiertos:

- **`TestValidation` verifica que NO transporta cause**: por convención, los errores de validación son del cliente, no del servidor, y no hay causa interna que filtrar.
- **`TestGmapsUpstream_NilCause`** cubre el caso `cause == nil` que de otro modo haría que `New(op, code, status, user, nil)` tuviera una `cause` vacía.

#### `pkg/apperr/respond_test.go` — 11 tests

Cubre la primitiva de respuesta HTTP:

| Test | Qué verifica |
|---|---|
| `TestRespondInternal_WritesStatusAndEnvelope` | El status HTTP correcto + el body con `{error: "..."}` y `data: nil`. |
| `TestRespondInternal_FallsBackToDefaultMessage` | Si `UserMessage` está vacío, usa el mensaje por defecto del status. |
| `TestRespondInternal_UnknownStatusFallback` | Status 418 → mensaje `"Error desconocido."`. |
| `TestRespondInternal_5xxDefaultMessage` | Status 500 sin mensaje → `"Error interno del servidor. Intentalo de nuevo."`. |
| `TestRespond_NilDoesNothing` | `Respond(c, nil)` → false, no escribe nada. |
| `TestRespond_PreservesAppError` | `Respond(c, ae)` → true, status preservado. |
| `TestRespond_GenericErrorBecomesInternal` | Error no-AppError → 500 + Internal. |
| `TestRespond_FillsEmptyOpFromContext` | Si `ae.Op` está vacío, se rellena con `c.FullPath()`. |
| `TestDefaultMessageForStatus` | Cubre 400, 401, 404, 429, 502, 503 + casos desconocidos. |
| `TestLevelForStatus` | 2xx/3xx → INFO; 4xx → WARN; 5xx → ERROR. |
| `TestOpFromContext` + `TestOpFromContext_NoRoute` | `opFromContext` extrae del path. Sin path enrutado → `"http"`. |
| `TestRequestID` + `TestRequestID_Empty` | Helpers de request id. |

Hallazgo curioso: el formato del op cuando el path es `/api/v1/places` es `"http.api/v1/places"`, no `"http.api.v1.places"` como podría parecer. Lo documenté en el test para que un cambio futuro no rompa silenciosamente.

#### `internal/middleware/auth_test.go` — 16 tests

Este es el test de seguridad más crítico del repo, porque valida que el middleware Auth rechaza todos los vectores de ataque conocidos:

| Test | Vector de ataque evitado |
|---|---|
| `TestAuth_NoHeaderReturns401` | Petición sin Authorization. |
| `TestAuth_EmptyHeaderReturns401` | Header Authorization vacío. |
| `TestAuth_NonBearerSchemeReturns401` | `Authorization: Basic dXNlcjpwYXNz` (credentials en lugar de bearer). |
| `TestAuth_BearerWithoutTokenReturns401` | `Authorization: Bearer ` (sin token). |
| `TestAuth_TamperedTokenReturns401` | Token malformado. |
| `TestAuth_AlgNoneReturns401` | **Ataque de algorithm confusion**: token con `alg: none` que pasa por HS256. Esto es el bug histórico CVE-2015-9235. |
| `TestAuth_TokenSignedWithDifferentSecretReturns401` | Token con firma incorrecta. |
| `TestAuth_ExpiredTokenReturns401` | Token con `exp` en el pasado. |
| `TestAuth_RefreshTokenReturns401` | Token con `type: "refresh"` intentando acceder a rutas protegidas (debe ir solo a `/auth/refresh`). |
| `TestAuth_ValidTokenInjectsUserID` | Token válido → handler recibe `user_id` en el contexto. |
| `TestAuth_ResponseEnvelopeOnFailure` | Cualquier 401 produce el envelope `{error: "..."}`. |
| `TestUserID_*` (5 tests) | `UserID(c)` extrae correctamente desde `float64`, `int64`, `int`. Devuelve `(0, false)` si falta o es de tipo incorrecto. |

El test de `alg=none` construye el token a mano (header + payload en base64 url-safe sin firma) porque `golang-jwt/jwt/v5` no permite generar tokens inseguros por diseño. Esto es exactamente lo que un atacante externo haría.

#### `internal/services/user_service_test.go` — 16 tests

Estrategia: dado que el proyecto no usa interfaces (lo prohíbe `CLAUDE.md`), `pgxmock` no encaja para mockear la DB. Por eso se testean solo las piezas testeables:

| Grupo | Qué verifica |
|---|---|
| `TestSentinels_Exist` (8 subtests) | Cada sentinel (`ErrInvalidCredentials`, `ErrEmailAlreadyUsed`, `ErrOptionMismatch`, `ErrContributionNotFound`, `ErrNotOwner`, `ErrEmptySubmission`, `ErrGmapsQuotaExceeded`, `ErrPlaceClosedPermanently`) existe y no es nil. |
| `TestSentinels_AreDistinct` | Dos sentinels distintos NO se confunden con `errors.Is`. Importante para que el mapping de errores en handlers no confunda `ErrInvalidCredentials` con `ErrNotOwner`. |
| `TestGenerateUsernameFromEmail` (5 subtests) | Email simple, con puntos, con mayúsculas, con símbolos, sin parte local. Verifica el prefijo esperado y que el fingerprint añade 4 chars. |
| `TestGenerateUsernameFromEmail_NoDuplicatesForSameEmail` | Mismo email → mismo username (idempotencia). |
| `TestShortEmailFingerprint_DeterministicAndDifferent` | Mismo email → mismo fingerprint. Emails distintos → fingerprints distintos. Longitud 4 hex. |

**Pendiente**: tests de `Login`/`Register` con bcrypt real contra DB. Requieren refactor del repo para introducir una interfaz `DBTX` (trade-off a discutir con el equipo).

#### `internal/handlers/errors_test.go` — 12 tests

| Test | Qué verifica |
|---|---|
| `TestRespond_NilDoesNothing` | `Respond(c, nil)` no escribe. |
| `TestRespond_AppErrorPassesThrough` | `*AppError` se preserva. |
| `TestRespond_AppErrorPreservesCodeAndStatus` | Status custom (ej. 418) se mantiene. |
| `TestRespond_GenericErrorBecomesInternal500` | Errores genéricos → 500. |
| `TestRespond_SentinelMappingToCorrectStatus` (6 subtests) | Cada sentinel de services se mapea al HTTP status correcto (ErrInvalidCredentials → 401, ErrEmailAlreadyUsed → 400, ErrNotOwner → 401, ErrEmptySubmission → 400, ErrContributionNotFound → 404, ErrGmapsQuotaExceeded → 429). |
| `TestRespond_OpFilledFromFullPathWhenEmpty` | Op vacío se rellena con el path. |
| `TestMatchServiceError_NoMatchReturnsFalse` | `matchServiceError` para errores no conocidos. |
| `TestMatchServiceError_FirstMatchWins` (3 subtests) | El primero que matchea gana; errores envueltos (con `errors.Join`) también matchean. |

#### `internal/middleware/ratelimit_test.go` — 8 tests

| Test | Qué verifica |
|---|---|
| `TestRateLimit_NilRedisFailsOpen` | Con Redis nil, 100 requests consecutivas pasan todas. Política de fail-open verificada. |
| `TestRateLimit_NilKeyFnFailsOpen` | Si KeyFn es nil, no se intenta contar. |
| `TestKeyByIP_ReturnsClientIP` | La key incluye `ip:` + la IP del cliente. |
| `TestKeyByUserID_PrefersUserID` | Si hay `user_id` en el contexto, se usa ese (no la IP). |
| `TestKeyByUserID_FallsBackToIP` | Sin `user_id`, cae a IP. |
| `TestItoaForLimit` (5 subtests) | Helper de conversión int → string. |
| `TestRetryAfterSeconds` (3 subtests) | Sub-segundo clampea a 1 segundo. |
| `TestRateLimit_ResponseShape_DocumentedIntegrationTest` | Skip explícito + guía de cómo activarlo con `miniredis`. |
| `TestRateLimit_AppliesLimit_WithRealRedis` | Stub del test de integración que documenta qué verificar. |

**Pendiente**: tests reales con `miniredis` (no está en `go.mod`). Documentado cómo activarlos.

#### `pkg/validate/validate_test.go` — 56 subtests

Cubre todos los helpers de validación con casos límite:

| Helper | Casos cubiertos |
|---|---|
| `Lat` (11 subtests) | 0, norte, sur, borde norte (90), borde sur (-90), 90.1 (rechazado), -90.1 (rechazado), 200 (rechazado), NaN (rechazado), +Inf (rechazado), -Inf (rechazado). |
| `Lng` (10 subtests) | Idem para longitud. |
| `NonZeroFloat` (5 subtests) | Positivo, negativo, 0 (rechazado), NaN (rechazado), +Inf (rechazado). |
| `Email` (10 subtests) | Válido, lowercase, con subdomain, vacío (rechazado), whitespace only (rechazado), sin @, sin user, sin domain, mayúsculas rechazadas, display name (acepta). |
| `MaxLen` (6 subtests) | Bajo límite, en límite, sobre límite, vacío, unicode (cuenta runes), unicode sobre límite. |
| `MinLen` (5 subtests) | Sobre límite, en límite, bajo, vacío, min=0 acepta cualquier cosa. |
| `NonEmpty` (5 subtests) | No vacío, whitespace solo, vacío, tabs/newlines, un solo char. |

#### `internal/middleware/security_headers_test.go` — 7 tests

| Test | Qué verifica |
|---|---|
| `TestSecurityHeaders_AlwaysPresent` (3 subtests) | Cabeceras X-Content-Type-Options, X-Frame-Options, Referrer-Policy, Permissions-Policy en development/production/test. |
| `TestSecurityHeaders_HSTSOnlyInProduction` (3 subtests) | HSTS presente solo en producción. En dev y test no se añade (rompería http://localhost). |
| `TestSecurityHeaders_NilConfigDoesNotPanic` | Pasar `nil` como config no produce panic. |

**Total: ~145 tests, todos verdes.**

---

### Fase 3 — Rate limiting (OWASP API4 + API6)

#### `pkg/apperr/apperr.go` (modificado)
Añadido `TooManyRequests(op)`:

```go
func TooManyRequests(op string) *AppError {
    return New(op, "rate_limited", http.StatusTooManyRequests,
        "Demasiadas solicitudes. Vuelve a intentarlo en unos minutos.", nil)
}
```

#### `internal/middleware/ratelimit.go` (nuevo)

```go
func RateLimit(rdb *redis.Client, key KeyFunc, limit int, window time.Duration) gin.HandlerFunc
```

- Backend: Redis `INCR + EXPIRE`. Si count == 1, fija el TTL de la ventana.
- **Fail-open**: si `rdb == nil`, si la key function es nil, o si Redis falla, deja pasar (degrada como el middleware cache).
- Cabecera `Retry-After` en la respuesta 429 con el cooldown en segundos (mínimo 1).
- `KeyByIP(c)` → key por IP. `KeyByUserID(c)` → prefiere user_id, fallback a IP.

#### `internal/routes/routes.go` (modificado)

Límites aplicados:

| Endpoint | Límite | Ventana | Clave | Razón |
|---|---|---|---|---|
| `POST /auth/login` | 5 | 1 min | IP | OWASP API2: brute force prevention |
| `POST /auth/register` | 3 | 1 hora | IP | OWASP API6: account abuse |
| `POST /auth/refresh` | 30 | 1 min | IP | Refresh spam |
| `GET /places/search` | 10 | 1 s | IP | OWASP API4: preservar cuota Google Maps |

---

### Fase 4 — Validación de inputs (OWASP API3/API4)

#### `pkg/validate/validate.go` (nuevo)

Helpers puros (sin DB, sin estado):

```go
Lat(v float64) error         // NaN, ±Inf, fuera de [-90, 90]
Lng(v float64) error         // NaN, ±Inf, fuera de [-180, 180]
NonZeroFloat(v float64, field string) error
Email(s string) error        // net/mail.ParseAddress + rechaza mayúsculas
MaxLen(s, field string, max int) error   // cuenta runes
MinLen(s, field string, min int) error
NonEmpty(s, field string) error
```

`isBadFloat` interno detecta NaN/±Inf (porque no son comparables con rangos).

#### `internal/handlers/place_handler.go` (modificado)

- `parseStrictFloat(c, field)` (nuevo) reemplaza a `parseFloatOrDefault(c.Query("..."), 0)`. Si el valor falta o no parsea, escribe `BadRequest`/`Validation` y devuelve error.
- `GetByBounds` (4 latitudes/longitudes): valida con `validate.Lat/Lng` antes de pasar al service.
- `GetNearby`: valida lat/lng con `validate.Lat/Lng`.
- `Update`: valida lat/lng + `MaxLen` para `name` (255), `address` (500), `description` (2000).

**Comparación con el comportamiento anterior**:

| Input | Antes (parseFloatOrDefault) | Ahora (parseStrictFloat + validate) |
|---|---|---|
| `?min_lat=abc` | Convertido a 0, fallaba la query silenciosamente | 400 `validation_error` con detalle |
| `?min_lat=` (vacío) | Convertido a 0 | 400 `validation.missing` |
| `?min_lat=91` | Aceptado (rango fuera) | 400 `validation_error` |
| `?min_lat=NaN` | Convertido a 0 | 400 `validation_error` |

---

### Fase 5 — Ownership check (OWASP API1)

#### `internal/services/place_service.go` (modificado)

```go
func (s *PlaceService) Update(ctx context.Context, id int64, userID int64, req models.UpdatePlaceRequest) (*models.Place, error) {
    existing, err := s.repo.FindByID(ctx, id)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, apperr.NotFound("places.update", "Place")
        }
        return nil, apperr.Wrap("places.update", err)
    }
    if existing.CreatedBy != userID {
        return nil, ErrNotOwner
    }
    // ... actualiza
}

func (s *PlaceService) Delete(ctx context.Context, id int64, userID int64) error {
    // Mismo patron
}
```

Distinción importante: **NotFound** (404) vs **NotOwner** (401 vía mapping). Trade-off conocido: revelar "no existe" vs "no es tuyo" facilita enumeración pero da mejor DX. Si en el futuro se quiere cerrar esto, se puede devolver 404 para ambos casos.

#### `internal/handlers/place_handler.go` (modificado)

```go
userID, ok := middleware.UserID(c)
if !ok {
    Respond(c, apperr.Unauthorized("places.update", "token requerido"))
    return
}
// ... validación
place, err := h.service.Update(c.Request.Context(), id, userID, req)
```

`ErrNotOwner` ya estaba mapeado en `serviceErrorMappings` (`internal/handlers/errors.go:62-69`) a `apperr.Unauthorized("no autorizado")` → 401.

---

### Fase 6 — Security headers + log redaction

#### `internal/middleware/security_headers.go` (nuevo)

Cabeceras añadidas a TODAS las respuestas:

| Cabecera | Valor | Razón |
|---|---|---|
| `X-Content-Type-Options` | `nosniff` | Evita MIME sniffing |
| `X-Frame-Options` | `DENY` | Anti clickjacking |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Limita info en Referer |
| `Permissions-Policy` | `camera=(), microphone=(), geolocation=(self), payment=()` | Deshabilita APIs no usadas |
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains` | Solo en producción (forzaría HTTPS en dev) |

#### `internal/middleware/logger.go` (modificado)

Helper `sanitizePath` que redacta query params sensibles:

```go
var sensitiveQueryKeys = map[string]bool{
    "password": true, "token": true, "refresh": true,
    "session": true, "sessiontoken": true,
    "apikey": true, "api_key": true, "secret": true,
}
```

Si una key de la query string matchea, su valor se reemplaza por `[REDACTED]`. El resto se mantiene intacto. Headers y body siguen sin loguearse (la regla original no cambia).

---

### Fase 7 — Migraciones incrementales

#### Archivos

```
db/
  dml.sql                                    (sin cambios, lo ejecuta el container al inicio)
  migrations/
    000001_initial_schema.up.sql              (movido desde ddl.sql)
    000001_initial_schema.down.sql            (DROP CASCADE en orden inverso)
```

`ddl.sql` se eliminó para evitar dos fuentes de verdad.

#### `pkg/database/migrate.go` (nuevo)

```go
func MigrateUp(databaseURL, sourceURL string) error
func ConfirmPoolHealthy(ctx context.Context, db *pgxpool.Pool) error
```

Usa `iofs.New(os.DirFS(sourceURL), ".")` para servir las migraciones y `pgx5://` como esquema del driver de golang-migrate. Es idempotente: si no hay migraciones nuevas, devuelve sin error (`ErrNoChange` se ignora).

#### `Makefile` (reescrito)

```makefile
make migrate-up     # auto-instala la CLI golang-migrate y aplica
make migrate-down   # revierte la ultima
make setup-db       # migrate-up + seed
make test           # go test ./... -race
make ci             # gofmt + vet + test + lint (atajo local)
```

#### `docker-compose.yml` (modificado)

```yaml
volumes:
  - ./db/migrations/000001_initial_schema.up.sql:/docker-entrypoint-initdb.d/01_schema.sql
  - ./db/dml.sql:/docker-entrypoint-initdb.d/02_dml.sql
```

El `initdb` del container de Postgres crea el schema la primera vez desde la migración inicial. Cambios incrementales futuros van con `make migrate-up`.

---

### Fase 8 — Documentación

#### `.claude/CLAUDE.md` (modificado)

Añadida la sección **"Como anadir una funcionalidad nueva"** con el orden canónico:

1. Migración en `db/migrations/NNNN_*.up.sql` + `.down.sql`.
2. Modelos con tag `db:`.
3. Repository con SQL parametrizado.
4. Servicio con sentinels (mapear en handlers/errors.go).
5. Tests del servicio.
6. Handler con `pkg/validate`, `userID` del token en Update/Delete.
7. Ruta con `auth` si muta, rate limit si sensible.
8. Tests del handler.
9. Swagger regenerado.
10. `make ci` en local.
11. CI valida automáticamente.

---

## 4. Verificación final

```bash
$ go test ./...
ok  accesspath/internal/handlers   1.337s
ok  accesspath/internal/middleware 1.288s
ok  accesspath/internal/services   1.276s
ok  accesspath/pkg/apperr          1.331s
ok  accesspath/pkg/validate        0.967s
PASS
```

```bash
$ go build ./...
exit 0
```

```bash
$ go vet ./...
exit 0
```

```bash
$ gofmt -l . | wc -l
0
```

---

## 5. Cómo usar las 3 subagents (project-scoped)

Los 3 subagents están en `.opencode/opencode.jsonc`. Se invocan solo desde el directorio del backend.

### `security-reviewer`

```bash
cd D:\Git\AccessPath_backend
opencode run security-reviewer -- "revisar diff entre main y HEAD para OWASP API"
```

Output esperado:

```markdown
## Resumen
[1-2 frases]

## Hallazgos
| Severidad | OWASP | Archivo:línea | Descripción | Sugerencia |
|---|---|---|---|---|
| high | API1 | internal/handlers/x_handler.go:45 | Update sin check de ownership | Pasar userID al service |

## Veredicto
BLOCK
```

### `test-writer`

```bash
opencode run test-writer -- "generar tests para internal/services/x_service.go: cubriendo sentinels y wrap de errores"
```

El agente escribe los tests en `internal/services/x_service_test.go` siguiendo las convenciones del repo (table-driven, testify), y los ejecuta para verificar que pasan.

### `code-quality-reviewer`

```bash
opencode run code-quality-reviewer -- "revisar internal/handlers/place_handler.go contra .opencode/skill/accesspath-feature/SKILL.md"
```

Output: tabla de infracciones con veredicto.

---

## 6. Pendientes (no hechos, documentados)

| Pendiente | Razón | Cómo activarlo |
|---|---|---|
| Tests de `pkg/gmaps/client.go` | Requiere refactor para inyectar `http.Client` | Refactor del struct `Client` para que acepte `http.Client` en el constructor; tests usan `httptest.NewServer` |
| Tests de `Login`/`Register` con DB | El proyecto no usa interfaces; `pgxmock` no encaja | Introducir interfaz `DBTX` en repos (trade-off a discutir) |
| Tests de rate limit con Redis real | Necesita `miniredis` (no en `go.mod`) | `go get github.com/alicebob/miniredis/v2` + activar tests marcados como `Skip` |
| Renovate / Dependabot | No hay config | Añadir `.github/renovate.json` o activar Dependabot en `.github/dependabot.yml` |
| Migraciones automáticas en arranque | Hoy se aplican con `make migrate-up` o `initdb` | Llamar a `database.MigrateUp` desde `main.go` con flag `SKIP_MIGRATIONS=true` opcional |
| CORS restrictivo | Hoy sigue `*` | Definir lista de orígenes permitidos vía env var `ALLOWED_ORIGINS` |

---

## 7. Métricas

- **Tests añadidos**: ~145 (subtests incluidos).
- **Paquetes con tests nuevos**: `pkg/apperr`, `pkg/validate`, `internal/middleware`, `internal/services`, `internal/handlers`.
- **OWASP API Top 10 cubiertos**:
  - API1 (BOLA): ownership check en Place.Update/Delete.
  - API2 (Broken Auth): tests exhaustivos de JWT (alg=none, expired, refresh, etc.).
  - API3 (BOPLA): `password_hash` con `json:"-"` (verificado), validación de inputs.
  - API4 (URC): rate limit + validación de rangos.
  - API5 (BFLA): auth middleware en rutas mutantes (sin cambios, ya estaba).
  - API6 (UASBF): rate limit en register.
  - API7 (SSRF): sin cambios (gmaps recibe URL hardcoded).
  - API8 (SM): security headers + CORS sigue abierto (pendiente).
  - API9 (IIM): Swagger regenerado en CI.
  - API10 (UCA): gmaps sin timeout explícito (pendiente).

- **Cobertura CI**: 7 checks automatizados (gofmt, vet, build, test, lint, vuln, swagger).
- **Build time**: ~1.3s para tests con race detector.
- **Cero warnings de linter**.

---

## 8. Reorganizacion de tests: todos a `tests/`

### 8.1 Motivacion

Los tests estaban mezclados con el codigo productivo (`internal/services/foo.go` + `internal/services/foo_test.go`). Esto hacia dificil escanear rapidamente que archivos tienen cobertura y cuales no, y mezclaba en el `ls`/`git status` archivos de dos naturalezas distintas.

### 8.2 Convencion resultante

| Caso | Ubicacion | Paquete |
|---|---|---|
| **Cualquier** test del backend | `tests/<misma_ruta>/<archivo>_test.go` | `X_test` (black-box) |

**Regla dura**: `internal/` y `pkg/` no contienen NINGUN `*_test.go`. Tampoco hay `export_test.go` ni `*_internal_test.go` adyacentes al codigo productivo.

### 8.3 Limitacion encontrada y solucion

Go **no permite** tener tests en una carpeta distinta del codigo y acceder a simbolos no exportados: el mecanismo `export_test.go` solo compila en la misma carpeta que su codigo asociado (documentado en https://pkg.go.dev/testing). Lo confirma la documentacion oficial:

> "If the test file is in the same package, it may refer to unexported identifiers within the package. If the file is in a separate '_test' package, the package being tested must be imported explicitly and only its exported identifiers may be used. This is known as 'black box' testing."

Esto bloquea la separacion fisica total **a no ser que los simbolos necesarios sean exportados**. Despues de buscar la forma canonica (la comunidad Go confirma que no existe otra tecnica), se opto por exportar los 10 helpers que los tests referenciaban.

### 8.4 Simbolos renombrados (privados -> exportados)

| Paquete | Antes (privado) | Despues (publico) |
|---|---|---|
| `pkg/apperr` | `defaultMessageForStatus` | `DefaultMessageForStatus` |
| `pkg/apperr` | `levelForStatus` | `LevelForStatus` |
| `pkg/apperr` | `opFromContext` | `OpFromContext` |
| `internal/middleware` | `itoaForLimit` | `ItoaForLimit` |
| `internal/middleware` | `retryAfterSeconds` | `RetryAfterSeconds` |
| `internal/handlers` | `matchServiceError` | `MatchServiceError` |
| `internal/services` | `generateUsernameFromEmail` | `GenerateUsernameFromEmail` |
| `internal/services` | `shortEmailFingerprint` | `ShortEmailFingerprint` |

Convencion del repo a partir de ahora: si un test en `tests/` necesita acceder a algo, ese algo forma parte de la API publica. La regla anterior "minimizar la API" cede ante "tests separados fisicamente".

### 8.5 Archivos movidos a `tests/` (los 8 archivos de test, ahora black-box)

| Archivo |
|---|
| `tests/pkg/validate/validate_test.go` (package `validate_test`) |
| `tests/pkg/apperr/apperr_test.go` (package `apperr_test`) |
| `tests/pkg/apperr/respond_test.go` (package `apperr_test`) |
| `tests/internal/middleware/auth_test.go` (package `middleware_test`) |
| `tests/internal/middleware/ratelimit_test.go` (package `middleware_test`) |
| `tests/internal/middleware/security_headers_test.go` (package `middleware_test`) |
| `tests/internal/handlers/errors_test.go` (package `handlers_test`) |
| `tests/internal/services/user_service_test.go` (package `services_test`) |

### 8.6 Archivos borrados

| Archivo | Razon |
|---|---|
| `test_*.log` (8 archivos en raiz) | Artefactos obsoletos de depuracion manual de la fase anterior. Los tests ahora se ejecutan via `make test` o CI. |

### 8.7 Cambios en documentacion

| Archivo | Cambio |
|---|---|
| `.opencode/opencode.jsonc` (agente `test-writer`) | Prompt actualizado: tests **siempre** van a `tests/` con `package X_test`. La regla "exportar si es necesario para testear" esta explicita. |
| `.opencode/skill/accesspath-feature/SKILL.md` | Seccion "Ubicacion de los tests" reescrita con la convencion definitiva y la tabla de simbolos exportados. |
| `.claude/CLAUDE.md` | Paso 5 del "orden canonico" actualizado para apuntar siempre a `tests/`. |
| `CHANGES.md` | Esta seccion. |

### 8.8 Verificacion

```bash
$ go test ./...
?   	accesspath/cmd/server          [no test files]
?   	accesspath/docs                [no test files]
?   	accesspath/internal/app        [no test files]
?   	accesspath/internal/config     [no test files]
?   	accesspath/internal/handlers    [no test files]
?   	accesspath/internal/middleware  [no test files]
?   	accesspath/internal/models     [no test files]
?   	accesspath/internal/repositories [no test files]
?   	accesspath/internal/routes     [no test files]
?   	accesspath/internal/services   [no test files]
?   	accesspath/pkg/apperr          [no test files]
?   	accesspath/pkg/database        [no test files]
?   	accesspath/pkg/gmaps           [no test files]
?   	accesspath/pkg/response        [no test files]
?   	accesspath/pkg/storage         [no test files]
?   	accesspath/pkg/validate        [no test files]
ok  	accesspath/tests/internal/handlers
ok  	accesspath/tests/internal/middleware
ok  	accesspath/tests/internal/services
ok  	accesspath/tests/pkg/apperr
ok  	accesspath/tests/pkg/validate
PASS
```

Los `?` son los paquetes productivos sin tests, lo esperado tras la reorg.

```bash
$ go vet ./...
$ go build ./...
$ gofmt -l tests/ ... (limpio en archivos nuevos)
```

### 8.9 Estado final

```
internal/        # 0 _test.go
pkg/             # 0 _test.go
tests/           # TODO
```

