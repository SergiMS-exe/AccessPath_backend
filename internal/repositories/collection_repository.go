package repositories

import (
	"context"

	"accesspath/internal/models"

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
		 FROM collections
		 WHERE user_id = $1 AND deleted_at IS NULL
		 ORDER BY is_default DESC, created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []models.Collection
	for rows.Next() {
		var c models.Collection
		if err := rows.Scan(&c.ID, &c.Code, &c.UserID, &c.Name, &c.IsDefault,
			&c.CreatedAt, &c.UpdatedAt, &c.DeletedAt); err != nil {
			return nil, err
		}
		cols = append(cols, c)
	}
	return cols, nil
}

func (r *CollectionRepository) FindByID(ctx context.Context, id int64) (*models.Collection, error) {
	var c models.Collection
	err := r.db.QueryRow(ctx,
		`SELECT id, code, user_id, name, is_default, created_at, updated_at, deleted_at
		 FROM collections WHERE id = $1 AND deleted_at IS NULL`, id).
		Scan(&c.ID, &c.Code, &c.UserID, &c.Name, &c.IsDefault, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CollectionRepository) Create(ctx context.Context, req models.CreateCollectionRequest) (*models.Collection, error) {
	var c models.Collection
	err := r.db.QueryRow(ctx,
		`INSERT INTO collections (user_id, name, is_default)
		 VALUES ($1, $2, $3)
		 RETURNING id, code, user_id, name, is_default, created_at, updated_at, deleted_at`,
		req.UserID, req.Name, req.IsDefault).
		Scan(&c.ID, &c.Code, &c.UserID, &c.Name, &c.IsDefault, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CollectionRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE collections SET deleted_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *CollectionRepository) AddPlace(ctx context.Context, collectionID, placeID int64) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO collection_places (collection_id, place_id)
		 VALUES ($1, $2)
		 ON CONFLICT DO NOTHING`,
		collectionID, placeID)
	return err
}

func (r *CollectionRepository) RemovePlace(ctx context.Context, collectionID, placeID int64) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM collection_places WHERE collection_id = $1 AND place_id = $2`,
		collectionID, placeID)
	return err
}

func (r *CollectionRepository) GetPlaces(ctx context.Context, collectionID int64) ([]models.Place, error) {
	rows, err := r.db.Query(ctx,
		`SELECT p.id, p.code, p.name, p.address, p.latitude, p.longitude, p.description, p.created_by,
		        p.created_at, p.updated_at, p.deleted_at
		 FROM places p
		 JOIN collection_places cp ON p.id = cp.place_id
		 WHERE cp.collection_id = $1 AND p.deleted_at IS NULL
		 ORDER BY cp.added_at DESC`, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var places []models.Place
	for rows.Next() {
		var p models.Place
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &p.Address, &p.Latitude, &p.Longitude,
			&p.Description, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt); err != nil {
			return nil, err
		}
		places = append(places, p)
	}
	return places, nil
}
