package repositories

import (
	"context"

	"accesspath/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PlaceRepository struct {
	db *pgxpool.Pool
}

func NewPlaceRepository(db *pgxpool.Pool) *PlaceRepository {
	return &PlaceRepository{db: db}
}

func (r *PlaceRepository) FindAll(ctx context.Context, filters models.PlaceFilters) ([]models.Place, error) {
	var query string
	var args []interface{}

	if filters.Limit == 0 {
		filters.Limit = 20
	}

	if filters.City != "" {
		query = `SELECT id, google_place_id, name, address, city, country, latitude, longitude, place_type, created_at, updated_at
				 FROM places WHERE city = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		args = []interface{}{filters.City, filters.Limit, filters.Offset}
	} else {
		query = `SELECT id, google_place_id, name, address, city, country, latitude, longitude, place_type, created_at, updated_at
				 FROM places ORDER BY created_at DESC LIMIT $1 OFFSET $2`
		args = []interface{}{filters.Limit, filters.Offset}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var places []models.Place
	for rows.Next() {
		var p models.Place
		err := rows.Scan(&p.ID, &p.GooglePlaceID, &p.Name, &p.Address, &p.City, &p.Country, &p.Latitude, &p.Longitude, &p.PlaceType, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		places = append(places, p)
	}

	return places, nil
}

func (r *PlaceRepository) FindByBounds(ctx context.Context, filters models.BoundsFilter) ([]models.Place, error) {
	if filters.Limit == 0 {
		filters.Limit = 100
	}

	query := `SELECT id, google_place_id, name, address, city, country, latitude, longitude, place_type, created_at, updated_at
			  FROM places
			  WHERE latitude BETWEEN $1 AND $2 AND longitude BETWEEN $3 AND $4
			  ORDER BY created_at DESC LIMIT $5`

	rows, err := r.db.Query(ctx, query, filters.MinLat, filters.MaxLat, filters.MinLng, filters.MaxLng, filters.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var places []models.Place
	for rows.Next() {
		var p models.Place
		err := rows.Scan(&p.ID, &p.GooglePlaceID, &p.Name, &p.Address, &p.City, &p.Country, &p.Latitude, &p.Longitude, &p.PlaceType, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		places = append(places, p)
	}

	return places, nil
}

func (r *PlaceRepository) FindNearby(ctx context.Context, filters models.NearbyFilter) ([]models.PlaceWithDistance, error) {
	if filters.Limit == 0 {
		filters.Limit = 20
	}
	if filters.Radius == 0 {
		filters.Radius = 5
	}

	query := `SELECT id, google_place_id, name, address, city, country, latitude, longitude, place_type, created_at, updated_at,
			  (6371 * acos(cos(radians($1)) * cos(radians(latitude)) * cos(radians(longitude) - radians($2)) + sin(radians($1)) * sin(radians(latitude)))) AS distance
			  FROM places
			  WHERE (6371 * acos(cos(radians($1)) * cos(radians(latitude)) * cos(radians(longitude) - radians($2)) + sin(radians($1)) * sin(radians(latitude)))) < $3
			  ORDER BY distance LIMIT $4 OFFSET $5`

	rows, err := r.db.Query(ctx, query, filters.Lat, filters.Lng, filters.Radius, filters.Limit, filters.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var places []models.PlaceWithDistance
	for rows.Next() {
		var p models.PlaceWithDistance
		err := rows.Scan(&p.ID, &p.GooglePlaceID, &p.Name, &p.Address, &p.City, &p.Country, &p.Latitude, &p.Longitude, &p.PlaceType, &p.CreatedAt, &p.UpdatedAt, &p.Distance)
		if err != nil {
			return nil, err
		}
		places = append(places, p)
	}

	return places, nil
}

func (r *PlaceRepository) FindByID(ctx context.Context, id string) (*models.Place, error) {
	query := `SELECT id, google_place_id, name, address, city, country, latitude, longitude, place_type, created_at, updated_at
			  FROM places WHERE id = $1`

	var p models.Place
	err := r.db.QueryRow(ctx, query, id).Scan(&p.ID, &p.GooglePlaceID, &p.Name, &p.Address, &p.City, &p.Country, &p.Latitude, &p.Longitude, &p.PlaceType, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *PlaceRepository) Create(ctx context.Context, req models.CreatePlaceRequest) (*models.Place, error) {
	query := `INSERT INTO places (google_place_id, name, address, city, country, latitude, longitude, place_type)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			  RETURNING id, google_place_id, name, address, city, country, latitude, longitude, place_type, created_at, updated_at`

	var p models.Place
	err := r.db.QueryRow(ctx, query, req.GooglePlaceID, req.Name, req.Address, req.City, req.Country, req.Latitude, req.Longitude, req.PlaceType).
		Scan(&p.ID, &p.GooglePlaceID, &p.Name, &p.Address, &p.City, &p.Country, &p.Latitude, &p.Longitude, &p.PlaceType, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *PlaceRepository) Update(ctx context.Context, id string, req models.CreatePlaceRequest) (*models.Place, error) {
	query := `UPDATE places SET name = $2, address = $3, city = $4, country = $5, latitude = $6, longitude = $7, place_type = $8, updated_at = NOW()
			  WHERE id = $1
			  RETURNING id, google_place_id, name, address, city, country, latitude, longitude, place_type, created_at, updated_at`

	var p models.Place
	err := r.db.QueryRow(ctx, query, id, req.Name, req.Address, req.City, req.Country, req.Latitude, req.Longitude, req.PlaceType).
		Scan(&p.ID, &p.GooglePlaceID, &p.Name, &p.Address, &p.City, &p.Country, &p.Latitude, &p.Longitude, &p.PlaceType, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *PlaceRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM places WHERE id = $1", id)
	return err
}
