package repositories

import (
	"context"

	"accesspath/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// placeRatingRow es una fila plana del join de place_rating_cache, antes de
// agruparla por categoria en Go.
type placeRatingRow struct {
	CategoryID      int     `db:"category_id"`
	CategoryName    string  `db:"category_name"`
	SubcategoryID   int     `db:"subcategory_id"`
	SubcategoryName string  `db:"subcategory_name"`
	AvgScore        float64 `db:"avg_score"`
	TotalRatings    int     `db:"total_ratings"`
}

type RatingRepository struct {
	db *pgxpool.Pool
}

func NewRatingRepository(db *pgxpool.Pool) *RatingRepository {
	return &RatingRepository{db: db}
}

// UpsertTx inserts or updates a score in review_ratings within the given transaction.
func (r *RatingRepository) UpsertTx(ctx context.Context, tx pgx.Tx, reviewID, subcategoryID int64, score int) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO review_rating (review_id, subcategory_id, score)
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
		 FROM review_rating rr
		 JOIN review rv ON rr.review_id = rv.id
		 WHERE rr.subcategory_id = $1
		   AND rv.place_id = (SELECT place_id FROM review WHERE id = $2)
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
		 JOIN subcategory s ON prc.subcategory_id = s.id
		 JOIN category   c ON s.category_id       = c.id
		 WHERE prc.place_id = $1
		 ORDER BY c.display_order, s.display_order`,
		placeID)
	if err != nil {
		return nil, err
	}
	flat, err := pgx.CollectRows(rows, pgx.RowToStructByName[placeRatingRow])
	if err != nil {
		return nil, err
	}

	type catKey struct {
		id   int
		name string
	}
	order := []catKey{}
	seen := map[int]bool{}
	groups := map[int]*models.CategoryRating{}

	for _, row := range flat {
		if !seen[row.CategoryID] {
			seen[row.CategoryID] = true
			order = append(order, catKey{row.CategoryID, row.CategoryName})
			groups[row.CategoryID] = &models.CategoryRating{
				CategoryID:   row.CategoryID,
				CategoryName: row.CategoryName,
			}
		}
		groups[row.CategoryID].Subcategories = append(groups[row.CategoryID].Subcategories, models.SubcategoryRating{
			SubcategoryID:   row.SubcategoryID,
			SubcategoryName: row.SubcategoryName,
			AvgScore:        row.AvgScore,
			TotalRatings:    row.TotalRatings,
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
