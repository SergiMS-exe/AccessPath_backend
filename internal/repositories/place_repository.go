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

const placeColumns = `id, code, name, address, latitude, longitude, description, created_by, created_at, updated_at, deleted_at`
const placeColumnsP = `p.id, p.code, p.name, p.address, p.latitude, p.longitude, p.description, p.created_by, p.created_at, p.updated_at, p.deleted_at`

func scanPlace(row interface {
	Scan(...any) error
}, p *models.Place) error {
	return row.Scan(&p.ID, &p.Code, &p.Name, &p.Address, &p.Latitude, &p.Longitude,
		&p.Description, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt)
}

const placeWhereFilters = `
	  AND ($1::text = '' OR p.name ILIKE '%' || $1 || '%' OR p.address ILIKE '%' || $1 || '%')
	  AND ($2::bigint = 0 OR EXISTS (
	        SELECT 1 FROM place_rating_cache prc
	        JOIN subcategories s ON s.id = prc.subcategory_id
	        WHERE prc.place_id = p.id AND s.category_id = $2
	  ))
	  AND ($3::numeric = 0 OR (
	        SELECT COALESCE(AVG(prc.avg_score), 0)
	        FROM place_rating_cache prc
	        JOIN subcategories s ON s.id = prc.subcategory_id
	        WHERE prc.place_id = p.id
	          AND ($2::bigint = 0 OR s.category_id = $2)
	  ) >= $3)`

func (r *PlaceRepository) FindAll(ctx context.Context, filters models.PlaceFilters) ([]models.Place, int, error) {
	if filters.Limit == 0 {
		filters.Limit = 20
	}

	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*)
		 FROM places p
		 WHERE p.deleted_at IS NULL`+placeWhereFilters,
		filters.Search, filters.CategoryID, filters.MinRating).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT `+placeColumnsP+`
		 FROM places p
		 WHERE p.deleted_at IS NULL`+placeWhereFilters+`
		 ORDER BY p.created_at DESC
		 LIMIT $4 OFFSET $5`,
		filters.Search, filters.CategoryID, filters.MinRating, filters.Limit, filters.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	places := make([]models.Place, 0, filters.Limit)
	for rows.Next() {
		var p models.Place
		if err := scanPlace(rows, &p); err != nil {
			return nil, 0, err
		}
		places = append(places, p)
	}
	return places, total, nil
}

func (r *PlaceRepository) FindByID(ctx context.Context, id int64) (*models.Place, error) {
	var p models.Place
	err := scanPlace(r.db.QueryRow(ctx,
		`SELECT `+placeColumns+` FROM places WHERE id = $1 AND deleted_at IS NULL`, id), &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PlaceRepository) FindByCode(ctx context.Context, code string) (*models.Place, error) {
	var p models.Place
	err := scanPlace(r.db.QueryRow(ctx,
		`SELECT `+placeColumns+` FROM places WHERE code = $1 AND deleted_at IS NULL`, code), &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PlaceRepository) FindByBounds(ctx context.Context, f models.BoundsFilter) ([]models.Place, error) {
	if f.Limit == 0 {
		f.Limit = 100
	}

	rows, err := r.db.Query(ctx,
		`SELECT `+placeColumnsP+`
		 FROM places p
		 WHERE p.deleted_at IS NULL
		   AND p.latitude  BETWEEN $1 AND $2
		   AND p.longitude BETWEEN $3 AND $4
		   AND ($5::bigint = 0 OR EXISTS (
		         SELECT 1 FROM place_rating_cache prc
		         JOIN subcategories s ON s.id = prc.subcategory_id
		         WHERE prc.place_id = p.id AND s.category_id = $5
		   ))
		 ORDER BY p.created_at DESC
		 LIMIT $6`,
		f.MinLat, f.MaxLat, f.MinLng, f.MaxLng, f.CategoryID, f.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	places := make([]models.Place, 0)
	for rows.Next() {
		var p models.Place
		if err := scanPlace(rows, &p); err != nil {
			return nil, err
		}
		places = append(places, p)
	}
	return places, nil
}

func (r *PlaceRepository) FindNearby(ctx context.Context, f models.NearbyFilter) ([]models.PlaceWithDistance, error) {
	if f.Limit == 0 {
		f.Limit = 20
	}
	if f.Radius == 0 {
		f.Radius = 5
	}

	rows, err := r.db.Query(ctx,
		`SELECT `+placeColumns+`,
		 (6371 * acos(cos(radians($1)) * cos(radians(latitude)) * cos(radians(longitude) - radians($2)) + sin(radians($1)) * sin(radians(latitude)))) AS distance
		 FROM places
		 WHERE deleted_at IS NULL
		   AND (6371 * acos(cos(radians($1)) * cos(radians(latitude)) * cos(radians(longitude) - radians($2)) + sin(radians($1)) * sin(radians(latitude)))) < $3
		 ORDER BY distance
		 LIMIT $4 OFFSET $5`,
		f.Lat, f.Lng, f.Radius, f.Limit, f.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var places []models.PlaceWithDistance
	for rows.Next() {
		var p models.PlaceWithDistance
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &p.Address, &p.Latitude, &p.Longitude,
			&p.Description, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt, &p.Distance); err != nil {
			return nil, err
		}
		places = append(places, p)
	}
	return places, nil
}

func (r *PlaceRepository) Create(ctx context.Context, req models.CreatePlaceRequest) (*models.Place, error) {
	var p models.Place
	err := scanPlace(r.db.QueryRow(ctx,
		`INSERT INTO places (name, address, latitude, longitude, description, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING `+placeColumns,
		req.Name, req.Address, req.Latitude, req.Longitude, req.Description, req.CreatedBy), &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PlaceRepository) Update(ctx context.Context, id int64, req models.UpdatePlaceRequest) (*models.Place, error) {
	var p models.Place
	err := scanPlace(r.db.QueryRow(ctx,
		`UPDATE places
		 SET name = $2, address = $3, latitude = $4, longitude = $5, description = $6, updated_at = NOW()
		 WHERE id = $1 AND deleted_at IS NULL
		 RETURNING `+placeColumns,
		id, req.Name, req.Address, req.Latitude, req.Longitude, req.Description), &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PlaceRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE places SET deleted_at = NOW() WHERE id = $1`, id)
	return err
}
