package services_test

import (
	"context"
	"errors"
	"testing"

	"accesspath/internal/config"
	"accesspath/internal/models"
	"accesspath/internal/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"accesspath/tests/internal/fakes"
)

func newQuestionSvc(
	cfg *config.AccessibilityThresholds,
	catalog *fakes.RepoCatalog,
	place *fakes.RepoPlace,
	contrib *fakes.RepoContribution,
	profile *fakes.RepoProfile,
	accSvc services.AccessibilityService,
) services.QuestionService {
	return services.NewQuestionService(cfg, catalog, place, contrib, profile, accSvc)
}

// catFixture devuelve un catalogo con un criterio starter en la dimension 1.
// El criterio tiene weight=1 (default), profile_tags=["silla"], y no depende
// de otros. Eso lo hace elegible como candidato.
func catFixture() []models.DimensionDetail {
	starterID := int64(100)
	return []models.DimensionDetail{
		{
			Dimension: models.Dimension{ID: 1, Key: "mov", Name: "Mov", SortOrder: 1},
			Criteria: []models.CriterionDetail{
				{
					Criterion: models.Criterion{
						ID: starterID, DimensionID: 1, Key: "ramp",
						Prompt: "Tiene rampa?", Weight: 1,
						IsBlocking: true, IsStarter: true,
						ProfileTags: []string{"silla"},
						SortOrder:   1, Active: true,
					},
					Options: []models.AnswerOption{
						{ID: 1, CriterionID: starterID, Label: "Si"},
						{ID: 2, CriterionID: starterID, Label: "No"},
					},
				},
			},
		},
	}
}

func TestQuestionService_Next_EmptyCatalogReturnsEmptyResponse(t *testing.T) {
	catalog := &fakes.RepoCatalog{}
	catalog.GetCatalogFn = func(_ context.Context) ([]models.DimensionDetail, error) {
		return nil, errors.New("catalog down")
	}
	svc := newQuestionSvc(defaultThresholds(), catalog, &fakes.RepoPlace{},
		&fakes.RepoContribution{}, &fakes.RepoProfile{},
		services.NewAccessibilityService(defaultThresholds()))

	_, err := svc.Next(context.Background(), 1, 1)
	require.Error(t, err)
}

func TestQuestionService_Next_CatalogErrorPropagates(t *testing.T) {
	// Catalog.GetCatalog falla -> la propagacion cubre el primer fmt.Errorf.
	catalog := &fakes.RepoCatalog{}
	catalog.GetCatalogFn = func(_ context.Context) ([]models.DimensionDetail, error) {
		return nil, errors.New("db unreachable")
	}
	place := &fakes.RepoPlace{}
	place.GetAccessibilityRowsFn = func(_ context.Context, _ int64) ([]models.CriterionAggRow, error) {
		return nil, nil
	}
	contrib := &fakes.RepoContribution{}
	contrib.AnsweredByPlaceFn = func(_ context.Context, _, _ int64) ([]models.AnsweredContribution, error) {
		return nil, nil
	}
	profile := &fakes.RepoProfile{}
	profile.GetNeedsFn = func(_ context.Context, _ int64) ([]string, error) {
		return nil, nil
	}
	svc := newQuestionSvc(defaultThresholds(), catalog, place, contrib, profile,
		services.NewAccessibilityService(defaultThresholds()))

	_, err := svc.Next(context.Background(), 1, 1)
	require.Error(t, err)
}

func TestQuestionService_Next_FirstQuestionPicksStarter(t *testing.T) {
	catalog := &fakes.RepoCatalog{}
	catalog.GetCatalogFn = func(_ context.Context) ([]models.DimensionDetail, error) {
		// Dos criterios, uno starter y otro no.
		return []models.DimensionDetail{
			{
				Dimension: models.Dimension{ID: 1, Key: "mov", Name: "Mov", SortOrder: 1},
				Criteria: []models.CriterionDetail{
					{
						Criterion: models.Criterion{
							ID: 100, DimensionID: 1, Key: "ramp",
							Prompt: "Tiene rampa?", Weight: 1,
							IsStarter: true, ProfileTags: []string{"silla"},
						},
						Options: []models.AnswerOption{{ID: 1, CriterionID: 100, Label: "Si"}},
					},
					{
						Criterion: models.Criterion{
							ID: 200, DimensionID: 1, Key: "elevator",
							Prompt: "Tiene ascensor?", Weight: 1,
							IsStarter: false, ProfileTags: []string{"silla"},
						},
						Options: []models.AnswerOption{{ID: 2, CriterionID: 200, Label: "Si"}},
					},
				},
			},
		}, nil
	}
	place := &fakes.RepoPlace{}
	place.GetAccessibilityRowsFn = func(_ context.Context, _ int64) ([]models.CriterionAggRow, error) {
		return nil, nil
	}
	contrib := &fakes.RepoContribution{}
	contrib.AnsweredByPlaceFn = func(_ context.Context, _, _ int64) ([]models.AnsweredContribution, error) {
		return nil, nil // sin respuestas -> es la primera
	}
	profile := &fakes.RepoProfile{}
	profile.GetNeedsFn = func(_ context.Context, _ int64) ([]string, error) {
		return []string{"silla"}, nil // match con profile_tags
	}
	svc := newQuestionSvc(defaultThresholds(), catalog, place, contrib, profile,
		services.NewAccessibilityService(defaultThresholds()))

	got, err := svc.Next(context.Background(), 1, 1)
	require.NoError(t, err)
	require.NotNil(t, got.Criterion, "primera pregunta debe elegir un starter")
	assert.Equal(t, "ramp", got.Criterion.Key,
		"entre los starters, elige el criterio de starter")
}

