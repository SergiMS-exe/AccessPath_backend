package services_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"accesspath/internal/models"
	"accesspath/internal/services"
	"accesspath/pkg/apperr"
	"accesspath/pkg/catalog"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"accesspath/tests/internal/fakes"
)

// placeSvcDeps agrupa todos los colaboradores de PlaceService para tests.
type placeSvcDeps struct {
	repo          *fakes.RepoPlace
	accSvc        services.AccessibilityService
	submissionSvc *fakes.SvcSubmission
	catalog       *fakes.CatalogProvider
	links         *fakes.LinkResolver
}

func newPlaceDeps() placeSvcDeps {
	return placeSvcDeps{
		repo:          &fakes.RepoPlace{},
		accSvc:        services.NewAccessibilityService(defaultThresholds()),
		submissionSvc: &fakes.SvcSubmission{},
	}
}

func newPlaceService(d placeSvcDeps, monthlyLimit int) services.PlaceService {
	// Un puntero nil pasado directamente a una interfaz no es una interfaz
	// nil: convertimos explicitamente para poder probar el modo legacy.
	var provider catalog.Provider
	if d.catalog != nil {
		provider = d.catalog
	}
	var links services.LinkResolver
	if d.links != nil {
		links = d.links
	}
	return services.NewPlaceService(
		d.repo, d.accSvc, d.submissionSvc, provider, links,
	)
}

// --- GetAll ----------------------------------------------------------------

func TestPlaceService_GetAll_HappyPath(t *testing.T) {
	d := newPlaceDeps()
	d.repo.FindAllFn = func(_ context.Context, f models.PlaceFilters) ([]models.Place, int, error) {
		assert.Equal(t, 10, f.Limit, "filtros del handler llegan al repo")
		return []models.Place{{ID: 1, Name: "x"}}, 1, nil
	}
	svc := newPlaceService(d, 0)

	got, err := svc.GetAll(context.Background(), models.PlaceFilters{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, got.Places, 1)
	assert.Equal(t, 1, got.Total)
}

func TestPlaceService_GetAll_RepoErrorWrapped(t *testing.T) {
	d := newPlaceDeps()
	d.repo.FindAllFn = func(_ context.Context, _ models.PlaceFilters) ([]models.Place, int, error) {
		return nil, 0, errors.New("db down")
	}
	svc := newPlaceService(d, 0)

	_, err := svc.GetAll(context.Background(), models.PlaceFilters{})
	require.Error(t, err)
	var ae *apperr.AppError
	require.True(t, errors.As(err, &ae), "errores del repo se envuelven en AppError")
	assert.Equal(t, "places.list", ae.Op)
	assert.Equal(t, http.StatusInternalServerError, ae.HTTPStatus)
}

// --- GetByBounds -----------------------------------------------------------

func TestPlaceService_GetByBounds_EmptyReturnsEmptySlice(t *testing.T) {
	d := newPlaceDeps()
	d.repo.FindByBoundsFn = func(_ context.Context, _ models.BoundsFilter) ([]models.Place, error) {
		return nil, nil
	}
	svc := newPlaceService(d, 0)

	items, err := svc.GetByBounds(context.Background(), models.BoundsFilter{})
	require.NoError(t, err)
	assert.NotNil(t, items, "siempre devuelve slice, nunca nil")
	assert.Len(t, items, 0)
}

func TestPlaceService_GetByBounds_BuildsDimensionsPerPlace(t *testing.T) {
	d := newPlaceDeps()
	d.repo.FindByBoundsFn = func(_ context.Context, _ models.BoundsFilter) ([]models.Place, error) {
		return []models.Place{{ID: 1}, {ID: 2}}, nil
	}
	d.repo.GetAggRowsByPlaceIDsFn = func(_ context.Context, ids []int64) ([]models.CriterionAggRow, error) {
		require.Len(t, ids, 2)
		// place 1: crit con muchos yes -> green; place 2: crit con muchos no -> red.
		rows := []models.CriterionAggRow{
			{PlaceID: 1, DimensionID: 10, DimensionKey: "mov", DimensionName: "Mov",
				CriterionID: 100, CriterionKey: "ramp", Prompt: "rampa?", IsBlocking: true,
				NYes: 5, NNo: 0, QualityP50: floatPtr(4.5)},
			{PlaceID: 2, DimensionID: 10, DimensionKey: "mov", DimensionName: "Mov",
				CriterionID: 100, CriterionKey: "ramp", Prompt: "rampa?", IsBlocking: true,
				NYes: 0, NNo: 5},
		}
		return rows, nil
	}
	svc := newPlaceService(d, 0)

	items, err := svc.GetByBounds(context.Background(), models.BoundsFilter{})
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, models.StateGreen, items[0].OverallState)
	assert.Equal(t, models.StateRed, items[1].OverallState)

	// Criterios se omiten en el mapa (no aparece criterio a criterio).
	for _, item := range items {
		for _, dim := range item.Dimensions {
			assert.Empty(t, dim.Criteria, "mapa no arrastra desglose criterio a criterio")
		}
	}
}

