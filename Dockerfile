# ---- Build stage ----
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Cache de capas
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/bin/server ./cmd/server

# ---- Runtime stage ----
FROM alpine:3.21
WORKDIR /app

# Certificados TLS para conexiones externas
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/bin/server .

# .env esta gitignored pero vive en el build context; godotenv lo lee al arrancar
COPY .env /app/.env

EXPOSE 8080
CMD ["./server"]
