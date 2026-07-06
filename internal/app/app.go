package app

import (
	"accesspath/internal/config"
	"accesspath/internal/handlers"
	"accesspath/internal/repositories"
	"accesspath/internal/routes"
	"accesspath/internal/services"
	"accesspath/pkg/gmaps"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
)

func BuildHandlers(db *pgxpool.Pool, minioClient *minio.Client, cfg *config.Config) *routes.Handlers {
	repos := repositories.New(db)

	accSvc := services.NewAccessibilityService(cfg.Accessibility)
	photoSvc := services.NewPhotoService(minioClient, cfg.MinioBucket, cfg.MinioPublicBaseURL)
	submissionSvc := services.NewSubmissionService(db, repos.Submission, repos.Photo, photoSvc)

	var gmapsClient *gmaps.Client
	if cfg.GMapsAPIKey != "" {
		gmapsClient = gmaps.New(cfg.GMapsAPIKey)
	}

	placeSvc := services.NewPlaceService(repos.Place, accSvc, submissionSvc, gmapsClient, repos.GmapsLog, cfg.GMapsMonthlyLimit)
	catalogSvc := services.NewCatalogService(repos.Catalog)
	contribSvc := services.NewContributionService(db, repos.Contribution, repos.Submission, repos.Catalog, repos.Place, accSvc)
	questionSvc := services.NewQuestionService(cfg.Accessibility, repos.Catalog, repos.Place, repos.Contribution, repos.Profile, accSvc)
	profileSvc := services.NewProfileService(db, repos.Profile)
	collectionSvc := services.NewCollectionService(repos.Collection)
	userSvc := services.NewUserService(repos.User)

	return &routes.Handlers{
		Place:        handlers.NewPlaceHandler(placeSvc),
		Catalog:      handlers.NewCatalogHandler(catalogSvc),
		Contribution: handlers.NewContributionHandler(contribSvc, questionSvc),
		Submission:   handlers.NewSubmissionHandler(submissionSvc),
		Profile:      handlers.NewProfileHandler(profileSvc),
		Collection:   handlers.NewCollectionHandler(collectionSvc),
		User:         handlers.NewUserHandler(userSvc, cfg.JWTSecret),
	}
}
