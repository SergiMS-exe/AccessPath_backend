package models

import "time"

type Collection struct {
	ID        int64      `json:"id"`
	Code      string     `json:"code"`
	UserID    int64      `json:"user_id"`
	Name      string     `json:"name"`
	IsDefault bool       `json:"is_default"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type CollectionPlace struct {
	CollectionID int64     `json:"collection_id"`
	PlaceID      int64     `json:"place_id"`
	AddedAt      time.Time `json:"added_at"`
}

type CollectionWithPlaces struct {
	Collection
	Places []Place `json:"places"`
}

type CreateCollectionRequest struct {
	UserID    int64  `json:"user_id" binding:"required"`
	Name      string `json:"name" binding:"required"`
	IsDefault bool   `json:"is_default"`
}
