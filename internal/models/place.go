package models

import "time"

type Place struct {
	ID            string    `json:"id"`
	GooglePlaceID *string   `json:"google_place_id,omitempty"`
	Name          string    `json:"name"`
	Address       *string   `json:"address,omitempty"`
	City          *string   `json:"city,omitempty"`
	Country       *string   `json:"country,omitempty"`
	Latitude      float64   `json:"latitude"`
	Longitude     float64   `json:"longitude"`
	PlaceType     *string   `json:"place_type,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type PlaceWithDistance struct {
	Place
	Distance float64 `json:"distance"`
}

type CreatePlaceRequest struct {
	GooglePlaceID *string `json:"google_place_id"`
	Name          string  `json:"name" binding:"required"`
	Address       *string `json:"address"`
	City          *string `json:"city"`
	Country       *string `json:"country"`
	Latitude      float64 `json:"latitude" binding:"required"`
	Longitude     float64 `json:"longitude" binding:"required"`
	PlaceType     *string `json:"place_type"`
}

type PlaceFilters struct {
	City   string
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
