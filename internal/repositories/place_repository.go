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

// aggSelect es la proyeccion comun del join criterion+dimension+cache a CriterionAggRow.
const aggSelect = `
	d.id AS dimension_id, d.key AS dimension_key, d.name AS dimension_name, d.sort_order AS dimension_sort,
	c.id AS criterion_id, c.key AS criterion_key, c.prompt, c.is_blocking, c.profile_tags, c.sort_order AS criterion_sort`

func (r *PlaceRepository) FindAll(ctx context.Context, filters models.PlaceFilters) ([]models.Place, int, error) {
	if filters.Limit == 0 {
		filters.Limit = 20
	}

	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*)
		 FROM place p
		 WHERE p.deleted_at IS NULL
		   AND ($1::text = '' OR p.name ILIKE '%' || $1 || '%' OR p.address ILIKE '%' || $1 || '%')`,
		filters.Search).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT `+placeColumnsP+`
		 FROM place p
		 WHERE p.deleted_at IS NULL
		   AND ($1::text = '' OR p.name ILIKE '%' || $1 || '%' OR p.address ILIKE '%' || $1 || '%')
		 ORDER BY p.created_at DESC
		 LIMIT $2 OFFSET $3`,
		filters.Search, filters.Limit, filters.Offset)
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

// FindByBounds devuelve lugares publicados dentro del bounding box (para el mapa).
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
		 ORDER BY p.created_at DESC
		 LIMIT $5`,
		f.MinLat, f.MaxLat, f.MinLng, f.MaxLng, f.Limit)
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

// MarkPublishedTx hace visible un lugar en el mapa tras su primera contribucion.
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

// --- Agregacion de accesibilidad ---

// GetAccessibilityRows devuelve TODOS los criterios activos (con su dimension)
// para un lugar, con los conteos del cache (LEFT JOIN: 0 si no hay datos -> gris).
// Base del desglose completo de GET /places/:id.
func (r *PlaceRepository) GetAccessibilityRows(ctx context.Context, placeID int64) ([]models.CriterionAggRow, error) {
	rows, err := r.db.Query(ctx,
		`SELECT
		     $1::bigint AS place_id,`+aggSelect+`,
		     COALESCE(pcc.n_yes, 0)    AS n_yes,
		     COALESCE(pcc.n_no, 0)     AS n_no,
		     COALESCE(pcc.n_unsure, 0) AS n_unsure,
		     pcc.quality_p50,
		     COALESCE(pcc.n_photos, 0) AS n_photos,
		     pcc.last_contribution_at
		 FROM dimension d
		 JOIN criterion c ON c.dimension_id = d.id AND c.active = TRUE
		 LEFT JOIN place_criterion_cache pcc ON pcc.criterion_id = c.id AND pcc.place_id = $1
		 WHERE d.active = TRUE
		 ORDER BY d.sort_order, c.sort_order, c.id`,
		placeID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.CriterionAggRow])
}

// GetCriterionAggRow devuelve la fila de agregacion de un unico (place, criterion),
// para el semaforo en vivo tras crear/borrar una contribucion.
func (r *PlaceRepository) GetCriterionAggRow(ctx context.Context, placeID, criterionID int64) (*models.CriterionAggRow, error) {
	rows, err := r.db.Query(ctx,
		`SELECT
		     $1::bigint AS place_id,`+aggSelect+`,
		     COALESCE(pcc.n_yes, 0)    AS n_yes,
		     COALESCE(pcc.n_no, 0)     AS n_no,
		     COALESCE(pcc.n_unsure, 0) AS n_unsure,
		     pcc.quality_p50,
		     COALESCE(pcc.n_photos, 0) AS n_photos,
		     pcc.last_contribution_at
		 FROM criterion c
		 JOIN dimension d ON d.id = c.dimension_id
		 LEFT JOIN place_criterion_cache pcc ON pcc.criterion_id = c.id AND pcc.place_id = $1
		 WHERE c.id = $2`,
		placeID, criterionID)
	if err != nil {
		return nil, err
	}
	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.CriterionAggRow])
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// GetAggRowsByPlaceIDs devuelve solo criterios CON datos (INNER JOIN cache) para
// un conjunto de lugares, para colorear el mapa. Lugares sin datos no aparecen.
func (r *PlaceRepository) GetAggRowsByPlaceIDs(ctx context.Context, placeIDs []int64) ([]models.CriterionAggRow, error) {
	rows, err := r.db.Query(ctx,
		`SELECT
		     pcc.place_id,`+aggSelect+`,
		     pcc.n_yes, pcc.n_no, pcc.n_unsure, pcc.quality_p50, pcc.n_photos, pcc.last_contribution_at
		 FROM place_criterion_cache pcc
		 JOIN criterion c ON c.id = pcc.criterion_id AND c.active = TRUE
		 JOIN dimension d ON d.id = c.dimension_id AND d.active = TRUE
		 WHERE pcc.place_id = ANY($1)
		 ORDER BY pcc.place_id, d.sort_order, c.sort_order, c.id`,
		placeIDs)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.CriterionAggRow])
}
