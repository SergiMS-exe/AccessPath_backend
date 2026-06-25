package models

import "time"

type Photo struct {
	ID        int64      `db:"id" json:"id"`
	Code      string     `db:"code" json:"code"`
	ReviewID  int64      `db:"review_id" json:"review_id"`
	URL       string     `db:"url" json:"url"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}
