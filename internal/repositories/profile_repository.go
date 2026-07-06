package repositories

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProfileRepository gestiona las necesidades funcionales y el consentimiento.
// El consentimiento vive en "user".accessibility_profile_consent_at.
type ProfileRepository struct {
	db *pgxpool.Pool
}

func NewProfileRepository(db *pgxpool.Pool) *ProfileRepository {
	return &ProfileRepository{db: db}
}

// GetNeeds devuelve las need_key elegidas por el usuario.
func (r *ProfileRepository) GetNeeds(ctx context.Context, userID int64) ([]string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT need_key FROM user_profile_need WHERE user_id = $1 ORDER BY need_key`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// GetConsent devuelve el timestamp de consentimiento (NULL si no lo dio).
func (r *ProfileRepository) GetConsent(ctx context.Context, userID int64) (*time.Time, error) {
	var consentAt *time.Time
	err := r.db.QueryRow(ctx,
		`SELECT accessibility_profile_consent_at FROM "user" WHERE id = $1 AND deleted_at IS NULL`, userID).
		Scan(&consentAt)
	if err != nil {
		return nil, err
	}
	return consentAt, nil
}

// ReplaceNeedsTx borra las necesidades actuales e inserta el nuevo conjunto.
func (r *ProfileRepository) ReplaceNeedsTx(ctx context.Context, tx pgx.Tx, userID int64, needs []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM user_profile_need WHERE user_id = $1`, userID); err != nil {
		return err
	}
	for _, need := range needs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO user_profile_need (user_id, need_key)
			 VALUES ($1, $2)
			 ON CONFLICT (user_id, need_key) DO NOTHING`, userID, need); err != nil {
			return err
		}
	}
	return nil
}

// SetConsentTx marca el consentimiento explicito (ahora).
func (r *ProfileRepository) SetConsentTx(ctx context.Context, tx pgx.Tx, userID int64) error {
	_, err := tx.Exec(ctx,
		`UPDATE "user" SET accessibility_profile_consent_at = NOW(), updated_at = NOW() WHERE id = $1`, userID)
	return err
}

// ClearConsentTx retira el consentimiento (al borrar el perfil).
func (r *ProfileRepository) ClearConsentTx(ctx context.Context, tx pgx.Tx, userID int64) error {
	_, err := tx.Exec(ctx,
		`UPDATE "user" SET accessibility_profile_consent_at = NULL, updated_at = NOW() WHERE id = $1`, userID)
	return err
}
