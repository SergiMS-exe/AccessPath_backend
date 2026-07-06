package models

import "time"

// Submission es la valoracion VIVA de un usuario sobre un lugar. Unica por
// (user, place); nace al pulsar "Quiero valorar" y se edita con el tiempo. El
// comentario y las fotos son opcionales; las contribuciones cuelgan de ella.
type Submission struct {
	ID        int64      `db:"id" json:"id"`
	Code      string     `db:"code" json:"code"`
	UserID    int64      `db:"user_id" json:"user_id"`
	PlaceID   int64      `db:"place_id" json:"place_id"`
	Comment   *string    `db:"comment" json:"comment,omitempty"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

// SubmissionWithDetails aniade el autor y las fotos para "que cuenta la gente".
type SubmissionWithDetails struct {
	Submission
	Username string  `db:"username" json:"username"`
	Photos   []Photo `db:"-" json:"photos"`
}

// SubmissionRequest es el body de PUT /submissions. user_id sale del token.
// get-or-create de la valoracion del usuario para el lugar; set comentario y/o
// adjuntar fotos base64. Las contribuciones ya cuelgan de la submission por si
// mismas (no se enlazan aqui).
type SubmissionRequest struct {
	PlaceID int64        `json:"place_id" binding:"required"`
	Comment *string      `json:"comment"`
	Photos  []PhotoInput `json:"photos"`
}

// PhotoInput es una foto en base64, opcionalmente asociada a una contribucion
// concreta y a un slot sugerido (p.ej. la dimension que ilustra).
type PhotoInput struct {
	Data           string `json:"data" binding:"required"` // base64
	ContributionID *int64 `json:"contribution_id"`
	SuggestedSlot  *string `json:"suggested_slot"`
}
