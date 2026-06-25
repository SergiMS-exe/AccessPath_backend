package repositories

import (
	"context"

	"accesspath/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReviewRepository struct {
	db *pgxpool.Pool
}

func NewReviewRepository(db *pgxpool.Pool) *ReviewRepository {
	return &ReviewRepository{db: db}
}

func (r *ReviewRepository) FindByPlace(ctx context.Context, placeID int64) ([]models.ReviewWithDetails, error) {
	rows, err := r.db.Query(ctx,
		`SELECT rv.id, rv.code, rv.user_id, rv.place_id, rv.comment, rv.created_at, rv.updated_at, rv.deleted_at,
		        u.username
		 FROM review rv
		 JOIN "user" u ON rv.user_id = u.id
		 WHERE rv.place_id = $1 AND rv.deleted_at IS NULL
		 ORDER BY rv.created_at DESC`, placeID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.ReviewWithDetails])
}

func (r *ReviewRepository) FindByUser(ctx context.Context, userID int64) ([]models.Review, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, code, user_id, place_id, comment, created_at, updated_at, deleted_at
		 FROM review
		 WHERE user_id = $1 AND deleted_at IS NULL
		 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.Review])
}

func (r *ReviewRepository) FindByID(ctx context.Context, id int64) (*models.Review, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, code, user_id, place_id, comment, created_at, updated_at, deleted_at
		 FROM review WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return nil, err
	}
	review, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Review])
	if err != nil {
		return nil, err
	}
	return &review, nil
}

// CreateTx inserts a review row inside an existing transaction.
func (r *ReviewRepository) CreateTx(ctx context.Context, tx pgx.Tx, req models.CreateReviewRequest) (*models.Review, error) {
	rows, err := tx.Query(ctx,
		`INSERT INTO review (user_id, place_id, comment)
		 VALUES ($1, $2, $3)
		 RETURNING id, code, user_id, place_id, comment, created_at, updated_at, deleted_at`,
		req.UserID, req.PlaceID, req.Comment)
	if err != nil {
		return nil, err
	}
	review, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Review])
	if err != nil {
		return nil, err
	}
	return &review, nil
}

func (r *ReviewRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE review SET deleted_at = NOW() WHERE id = $1`, id)
	return err
}
