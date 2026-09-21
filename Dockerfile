# Multi-stage build: Tailwind -> Go binary -> minimal Alpine runtime.
# Classic Docker builder syntax (no BuildKit required).

# ---------- 1. CSS (Tailwind v4 native) ----------
FROM node:22-alpine AS css-builder
WORKDIR /app

# Use the project's real package.json (Tailwind v4) — no more docker.css
# shim, no npm-init workaround, no tailwind.config.js plugin.
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund

COPY front/setup.css ./front/
COPY templates/ ./templates/
COPY main.go ./

RUN npx @tailwindcss/cli -i ./front/setup.css -o ./style/tailwind.css --minify

# ---------- 2. Go ----------
FROM golang:1.25-alpine AS go-builder
WORKDIR /app

# Download prebuilt templ + sqlc binaries (~5s) instead of `go install` (~240s).
RUN apk add --no-cache curl ca-certificates \
 && TEMPL_VERSION=v0.3.1020 \
 && SQLC_VERSION=v1.30.0 \
 && curl -fsSL "https://github.com/a-h/templ/releases/download/${TEMPL_VERSION}/templ_Linux_x86_64.tar.gz" \
      | tar xz -C /usr/local/bin templ \
 && curl -fsSL "https://github.com/sqlc-dev/sqlc/releases/download/${SQLC_VERSION}/sqlc_${SQLC_VERSION#v}_linux_amd64.tar.gz" \
      | tar xz -C /usr/local/bin sqlc \
 && chmod +x /usr/local/bin/templ /usr/local/bin/sqlc

# Cache modules separately so source-only edits skip the download step.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=css-builder /app/style/tailwind.css ./style/

ENV CGO_ENABLED=0 GOOS=linux
RUN sqlc generate \
 && templ generate \
 && go build -trimpath -ldflags="-s -w" -o main .

# ---------- 3. Runtime ----------
FROM alpine:3.21
RUN apk --no-cache add ca-certificates
COPY --from=go-builder /app/main /main
COPY --from=go-builder /app/style /style
COPY --from=go-builder /app/db /db

EXPOSE 8080
ENTRYPOINT ["/main"]
