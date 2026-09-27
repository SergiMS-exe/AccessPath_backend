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

func newColSvc(t *testing.T, repo *fakes.RepoCollection) services.CollectionService {
	t.Helper()
	return services.NewCollectionService(repo)
}

func TestCollectionService_GetByUser_Delegates(t *testing.T) {
	repo := &fakes.RepoCollection{}
	repo.FindByUserFn = func(_ context.Context, userID int64) ([]models.Collection, error) {
		assert.Equal(t, int64(42), userID)
		return []models.Collection{{ID: 1, UserID: userID, Name: "favoritos"}}, nil
	}
	svc := newColSvc(t, repo)

	got, err := svc.GetByUser(context.Background(), 42)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "favoritos", got[0].Name)
}

func TestCollectionService_GetByID_Delegates(t *testing.T) {
	repo := &fakes.RepoCollection{}
	repo.FindByIDFn = func(_ context.Context, id int64) (*models.Collection, error) {
		return &models.Collection{ID: id}, nil
	}
	svc := newColSvc(t, repo)

	got, err := svc.GetByID(context.Background(), 99)
	require.NoError(t, err)
	assert.Equal(t, int64(99), got.ID)
}

func TestCollectionService_Create_Delegates(t *testing.T) {
	repo := &fakes.RepoCollection{}
	repo.CreateFn = func(_ context.Context, req models.CreateCollectionRequest) (*models.Collection, error) {
		return &models.Collection{ID: 1, UserID: req.UserID, Name: req.Name, IsDefault: req.IsDefault}, nil
	}
	svc := newColSvc(t, repo)

	got, err := svc.Create(context.Background(), models.CreateCollectionRequest{
		UserID: 42, Name: "x", IsDefault: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "x", got.Name)
	assert.True(t, got.IsDefault)
}

func TestCollectionService_Delete_Delegates(t *testing.T) {
	repo := &fakes.RepoCollection{}
	repo.DeleteFn = func(_ context.Context, id int64) error {
		assert.Equal(t, int64(99), id)
		return nil
	}
	svc := newColSvc(t, repo)

	require.NoError(t, svc.Delete(context.Background(), 99))
}

func TestCollectionService_Delete_RepoErrorPropagates(t *testing.T) {
	repo := &fakes.RepoCollection{}
	repo.DeleteFn = func(_ context.Context, _ int64) error {
		return errors.New("fk violation")
	}
	svc := newColSvc(t, repo)

	require.Error(t, svc.Delete(context.Background(), 99))
}

func TestCollectionService_AddPlace_Delegates(t *testing.T) {
	repo := &fakes.RepoCollection{}
	repo.AddPlaceFn = func(_ context.Context, cid, pid int64) error {
		assert.Equal(t, int64(1), cid)
		assert.Equal(t, int64(2), pid)
		return nil
	}
	svc := newColSvc(t, repo)

	require.NoError(t, svc.AddPlace(context.Background(), 1, 2))
}

func TestCollectionService_RemovePlace_Delegates(t *testing.T) {
	repo := &fakes.RepoCollection{}
	repo.RemovePlaceFn = func(_ context.Context, cid, pid int64) error {
		return nil
	}
	svc := newColSvc(t, repo)

	require.NoError(t, svc.RemovePlace(context.Background(), 1, 2))
}

func TestCollectionService_GetPlaces_Delegates(t *testing.T) {
	repo := &fakes.RepoCollection{}
	repo.GetPlacesFn = func(_ context.Context, _ int64) ([]models.Place, error) {
		return []models.Place{{ID: 1, Name: "x"}}, nil
	}
	svc := newColSvc(t, repo)

	got, err := svc.GetPlaces(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, got, 1)
}
