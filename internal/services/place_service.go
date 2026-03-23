package services

import (
	"context"
	"fmt"

	"accesspath/internal/models"
	"accesspath/internal/repositories"
)

type PlaceService struct {
	repo      *repositories.PlaceRepository
	ratingSvc *RatingService
}

func NewPlaceService(repo *repositories.PlaceRepository, ratingSvc *RatingService) *PlaceService {
	return &PlaceService{repo: repo, ratingSvc: ratingSvc}
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

// GetByID returns the place detail including its rating cache grouped by category.
func (s *PlaceService) GetByID(ctx context.Context, id int64) (*models.PlaceDetail, error) {
	place, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("place: find: %w", err)
	}
	ratings, err := s.ratingSvc.GetPlaceRatings(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("place: ratings: %w", err)
	}
	return &models.PlaceDetail{Place: *place, Ratings: ratings}, nil
}

func (s *PlaceService) Create(ctx context.Context, req models.CreatePlaceRequest) (*models.Place, error) {
	return s.repo.Create(ctx, req)
}

func (s *PlaceService) Update(ctx context.Context, id int64, req models.UpdatePlaceRequest) (*models.Place, error) {
	return s.repo.Update(ctx, id, req)
}

func (s *PlaceService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
