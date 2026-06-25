package app

import (
	"accesspath/internal/handlers"
	"accesspath/internal/repositories"
	"accesspath/internal/routes"
	"accesspath/internal/services"
	"accesspath/pkg/gmaps"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
)

func BuildHandlers(db *pgxpool.Pool, minioClient *minio.Client, minioBucket, jwtSecret, gmapsAPIKey string, gmapsMonthlyLimit int) *routes.Handlers {
	repos := repositories.New(db)

	ratingSvc := services.NewRatingService(repos.Rating)
	photoSvc  := services.NewPhotoService(minioClient, minioBucket)

	var gmapsClient *gmaps.Client
	if gmapsAPIKey != "" {
		gmapsClient = gmaps.New(gmapsAPIKey)
	}

	placeSvc      := services.NewPlaceService(repos.Place, ratingSvc, gmapsClient, repos.GmapsLog, gmapsMonthlyLimit)
	categorySvc   := services.NewCategoryService(repos.Category)
	reviewSvc     := services.NewReviewService(db, repos.Review, repos.Photo, repos.Place, ratingSvc, photoSvc)
	collectionSvc := services.NewCollectionService(repos.Collection)
	userSvc       := services.NewUserService(repos.User)

	return &routes.Handlers{
		Place:      handlers.NewPlaceHandler(placeSvc),
		Category:   handlers.NewCategoryHandler(categorySvc),
		Review:     handlers.NewReviewHandler(reviewSvc),
		Collection: handlers.NewCollectionHandler(collectionSvc),
		User:       handlers.NewUserHandler(userSvc, jwtSecret),
	}
}
