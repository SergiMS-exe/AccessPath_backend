package repositories

import (
	"context"

	"accesspath/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SubmissionRepository struct {
	db *pgxpool.Pool
}

func NewSubmissionRepository(db *pgxpool.Pool) *SubmissionRepository {
	return &SubmissionRepository{db: db}
}

const submissionColumns = `id, code, user_id, place_id, comment, created_at, updated_at, deleted_at`

// GetOrCreateLiveTx devuelve la valoracion viva del usuario para el lugar,
// creandola si no existe. Idempotente por el indice unico parcial
// (user_id, place_id) WHERE deleted_at IS NULL. Refresca updated_at.
func (r *SubmissionRepository) GetOrCreateLiveTx(ctx context.Context, tx pgx.Tx, userID, placeID int64) (*models.Submission, error) {
	rows, err := tx.Query(ctx,
		`INSERT INTO submission (user_id, place_id)
		 VALUES ($1, $2)
		 ON CONFLICT (user_id, place_id) WHERE deleted_at IS NULL
		 DO UPDATE SET updated_at = NOW()
		 RETURNING `+submissionColumns,
		userID, placeID)
	if err != nil {
		return nil, err
	}
	submission, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Submission])
	if err != nil {
		return nil, err
	}
	return &submission, nil
}

// SetCommentTx fija el comentario de la submission y refresca updated_at.
func (r *SubmissionRepository) SetCommentTx(ctx context.Context, tx pgx.Tx, submissionID int64, comment *string) error {
	_, err := tx.Exec(ctx,
		`UPDATE submission SET comment = $1, updated_at = NOW() WHERE id = $2`,
		comment, submissionID)
	return err
}

// FindByPlace devuelve las valoraciones vivas de un lugar que aportan algo que
// contar (comentario no vacio o al menos una foto), con su autor. Las fotos se
// adjuntan por separado via PhotoRepository.
func (r *SubmissionRepository) FindByPlace(ctx context.Context, placeID int64) ([]models.SubmissionWithDetails, error) {
	rows, err := r.db.Query(ctx,
		`SELECT s.id, s.code, s.user_id, s.place_id, s.comment, s.created_at, s.updated_at, s.deleted_at,
		        u.username
		 FROM submission s
		 JOIN "user" u ON s.user_id = u.id
		 WHERE s.place_id = $1 AND s.deleted_at IS NULL
		   AND (
		       (s.comment IS NOT NULL AND btrim(s.comment) <> '')
		       OR EXISTS (SELECT 1 FROM photo p WHERE p.submission_id = s.id AND p.deleted_at IS NULL)
		   )
		 ORDER BY s.created_at DESC`, placeID)
	if err != nil {
		return nil, err
	}
	// Lax: SubmissionWithDetails tiene un campo Photos sin columna (db:"-").
	return pgx.CollectRows(rows, pgx.RowToStructByNameLax[models.SubmissionWithDetails])
}
