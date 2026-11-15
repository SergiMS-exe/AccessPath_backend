package catalog

import (
	"context"
	"errors"
)

// ErrNotFound indica que el (source, external_id) no existe en el catalogo.
var ErrNotFound = errors.New("catalog item not found")

// Provider abstrae el catalogo de lugares. Hoy solo hay una implementacion
// (PostgreSQL + Overture), pero la interfaz permite anadir Foursquare o un
// proveedor legacy sin tocar los servicios.
type Provider interface {
	// Source devuelve el identificador del proveedor ('overture', ...).
	Source() string
	// Search busca por texto libre. limit <= 0 usa un valor por defecto.
	Search(ctx context.Context, query string, limit int) ([]Item, error)
	// Details devuelve una ficha concreta del catalogo.
	Details(ctx context.Context, source, externalID string) (*Item, error)
	// Nearby devuelve candidatos dentro de un radio, ordenados por similitud
	// de nombre. La decision final (distancia exacta + score) la toma el
	// servicio, para que sea testeable sin base de datos.
	Nearby(ctx context.Context, lat, lng, radiusMeters float64, name string) ([]Item, error)
}
