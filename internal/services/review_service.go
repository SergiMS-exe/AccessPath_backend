package services

import (
	"context"
	"errors"

	"accesspath/internal/models"
	"accesspath/internal/repositories"
)

type ReviewService struct {
	repo *repositories.ReviewRepository
}

func NewReviewService(repo *repositories.ReviewRepository) *ReviewService {
	return &ReviewService{repo: repo}
}

func (s *ReviewService) GetByPlace(ctx context.Context, placeID string) ([]models.FeatureReviewWithDetails, error) {
	return s.repo.FindByPlace(ctx, placeID)
}

func (s *ReviewService) GetPlaceAccessibility(ctx context.Context, placeID string) ([]models.FeatureAverage, error) {
	return s.repo.GetPlaceAverages(ctx, placeID)
}

func (s *ReviewService) Create(ctx context.Context, placeID string, req models.CreateReviewRequest) (*models.FeatureReview, error) {
	if req.Rating < 1 || req.Rating > 5 {
		return nil, errors.New("rating must be between 1 and 5")
	}
	return s.repo.Create(ctx, placeID, req)
}

func (s *ReviewService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
