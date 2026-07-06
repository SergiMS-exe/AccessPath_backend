package models

import "time"

// Photo pertenece a una submission; contribution_id es opcional (evidencia de
// un hecho concreto). suggested_slot marca en que contexto mostrarla.
type Photo struct {
	ID             int64      `db:"id" json:"id"`
	Code           string     `db:"code" json:"code"`
	SubmissionID   int64      `db:"submission_id" json:"submission_id"`
	ContributionID *int64     `db:"contribution_id" json:"contribution_id,omitempty"`
	URL            string     `db:"url" json:"url"`
	ObjectKey      *string    `db:"object_key" json:"object_key,omitempty"`
	SuggestedSlot  *string    `db:"suggested_slot" json:"suggested_slot,omitempty"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	DeletedAt      *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}
