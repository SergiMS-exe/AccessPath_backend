# CLAUDE.md — AccessPath Backend

Instrucciones para Claude Code al trabajar en este repositorio.
Para arquitectura detallada ver `.claude/PROJECT.md`.

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
