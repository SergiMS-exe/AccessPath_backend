package repositories

import (
	"context"

	"accesspath/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PhotoRepository struct {
	db *pgxpool.Pool
}

func NewPhotoRepository(db *pgxpool.Pool) *PhotoRepository {
	return &PhotoRepository{db: db}
}

// SaveTx inserts a photo record into review_photos within the given transaction.
func (r *PhotoRepository) SaveTx(ctx context.Context, tx pgx.Tx, reviewID int64, url string) (*models.Photo, error) {
	var ph models.Photo
	err := tx.QueryRow(ctx,
		`INSERT INTO review_photos (review_id, url)
		 VALUES ($1, $2)
		 RETURNING id, code, review_id, url, created_at, deleted_at`,
		reviewID, url).
		Scan(&ph.ID, &ph.Code, &ph.ReviewID, &ph.URL, &ph.CreatedAt, &ph.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &ph, nil
}
