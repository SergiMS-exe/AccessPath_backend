package models

import "accesspath/pkg/catalog"

// CatalogItem es una ficha del catalogo externo (Overture Maps).
// El alias evita duplicar el struct entre la capa de datos y la de modelos.
type CatalogItem = catalog.Item

// ImportFromCatalogRequest importa un sitio del catalogo a nuestra tabla place.
//
// Dos modos excluyentes:
//   - source + external_id: importa la ficha del catalogo.
//   - place_id: devuelve ese lugar local (el buscador ya lo devolvio).
//
// source es opcional: si falta se usa el proveedor activo.
type ImportFromCatalogRequest struct {
	Source     string `json:"source"`
	ExternalID string `json:"external_id"`
	PlaceID    int64  `json:"place_id"`
}

// ResolveLinkRequest es el body de POST /places/resolve-link.
type ResolveLinkRequest struct {
	URL string `json:"url" binding:"required"`
}

// PlacePrefill son los datos que el backend puede prerellenar a partir de un
// enlace cuando el sitio NO esta en el catalogo.
type PlacePrefill struct {
	Name       string   `json:"name"`
	Address    *string  `json:"address,omitempty"`
	Latitude   *float64 `json:"latitude,omitempty"`
	Longitude  *float64 `json:"longitude,omitempty"`
	Category   *string  `json:"category,omitempty"`
	GoogleFTID string   `json:"google_ftid,omitempty"`
	SourceURL  string   `json:"source_url,omitempty"`
}

// ResolvedLink es la respuesta de POST /places/resolve-link.
//
//   - matched=true  -> catalog trae la ficha de Overture; el cliente puede
//     importarla con POST /places/from-catalog.
//   - matched=false -> prefill trae nombre/coordenadas para que el usuario
//     complete el formulario y cree el sitio (queda pendiente de moderacion).
type ResolvedLink struct {
	Matched bool          `json:"matched"`
	Catalog *CatalogItem  `json:"catalog,omitempty"`
	Prefill *PlacePrefill `json:"prefill,omitempty"`
	// Message explica por que no hubo match (util para la UI).
	Message string `json:"message,omitempty"`
}

// CreateCommunityPlaceRequest es el body de POST /places/community: el sitio
// que el usuario no encontro, creado a mano a partir de un enlace de Google.
type CreateCommunityPlaceRequest struct {
	Name        string  `json:"name" binding:"required"`
	Address     string  `json:"address"`
	Latitude    float64 `json:"latitude" binding:"required"`
	Longitude   float64 `json:"longitude" binding:"required"`
	Category    *string `json:"category"`
	Description *string `json:"description"`
	SourceURL   *string `json:"source_url"`
	GoogleFTID  *string `json:"google_ftid"`
}
