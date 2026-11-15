package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
	Env         string

	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioUseSSL    bool
	// Base URL publica (resoluble desde el cliente movil) para servir las fotos,
	// p.ej. https://cdn.example.com via Caddy. Vacio => se usa el endpoint interno
	// de MinIO (solo valido en desarrollo local).
	MinioPublicBaseURL string

	// PlacesLinkTimeout y PlacesLinkMaxRedirects acotan la resolucion de
	// enlaces de Google Maps (timeout por peticion y saltos maximos).
	PlacesLinkTimeout      time.Duration
	PlacesLinkMaxRedirects int

	Accessibility *AccessibilityThresholds
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no se encontró .env, usando variables del sistema")
	}

	return &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://user:pass@localhost:5432/accesspath?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:   getEnv("JWT_SECRET", "cambia-esto-en-produccion"),
		Env:         getEnv("ENV", "development"),

		MinioEndpoint:      getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey:     getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey:     getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinioBucket:        getEnv("MINIO_BUCKET", "accesspath"),
		MinioUseSSL:        getEnv("MINIO_USE_SSL", "false") == "true",
		MinioPublicBaseURL: getEnv("MINIO_PUBLIC_BASE_URL", ""),

		PlacesLinkTimeout:      getEnvDuration("PLACES_LINK_TIMEOUT", 5*time.Second),
		PlacesLinkMaxRedirects: getEnvInt("PLACES_LINK_MAX_REDIRECTS", 3),

		Accessibility: LoadAccessibilityThresholds(getEnv("ACCESSIBILITY_CONFIG_PATH", "config/accessibility.json")),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return fallback
}
