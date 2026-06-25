package models

import "time"

type Collection struct {
	ID        int64      `db:"id" json:"id"`
	Code      string     `db:"code" json:"code"`
	UserID    int64      `db:"user_id" json:"user_id"`
	Name      string     `db:"name" json:"name"`
	IsDefault bool       `db:"is_default" json:"is_default"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
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
