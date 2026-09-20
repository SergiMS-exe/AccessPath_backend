// Package main is the entry point of the AccessPath API.
//
// @title           AccessPath API
// @version         1.0
// @description     API para gestión de lugares accesibles, reseñas, colecciones y categorías.
//
// @contact.name    AccessPath Team
//
// @host            localhost:8080
// @BasePath        /api/v1
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Ingresa el token JWT con el prefijo "Bearer ". Ejemplo: "Bearer eyJhbGci..."
package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "accesspath/docs"
	"accesspath/internal/app"
	"accesspath/internal/config"
	"accesspath/internal/routes"
	"accesspath/pkg/database"
	"accesspath/pkg/storage"
)

func main() {
	// 0. Configurar logger estructurado (slog).
	//    Dev  → texto con nivel, legible a ojo.
	//    Prod → JSON, una linea por evento, parseable por cualquier colector.
	initLogger(os.Getenv("APP_ENV") == "production")

	// 1. Cargar configuración
	cfg := config.Load()

	// 2. Conectar a PostgreSQL
	db, err := database.NewPostgresConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("no se pudo conectar a la base de datos: %v", err)
	}
	defer db.Close()
	log.Println("Conectado a PostgreSQL")

	// 3. Conectar a Redis (opcional, puede ser nil)
	cache := database.NewRedisClient(cfg.RedisURL)
	if cache != nil {
		defer cache.Close()
	}

	// 4. Conectar a MinIO
	minioClient, err := storage.NewMinioClient(
		cfg.MinioEndpoint,
		cfg.MinioAccessKey,
		cfg.MinioSecretKey,
		cfg.MinioUseSSL,
	)
	if err != nil {
		log.Fatalf("no se pudo conectar a MinIO: %v", err)
	}
	if err := storage.EnsureBucket(context.Background(), minioClient, cfg.MinioBucket); err != nil {
		log.Fatalf("error al verificar bucket MinIO: %v", err)
	}
	log.Println("Conectado a MinIO")

	// 5. Inicializar handlers (repos + servicios + handlers)
	h := app.BuildHandlers(db, minioClient, cfg)

	// 6. Montar router
	r := routes.Setup(h, cache, cfg)

	// 7. Configurar servidor HTTP
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 8. Arrancar en goroutine para graceful shutdown
	go func() {
		log.Printf("servidor arrancado en el puerto %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("error al arrancar el servidor: %v", err)
		}
	}()

	// 9. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("apagando servidor...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("error en el shutdown: %v", err)
	}

	log.Println("servidor apagado correctamente")
}

// initLogger configura el handler por defecto de slog. Ademas redirige la
// salida del package log estandar (log.Printf/log.Println) para que los
// modulos que aun no usan slog se integren en el mismo flujo.
func initLogger(json bool) {
	var h slog.Handler
	if json {
		h = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		h = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}
	logger := slog.New(h)
	slog.SetDefault(logger)

	// Redirige el package log estandar a slog, para no perder los logs de
	// codigo legacy (database, redis, minio, etc).
	slog.SetLogLoggerLevel(slog.LevelInfo)
	log.SetOutput(slogLogWriter{logger})
}

// slogLogWriter implementa io.Writer para que log.Printf pase por slog.
// Cada Write genera una linea a nivel INFO con msg=stdlib.
type slogLogWriter struct{ logger *slog.Logger }

func (w slogLogWriter) Write(p []byte) (int, error) {
	msg := string(p)
	// log.Printf/log.Println anaden \n al final; lo recortamos para no duplicar.
	if n := len(msg); n > 0 && msg[n-1] == '\n' {
		msg = msg[:n-1]
	}
	w.logger.Info("stdlib", slog.String("msg", msg))
	return len(p), nil
}
