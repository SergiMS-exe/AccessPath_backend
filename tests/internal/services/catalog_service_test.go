package services_test

import (
	"context"
	"errors"
	"testing"

	"accesspath/internal/models"
	"accesspath/internal/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"accesspath/tests/internal/fakes"
)

func newCatalogSvc(t *testing.T, repo *fakes.RepoCatalog) services.CatalogService {
	t.Helper()
	return services.NewCatalogService(repo)
}

func TestCatalogService_GetCatalog_Delegates(t *testing.T) {
	repo := &fakes.RepoCatalog{}
	repo.GetCatalogFn = func(_ context.Context) ([]models.DimensionDetail, error) {
		return []models.DimensionDetail{
			{Dimension: models.Dimension{ID: 1, Name: "Mov"}},
		}, nil
	}
	svc := newCatalogSvc(t, repo)

	got, err := svc.GetCatalog(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Mov", got[0].Name)
}

func TestCatalogService_GetCatalog_RepoErrorPropagates(t *testing.T) {
	repo := &fakes.RepoCatalog{}
	repo.GetCatalogFn = func(_ context.Context) ([]models.DimensionDetail, error) {
		return nil, errors.New("db down")
	}
	svc := newCatalogSvc(t, repo)

	_, err := svc.GetCatalog(context.Background())
	require.Error(t, err)
}
