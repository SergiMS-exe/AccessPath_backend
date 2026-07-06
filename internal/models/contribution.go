package models

import "time"

// Contribution es una respuesta atomica del usuario a un criterio en un lugar.
// Siempre cuelga de la submission viva del usuario para ese lugar (submission_id
// NOT NULL). exists_flag y quality se copian de la opcion al crear (estables).
// updated_at es la fecha de validez del dato (se refresca al editar in-place).
type Contribution struct {
	ID             int64      `db:"id" json:"id"`
	Code           string     `db:"code" json:"code"`
	SubmissionID   int64      `db:"submission_id" json:"submission_id"`
	UserID         int64      `db:"user_id" json:"user_id"`
	PlaceID        int64      `db:"place_id" json:"place_id"`
	CriterionID    int64      `db:"criterion_id" json:"criterion_id"`
	AnswerOptionID int64      `db:"answer_option_id" json:"answer_option_id"`
	ExistsFlag     *bool      `db:"exists_flag" json:"exists,omitempty"`
	Quality        *int       `db:"quality" json:"quality,omitempty"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt      *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

// AnsweredContribution es una respuesta viva del usuario en un lugar, usada por
// el algoritmo de seleccion (exclusion de respondidos, depends_on por respuesta propia).
type AnsweredContribution struct {
	CriterionID int64 `db:"criterion_id"`
	ExistsFlag  *bool `db:"exists_flag"`
}

// ContributionRequest es el body de POST /contributions. user_id sale del token.
type ContributionRequest struct {
	PlaceID        int64 `json:"place_id" binding:"required"`
	CriterionID    int64 `json:"criterion_id" binding:"required"`
	AnswerOptionID int64 `json:"answer_option_id" binding:"required"`
}

// ContributionResult devuelve el semaforo en vivo tras crear/borrar una contribucion:
// el estado nuevo del criterio y de su dimension.
type ContributionResult struct {
	ContributionID *int64          `json:"contribution_id,omitempty"`
	Criterion      CriterionScore  `json:"criterion"`
	Dimension      DimensionScore  `json:"dimension"`
}
