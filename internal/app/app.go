package app

import (
	"accesspath/internal/handlers"
	"accesspath/internal/repositories"
	"accesspath/internal/routes"
	"accesspath/internal/services"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
)

func BuildHandlers(db *pgxpool.Pool, minioClient *minio.Client, minioBucket string) *routes.Handlers {
	repos := repositories.New(db)

	ratingSvc := services.NewRatingService(repos.Rating)
	photoSvc  := services.NewPhotoService(minioClient, minioBucket)

	placeSvc      := services.NewPlaceService(repos.Place, ratingSvc)
	categorySvc   := services.NewCategoryService(repos.Category)
	reviewSvc     := services.NewReviewService(db, repos.Review, repos.Photo, ratingSvc, photoSvc)
	collectionSvc := services.NewCollectionService(repos.Collection)
	userSvc       := services.NewUserService(repos.User)

	return &routes.Handlers{
		Place:      handlers.NewPlaceHandler(placeSvc),
		Category:   handlers.NewCategoryHandler(categorySvc),
		Review:     handlers.NewReviewHandler(reviewSvc),
		Collection: handlers.NewCollectionHandler(collectionSvc),
		User:       handlers.NewUserHandler(userSvc),
	}
}
