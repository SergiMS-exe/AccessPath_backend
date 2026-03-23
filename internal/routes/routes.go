package routes

import (
	"accesspath/internal/config"
	"accesspath/internal/handlers"
	"accesspath/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type Handlers struct {
	Place      *handlers.PlaceHandler
	Category   *handlers.CategoryHandler
	Review     *handlers.ReviewHandler
	Collection *handlers.CollectionHandler
	User       *handlers.UserHandler
}

func Setup(h *Handlers, cache *redis.Client, cfg *config.Config) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")
	{
		// Auth - pública
		auth := v1.Group("/auth")
		{
			auth.POST("/register", h.User.Register)
			auth.POST("/login", h.User.Login)
		}

		// Users
		users := v1.Group("/users")
		{
			users.GET("/:id", h.User.GetProfile)
			users.GET("/:id/collections", h.Collection.GetByUser)
		}

		// Places — GET /:id returns PlaceDetail (place + rating cache)
		places := v1.Group("/places")
		{
			places.GET("", middleware.Cache(cache, "places"), h.Place.GetAll)
			places.GET("/map", h.Place.GetByBounds)
			places.GET("/nearby", middleware.Cache(cache, "nearby"), h.Place.GetNearby)
			places.GET("/:id", h.Place.GetByID)
			places.POST("", middleware.Auth(cfg.JWTSecret), h.Place.Create)
			places.PUT("/:id", middleware.Auth(cfg.JWTSecret), h.Place.Update)
			places.DELETE("/:id", middleware.Auth(cfg.JWTSecret), h.Place.Delete)

			places.GET("/:id/reviews", h.Review.GetByPlace)
		}

		// Reviews — POST creates review + ratings + photos in a single transaction
		reviews := v1.Group("/reviews")
		reviews.Use(middleware.Auth(cfg.JWTSecret))
		{
			reviews.POST("", h.Review.Create)
			reviews.DELETE("/:id", h.Review.Delete)
		}

		// Collections
		collections := v1.Group("/collections")
		collections.Use(middleware.Auth(cfg.JWTSecret))
		{
			collections.POST("", h.Collection.Create)
			collections.DELETE("/:id", h.Collection.Delete)
			collections.GET("/:id/places", h.Collection.GetPlaces)
			collections.POST("/:id/places/:placeId", h.Collection.AddPlace)
			collections.DELETE("/:id/places/:placeId", h.Collection.RemovePlace)
		}

		// Categories
		categories := v1.Group("/categories")
		{
			categories.GET("", h.Category.GetAllCategories)
			categories.GET("/:id", h.Category.GetCategoryByID)
			categories.POST("", middleware.Auth(cfg.JWTSecret), h.Category.CreateCategory)
			categories.GET("/:id/subcategories", h.Category.GetSubcategoriesByCategory)
			categories.GET("/subcategories", h.Category.GetAllSubcategories)
			categories.POST("/subcategories", middleware.Auth(cfg.JWTSecret), h.Category.CreateSubcategory)
		}
	}

	return r
}
