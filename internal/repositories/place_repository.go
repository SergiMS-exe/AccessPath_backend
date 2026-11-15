package repositories

import (
	"context"

	"accesspath/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PlaceRepository define las operaciones del repositorio de lugares que
// pueden invocar los servicios. La implementacion concreta es privada.
type PlaceRepository interface {
	FindAll(ctx context.Context, filters models.PlaceFilters) ([]models.Place, int, error)
	FindByID(ctx context.Context, id int64) (*models.Place, error)
	FindByCode(ctx context.Context, code string) (*models.Place, error)
	FindByBounds(ctx context.Context, f models.BoundsFilter) ([]models.Place, error)
	FindNearby(ctx context.Context, f models.NearbyFilter) ([]models.PlaceWithDistance, error)
	Create(ctx context.Context, req models.CreatePlaceRequest) (*models.Place, error)
	FindBySourceExternalID(ctx context.Context, source, externalID string) (*models.Place, error)
	SearchVerified(ctx context.Context, query string, limit int) ([]models.Place, error)
	FindSimilar(ctx context.Context, lat, lng, radiusMeters float64, name string) (*models.Place, error)
	MarkPublishedTx(ctx context.Context, tx pgx.Tx, placeID int64) error
	Update(ctx context.Context, id int64, req models.UpdatePlaceRequest) (*models.Place, error)
	Delete(ctx context.Context, id int64) error
	GetAccessibilityRows(ctx context.Context, placeID int64) ([]models.CriterionAggRow, error)
	GetCriterionAggRow(ctx context.Context, placeID, criterionID int64) (*models.CriterionAggRow, error)
	GetAggRowsByPlaceIDs(ctx context.Context, placeIDs []int64) ([]models.CriterionAggRow, error)
}

type pgPlaceRepository struct {
	db *pgxpool.Pool
}

func NewPlaceRepository(db *pgxpool.Pool) PlaceRepository {
	return &pgPlaceRepository{db: db}
}

var _ PlaceRepository = (*pgPlaceRepository)(nil)

// Listas de columnas reutilizables. El escaneo a struct es por nombre (tags `db`),
// asi que el ORDEN aqui no tiene que coincidir con el de los campos del struct.
const placeColumns = `id, code, name, address, latitude, longitude, description, source, external_id, google_ftid, source_url, published, moderation_status, created_by, created_at, updated_at, deleted_at`
const placeColumnsP = `p.id, p.code, p.name, p.address, p.latitude, p.longitude, p.description, p.source, p.external_id, p.google_ftid, p.source_url, p.published, p.moderation_status, p.created_by, p.created_at, p.updated_at, p.deleted_at`

// aggSelect es la proyeccion comun del join criterion+dimension+cache a CriterionAggRow.
const aggSelect = `
	d.id AS dimension_id, d.key AS dimension_key, d.name AS dimension_name, d.sort_order AS dimension_sort,
	c.id AS criterion_id, c.key AS criterion_key, c.prompt, c.is_blocking, c.profile_tags, c.sort_order AS criterion_sort`

func (r *pgPlaceRepository) FindAll(ctx context.Context, filters models.PlaceFilters) ([]models.Place, int, error) {
	if filters.Limit == 0 {
		filters.Limit = 20
	}

	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*)
		 FROM place p
		 WHERE p.deleted_at IS NULL
		   AND p.moderation_status = 'verified'
		   AND ($1::text = '' OR p.name ILIKE '%' || $1 || '%' OR p.address ILIKE '%' || $1 || '%')`,
		filters.Search).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT `+placeColumnsP+`
		 FROM place p
		 WHERE p.deleted_at IS NULL
		   AND p.moderation_status = 'verified'
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

func (r *pgPlaceRepository) FindByID(ctx context.Context, id int64) (*models.Place, error) {
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

func (r *pgPlaceRepository) FindByCode(ctx context.Context, code string) (*models.Place, error) {
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
func (r *pgPlaceRepository) FindByBounds(ctx context.Context, f models.BoundsFilter) ([]models.Place, error) {
	if f.Limit == 0 {
		f.Limit = 100
	}

	rows, err := r.db.Query(ctx,
		`SELECT `+placeColumnsP+`
		 FROM place p
		 WHERE p.deleted_at IS NULL
		   AND p.published = TRUE
		   AND p.moderation_status = 'verified'
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

func (r *pgPlaceRepository) FindNearby(ctx context.Context, f models.NearbyFilter) ([]models.PlaceWithDistance, error) {
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
		   AND moderation_status = 'verified'
		   AND (6371 * acos(cos(radians($1)) * cos(radians(latitude)) * cos(radians(longitude) - radians($2)) + sin(radians($1)) * sin(radians(latitude)))) < $3
		 ORDER BY distance
		 LIMIT $4 OFFSET $5`,
		f.Lat, f.Lng, f.Radius, f.Limit, f.Offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.PlaceWithDistance])
}

func (r *pgPlaceRepository) Create(ctx context.Context, req models.CreatePlaceRequest) (*models.Place, error) {
	rows, err := r.db.Query(ctx,
		`INSERT INTO place (name, address, latitude, longitude, description,
		                    source, external_id, google_ftid, source_url, moderation_status, created_by)
		 VALUES ($1, $2, $3, $4, $5,
		         COALESCE($6::text, 'community'), $7, $8, $9,
		         COALESCE($10::text, 'verified'), $11)
		 RETURNING `+placeColumns,
		req.Name, req.Address, req.Latitude, req.Longitude, req.Description,
		req.Source, req.ExternalID, req.GoogleFTID, req.SourceURL, req.ModerationStatus, req.CreatedBy)
	if err != nil {
		return nil, err
	}
	place, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Place])
	if err != nil {
		return nil, err
	}
	return &place, nil
}

// FindBySourceExternalID busca un lugar ya importado desde un proveedor
// concreto. Devuelve pgx.ErrNoRows si no existe.
func (r *pgPlaceRepository) FindBySourceExternalID(ctx context.Context, source, externalID string) (*models.Place, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+placeColumns+`
		 FROM place
		 WHERE source = $1 AND external_id = $2 AND deleted_at IS NULL`,
		source, externalID)
	if err != nil {
		return nil, err
	}
	place, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Place])
	if err != nil {
		return nil, err
	}
	return &place, nil
}

// SearchVerified busca lugares creados o verificados en nuestra propia tabla
// (los que no vienen del catalogo externo). Solo devuelve filas verificadas.
func (r *pgPlaceRepository) SearchVerified(ctx context.Context, query string, limit int) ([]models.Place, error) {
	if limit <= 0 {
		limit = 10
	}

	rows, err := r.db.Query(ctx,
		`SELECT `+placeColumns+`
		 FROM place
		 WHERE deleted_at IS NULL
		   AND moderation_status = 'verified'
		   AND (
		         strpos(lower(name), lower($1)) > 0
		      OR strpos(lower(COALESCE(address, '')), lower($1)) > 0
		      OR similarity(name, $1) > 0.2
		   )
		 ORDER BY
		     (lower(name) = lower($1)) DESC,
		     starts_with(lower(name), lower($1)) DESC,
		     similarity(name, $1) DESC,
		     created_at DESC
		 LIMIT $2`,
		query, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.Place])
}

// FindSimilar busca un lugar ya existente (a menos de radiusMeters y con un
// nombre parecido) para no crear duplicados comunitarios. Devuelve
// pgx.ErrNoRows si no hay ninguno.
func (r *pgPlaceRepository) FindSimilar(ctx context.Context, lat, lng, radiusMeters float64, name string) (*models.Place, error) {
	if radiusMeters <= 0 {
		radiusMeters = 50
	}

	rows, err := r.db.Query(ctx,
		`SELECT `+placeColumns+`
		 FROM place
		 WHERE deleted_at IS NULL
		   AND moderation_status <> 'rejected'
		   AND latitude BETWEEN $1 - ($3 / 111320.0) AND $1 + ($3 / 111320.0)
		   AND longitude BETWEEN $2 - ($3 / (111320.0 * GREATEST(cos(radians($1)), 0.01)))
		                        AND $2 + ($3 / (111320.0 * GREATEST(cos(radians($1)), 0.01)))
		   AND similarity(name, $4) > 0.3
		 ORDER BY (6371 * acos(cos(radians($1)) * cos(radians(latitude)) *
		           cos(radians(longitude) - radians($2)) + sin(radians($1)) * sin(radians(latitude)))) ASC
		 LIMIT 1`,
		lat, lng, radiusMeters, name)
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
func (r *pgPlaceRepository) MarkPublishedTx(ctx context.Context, tx pgx.Tx, placeID int64) error {
	_, err := tx.Exec(ctx,
		`UPDATE place SET published = TRUE, updated_at = NOW()
		 WHERE id = $1 AND published = FALSE`, placeID)
	return err
}

func (r *pgPlaceRepository) Update(ctx context.Context, id int64, req models.UpdatePlaceRequest) (*models.Place, error) {
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

func (r *pgPlaceRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE place SET deleted_at = NOW() WHERE id = $1`, id)
	return err
}

// --- Agregacion de accesibilidad ---

// GetAccessibilityRows devuelve TODOS los criterios activos (con su dimension)
// para un lugar, con los conteos del cache (LEFT JOIN: 0 si no hay datos -> gris).
// Base del desglose completo de GET /places/:id.
func (r *pgPlaceRepository) GetAccessibilityRows(ctx context.Context, placeID int64) ([]models.CriterionAggRow, error) {
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
func (r *pgPlaceRepository) GetCriterionAggRow(ctx context.Context, placeID, criterionID int64) (*models.CriterionAggRow, error) {
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
func (r *pgPlaceRepository) GetAggRowsByPlaceIDs(ctx context.Context, placeIDs []int64) ([]models.CriterionAggRow, error) {
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
