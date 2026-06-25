package repositories

import (
	"context"

	"accesspath/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PlaceRepository struct {
	db *pgxpool.Pool
}

func NewPlaceRepository(db *pgxpool.Pool) *PlaceRepository {
	return &PlaceRepository{db: db}
}

// Listas de columnas reutilizables. El escaneo a struct es por nombre (tags `db`),
// asi que el ORDEN aqui no tiene que coincidir con el de los campos del struct.
const placeColumns = `id, code, name, address, latitude, longitude, description, google_place_id, published, created_by, created_at, updated_at, deleted_at`
const placeColumnsP = `p.id, p.code, p.name, p.address, p.latitude, p.longitude, p.description, p.google_place_id, p.published, p.created_by, p.created_at, p.updated_at, p.deleted_at`

const placeWhereFilters = `
	  AND ($1::text = '' OR p.name ILIKE '%' || $1 || '%' OR p.address ILIKE '%' || $1 || '%')
	  AND ($2::bigint = 0 OR EXISTS (
	        SELECT 1 FROM place_rating_cache prc
	        JOIN subcategory s ON s.id = prc.subcategory_id
	        WHERE prc.place_id = p.id AND s.category_id = $2
	  ))
	  AND ($3::numeric = 0 OR (
	        SELECT COALESCE(AVG(prc.avg_score), 0)
	        FROM place_rating_cache prc
	        JOIN subcategory s ON s.id = prc.subcategory_id
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
		 FROM place p
		 WHERE p.deleted_at IS NULL`+placeWhereFilters,
		filters.Search, filters.CategoryID, filters.MinRating).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT `+placeColumnsP+`
		 FROM place p
		 WHERE p.deleted_at IS NULL`+placeWhereFilters+`
		 ORDER BY p.created_at DESC
		 LIMIT $4 OFFSET $5`,
		filters.Search, filters.CategoryID, filters.MinRating, filters.Limit, filters.Offset)
	if err != nil {
		return nil, 0, err
	}
	places, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.Place])
	if err != nil {
		return nil, 0, err
	}
	return places, total, nil
}

func (r *PlaceRepository) FindByID(ctx context.Context, id int64) (*models.Place, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+placeColumns+` FROM place WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return nil, err
	}
	place, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Place])
	if err != nil {
		return nil, err
	}
	return &place, nil
}

func (r *PlaceRepository) FindByCode(ctx context.Context, code string) (*models.Place, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+placeColumns+` FROM place WHERE code = $1 AND deleted_at IS NULL`, code)
	if err != nil {
		return nil, err
	}
	place, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Place])
	if err != nil {
		return nil, err
	}
	return &place, nil
}

func (r *PlaceRepository) FindByBounds(ctx context.Context, f models.BoundsFilter) ([]models.Place, error) {
	if f.Limit == 0 {
		f.Limit = 100
	}

	rows, err := r.db.Query(ctx,
		`SELECT `+placeColumnsP+`
		 FROM place p
		 WHERE p.deleted_at IS NULL
		   AND p.published = TRUE
		   AND p.latitude  BETWEEN $1 AND $2
		   AND p.longitude BETWEEN $3 AND $4
		   AND ($5::bigint = 0 OR EXISTS (
		         SELECT 1 FROM place_rating_cache prc
		         JOIN subcategory s ON s.id = prc.subcategory_id
		         WHERE prc.place_id = p.id AND s.category_id = $5
		   ))
		 ORDER BY p.created_at DESC
		 LIMIT $6`,
		f.MinLat, f.MaxLat, f.MinLng, f.MaxLng, f.CategoryID, f.Limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.Place])
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
		 FROM place
		 WHERE deleted_at IS NULL
		   AND (6371 * acos(cos(radians($1)) * cos(radians(latitude)) * cos(radians(longitude) - radians($2)) + sin(radians($1)) * sin(radians(latitude)))) < $3
		 ORDER BY distance
		 LIMIT $4 OFFSET $5`,
		f.Lat, f.Lng, f.Radius, f.Limit, f.Offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.PlaceWithDistance])
}

func (r *PlaceRepository) Create(ctx context.Context, req models.CreatePlaceRequest) (*models.Place, error) {
	rows, err := r.db.Query(ctx,
		`INSERT INTO place (name, address, latitude, longitude, description, google_place_id, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING `+placeColumns,
		req.Name, req.Address, req.Latitude, req.Longitude, req.Description, req.GooglePlaceID, req.CreatedBy)
	if err != nil {
		return nil, err
	}
	place, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Place])
	if err != nil {
		return nil, err
	}
	return &place, nil
}

func (r *PlaceRepository) FindByGooglePlaceID(ctx context.Context, googlePlaceID string) (*models.Place, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+placeColumns+` FROM place WHERE google_place_id = $1 AND deleted_at IS NULL`,
		googlePlaceID)
	if err != nil {
		return nil, err
	}
	place, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Place])
	if err != nil {
		return nil, err
	}
	return &place, nil
}

// MarkPublishedTx hace visible un lugar en el mapa tras su primera valoracion.
// Idempotente: solo escribe si aun no estaba publicado.
func (r *PlaceRepository) MarkPublishedTx(ctx context.Context, tx pgx.Tx, placeID int64) error {
	_, err := tx.Exec(ctx,
		`UPDATE place SET published = TRUE, updated_at = NOW()
		 WHERE id = $1 AND published = FALSE`, placeID)
	return err
}

func (r *PlaceRepository) Update(ctx context.Context, id int64, req models.UpdatePlaceRequest) (*models.Place, error) {
	rows, err := r.db.Query(ctx,
		`UPDATE place
		 SET name = $2, address = $3, latitude = $4, longitude = $5, description = $6, updated_at = NOW()
		 WHERE id = $1 AND deleted_at IS NULL
		 RETURNING `+placeColumns,
		id, req.Name, req.Address, req.Latitude, req.Longitude, req.Description)
	if err != nil {
		return nil, err
	}
	place, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Place])
	if err != nil {
		return nil, err
	}
	return &place, nil
}

func (r *PlaceRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE place SET deleted_at = NOW() WHERE id = $1`, id)
	return err
}
