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

const photoColumns = `id, code, submission_id, contribution_id, url, object_key, suggested_slot, created_at, deleted_at`

// SaveTx inserta una foto de una submission. contribution_id/object_key/
// suggested_slot son opcionales.
func (r *PhotoRepository) SaveTx(ctx context.Context, tx pgx.Tx, submissionID int64, contributionID *int64, url string, objectKey, suggestedSlot *string) (*models.Photo, error) {
	rows, err := tx.Query(ctx,
		`INSERT INTO photo (submission_id, contribution_id, url, object_key, suggested_slot)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+photoColumns,
		submissionID, contributionID, url, objectKey, suggestedSlot)
	if err != nil {
		return nil, err
	}
	photo, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Photo])
	if err != nil {
		return nil, err
	}
	return &photo, nil
}

// FindBySubmissionIDs devuelve las fotos vivas de un conjunto de submissions.
func (r *PhotoRepository) FindBySubmissionIDs(ctx context.Context, submissionIDs []int64) ([]models.Photo, error) {
	if len(submissionIDs) == 0 {
		return nil, nil
	}
	rows, err := r.db.Query(ctx,
		`SELECT `+photoColumns+`
		 FROM photo
		 WHERE submission_id = ANY($1) AND deleted_at IS NULL
		 ORDER BY created_at`, submissionIDs)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.Photo])
}
