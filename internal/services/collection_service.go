package services

import (
	"context"

	"accesspath/internal/models"
	"accesspath/internal/repositories"
)

type CollectionService struct {
	repo *repositories.CollectionRepository
}

func NewCollectionService(repo *repositories.CollectionRepository) *CollectionService {
	return &CollectionService{repo: repo}
}

func (s *CollectionService) GetByUser(ctx context.Context, userID int64) ([]models.Collection, error) {
	return s.repo.FindByUser(ctx, userID)
}

func (s *CollectionService) GetByID(ctx context.Context, id int64) (*models.Collection, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *CollectionService) Create(ctx context.Context, req models.CreateCollectionRequest) (*models.Collection, error) {
	return s.repo.Create(ctx, req)
}

func (s *CollectionService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *CollectionService) AddPlace(ctx context.Context, collectionID, placeID int64) error {
	return s.repo.AddPlace(ctx, collectionID, placeID)
}

func (s *CollectionService) RemovePlace(ctx context.Context, collectionID, placeID int64) error {
	return s.repo.RemovePlace(ctx, collectionID, placeID)
}

func (s *CollectionService) GetPlaces(ctx context.Context, collectionID int64) ([]models.Place, error) {
	return s.repo.GetPlaces(ctx, collectionID)
}
