package models

import "time"

// ValidNeedKeys son las necesidades funcionales permitidas. Deben casar con
// criterion.profile_tags. NO es un diagnostico medico.
var ValidNeedKeys = map[string]bool{
	"silla":              true,
	"movilidad_reducida": true,
	"baja_vision":        true,
	"ceguera":            true,
	"auditiva":           true,
	"cognitiva":          true,
	"sensorial":          true,
}

// UserProfileNeed es una necesidad funcional elegida por el usuario (opt-in).
type UserProfileNeed struct {
	ID        int64     `db:"id" json:"id"`
	UserID    int64     `db:"user_id" json:"user_id"`
	NeedKey   string    `db:"need_key" json:"need_key"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// ProfileResponse es GET /me/profile: necesidades + estado de consentimiento.
type ProfileResponse struct {
	Needs       []string   `json:"needs"`
	ConsentAt   *time.Time `json:"consent_at,omitempty"`
	HasConsent  bool       `json:"has_consent"`
}

// ProfileRequest es PUT /me/profile: fija necesidades + consentimiento opt-in.
type ProfileRequest struct {
	Needs   []string `json:"needs"`
	Consent bool     `json:"consent" binding:"required"`
}
