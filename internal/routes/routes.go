package routes

import (
	"accesspath/internal/config"
	"accesspath/internal/handlers"
	"accesspath/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Handlers struct {
	Place        *handlers.PlaceHandler
	Catalog      *handlers.CatalogHandler
	Contribution *handlers.ContributionHandler
	Submission   *handlers.SubmissionHandler
	Profile      *handlers.ProfileHandler
	Collection   *handlers.CollectionHandler
	User         *handlers.UserHandler
}

func Setup(h *Handlers, cache *redis.Client, cfg *config.Config) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	auth := middleware.Auth(cfg.JWTSecret)

	v1 := r.Group("/api/v1")
	{
		// Auth - publica
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/register", h.User.Register)
			authGroup.POST("/login", h.User.Login)
			authGroup.POST("/refresh", h.User.Refresh)
		}

		// Users
		users := v1.Group("/users")
		{
			users.GET("/:id", h.User.GetProfile)
			users.GET("/:id/collections", h.Collection.GetByUser)
		}

		// Catalogo del formulario (reemplaza /categories)
		v1.GET("/dimensions", h.Catalog.GetDimensions)

		// Places
		places := v1.Group("/places")
		{
			places.GET("", middleware.Cache(cache, "places"), h.Place.GetAll)
			places.GET("/map", h.Place.GetByBounds)
			places.GET("/nearby", middleware.Cache(cache, "nearby"), h.Place.GetNearby)
			places.GET("/search", h.Place.Search)
			places.GET("/:id", h.Place.GetByID)
			places.POST("", auth, h.Place.Create)
			places.POST("/from-google", auth, h.Place.ImportFromGoogle)
			places.PUT("/:id", auth, h.Place.Update)
			places.DELETE("/:id", auth, h.Place.Delete)

			places.GET("/:id/submissions", h.Submission.GetByPlace)
			places.GET("/:id/next-question", auth, h.Contribution.NextQuestion)
		}

		// Contributions (contribucion atomica; user_id del token)
		contributions := v1.Group("/contributions")
		contributions.Use(auth)
		{
			contributions.POST("", h.Contribution.Create)
			contributions.DELETE("/:id", h.Contribution.Delete)
		}

		// Submissions (valoracion viva del usuario: comentario + fotos)
		submissions := v1.Group("/submissions")
		submissions.Use(auth)
		{
			submissions.PUT("", h.Submission.Save)
		}

		// Perfil funcional del usuario (opt-in)
		me := v1.Group("/me")
		me.Use(auth)
		{
			me.GET("/profile", h.Profile.Get)
			me.PUT("/profile", h.Profile.Set)
			me.DELETE("/profile", h.Profile.Delete)
		}

		// Collections
		collections := v1.Group("/collections")
		collections.Use(auth)
		{
			collections.POST("", h.Collection.Create)
			collections.DELETE("/:id", h.Collection.Delete)
			collections.GET("/:id/places", h.Collection.GetPlaces)
			collections.POST("/:id/places/:placeId", h.Collection.AddPlace)
			collections.DELETE("/:id/places/:placeId", h.Collection.RemovePlace)
		}
	}

	return r
}
