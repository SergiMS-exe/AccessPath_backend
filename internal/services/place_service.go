package services

import (
	"context"

	"accesspath/internal/models"
	"accesspath/internal/repositories"
)

type PlaceService struct {
	repo *repositories.PlaceRepository
}

func NewPlaceService(repo *repositories.PlaceRepository) *PlaceService {
	return &PlaceService{repo: repo}
}

func (s *PlaceService) GetAll(ctx context.Context, filters models.PlaceFilters) ([]models.Place, error) {
	return s.repo.FindAll(ctx, filters)
}

func (s *PlaceService) GetByBounds(ctx context.Context, filters models.BoundsFilter) ([]models.Place, error) {
	return s.repo.FindByBounds(ctx, filters)
}

func (s *PlaceService) GetNearby(ctx context.Context, filters models.NearbyFilter) ([]models.PlaceWithDistance, error) {
	return s.repo.FindNearby(ctx, filters)
}

func (s *PlaceService) GetByID(ctx context.Context, id string) (*models.Place, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *PlaceService) Create(ctx context.Context, req models.CreatePlaceRequest) (*models.Place, error) {
	return s.repo.Create(ctx, req)
}

func (s *PlaceService) Update(ctx context.Context, id string, req models.CreatePlaceRequest) (*models.Place, error) {
	return s.repo.Update(ctx, id, req)
}

func (s *PlaceService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
