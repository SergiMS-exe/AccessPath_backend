# ---- Build stage ----
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Descargar dependencias primero (aprovecha caché de Docker)
COPY go.mod go.sum ./
RUN go mod download

# Copiar código fuente y compilar
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/server ./cmd/server

# ---- Runtime stage ----
FROM alpine:3.19

WORKDIR /app

# Certificados TLS para conexiones externas (PostgreSQL, etc.)
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/bin/server .

EXPOSE 8080

CMD ["./server"]