// --- GetByID ----------------------------------------------------------------

func TestPlaceService_GetByID_NotFoundReturns404AppError(t *testing.T) {
	d := newPlaceDeps()
	d.repo.FindByIDFn = func(_ context.Context, _ int64) (*models.Place, error) {
		return nil, pgx.ErrNoRows
	}
	svc := newPlaceService(d, 0)

	_, err := svc.GetByID(context.Background(), 99)
	require.Error(t, err)
	var ae *apperr.AppError
	require.True(t, errors.As(err, &ae))
	assert.Equal(t, http.StatusNotFound, ae.HTTPStatus)
}

func TestPlaceService_GetByID_RepoErrorWrapped(t *testing.T) {
	d := newPlaceDeps()
	d.repo.FindByIDFn = func(_ context.Context, _ int64) (*models.Place, error) {
		return nil, errors.New("connection refused")
	}
	svc := newPlaceService(d, 0)

	_, err := svc.GetByID(context.Background(), 1)
	require.Error(t, err)
	var ae *apperr.AppError
	require.True(t, errors.As(err, &ae))
	assert.Equal(t, "places.detail", ae.Op)
}

func TestPlaceService_GetByID_AssemblesPlaceAccessibilityAndSubmissions(t *testing.T) {
	d := newPlaceDeps()
	d.repo.FindByIDFn = func(_ context.Context, id int64) (*models.Place, error) {
		return &models.Place{ID: id, Name: "x"}, nil
	}
	d.repo.GetAccessibilityRowsFn = func(_ context.Context, _ int64) ([]models.CriterionAggRow, error) {
		return []models.CriterionAggRow{
			{PlaceID: 1, DimensionID: 10, DimensionKey: "mov", DimensionName: "Mov",
				CriterionID: 100, CriterionKey: "ramp", Prompt: "?", IsBlocking: true,
				NYes: 5, NNo: 0, QualityP50: floatPtr(4.5)},
		}, nil
	}
	d.submissionSvc.GetByPlaceFn = func(_ context.Context, _ int64) ([]models.SubmissionWithDetails, error) {
		return []models.SubmissionWithDetails{{Submission: models.Submission{ID: 99, PlaceID: 1}}}, nil
	}
	svc := newPlaceService(d, 0)

	got, err := svc.GetByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), got.ID)
	assert.NotEmpty(t, got.Accessibility.Dimensions)
	assert.Equal(t, models.StateGreen, got.Accessibility.OverallState)
	assert.Len(t, got.Submissions, 1)
}

func TestPlaceService_GetByID_NilSubmissionsReplacedByEmptySlice(t *testing.T) {
	d := newPlaceDeps()
	d.repo.FindByIDFn = func(_ context.Context, id int64) (*models.Place, error) {
		return &models.Place{ID: id}, nil
	}
	d.repo.GetAccessibilityRowsFn = func(_ context.Context, _ int64) ([]models.CriterionAggRow, error) {
		return nil, nil
	}
	d.submissionSvc.GetByPlaceFn = func(_ context.Context, _ int64) ([]models.SubmissionWithDetails, error) {
		return nil, nil // explicit nil
	}
	svc := newPlaceService(d, 0)

	got, err := svc.GetByID(context.Background(), 1)
	require.NoError(t, err)
	assert.NotNil(t, got.Submissions, "nil del servicio se traduce a [] vacio")
	assert.Len(t, got.Submissions, 0)
}

