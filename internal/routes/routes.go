package routes

import (
	"accesspath/internal/handlers"

	"github.com/gin-gonic/gin"
)

func RegisterPlaceRoutes(r *gin.Engine, placeHandler *handlers.PlaceHandler, reviewHandler *handlers.ReviewHandler) {
	places := r.Group("/api/places")
	{
		places.GET("", placeHandler.GetAll)
		places.GET("/map", placeHandler.GetByBounds)
		places.GET("/nearby", placeHandler.GetNearby)
		places.GET("/:id", placeHandler.GetByID)
		places.POST("", placeHandler.Create)
		places.PUT("/:id", placeHandler.Update)
		places.DELETE("/:id", placeHandler.Delete)

		// Reviews nested under places
		places.GET("/:id/reviews", reviewHandler.GetByPlace)
		places.POST("/:id/reviews", reviewHandler.Create)
		places.GET("/:id/accessibility", reviewHandler.GetPlaceAccessibility)
	}
}

func RegisterFeatureRoutes(r *gin.Engine, featureHandler *handlers.FeatureHandler) {
	features := r.Group("/api/features")
	{
		features.GET("", featureHandler.GetAll)
		features.GET("/categories", featureHandler.GetCategories)
		features.GET("/:id", featureHandler.GetByID)
	}
}

func RegisterUserRoutes(r *gin.Engine, userHandler *handlers.UserHandler) {
	auth := r.Group("/api/auth")
	{
		auth.POST("/register", userHandler.Register)
		auth.POST("/login", userHandler.Login)
	}

	users := r.Group("/api/users")
	{
		users.GET("/:id", userHandler.GetProfile)
		users.GET("/:id/saved-places", userHandler.GetSavedPlaces)
		users.POST("/:id/saved-places/:placeId", userHandler.SavePlace)
		users.DELETE("/:id/saved-places/:placeId", userHandler.UnsavePlace)
	}
}

func RegisterHealthRoutes(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}
