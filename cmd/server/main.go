package main

import (
	"log"

	"accesspath/internal/config"
	"accesspath/internal/handlers"
	"accesspath/internal/repositories"
	"accesspath/internal/routes"
	"accesspath/internal/services"
	"accesspath/pkg/database"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.NewPostgresConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Connected to database")

	// Initialize repositories
	placeRepo := repositories.NewPlaceRepository(db)
	featureRepo := repositories.NewFeatureRepository(db)
	reviewRepo := repositories.NewReviewRepository(db)
	userRepo := repositories.NewUserRepository(db)

	// Initialize services
	placeService := services.NewPlaceService(placeRepo)
	featureService := services.NewFeatureService(featureRepo)
	reviewService := services.NewReviewService(reviewRepo)
	userService := services.NewUserService(userRepo)

	// Initialize handlers
	placeHandler := handlers.NewPlaceHandler(placeService)
	featureHandler := handlers.NewFeatureHandler(featureService)
	reviewHandler := handlers.NewReviewHandler(reviewService)
	userHandler := handlers.NewUserHandler(userService)

	// Initialize Gin
	r := gin.Default()

	// Register routes
	routes.RegisterPlaceRoutes(r, placeHandler, reviewHandler)
	routes.RegisterFeatureRoutes(r, featureHandler)
	routes.RegisterUserRoutes(r, userHandler)
	routes.RegisterHealthRoutes(r)

	// Start server
	log.Printf("Server starting on port %s", cfg.Port)
	r.Run(":" + cfg.Port)
}
