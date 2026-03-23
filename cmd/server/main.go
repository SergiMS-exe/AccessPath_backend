package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"accesspath/internal/app"
	"accesspath/internal/config"
	"accesspath/internal/routes"
	"accesspath/pkg/database"
	"accesspath/pkg/storage"
)

func main() {
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
	h := app.BuildHandlers(db, minioClient, cfg.MinioBucket)

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
