package models

import "time"

type Place struct {
	ID          int64      `json:"id"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Address     *string    `json:"address,omitempty"`
	Latitude    float64    `json:"latitude"`
	Longitude   float64    `json:"longitude"`
	Description *string    `json:"description,omitempty"`
	CreatedBy   int64      `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type PlaceWithDistance struct {
	Place
	Distance float64 `json:"distance"`
}

// PlaceDetail is returned by GET /places/:id and embeds the rating cache.
type PlaceDetail struct {
	Place
	Ratings []CategoryRating `json:"ratings"`
}

type CreatePlaceRequest struct {
	Name        string  `json:"name" binding:"required"`
	Address     *string `json:"address"`
	Latitude    float64 `json:"latitude" binding:"required"`
	Longitude   float64 `json:"longitude" binding:"required"`
	Description *string `json:"description"`
	CreatedBy   int64   `json:"created_by" binding:"required"`
}

type UpdatePlaceRequest struct {
	Name        string  `json:"name" binding:"required"`
	Address     *string `json:"address"`
	Latitude    float64 `json:"latitude" binding:"required"`
	Longitude   float64 `json:"longitude" binding:"required"`
	Description *string `json:"description"`
}

type PlaceFilters struct {
	Limit  int
	Offset int
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
