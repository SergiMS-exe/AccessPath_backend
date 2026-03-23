package models

import "time"

type Photo struct {
	ID        int64      `json:"id"`
	Code      string     `json:"code"`
	ReviewID  int64      `json:"review_id"`
	URL       string     `json:"url"`
	CreatedAt time.Time  `json:"created_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}
