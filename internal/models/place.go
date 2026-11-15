package models

import "time"

type Place struct {
	ID          int64   `db:"id" json:"id"`
	Code        string  `db:"code" json:"code"`
	Name        string  `db:"name" json:"name"`
	Address     *string `db:"address" json:"address,omitempty"`
	Latitude    float64 `db:"latitude" json:"latitude"`
	Longitude   float64 `db:"longitude" json:"longitude"`
	Description *string `db:"description" json:"description,omitempty"`
	// Source indica el origen del dato: 'overture' o 'community'.
	Source string `db:"source" json:"source"`
	// ExternalID es el id del sitio en el proveedor de origen (GERS id, place id).
	ExternalID *string `db:"external_id" json:"external_id,omitempty"`
	// GoogleFTID es el feature id del enlace de Google; se guarda como
	// trazabilidad y para futuras comprobaciones, no para deduplicar.
	GoogleFTID *string `db:"google_ftid" json:"google_ftid,omitempty"`
	// SourceURL es el enlace que aporto el usuario, si lo hubo.
	SourceURL *string `db:"source_url" json:"source_url,omitempty"`
	// Published significa "tiene datos de accesibilidad" (NO es moderacion).
	Published bool `db:"published" json:"published"`
	// ModerationStatus: verified | pending | rejected.
	ModerationStatus string     `db:"moderation_status" json:"moderation_status"`
	CreatedBy        int64      `db:"created_by" json:"created_by"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt        *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

type PlaceWithDistance struct {
	Place
	Distance float64 `db:"distance" json:"distance"`
}

// PlaceDetail is returned by GET /places/:id: place + accessibility breakdown
// (por dimension y criterio) + submissions ("que cuenta la gente").
type PlaceDetail struct {
	Place
	Accessibility PlaceAccessibility      `json:"accessibility"`
	Submissions   []SubmissionWithDetails `json:"submissions"`
}

// CreatePlaceRequest: created_by NO viene del body, se deriva del token.
// Los campos marcados con json:"-" los rellena el servicio (importacion desde
// catalogo o creacion comunitaria); un cliente no puede fijarlos.
type CreatePlaceRequest struct {
	Name        string  `json:"name" binding:"required"`
	Address     *string `json:"address"`
	Latitude    float64 `json:"latitude" binding:"required"`
	Longitude   float64 `json:"longitude" binding:"required"`
	Description *string `json:"description"`
	CreatedBy   int64   `json:"-"`

	Source           *string `json:"-"`
	ExternalID       *string `json:"-"`
	GoogleFTID       *string `json:"-"`
	SourceURL        *string `json:"-"`
	ModerationStatus *string `json:"-"`
}

// PlaceSearchItem es un resultado del buscador de lugares. Los cuatro
// primeros campos son el contrato que ya consume la app movil; el resto
// permite importar la ficha sin ambiguedad.
type PlaceSearchItem struct {
	PlaceID       string `json:"place_id"`
	Description   string `json:"description"`
	MainText      string `json:"main_text"`
	SecondaryText string `json:"secondary_text"`

	Source         string   `json:"source"`
	ExternalID     string   `json:"external_id"`
	Address        *string  `json:"address,omitempty"`
	Latitude       *float64 `json:"latitude,omitempty"`
	Longitude      *float64 `json:"longitude,omitempty"`
	Category       *string  `json:"category,omitempty"`
	DistanceMeters *float64 `json:"distance_meters,omitempty"`
	// LocalPlaceID se rellena si el sitio ya existe en nuestra tabla place.
	LocalPlaceID *int64 `json:"local_place_id,omitempty"`
}

type UpdatePlaceRequest struct {
	Name        string  `json:"name" binding:"required"`
	Address     *string `json:"address"`
	Latitude    float64 `json:"latitude" binding:"required"`
	Longitude   float64 `json:"longitude" binding:"required"`
	Description *string `json:"description"`
}

type PlaceFilters struct {
	Search string
	Limit  int
	Offset int
}

// SearchOptions agrupa los parametros del buscador de lugares. Latitude y
// Longitude son opcionales: si vienen, cada resultado incluye la distancia.
type SearchOptions struct {
	Query     string
	Latitude  *float64
	Longitude *float64
	Limit     int
}

// PlaceListResult wraps the paginated list response for GET /places.
type PlaceListResult struct {
	Places []Place `json:"places"`
	Total  int     `json:"total"`
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
}

type BoundsFilter struct {
	MinLat float64
	MaxLat float64
	MinLng float64
	MaxLng float64
	Limit  int
}

type NearbyFilter struct {
	Lat    float64
	Lng    float64
	Radius float64
	Limit  int
	Offset int
}
