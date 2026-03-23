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
		 FROM reviews rv
		 JOIN users u ON rv.user_id = u.id
		 WHERE rv.place_id = $1 AND rv.deleted_at IS NULL
		 ORDER BY rv.created_at DESC`, placeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []models.ReviewWithDetails
	for rows.Next() {
		var rv models.ReviewWithDetails
		if err := rows.Scan(&rv.ID, &rv.Code, &rv.UserID, &rv.PlaceID, &rv.Comment,
			&rv.CreatedAt, &rv.UpdatedAt, &rv.DeletedAt, &rv.Username); err != nil {
			return nil, err
		}
		reviews = append(reviews, rv)
	}
	return reviews, nil
}

func (r *ReviewRepository) FindByUser(ctx context.Context, userID int64) ([]models.Review, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, code, user_id, place_id, comment, created_at, updated_at, deleted_at
		 FROM reviews
		 WHERE user_id = $1 AND deleted_at IS NULL
		 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []models.Review
	for rows.Next() {
		var rv models.Review
		if err := rows.Scan(&rv.ID, &rv.Code, &rv.UserID, &rv.PlaceID, &rv.Comment,
			&rv.CreatedAt, &rv.UpdatedAt, &rv.DeletedAt); err != nil {
			return nil, err
		}
		reviews = append(reviews, rv)
	}
	return reviews, nil
}

func (r *ReviewRepository) FindByID(ctx context.Context, id int64) (*models.Review, error) {
	var rv models.Review
	err := r.db.QueryRow(ctx,
		`SELECT id, code, user_id, place_id, comment, created_at, updated_at, deleted_at
		 FROM reviews WHERE id = $1 AND deleted_at IS NULL`, id).
		Scan(&rv.ID, &rv.Code, &rv.UserID, &rv.PlaceID, &rv.Comment,
			&rv.CreatedAt, &rv.UpdatedAt, &rv.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &rv, nil
}

// CreateTx inserts a review row inside an existing transaction.
func (r *ReviewRepository) CreateTx(ctx context.Context, tx pgx.Tx, req models.CreateReviewRequest) (*models.Review, error) {
	var rv models.Review
	err := tx.QueryRow(ctx,
		`INSERT INTO reviews (user_id, place_id, comment)
		 VALUES ($1, $2, $3)
		 RETURNING id, code, user_id, place_id, comment, created_at, updated_at, deleted_at`,
		req.UserID, req.PlaceID, req.Comment).
		Scan(&rv.ID, &rv.Code, &rv.UserID, &rv.PlaceID, &rv.Comment,
			&rv.CreatedAt, &rv.UpdatedAt, &rv.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &rv, nil
}

func (r *ReviewRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE reviews SET deleted_at = NOW() WHERE id = $1`, id)
	return err
}
