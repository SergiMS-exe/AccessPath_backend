package catalog

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SourceOverture es el source con el que se cargan los datos de Overture Maps.
const SourceOverture = "overture"

// OperatingStatusPermanentlyClosed es el valor de operating_status de Overture
// para un sitio cerrado definitivamente. No debe importarse.
const OperatingStatusPermanentlyClosed = "permanently_closed"

// itemColumns es la proyeccion comun a todas las consultas del catalogo.
// El mapeo a Item es por nombre (tags `db`), no por orden.
const itemColumns = `id, source, name, address, city, region, postcode, country,
	latitude, longitude, category, subcategory, confidence, operating_status,
	phones, websites, refreshed_at`

// postgresProvider implementa Provider sobre la tabla geo.place_catalog.
type postgresProvider struct {
	db     *pgxpool.Pool
	source string
}

// NewPostgresProvider construye el proveedor sobre el catalogo cargado en
// PostgreSQL. source permite reutilizar la misma tabla para otras fuentes.
func NewPostgresProvider(db *pgxpool.Pool, source string) Provider {
	if source == "" {
		source = SourceOverture
	}
	return &postgresProvider{db: db, source: source}
}

var _ Provider = (*postgresProvider)(nil)

func (p *postgresProvider) Source() string { return p.source }

func (p *postgresProvider) Search(ctx context.Context, query string, limit int) ([]Item, error) {
	if limit <= 0 {
		limit = 10
	}

	rows, err := p.db.Query(ctx,
		`SELECT `+itemColumns+`
		 FROM geo.place_catalog
		 WHERE COALESCE(operating_status, 'open') <> 'permanently_closed'
		   AND (
		         strpos(lower(name), lower($1)) > 0
		      OR strpos(lower(COALESCE(address, '')), lower($1)) > 0
		      OR similarity(name, $1) > 0.2
		   )
		 ORDER BY
		     (lower(name) = lower($1)) DESC,
		     starts_with(lower(name), lower($1)) DESC,
		     similarity(name, $1) DESC,
		     confidence DESC NULLS LAST,
		     refreshed_at DESC NULLS LAST
		 LIMIT $2`,
		query, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[Item])
}

func (p *postgresProvider) Details(ctx context.Context, source, externalID string) (*Item, error) {
	if source == "" {
		source = p.source
	}

	rows, err := p.db.Query(ctx,
		`SELECT `+itemColumns+`
		 FROM geo.place_catalog
		 WHERE source = $1 AND id = $2`,
		source, externalID)
	if err != nil {
		return nil, err
	}
	item, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[Item])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (p *postgresProvider) Nearby(ctx context.Context, lat, lng, radiusMeters float64, name string) ([]Item, error) {
	minLat, maxLat, minLng, maxLng := BoundingBox(lat, lng, radiusMeters)

	rows, err := p.db.Query(ctx,
		`SELECT `+itemColumns+`
		 FROM geo.place_catalog
		 WHERE latitude BETWEEN $1 AND $2
		   AND longitude BETWEEN $3 AND $4
		   AND COALESCE(operating_status, 'open') <> 'permanently_closed'
		   AND similarity(name, $5) > 0.1
		 ORDER BY similarity(name, $5) DESC, confidence DESC NULLS LAST
		 LIMIT 20`,
		minLat, maxLat, minLng, maxLng, name)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[Item])
}
