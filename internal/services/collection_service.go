package services

import (
	"context"

	"accesspath/internal/models"
	"accesspath/internal/repositories"
)

type CollectionService interface {
	GetByUser(ctx context.Context, userID int64) ([]models.Collection, error)
	GetByID(ctx context.Context, id int64) (*models.Collection, error)
	Create(ctx context.Context, req models.CreateCollectionRequest) (*models.Collection, error)
	Delete(ctx context.Context, id int64) error
	AddPlace(ctx context.Context, collectionID, placeID int64) error
	RemovePlace(ctx context.Context, collectionID, placeID int64) error
	GetPlaces(ctx context.Context, collectionID int64) ([]models.Place, error)
}

type pgCollectionService struct {
	repo repositories.CollectionRepository
}

func NewCollectionService(repo repositories.CollectionRepository) CollectionService {
	return &pgCollectionService{repo: repo}
}

var _ CollectionService = (*pgCollectionService)(nil)

func (s *pgCollectionService) GetByUser(ctx context.Context, userID int64) ([]models.Collection, error) {
	return s.repo.FindByUser(ctx, userID)
}

func (s *pgCollectionService) GetByID(ctx context.Context, id int64) (*models.Collection, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *pgCollectionService) Create(ctx context.Context, req models.CreateCollectionRequest) (*models.Collection, error) {
	return s.repo.Create(ctx, req)
}

func (s *pgCollectionService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *pgCollectionService) AddPlace(ctx context.Context, collectionID, placeID int64) error {
	return s.repo.AddPlace(ctx, collectionID, placeID)
}

func (s *pgCollectionService) RemovePlace(ctx context.Context, collectionID, placeID int64) error {
	return s.repo.RemovePlace(ctx, collectionID, placeID)
}

func (s *pgCollectionService) GetPlaces(ctx context.Context, collectionID int64) ([]models.Place, error) {
	return s.repo.GetPlaces(ctx, collectionID)
}