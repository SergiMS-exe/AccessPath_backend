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
	rows, err := tx.Query(ctx,
		`INSERT INTO review_photo (review_id, url)
		 VALUES ($1, $2)
		 RETURNING id, code, review_id, url, created_at, deleted_at`,
		reviewID, url)
	if err != nil {
		return nil, err
	}
	photo, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Photo])
	if err != nil {
		return nil, err
	}
	return &photo, nil
}
