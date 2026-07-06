package repositories

import (
	"context"

	"accesspath/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ContributionRepository struct {
	db *pgxpool.Pool
}

func NewContributionRepository(db *pgxpool.Pool) *ContributionRepository {
	return &ContributionRepository{db: db}
}

const contributionColumns = `id, code, submission_id, user_id, place_id, criterion_id, answer_option_id, exists_flag, quality, created_at, updated_at, deleted_at`

// UpsertLiveTx inserta o actualiza la contribucion viva de la submission para
// un criterio, respetando el indice unico parcial (submission_id, criterion_id).
// Editar = UPDATE in-place: refresca updated_at (fecha de validez), created_at
// permanece estable. exists/quality vienen copiados de la opcion en el servicio.
func (r *ContributionRepository) UpsertLiveTx(ctx context.Context, tx pgx.Tx, submissionID, userID, placeID, criterionID, answerOptionID int64, existsFlag *bool, quality *int) (*models.Contribution, error) {
	rows, err := tx.Query(ctx,
		`INSERT INTO contribution (submission_id, user_id, place_id, criterion_id, answer_option_id, exists_flag, quality)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (submission_id, criterion_id) WHERE deleted_at IS NULL
		 DO UPDATE SET answer_option_id = EXCLUDED.answer_option_id,
		               exists_flag      = EXCLUDED.exists_flag,
		               quality          = EXCLUDED.quality,
		               updated_at       = NOW()
		 RETURNING `+contributionColumns,
		submissionID, userID, placeID, criterionID, answerOptionID, existsFlag, quality)
	if err != nil {
		return nil, err
	}
	contribution, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Contribution])
	if err != nil {
		return nil, err
	}
	return &contribution, nil
}

// FindByID lee una contribucion viva (para validar propiedad al borrar).
func (r *ContributionRepository) FindByID(ctx context.Context, id int64) (*models.Contribution, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+contributionColumns+` FROM contribution WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return nil, err
	}
	contribution, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Contribution])
	if err != nil {
		return nil, err
	}
	return &contribution, nil
}

// SoftDeleteTx marca la contribucion como borrada (Deshacer).
func (r *ContributionRepository) SoftDeleteTx(ctx context.Context, tx pgx.Tx, id int64) error {
	_, err := tx.Exec(ctx,
		`UPDATE contribution SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}

// AnsweredByPlace devuelve las respuestas vivas del usuario en un lugar.
func (r *ContributionRepository) AnsweredByPlace(ctx context.Context, userID, placeID int64) ([]models.AnsweredContribution, error) {
	rows, err := r.db.Query(ctx,
		`SELECT criterion_id, exists_flag
		 FROM contribution
		 WHERE user_id = $1 AND place_id = $2 AND deleted_at IS NULL`, userID, placeID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.AnsweredContribution])
}

// RecalculateCacheTx recalcula place_criterion_cache para (place, criterion):
// conteos por exists, mediana de quality (percentile_cont) y n_photos. Se ejecuta
// en la misma TX que la contribucion o su borrado. Si no quedan contribuciones
// vivas, deja la fila a cero (definidos=0 => gris al derivar).
func (r *ContributionRepository) RecalculateCacheTx(ctx context.Context, tx pgx.Tx, placeID, criterionID int64) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO place_criterion_cache
		     (place_id, criterion_id, n_yes, n_no, n_unsure, quality_p50, n_photos, last_contribution_at, updated_at)
		 SELECT
		     $1, $2,
		     COUNT(*) FILTER (WHERE c.exists_flag IS TRUE),
		     COUNT(*) FILTER (WHERE c.exists_flag IS FALSE),
		     COUNT(*) FILTER (WHERE c.exists_flag IS NULL),
		     PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY c.quality)
		         FILTER (WHERE c.exists_flag IS TRUE AND c.quality IS NOT NULL),
		     COUNT(*) FILTER (WHERE EXISTS (
		         SELECT 1 FROM photo p WHERE p.contribution_id = c.id AND p.deleted_at IS NULL)),
		     MAX(c.updated_at),
		     NOW()
		 FROM contribution c
		 WHERE c.place_id = $1 AND c.criterion_id = $2 AND c.deleted_at IS NULL
		 ON CONFLICT (place_id, criterion_id) DO UPDATE SET
		     n_yes                = EXCLUDED.n_yes,
		     n_no                 = EXCLUDED.n_no,
		     n_unsure             = EXCLUDED.n_unsure,
		     quality_p50          = EXCLUDED.quality_p50,
		     n_photos             = EXCLUDED.n_photos,
		     last_contribution_at = EXCLUDED.last_contribution_at,
		     updated_at           = NOW()`,
		placeID, criterionID)
	return err
}
