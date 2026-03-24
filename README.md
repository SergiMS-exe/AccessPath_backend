# ♿ AccessPath

**Busca, descubre y valora lugares accesibles para personas con discapacidad.**

AccessPath es una plataforma de código abierto donde cualquier persona puede consultar la accesibilidad real de sitios públicos —restaurantes, museos, tiendas, transporte— y contribuir con valoraciones propias. El objetivo es construir, entre todos, un mapa de accesibilidad honesto y útil.

> 🌍 **Proyecto abierto.** Si quieres contribuir al código, alojar tu propia instancia o adaptar el proyecto a tu ciudad o región, eres bienvenido. Consulta la sección [Contribuir](#contribuir).

---

## Contenido

- [Tecnologías](#tecnologías)
- [Estructura del proyecto](#estructura-del-proyecto)
- [Puesta en marcha](#puesta-en-marcha)
  - [Requisitos](#requisitos)
  - [Modo desarrollo](#modo-desarrollo)
  - [Modo producción](#modo-producción)
- [Variables de entorno](#variables-de-entorno)
- [Contribuir](#contribuir)
- [Licencia](#licencia)

---

## Tecnologías

| Capa | Tecnología |
|---|---|
| Base de datos | PostgreSQL 16 |
| Caché | Redis 7 |
| Almacenamiento | MinIO (S3-compatible) |
| Proxy inverso | Caddy |
| Contenedores | Docker + Docker Compose |

---

## Estructura del proyecto

```
accesspath/
├── db/
│   ├── ddl.sql          # Esquema de la base de datos
│   └── dml.sql          # Datos iniciales
├── .env.example         # Plantilla de variables de entorno
├── docker-compose.yml
└── README.md
```

---

## Puesta en marcha

### Requisitos

- [Docker](https://docs.docker.com/get-docker/) y Docker Compose
- Git

```bash
git clone https://github.com/tu-usuario/accesspath.git
cd accesspath
```

---

### Modo desarrollo

Pensado para desarrollar la API en tu máquina local mientras la infraestructura (postgres, redis, minio) corre en un servidor o en local.

**1. Crea tu archivo de entorno:**

```bash
cp .env.example .env
```

**2. Ajusta las URLs en `.env`** apuntando a donde estén los servicios:

- Si los servicios corren en **tu propia máquina**: usa `localhost`
- Si corren en un **servidor de tu red local**: usa su IP (ej. `192.168.1.149`)

```env
DATABASE_URL=postgres://accesspath:pass@192.168.1.149:5432/accesspath_db?sslmode=disable
REDIS_URL=redis://192.168.1.149:6379
MINIO_ENDPOINT=192.168.1.149:9000
```

**3. Levanta solo la infraestructura** (sin la API, que correrá en tu máquina):

```bash
docker compose --env-file .env up -d
```

**4. Arranca la API localmente** con tu método habitual (`go run`, `npm run dev`, etc.).

---

### Modo producción

Todo corre en el servidor via Docker. La API se levanta con el perfil `full`.

**1. Crea el archivo de entorno de producción:**

```bash
cp .env.example .env
```

**2. Edita `.env`** con valores de producción. Usa los **nombres de servicio Docker** como hosts, no IPs:

```env
APP_ENV=production
DATABASE_URL=postgres://accesspath:pass@postgres:5432/accesspath_db?sslmode=disable
REDIS_URL=redis://redis:6379
MINIO_ENDPOINT=minio:9000
```

> ⚠️ Cambia **todas** las contraseñas y el `JWT_SECRET` por valores seguros y aleatorios.

**3. Levanta todo:**

```bash
docker compose --profile full up -d
```

---

## Variables de entorno

Copia `.env.example` a `.env` y rellena los valores. La tabla resume qué cambia entre entornos:

| Variable | Desarrollo | Producción |
|---|---|---|
| `APP_ENV` | `development` | `production` |
| `DATABASE_URL` | `...@<ip-servidor>:5432/...` | `...@postgres:5432/...` |
| `REDIS_URL` | `redis://<ip-servidor>:6379` | `redis://redis:6379` |
| `MINIO_ENDPOINT` | `<ip-servidor>:9000` | `minio:9000` |
| `JWT_SECRET` | cualquier string | **string largo y aleatorio** |

En desarrollo, los hosts son IPs o `localhost` porque la API corre fuera de Docker. En producción, son los nombres de servicio internos de Docker Compose porque todo corre dentro de la misma red.

---

## Contribuir

AccessPath está pensado para crecer con la comunidad. Hay varias formas de participar:

- 🐛 **Reportar bugs** — Abre un issue describiendo el problema
- 💡 **Proponer mejoras** — Nuevas categorías de accesibilidad, filtros, integraciones
- 🌐 **Alojar una instancia** — Despliégalo en tu ciudad o región y contribuye datos locales
- 🔧 **Contribuir código** — Haz un fork, crea una rama y abre un Pull Request

Por favor, lee el [Código de Conducta](CODE_OF_CONDUCT.md) antes de contribuir.

---

## Licencia

Este proyecto está bajo la licencia [MIT](LICENSE). Puedes usarlo, modificarlo y distribuirlo libremente.

---

<p align="center">Hecho con ❤️ para que el mundo sea un poco más accesible para todos.</p>