func TestQuestionService_Next_ExcludesAlreadyAnswered(t *testing.T) {
	catalog := &fakes.RepoCatalog{}
	catalog.GetCatalogFn = func(_ context.Context) ([]models.DimensionDetail, error) {
		return catFixture(), nil
	}
	place := &fakes.RepoPlace{}
	place.GetAccessibilityRowsFn = func(_ context.Context, _ int64) ([]models.CriterionAggRow, error) {
		return nil, nil
	}
	contrib := &fakes.RepoContribution{}
	contrib.AnsweredByPlaceFn = func(_ context.Context, _, _ int64) ([]models.AnsweredContribution, error) {
		// El usuario ya respondio el criterio 100.
		return []models.AnsweredContribution{{CriterionID: 100}}, nil
	}
	profile := &fakes.RepoProfile{}
	profile.GetNeedsFn = func(_ context.Context, _ int64) ([]string, error) {
		return nil, nil
	}
	svc := newQuestionSvc(defaultThresholds(), catalog, place, contrib, profile,
		services.NewAccessibilityService(defaultThresholds()))

	got, err := svc.Next(context.Background(), 1, 1)
	require.NoError(t, err)
	assert.Nil(t, got.Criterion,
		"si todos los criterios estan respondidos, devuelve vacio (no hay nada que preguntar)")
}

func TestQuestionService_Next_NoCandidatesReturnsEmpty(t *testing.T) {
	catalog := &fakes.RepoCatalog{}
	catalog.GetCatalogFn = func(_ context.Context) ([]models.DimensionDetail, error) {
		// Criterio ya respondido por el usuario.
		return catFixture(), nil
	}
	place := &fakes.RepoPlace{}
	place.GetAccessibilityRowsFn = func(_ context.Context, _ int64) ([]models.CriterionAggRow, error) {
		return nil, nil
	}
	contrib := &fakes.RepoContribution{}
	contrib.AnsweredByPlaceFn = func(_ context.Context, _, _ int64) ([]models.AnsweredContribution, error) {
		existsTrue := true
		return []models.AnsweredContribution{{CriterionID: 100, ExistsFlag: &existsTrue}}, nil
	}
	profile := &fakes.RepoProfile{}
	profile.GetNeedsFn = func(_ context.Context, _ int64) ([]string, error) {
		return nil, nil
	}
	svc := newQuestionSvc(defaultThresholds(), catalog, place, contrib, profile,
		services.NewAccessibilityService(defaultThresholds()))

	got, err := svc.Next(context.Background(), 1, 1)
	require.NoError(t, err)
	assert.Nil(t, got.Criterion, "sin candidatos -> respuesta vacia")
}

func TestQuestionService_Next_DependsOnExcludes(t *testing.T) {
	// Criterio B depende de A. A no esta respondido por el usuario y no
	// tiene estado positivo en el catalogo -> B queda excluido.
	depID := int64(50)
	catalog := &fakes.RepoCatalog{}
	catalog.GetCatalogFn = func(_ context.Context) ([]models.DimensionDetail, error) {
		return []models.DimensionDetail{
			{
				Dimension: models.Dimension{ID: 1, Key: "mov", Name: "Mov", SortOrder: 1},
				Criteria: []models.CriterionDetail{
					{
						Criterion: models.Criterion{
							ID: 100, DimensionID: 1, Key: "ramp",
							Prompt: "Tiene rampa?", Weight: 1, IsStarter: true,
						},
						Options: []models.AnswerOption{{ID: 1, CriterionID: 100}},
					},
					{
						Criterion: models.Criterion{
							ID: 200, DimensionID: 1, Key: "elevator",
							Prompt: "Tiene ascensor?", Weight: 1,
							DependsOnCriterionID: &depID,
						},
						Options: []models.AnswerOption{{ID: 2, CriterionID: 200}},
					},
				},
			},
		}, nil
	}
	place := &fakes.RepoPlace{}
	place.GetAccessibilityRowsFn = func(_ context.Context, _ int64) ([]models.CriterionAggRow, error) {
		return nil, nil
	}
	contrib := &fakes.RepoContribution{}
	contrib.AnsweredByPlaceFn = func(_ context.Context, _, _ int64) ([]models.AnsweredContribution, error) {
		return nil, nil
	}
	profile := &fakes.RepoProfile{}
	profile.GetNeedsFn = func(_ context.Context, _ int64) ([]string, error) {
		return nil, nil
	}
	svc := newQuestionSvc(defaultThresholds(), catalog, place, contrib, profile,
		services.NewAccessibilityService(defaultThresholds()))

	got, err := svc.Next(context.Background(), 1, 1)
	require.NoError(t, err)
	require.NotNil(t, got.Criterion, "debe quedar al menos un candidato (ramp)")
	assert.Equal(t, "ramp", got.Criterion.Key,
		"elevator queda excluido por depends_on no satisfecho; ramp es elegible")
}
