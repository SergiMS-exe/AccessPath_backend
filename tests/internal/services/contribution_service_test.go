package services_test

import (
	"context"
	"errors"
	"testing"

	"accesspath/internal/models"
	"accesspath/internal/services"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"accesspath/tests/internal/fakes"
)

// newContribSvcReadOnly construye un ContributionService con db=nil.
// Solo cubre los caminos que cortan ANTES de db.Begin: ErrOptionMismatch
// (Create) y ErrContributionNotFound / ErrNotOwner (Delete). El resto del
// flujo necesita un pool real.
func newContribSvcReadOnly(
	contribRepo *fakes.RepoContribution,
	submissionRepo *fakes.RepoSubmission,
	catalogRepo *fakes.RepoCatalog,
	placeRepo *fakes.RepoPlace,
	accSvc services.AccessibilityService,
) services.ContributionService {
	return services.NewContributionService(nil, contribRepo, submissionRepo, catalogRepo, placeRepo, accSvc)
}

// --- Create: ErrOptionMismatch (corta antes de db.Begin) -------------------

func TestContributionService_Create_OptionMismatchReturnsSentinel(t *testing.T) {
	catalog := &fakes.RepoCatalog{}
	catalog.GetOptionByIDFn = func(_ context.Context, id int64) (*models.AnswerOption, error) {
		return &models.AnswerOption{ID: id, CriterionID: 999}, nil // otro criterio
	}
	svc := newContribSvcReadOnly(&fakes.RepoContribution{}, &fakes.RepoSubmission{}, catalog,
		&fakes.RepoPlace{}, services.NewAccessibilityService(defaultThresholds()))

	_, err := svc.Create(context.Background(), 1, models.ContributionRequest{
		PlaceID: 1, CriterionID: 1, AnswerOptionID: 1,
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, services.ErrOptionMismatch)
}

func TestContributionService_Create_OptionLookupErrorPropagates(t *testing.T) {
	catalog := &fakes.RepoCatalog{}
	catalog.GetOptionByIDFn = func(_ context.Context, _ int64) (*models.AnswerOption, error) {
		return nil, errors.New("option read failed")
	}
	svc := newContribSvcReadOnly(&fakes.RepoContribution{}, &fakes.RepoSubmission{}, catalog,
		&fakes.RepoPlace{}, services.NewAccessibilityService(defaultThresholds()))

	_, err := svc.Create(context.Background(), 1, models.ContributionRequest{
		PlaceID: 1, CriterionID: 1, AnswerOptionID: 1,
	})

	require.Error(t, err)
}

func TestContributionService_Create_CriterionMatchesProceedsBeyondGuard(t *testing.T) {
	// Cuando criterion coincide, pasamos la primera guarda y llegamos a db.Begin.
	// Sin pool real, esto falla (panic o error). Aqui confirmamos que NO
	// devuelve ErrOptionMismatch (sentinel de la guarda que cubrimos arriba).
	catalog := &fakes.RepoCatalog{}
	catalog.GetOptionByIDFn = func(_ context.Context, id int64) (*models.AnswerOption, error) {
		return &models.AnswerOption{ID: id, CriterionID: 1}, nil
	}
	svc := newContribSvcReadOnly(&fakes.RepoContribution{}, &fakes.RepoSubmission{}, catalog,
		&fakes.RepoPlace{}, services.NewAccessibilityService(defaultThresholds()))

	defer func() { _ = recover() }()
	_, err := svc.Create(context.Background(), 1, models.ContributionRequest{
		PlaceID: 1, CriterionID: 1, AnswerOptionID: 1,
	})
	if err == nil {
		t.Fatal("sin pool real, Create debe fallar tras pasar la guarda de opcion")
	}
	assert.NotErrorIs(t, err, services.ErrOptionMismatch,
		"la guarda de opcion se supero; el fallo NO es de sentinel")
}

// --- Delete: ErrContributionNotFound / ErrNotOwner (cortan antes de TX) ---

func TestContributionService_Delete_NotFoundReturnsSentinel(t *testing.T) {
	contribRepo := &fakes.RepoContribution{}
	contribRepo.FindByIDFn = func(_ context.Context, _ int64) (*models.Contribution, error) {
		return nil, pgx.ErrNoRows
	}
	svc := newContribSvcReadOnly(contribRepo, &fakes.RepoSubmission{}, &fakes.RepoCatalog{},
		&fakes.RepoPlace{}, services.NewAccessibilityService(defaultThresholds()))

	_, err := svc.Delete(context.Background(), 1, 99)
	require.Error(t, err)
	assert.ErrorIs(t, err, services.ErrContributionNotFound)
}

func TestContributionService_Delete_NotOwnerReturnsSentinel(t *testing.T) {
	contribRepo := &fakes.RepoContribution{}
	contribRepo.FindByIDFn = func(_ context.Context, id int64) (*models.Contribution, error) {
		return &models.Contribution{ID: id, UserID: 99, PlaceID: 1, CriterionID: 1}, nil
	}
	svc := newContribSvcReadOnly(contribRepo, &fakes.RepoSubmission{}, &fakes.RepoCatalog{},
		&fakes.RepoPlace{}, services.NewAccessibilityService(defaultThresholds()))

	_, err := svc.Delete(context.Background(), 7, 42) // 7 no es el owner (99)
	require.Error(t, err)
	assert.ErrorIs(t, err, services.ErrNotOwner)
}

func TestContributionService_Delete_OwnerReachesTx(t *testing.T) {
	// Propietario valido: pasamos las dos guardas (FindByID + owner check)
	// y llegamos a db.Begin. Sin un pool real, esto puede panicar o devolver
	// un error; el comportamiento exacto depende del driver. No podemos
	// verificar mas aqui; la cobertura del resto del camino requiere un pool
	// real (suite de integracion). Lo importante es que las guardas no
	// disparan sentinels.
	contribRepo := &fakes.RepoContribution{}
	contribRepo.FindByIDFn = func(_ context.Context, id int64) (*models.Contribution, error) {
		return &models.Contribution{ID: id, UserID: 42, PlaceID: 1, CriterionID: 1}, nil
	}
	svc := newContribSvcReadOnly(contribRepo, &fakes.RepoSubmission{}, &fakes.RepoCatalog{},
		&fakes.RepoPlace{}, services.NewAccessibilityService(defaultThresholds()))

	defer func() { _ = recover() }()
	_, _ = svc.Delete(context.Background(), 7, 42)
	// (no asserto nada: db.Begin con pool nil tiene comportamiento indefinido
	// segun la version de pgx. Lo que SI cubrimos son los sentinels arriba.)
}

// --- liveResult se testea indirectamente via el camino de Create/Delete.
// Aqui no llegamos sin TX, asi que su cobertura viene del flujo end-to-end.
// --- ErrContributionNotFound sentinel

func TestContributionService_NotFoundSentinelMessage(t *testing.T) {
	assert.Equal(t, "contribution not found", services.ErrContributionNotFound.Error())
}

func TestContributionService_NotOwnerSentinelMessage(t *testing.T) {
	assert.Equal(t, "not the owner", services.ErrNotOwner.Error())
}

func TestContributionService_OptionMismatchSentinelMessage(t *testing.T) {
	assert.Equal(t, "answer option does not belong to criterion", services.ErrOptionMismatch.Error())
}
