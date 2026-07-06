package models

import "time"

// AccessibilityState es el semaforo de 4 estados. NoData (gris) es "sin datos",
// distinto de rojo. Nunca se colapsa a una media o numero global.
type AccessibilityState string

const (
	StateGreen  AccessibilityState = "green"
	StateYellow AccessibilityState = "yellow"
	StateRed    AccessibilityState = "red"
	StateNoData AccessibilityState = "no_data"
)

// Confidence es el eje SEPARADO del score. No altera el color.
// LastContributionAt es informativo ("ultima valoracion: hace X"): la recencia
// se muestra para que el usuario juzgue, nunca penaliza ni decae el estado.
type Confidence struct {
	Level              string     `json:"level"` // none | low | medium | high
	NDefined           int        `json:"n_defined"`
	NUnsure            int        `json:"n_unsure"`
	PhotoRatio         float64    `json:"photo_ratio"`
	LastContributionAt *time.Time `json:"last_contribution_at,omitempty"`
}

// CriterionScore es el estado derivado de un criterio en un lugar.
// ProfileTags permite al cliente agrupar/priorizar por las necesidades del
// perfil del usuario (p.ej. "para ti" en el detalle).
type CriterionScore struct {
	CriterionID int64              `json:"criterion_id"`
	Key         string             `json:"key"`
	Prompt      string             `json:"prompt"`
	IsBlocking  bool               `json:"is_blocking"`
	ProfileTags []string           `json:"profile_tags,omitempty"`
	State       AccessibilityState `json:"state"`
	Conflict    bool               `json:"conflict"`
	NYes        int                `json:"n_yes"`
	NNo         int                `json:"n_no"`
	NUnsure     int                `json:"n_unsure"`
	QualityP50  *float64           `json:"quality_p50,omitempty"`
	Confidence  Confidence         `json:"confidence"`
}

// DimensionScore es el rollup (NO media) de una dimension y su desglose.
type DimensionScore struct {
	DimensionID int64              `json:"dimension_id"`
	Key         string             `json:"key"`
	Name        string             `json:"name"`
	State       AccessibilityState `json:"state"`
	Conflict    bool               `json:"conflict"`
	Criteria    []CriterionScore   `json:"criteria,omitempty"`
}

// PlaceAccessibility es el bloque de accesibilidad del detalle de un lugar.
type PlaceAccessibility struct {
	Dimensions   []DimensionScore   `json:"dimensions"`
	OverallState AccessibilityState `json:"overall_state"`
}

// CriterionAggRow es una fila del join criterion+dimension+cache, antes de
// derivar estado en el servicio. Los conteos vienen COALESCE'd a 0 en SQL.
type CriterionAggRow struct {
	PlaceID            int64      `db:"place_id"`
	DimensionID        int64      `db:"dimension_id"`
	DimensionKey       string     `db:"dimension_key"`
	DimensionName      string     `db:"dimension_name"`
	DimensionSort      int        `db:"dimension_sort"`
	CriterionID        int64      `db:"criterion_id"`
	CriterionKey       string     `db:"criterion_key"`
	Prompt             string     `db:"prompt"`
	IsBlocking         bool       `db:"is_blocking"`
	ProfileTags        []string   `db:"profile_tags"`
	CriterionSort      int        `db:"criterion_sort"`
	NYes               int        `db:"n_yes"`
	NNo                int        `db:"n_no"`
	NUnsure            int        `db:"n_unsure"`
	QualityP50         *float64   `db:"quality_p50"`
	NPhotos            int        `db:"n_photos"`
	LastContributionAt *time.Time `db:"last_contribution_at"`
}

// PlaceMapItem es el resumen por lugar para GET /places/map: place + estado por
// dimension (sin desglose criterio a criterio) + estado global del marcador.
type PlaceMapItem struct {
	Place
	Dimensions   []DimensionScore   `json:"dimensions"`
	OverallState AccessibilityState `json:"overall_state"`
}
