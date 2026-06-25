package models

import "time"

type Place struct {
	ID             int64      `db:"id" json:"id"`
	Code           string     `db:"code" json:"code"`
	Name           string     `db:"name" json:"name"`
	Address        *string    `db:"address" json:"address,omitempty"`
	Latitude       float64    `db:"latitude" json:"latitude"`
	Longitude      float64    `db:"longitude" json:"longitude"`
	Description    *string    `db:"description" json:"description,omitempty"`
	GooglePlaceID  *string    `db:"google_place_id" json:"google_place_id,omitempty"`
	Published      bool       `db:"published" json:"published"`
	CreatedBy      int64      `db:"created_by" json:"created_by"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt      *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

type PlaceWithDistance struct {
	Place
	Distance float64 `db:"distance" json:"distance"`
}

// PlaceDetail is returned by GET /places/:id and embeds the rating cache.
type PlaceDetail struct {
	Place
	Ratings []CategoryRating `json:"ratings"`
}

type CreatePlaceRequest struct {
	Name          string  `json:"name" binding:"required"`
	Address       *string `json:"address"`
	Latitude      float64 `json:"latitude" binding:"required"`
	Longitude     float64 `json:"longitude" binding:"required"`
	Description   *string `json:"description"`
	GooglePlaceID *string `json:"google_place_id"`
	CreatedBy     int64   `json:"created_by" binding:"required"`
}

type ImportFromGoogleRequest struct {
	GooglePlaceID string `json:"google_place_id" binding:"required"`
	SessionToken  string `json:"session_token" binding:"required"`
}

type GoogleAutocompleteItem struct {
	PlaceID       string `json:"place_id"`
	Description   string `json:"description"`
	MainText      string `json:"main_text"`
	SecondaryText string `json:"secondary_text"`
}

type UpdatePlaceRequest struct {
	Name        string  `json:"name" binding:"required"`
	Address     *string `json:"address"`
	Latitude    float64 `json:"latitude" binding:"required"`
	Longitude   float64 `json:"longitude" binding:"required"`
	Description *string `json:"description"`
}

type PlaceFilters struct {
	Search     string
	CategoryID int64
	MinRating  float64
	Limit      int
	Offset     int
}

// PlaceListResult wraps the paginated list response for GET /places.
type PlaceListResult struct {
	Places []Place `json:"places"`
	Total  int     `json:"total"`
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
}

type BoundsFilter struct {
	MinLat     float64
	MaxLat     float64
	MinLng     float64
	MaxLng     float64
	CategoryID int64
	Limit      int
}

type NearbyFilter struct {
	Lat    float64
	Lng    float64
	Radius float64
	Limit  int
	Offset int
}
