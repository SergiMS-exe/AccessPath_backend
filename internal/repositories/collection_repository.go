package repositories

import (
	"context"

	"accesspath/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CollectionRepository struct {
	db *pgxpool.Pool
}

func NewCollectionRepository(db *pgxpool.Pool) *CollectionRepository {
	return &CollectionRepository{db: db}
}

func (r *CollectionRepository) FindByUser(ctx context.Context, userID int64) ([]models.Collection, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, code, user_id, name, is_default, created_at, updated_at, deleted_at
		 FROM collection
		 WHERE user_id = $1 AND deleted_at IS NULL
		 ORDER BY is_default DESC, created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.Collection])
}

func (r *CollectionRepository) FindByID(ctx context.Context, id int64) (*models.Collection, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, code, user_id, name, is_default, created_at, updated_at, deleted_at
		 FROM collection WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return nil, err
	}
	col, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Collection])
	if err != nil {
		return nil, err
	}
	return &col, nil
}

func (r *CollectionRepository) Create(ctx context.Context, req models.CreateCollectionRequest) (*models.Collection, error) {
	rows, err := r.db.Query(ctx,
		`INSERT INTO collection (user_id, name, is_default)
		 VALUES ($1, $2, $3)
		 RETURNING id, code, user_id, name, is_default, created_at, updated_at, deleted_at`,
		req.UserID, req.Name, req.IsDefault)
	if err != nil {
		return nil, err
	}
	col, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Collection])
	if err != nil {
		return nil, err
	}
	return &col, nil
}

func (r *CollectionRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE collection SET deleted_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *CollectionRepository) AddPlace(ctx context.Context, collectionID, placeID int64) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO collection_place (collection_id, place_id)
		 VALUES ($1, $2)
		 ON CONFLICT DO NOTHING`,
		collectionID, placeID)
	return err
}

func (r *CollectionRepository) RemovePlace(ctx context.Context, collectionID, placeID int64) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM collection_place WHERE collection_id = $1 AND place_id = $2`,
		collectionID, placeID)
	return err
}

func (r *CollectionRepository) GetPlaces(ctx context.Context, collectionID int64) ([]models.Place, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+placeColumnsP+`
		 FROM place p
		 JOIN collection_place cp ON p.id = cp.place_id
		 WHERE cp.collection_id = $1 AND p.deleted_at IS NULL
		 ORDER BY cp.added_at DESC`, collectionID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.Place])
}
