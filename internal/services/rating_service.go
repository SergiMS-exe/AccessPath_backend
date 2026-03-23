package services

import (
	"context"

	"accesspath/internal/models"
	"accesspath/internal/repositories"

	"github.com/jackc/pgx/v5"
)

type RatingService struct {
	repo *repositories.RatingRepository
}

func NewRatingService(repo *repositories.RatingRepository) *RatingService {
	return &RatingService{repo: repo}
}

// UpsertRating upserts a score in review_ratings and recalculates
// place_rating_cache for the affected (place_id, subcategory_id) pair,
// all within the provided transaction.
func (s *RatingService) UpsertRating(ctx context.Context, tx pgx.Tx, reviewID, subcategoryID int64, score int) error {
	if err := s.repo.UpsertTx(ctx, tx, reviewID, subcategoryID, score); err != nil {
		return err
	}
	return s.repo.RecalculateCacheTx(ctx, tx, reviewID, subcategoryID)
}

// GetPlaceRatings returns ratings grouped by category from place_rating_cache.
func (s *RatingService) GetPlaceRatings(ctx context.Context, placeID int64) ([]models.CategoryRating, error) {
	return s.repo.GetPlaceRatings(ctx, placeID)
}
