package repositories

import (
	"context"

	"accesspath/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ReviewRepository struct {
	db *pgxpool.Pool
}

func NewReviewRepository(db *pgxpool.Pool) *ReviewRepository {
	return &ReviewRepository{db: db}
}

func (r *ReviewRepository) FindByPlace(ctx context.Context, placeID string) ([]models.FeatureReviewWithDetails, error) {
	query := `SELECT fr.id, fr.place_id, fr.user_id, fr.feature_id, fr.rating, fr.comment, fr.created_at, fr.updated_at,
			  u.name as user_name, af.name as feature_name
			  FROM feature_reviews fr
			  JOIN users u ON fr.user_id = u.id
			  JOIN accessibility_features af ON fr.feature_id = af.id
			  WHERE fr.place_id = $1
			  ORDER BY fr.created_at DESC`

	rows, err := r.db.Query(ctx, query, placeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []models.FeatureReviewWithDetails
	for rows.Next() {
		var rev models.FeatureReviewWithDetails
		if err := rows.Scan(&rev.ID, &rev.PlaceID, &rev.UserID, &rev.FeatureID, &rev.Rating, &rev.Comment, &rev.CreatedAt, &rev.UpdatedAt, &rev.UserName, &rev.FeatureName); err != nil {
			return nil, err
		}
		reviews = append(reviews, rev)
	}

	return reviews, nil
}

func (r *ReviewRepository) GetPlaceAverages(ctx context.Context, placeID string) ([]models.FeatureAverage, error) {
	query := `SELECT af.id, af.name, af.category_id,
			  COALESCE(AVG(fr.rating), 0) as average_rate,
			  COUNT(fr.id) as total_votes
			  FROM accessibility_features af
			  LEFT JOIN feature_reviews fr ON af.id = fr.feature_id AND fr.place_id = $1
			  GROUP BY af.id, af.name, af.category_id
			  ORDER BY af.category_id, af.id`

	rows, err := r.db.Query(ctx, query, placeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var averages []models.FeatureAverage
	for rows.Next() {
		var avg models.FeatureAverage
		if err := rows.Scan(&avg.FeatureID, &avg.FeatureName, &avg.CategoryID, &avg.AverageRate, &avg.TotalVotes); err != nil {
			return nil, err
		}
		averages = append(averages, avg)
	}

	return averages, nil
}

func (r *ReviewRepository) Create(ctx context.Context, placeID string, req models.CreateReviewRequest) (*models.FeatureReview, error) {
	query := `INSERT INTO feature_reviews (place_id, user_id, feature_id, rating, comment)
			  VALUES ($1, $2, $3, $4, $5)
			  ON CONFLICT (place_id, user_id, feature_id)
			  DO UPDATE SET rating = $4, comment = $5, updated_at = NOW()
			  RETURNING id, place_id, user_id, feature_id, rating, comment, created_at, updated_at`

	var rev models.FeatureReview
	err := r.db.QueryRow(ctx, query, placeID, req.UserID, req.FeatureID, req.Rating, req.Comment).
		Scan(&rev.ID, &rev.PlaceID, &rev.UserID, &rev.FeatureID, &rev.Rating, &rev.Comment, &rev.CreatedAt, &rev.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &rev, nil
}

func (r *ReviewRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM feature_reviews WHERE id = $1", id)
	return err
}
