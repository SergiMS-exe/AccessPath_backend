package services_test

import (
	"testing"
	"time"

	"accesspath/internal/config"
	"accesspath/internal/models"
	"accesspath/internal/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// defaultThresholds usa los valores por defecto del paquete config, que es lo
// que la app carga si no hay archivo JSON externo. Asi los tests son
// deterministas y reproducen exactamente el comportamiento en runtime.
func defaultThresholds() *config.AccessibilityThresholds {
	return &config.AccessibilityThresholds{
		ConfidenceMin: 1,
		ConsensoSi:    0.60,
		ConsensoNo:    0.40,
		CalBuena:      4.0,
		CalRegular:    2.5,
		Selection: config.AccessibilitySelection{
			RelevanceMatch:   1.0,
			RelevanceNoMatch: 0.4,
			ConflictBonus:    0.5,
			TopK:             3,
		},
	}
}

func ptrFloat(v float64) *float64 { return &v }

func newSvc(t *testing.T) services.AccessibilityService {
	t.Helper()
	return services.NewAccessibilityService(defaultThresholds())
}

// baseRow devuelve una fila neutra: counts=0, sin quality, sin fecha. Los tests
// modifican solo los campos que les interesan.
func baseRow() models.CriterionAggRow {
	return models.CriterionAggRow{
		PlaceID:       1,
		DimensionID:   10,
		DimensionKey:  "mobility",
		DimensionName: "Movilidad",
		CriterionID:   100,
		CriterionKey:  "ramp",
		Prompt:        "Tiene rampa?",
		IsBlocking:    true,
		ProfileTags:   []string{"silla"},
	}
}

// --- DeriveCriterion -------------------------------------------------------

func TestDeriveCriterion_NoDataWhenDefinedBelowMin(t *testing.T) {
	svc := newSvc(t)
	row := baseRow() // defined=0 (< ConfidenceMin=1)

	cs := svc.DeriveCriterion(row)

	assert.Equal(t, models.StateNoData, cs.State)
	assert.Equal(t, row.CriterionID, cs.CriterionID)
	assert.Equal(t, row.CriterionKey, cs.Key)
	assert.Equal(t, row.Prompt, cs.Prompt)
	assert.Equal(t, row.IsBlocking, cs.IsBlocking)
	assert.Equal(t, row.ProfileTags, cs.ProfileTags)
}

func TestDeriveCriterion_RedWhenConsensusBelowNo(t *testing.T) {
	svc := newSvc(t)
	row := baseRow()
	row.NYes = 1
	row.NNo = 9 // defined=10, consensus=0.1 < 0.40 (ConsensoNo)

	cs := svc.DeriveCriterion(row)

	assert.Equal(t, models.StateRed, cs.State)
	assert.False(t, cs.Conflict, "rojo no se marca como conflicto (consenso claro)")
}

func TestDeriveCriterion_GreenWhenConsensusYesAndQualityBuena(t *testing.T) {
	svc := newSvc(t)
	row := baseRow()
	row.NYes = 8
	row.NNo = 2 // consensus=0.8 >= 0.60 (ConsensoSi)
	row.QualityP50 = ptrFloat(4.5)

	cs := svc.DeriveCriterion(row)

	assert.Equal(t, models.StateGreen, cs.State)
	assert.False(t, cs.Conflict)
}

func TestDeriveCriterion_YellowWhenConsensusYesButQualityMissing(t *testing.T) {
	svc := newSvc(t)
	row := baseRow()
	row.NYes = 8
	row.NNo = 2
	row.QualityP50 = nil // existe pero sin calidad -> amarillo, nunca verde

	cs := svc.DeriveCriterion(row)

	assert.Equal(t, models.StateYellow, cs.State)
	assert.False(t, cs.Conflict, "amarillo por calidad faltante NO es conflicto")
}

func TestDeriveCriterion_YellowWhenConsensusYesButQualityBelowBuena(t *testing.T) {
	svc := newSvc(t)
	row := baseRow()
	row.NYes = 8
	row.NNo = 2
	row.QualityP50 = ptrFloat(3.0) // < CalBuena (4.0)

	cs := svc.DeriveCriterion(row)

	assert.Equal(t, models.StateYellow, cs.State)
}

func TestDeriveCriterion_YellowWithConflictWhenConsensusMiddle(t *testing.T) {
	svc := newSvc(t)
	row := baseRow()
	row.NYes = 5
	row.NNo = 5 // consensus=0.5, entre 0.40 y 0.60

	cs := svc.DeriveCriterion(row)

	assert.Equal(t, models.StateYellow, cs.State)
	assert.True(t, cs.Conflict, "consenso intermedio SI se marca como conflicto")
}

func TestDeriveCriterion_PopulatesConfidenceCounts(t *testing.T) {
	svc := newSvc(t)
	row := baseRow()
	row.NYes = 5
	row.NNo = 3
	row.NUnsure = 2

	cs := svc.DeriveCriterion(row)

	assert.Equal(t, 8, cs.Confidence.NDefined, "NDefined = NYes + NNo")
	assert.Equal(t, 2, cs.Confidence.NUnsure)
}

// --- confidence (vía DeriveCriterion, único punto de entrada público) ------

func TestConfidence_NoneWhenNoDefined(t *testing.T) {
	svc := newSvc(t)
	row := baseRow()
	row.NYes = 0
	row.NNo = 0
	row.NUnsure = 5

	cs := svc.DeriveCriterion(row)

	assert.Equal(t, "none", cs.Confidence.Level)
}

func TestConfidence_LowForOneDefined(t *testing.T) {
	svc := newSvc(t)
	row := baseRow()
	row.NYes = 1
	row.NNo = 0 // defined=1, n_unsure=0 -> photo_ratio=0

	cs := svc.DeriveCriterion(row)

	assert.Equal(t, "low", cs.Confidence.Level)
}

func TestConfidence_LowBecomesMediumIfHasPhoto(t *testing.T) {
	svc := newSvc(t)
	row := baseRow()
	row.NYes = 1
	row.NNo = 0
	row.NPhotos = 1 // photo_ratio = 1/1 = 1.0 > 0 -> sube low a medium

	cs := svc.DeriveCriterion(row)

	assert.Equal(t, "medium", cs.Confidence.Level)
}

func TestConfidence_MediumForTwoOrFour(t *testing.T) {
	svc := newSvc(t)
	row := baseRow()
	row.NYes = 3
	row.NNo = 1 // defined=4

	cs := svc.DeriveCriterion(row)

	assert.Equal(t, "medium", cs.Confidence.Level)
}

func TestConfidence_HighForFiveOrMore(t *testing.T) {
	svc := newSvc(t)
	row := baseRow()
	row.NYes = 5
	row.NNo = 0 // defined=5

	cs := svc.DeriveCriterion(row)

	assert.Equal(t, "high", cs.Confidence.Level)
}

func TestConfidence_PhotoRatioConsidersUnsure(t *testing.T) {
	svc := newSvc(t)
	row := baseRow()
	row.NYes = 2
	row.NNo = 0
	row.NUnsure = 2
	row.NPhotos = 1 // ratio = 1 / (2+2) = 0.25

	cs := svc.DeriveCriterion(row)

	assert.InDelta(t, 0.25, cs.Confidence.PhotoRatio, 0.001)
	assert.Equal(t, "medium", cs.Confidence.Level)
}

func TestConfidence_PreservesLastContributionAt(t *testing.T) {
	svc := newSvc(t)
	when := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	row := baseRow()
	row.NYes = 5
	row.LastContributionAt = &when

	cs := svc.DeriveCriterion(row)

	assert.NotNil(t, cs.Confidence.LastContributionAt)
	assert.Equal(t, when, *cs.Confidence.LastContributionAt)
}

// --- BuildDimensions / rollup ---------------------------------------------

func TestBuildDimensions_GroupsByDimensionAndOrdersByInput(t *testing.T) {
	svc := newSvc(t)

	rows := []models.CriterionAggRow{
		baseRow(), // dim 10, crit 100
		func() models.CriterionAggRow {
			r := baseRow()
			r.DimensionID = 10
			r.CriterionID = 101
			r.CriterionKey = "elevator"
			r.NYes = 1
			return r
		}(),
		func() models.CriterionAggRow {
			r := baseRow()
			r.DimensionID = 20
			r.DimensionKey = "visual"
			r.DimensionName = "Visual"
			r.CriterionID = 200
			r.CriterionKey = "braille"
			r.NYes = 1
			return r
		}(),
	}

	dims := svc.BuildDimensions(rows)

	assert.Len(t, dims, 2, "debe haber dos dimensiones (10 y 20)")
	assert.Equal(t, int64(10), dims[0].DimensionID, "orden por DimensionID del input")
	assert.Equal(t, int64(20), dims[1].DimensionID)
	assert.Len(t, dims[0].Criteria, 2)
	assert.Len(t, dims[1].Criteria, 1)
}

func TestBuildDimensions_RollupIsWorstBlocking(t *testing.T) {
	svc := newSvc(t)

	rows := []models.CriterionAggRow{
		func() models.CriterionAggRow {
			r := baseRow()
			r.IsBlocking = true
			r.NYes = 1
			r.NNo = 9 // red
			return r
		}(),
		func() models.CriterionAggRow {
			r := baseRow()
			r.CriterionID = 101
			r.IsBlocking = true
			r.NYes = 10 // green
			return r
		}(),
		func() models.CriterionAggRow {
			r := baseRow()
			r.CriterionID = 102
			r.IsBlocking = false
			r.NYes = 1
			r.NNo = 0 // green, no bloqueante
			return r
		}(),
	}

	dims := svc.BuildDimensions(rows)

	require.Len(t, dims, 1)
	assert.Equal(t, models.StateRed, dims[0].State,
		"rollup = peor entre bloqueantes con datos")
}

func TestBuildDimensions_RollupFallsBackToAllWhenNoBlockingWithData(t *testing.T) {
	svc := newSvc(t)

	rows := []models.CriterionAggRow{
		func() models.CriterionAggRow {
			r := baseRow()
			r.IsBlocking = false
			r.NYes = 1
			r.NNo = 9 // red, no bloqueante
			return r
		}(),
		func() models.CriterionAggRow {
			r := baseRow()
			r.CriterionID = 101
			r.IsBlocking = false
			r.NYes = 10 // green, no bloqueante
			return r
		}(),
	}

	dims := svc.BuildDimensions(rows)

	require.Len(t, dims, 1)
	assert.Equal(t, models.StateRed, dims[0].State,
		"sin bloqueantes con datos, rollup = peor entre todos")
}

func TestBuildDimensions_RollupNoDataWhenAllCriteriaNoData(t *testing.T) {
	svc := newSvc(t)
	// baseRow tiene defined=0 -> cada criterio es NoData
	dims := svc.BuildDimensions([]models.CriterionAggRow{baseRow()})

	require.Len(t, dims, 1)
	assert.Equal(t, models.StateNoData, dims[0].State,
		"si todos los criterios son NoData, dimension = NoData")
}

func TestBuildDimensions_RollupPropagatesConflict(t *testing.T) {
	svc := newSvc(t)

	rows := []models.CriterionAggRow{
		func() models.CriterionAggRow {
			r := baseRow()
			r.NYes = 5
			r.NNo = 5 // yellow con conflict=true
			return r
		}(),
	}

	dims := svc.BuildDimensions(rows)

	require.Len(t, dims, 1)
	assert.Equal(t, models.StateYellow, dims[0].State)
	assert.True(t, dims[0].Conflict, "la dimension propaga el conflict")
}

func TestBuildDimensions_EmptyInputProducesEmptyResult(t *testing.T) {
	svc := newSvc(t)

	dims := svc.BuildDimensions(nil)

	assert.Empty(t, dims)
}

// --- OverallState ----------------------------------------------------------

func TestOverallState_NoDataForEmptyDims(t *testing.T) {
	svc := newSvc(t)

	got := svc.OverallState(nil)

	assert.Equal(t, models.StateNoData, got)
}

func TestOverallState_AllNoDataYieldsNoData(t *testing.T) {
	svc := newSvc(t)

	dims := []models.DimensionScore{
		{State: models.StateNoData},
		{State: models.StateNoData},
	}

	got := svc.OverallState(dims)

	assert.Equal(t, models.StateNoData, got)
}

func TestOverallState_PicksWorstNonNoData(t *testing.T) {
	svc := newSvc(t)

	dims := []models.DimensionScore{
		{State: models.StateGreen},
		{State: models.StateYellow},
		{State: models.StateRed},
		{State: models.StateNoData}, // ignorado
	}

	got := svc.OverallState(dims)

	assert.Equal(t, models.StateRed, got)
}

// --- BuildPlaceAccessibility ----------------------------------------------

func TestBuildPlaceAccessibility_PopulatesOverallState(t *testing.T) {
	svc := newSvc(t)

	rows := []models.CriterionAggRow{
		func() models.CriterionAggRow {
			r := baseRow()
			r.NYes = 8
			r.NNo = 2 // green
			return r
		}(),
		func() models.CriterionAggRow {
			r := baseRow()
			r.DimensionID = 20
			r.CriterionID = 200
			r.NYes = 1
			r.NNo = 9 // red
			return r
		}(),
	}

	got := svc.BuildPlaceAccessibility(rows)

	assert.NotNil(t, got.Dimensions)
	assert.Equal(t, models.StateRed, got.OverallState)
}

func TestBuildPlaceAccessibility_AllNoDataOverallIsNoData(t *testing.T) {
	svc := newSvc(t)
	rows := []models.CriterionAggRow{baseRow()} // defined=0

	got := svc.BuildPlaceAccessibility(rows)

	assert.Equal(t, models.StateNoData, got.OverallState)
}
