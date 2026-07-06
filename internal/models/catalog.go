package models

// Catalogo del formulario: dimension -> criterion -> answer_option.
// Reemplaza category/subcategory del modelo v1.

type Dimension struct {
	ID          int64   `db:"id" json:"id"`
	Code        string  `db:"code" json:"code"`
	Key         string  `db:"key" json:"key"`
	Name        string  `db:"name" json:"name"`
	Description *string `db:"description" json:"description,omitempty"`
	SortOrder   int     `db:"sort_order" json:"sort_order"`
	Active      bool    `db:"active" json:"active"`
}

type Criterion struct {
	ID                   int64    `db:"id" json:"id"`
	Code                 string   `db:"code" json:"code"`
	DimensionID          int64    `db:"dimension_id" json:"dimension_id"`
	Key                  string   `db:"key" json:"key"`
	Prompt               string   `db:"prompt" json:"prompt"`
	ProfileTags          []string `db:"profile_tags" json:"profile_tags,omitempty"`
	Weight               int      `db:"weight" json:"weight"`
	IsBlocking           bool     `db:"is_blocking" json:"is_blocking"`
	IsStarter            bool     `db:"is_starter" json:"is_starter"`
	DependsOnCriterionID *int64   `db:"depends_on_criterion_id" json:"depends_on_criterion_id,omitempty"`
	SortOrder            int      `db:"sort_order" json:"sort_order"`
	Active               bool     `db:"active" json:"active"`
}

type AnswerOption struct {
	ID           int64   `db:"id" json:"id"`
	CriterionID  int64   `db:"criterion_id" json:"criterion_id"`
	Label        string  `db:"label" json:"label"`
	ExistsValue  *bool   `db:"exists_value" json:"exists_value,omitempty"`
	QualityValue *int    `db:"quality_value" json:"quality_value,omitempty"`
	IsUnsure     bool    `db:"is_unsure" json:"is_unsure"`
	SortOrder    int     `db:"sort_order" json:"sort_order"`
}

// --- DTOs de lectura del catalogo (GET /dimensions) ---

// CriterionDetail es un criterio con sus opciones, para el catalogo.
type CriterionDetail struct {
	Criterion
	Options []AnswerOption `json:"options"`
}

// DimensionDetail es una dimension con sus criterios y opciones.
type DimensionDetail struct {
	Dimension
	Criteria []CriterionDetail `json:"criteria"`
}

// NextQuestionResponse es la pregunta elegida para el usuario (seccion 4).
// Criterion es nil cuando no hay nada util que preguntar.
type NextQuestionResponse struct {
	Criterion *Criterion     `json:"criterion"`
	Options   []AnswerOption `json:"options"`
}
