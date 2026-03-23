package repositories

import (
	"context"

	"accesspath/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RatingRepository struct {
	db *pgxpool.Pool
}

func NewRatingRepository(db *pgxpool.Pool) *RatingRepository {
	return &RatingRepository{db: db}
}

// UpsertTx inserts or updates a score in review_ratings within the given transaction.
func (r *RatingRepository) UpsertTx(ctx context.Context, tx pgx.Tx, reviewID, subcategoryID int64, score int) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO review_ratings (review_id, subcategory_id, score)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (review_id, subcategory_id)
		 DO UPDATE SET score = $3, updated_at = NOW()`,
		reviewID, subcategoryID, score)
	return err
}

// RecalculateCacheTx recomputes place_rating_cache for the (place, subcategory) pair
// affected by reviewID, running inside the given transaction.
func (r *RatingRepository) RecalculateCacheTx(ctx context.Context, tx pgx.Tx, reviewID, subcategoryID int64) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO place_rating_cache (place_id, subcategory_id, avg_score, total_ratings, updated_at)
		 SELECT
		     rv.place_id,
		     rr.subcategory_id,
		     ROUND(AVG(rr.score)::NUMERIC, 2),
		     COUNT(rr.review_id),
		     NOW()
		 FROM review_ratings rr
		 JOIN reviews rv ON rr.review_id = rv.id
		 WHERE rr.subcategory_id = $1
		   AND rv.place_id = (SELECT place_id FROM reviews WHERE id = $2)
		   AND rv.deleted_at IS NULL
		 GROUP BY rv.place_id, rr.subcategory_id
		 ON CONFLICT (place_id, subcategory_id)
		 DO UPDATE SET
		     avg_score     = EXCLUDED.avg_score,
		     total_ratings = EXCLUDED.total_ratings,
		     updated_at    = NOW()`,
		subcategoryID, reviewID)
	return err
}

// GetPlaceRatings reads from place_rating_cache and groups results by category.
// CategoryRating.AvgScore is computed in Go as the mean of its subcategories.
func (r *RatingRepository) GetPlaceRatings(ctx context.Context, placeID int64) ([]models.CategoryRating, error) {
	rows, err := r.db.Query(ctx,
		`SELECT
		     c.id   AS category_id,
		     c.name AS category_name,
		     s.id   AS subcategory_id,
		     s.name AS subcategory_name,
		     prc.avg_score,
		     prc.total_ratings
		 FROM place_rating_cache prc
		 JOIN subcategories s ON prc.subcategory_id = s.id
		 JOIN categories   c ON s.category_id       = c.id
		 WHERE prc.place_id = $1
		 ORDER BY c.display_order, s.display_order`,
		placeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type catKey struct {
		id   int
		name string
	}
	order := []catKey{}
	seen := map[int]bool{}
	groups := map[int]*models.CategoryRating{}

	for rows.Next() {
		var (
			catID, subID     int
			catName, subName string
			avgScore         float64
			total            int
		)
		if err := rows.Scan(&catID, &catName, &subID, &subName, &avgScore, &total); err != nil {
			return nil, err
		}
		if !seen[catID] {
			seen[catID] = true
			order = append(order, catKey{catID, catName})
			groups[catID] = &models.CategoryRating{
				CategoryID:   catID,
				CategoryName: catName,
			}
		}
		groups[catID].Subcategories = append(groups[catID].Subcategories, models.SubcategoryRating{
			SubcategoryID:   subID,
			SubcategoryName: subName,
			AvgScore:        avgScore,
			TotalRatings:    total,
		})
	}

	result := make([]models.CategoryRating, 0, len(order))
	for _, k := range order {
		cat := groups[k.id]
		var sum float64
		for _, sub := range cat.Subcategories {
			sum += sub.AvgScore
		}
		if len(cat.Subcategories) > 0 {
			cat.AvgScore = sum / float64(len(cat.Subcategories))
		}
		result = append(result, *cat)
	}
	return result, nil
}