// --- Update ---------------------------------------------------------------

func TestPlaceService_Update_NotOwnerReturnsSentinel(t *testing.T) {
	d := newPlaceDeps()
	d.repo.FindByIDFn = func(_ context.Context, _ int64) (*models.Place, error) {
		return &models.Place{ID: 1, CreatedBy: 99}, nil // creado por otro
	}
	svc := newPlaceService(d, 0)

	_, err := svc.Update(context.Background(), 1, 42, models.UpdatePlaceRequest{})
	require.Error(t, err)
	assert.ErrorIs(t, err, services.ErrNotOwner)
}

func TestPlaceService_Update_NotFoundReturns404(t *testing.T) {
	d := newPlaceDeps()
	d.repo.FindByIDFn = func(_ context.Context, _ int64) (*models.Place, error) {
		return nil, pgx.ErrNoRows
	}
	svc := newPlaceService(d, 0)

	_, err := svc.Update(context.Background(), 99, 42, models.UpdatePlaceRequest{})
	require.Error(t, err)
	var ae *apperr.AppError
	require.True(t, errors.As(err, &ae))
	assert.Equal(t, http.StatusNotFound, ae.HTTPStatus)
}

func TestPlaceService_Update_HappyPath(t *testing.T) {
	d := newPlaceDeps()
	d.repo.FindByIDFn = func(_ context.Context, id int64) (*models.Place, error) {
		return &models.Place{ID: id, CreatedBy: 42}, nil
	}
	d.repo.UpdateFn = func(_ context.Context, id int64, req models.UpdatePlaceRequest) (*models.Place, error) {
		return &models.Place{ID: id, Name: req.Name}, nil
	}
	svc := newPlaceService(d, 0)

	got, err := svc.Update(context.Background(), 1, 42,
		models.UpdatePlaceRequest{Name: "new", Latitude: 1, Longitude: 2})
	require.NoError(t, err)
	assert.Equal(t, "new", got.Name)
}

// --- Delete ---------------------------------------------------------------

func TestPlaceService_Delete_NotOwnerReturnsSentinel(t *testing.T) {
	d := newPlaceDeps()
	d.repo.FindByIDFn = func(_ context.Context, _ int64) (*models.Place, error) {
		return &models.Place{ID: 1, CreatedBy: 99}, nil
	}
	svc := newPlaceService(d, 0)

	err := svc.Delete(context.Background(), 1, 42)
	require.Error(t, err)
	assert.ErrorIs(t, err, services.ErrNotOwner)
}

func TestPlaceService_Delete_NotFoundReturns404(t *testing.T) {
	d := newPlaceDeps()
	d.repo.FindByIDFn = func(_ context.Context, _ int64) (*models.Place, error) {
		return nil, pgx.ErrNoRows
	}
	svc := newPlaceService(d, 0)

	err := svc.Delete(context.Background(), 99, 42)
	require.Error(t, err)
	var ae *apperr.AppError
	require.True(t, errors.As(err, &ae))
	assert.Equal(t, http.StatusNotFound, ae.HTTPStatus)
}

func TestPlaceService_Delete_HappyPath(t *testing.T) {
	d := newPlaceDeps()
	d.repo.FindByIDFn = func(_ context.Context, id int64) (*models.Place, error) {
		return &models.Place{ID: id, CreatedBy: 42}, nil
	}
	d.repo.DeleteFn = func(_ context.Context, id int64) error {
		assert.Equal(t, int64(1), id)
		return nil
	}
	svc := newPlaceService(d, 0)

	err := svc.Delete(context.Background(), 1, 42)
	require.NoError(t, err)
}

// --- helpers --------------------------------------------------------------

func floatPtr(v float64) *float64 { return &v }
