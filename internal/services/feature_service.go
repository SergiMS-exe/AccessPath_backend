package services

import (
	"context"

	"accesspath/internal/models"
	"accesspath/internal/repositories"
)

type FeatureService struct {
	repo *repositories.FeatureRepository
}

func NewFeatureService(repo *repositories.FeatureRepository) *FeatureService {
	return &FeatureService{repo: repo}
}

func (s *FeatureService) GetAllCategories(ctx context.Context) ([]models.FeatureCategory, error) {
	return s.repo.FindAllCategories(ctx)
}

func (s *FeatureService) GetAllFeatures(ctx context.Context) ([]models.FeatureWithCategory, error) {
	return s.repo.FindAllFeatures(ctx)
}

func (s *FeatureService) GetByCategory(ctx context.Context, categoryID int32) ([]models.AccessibilityFeature, error) {
	return s.repo.FindByCategory(ctx, categoryID)
}

func (s *FeatureService) GetByID(ctx context.Context, id int32) (*models.FeatureWithCategory, error) {
	return s.repo.FindByID(ctx, id)
}
