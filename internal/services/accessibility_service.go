package services

import (
	"accesspath/internal/config"
	"accesspath/internal/models"
)

// AccessibilityService deriva estado (semaforo) y confianza a partir de los
// conteos del cache. Logica pura; umbrales desde config externo (nunca inline).
type AccessibilityService struct {
	cfg *config.AccessibilityThresholds
}

func NewAccessibilityService(cfg *config.AccessibilityThresholds) *AccessibilityService {
	return &AccessibilityService{cfg: cfg}
}

// severity ordena estados para el rollup (peor = mayor). NoData no participa.
func severity(s models.AccessibilityState) int {
	switch s {
	case models.StateRed:
		return 3
	case models.StateYellow:
		return 2
	case models.StateGreen:
		return 1
	default:
		return 0
	}
}

// DeriveCriterion aplica las reglas de la seccion 3 a una fila de agregacion.
func (s *AccessibilityService) DeriveCriterion(row models.CriterionAggRow) models.CriterionScore {
	t := s.cfg
	defined := row.NYes + row.NNo

	cs := models.CriterionScore{
		CriterionID: row.CriterionID,
		Key:         row.CriterionKey,
		Prompt:      row.Prompt,
		IsBlocking:  row.IsBlocking,
		ProfileTags: row.ProfileTags,
		NYes:        row.NYes,
		NNo:         row.NNo,
		NUnsure:     row.NUnsure,
		QualityP50:  row.QualityP50,
		Confidence:  s.confidence(row, defined),
	}

	if defined < t.ConfidenceMin {
		cs.State = models.StateNoData
		return cs
	}

	consenso := float64(row.NYes) / float64(defined)
	switch {
	case consenso < t.ConsensoNo:
		cs.State = models.StateRed
	case consenso >= t.ConsensoSi:
		// Solo verde con calidad buena confirmada. Existe pero pobre o sin
		// calidad -> amarillo (nunca verde).
		if row.QualityP50 != nil && *row.QualityP50 >= t.CalBuena {
			cs.State = models.StateGreen
		} else {
			cs.State = models.StateYellow
		}
	default:
		// Consenso intermedio: existe segun mayoria debil, marcado como conflicto.
		cs.State = models.StateYellow
		cs.Conflict = true
	}
	return cs
}

// confidence calcula el eje de confianza (no altera el color). La recencia
// (last_contribution_at) se expone como dato informativo, nunca como penalizacion.
func (s *AccessibilityService) confidence(row models.CriterionAggRow, defined int) models.Confidence {
	c := models.Confidence{
		NDefined:           defined,
		NUnsure:            row.NUnsure,
		LastContributionAt: row.LastContributionAt,
	}

	total := defined + row.NUnsure
	if total > 0 {
		c.PhotoRatio = float64(row.NPhotos) / float64(total)
	}

	switch {
	case defined == 0:
		c.Level = "none"
	case defined >= 5:
		c.Level = "high"
	case defined >= 2:
		c.Level = "medium"
	default:
		c.Level = "low"
	}
	// Una foto sube la confianza (no el estado).
	if c.PhotoRatio > 0 && c.Level == "low" {
		c.Level = "medium"
	}
	return c
}

// BuildDimensions agrupa las filas por dimension (ya vienen ordenadas), deriva
// cada criterio y hace el rollup por dimension.
func (s *AccessibilityService) BuildDimensions(rows []models.CriterionAggRow) []models.DimensionScore {
	dims := []models.DimensionScore{}
	index := map[int64]int{}

	for _, row := range rows {
		pos, ok := index[row.DimensionID]
		if !ok {
			pos = len(dims)
			index[row.DimensionID] = pos
			dims = append(dims, models.DimensionScore{
				DimensionID: row.DimensionID,
				Key:         row.DimensionKey,
				Name:        row.DimensionName,
				State:       models.StateNoData,
			})
		}
		dims[pos].Criteria = append(dims[pos].Criteria, s.DeriveCriterion(row))
	}

	for i := range dims {
		s.rollup(&dims[i])
	}
	return dims
}

// rollup fija el estado de la dimension: peor estado confirmado entre los
// criterios bloqueantes con datos; si no hay bloqueantes con datos, entre todos
// los criterios con datos. Los criterios sin datos no arrastran a rojo.
func (s *AccessibilityService) rollup(dim *models.DimensionScore) {
	var withData, blocking []models.CriterionScore
	for _, cr := range dim.Criteria {
		if cr.State == models.StateNoData {
			continue
		}
		withData = append(withData, cr)
		if cr.IsBlocking {
			blocking = append(blocking, cr)
		}
	}
	if len(withData) == 0 {
		dim.State = models.StateNoData
		return
	}
	candidates := withData
	if len(blocking) > 0 {
		candidates = blocking
	}

	worst := models.StateGreen
	conflict := false
	for _, cr := range candidates {
		if severity(cr.State) > severity(worst) {
			worst = cr.State
		}
		if cr.Conflict {
			conflict = true
		}
	}
	dim.State = worst
	dim.Conflict = conflict
}

// OverallState es el estado global del lugar (peor dimension con datos), para el
// color del marcador en el mapa.
func (s *AccessibilityService) OverallState(dims []models.DimensionScore) models.AccessibilityState {
	worst := models.StateNoData
	for _, d := range dims {
		if d.State == models.StateNoData {
			continue
		}
		if severity(d.State) > severity(worst) {
			worst = d.State
		}
	}
	return worst
}

// BuildPlaceAccessibility ensambla el bloque completo para el detalle.
func (s *AccessibilityService) BuildPlaceAccessibility(rows []models.CriterionAggRow) models.PlaceAccessibility {
	dims := s.BuildDimensions(rows)
	return models.PlaceAccessibility{
		Dimensions:   dims,
		OverallState: s.OverallState(dims),
	}
}
