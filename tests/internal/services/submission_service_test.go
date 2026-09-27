package services_test

import (
	"context"
	"testing"

	"accesspath/internal/models"
	"accesspath/internal/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"accesspath/tests/internal/fakes"
)

// newSubmissionSvcReadOnly construye un SubmissionService con db=nil.
// GetByPlace no toca la TX, asi que es 100% testeable. Save requiere TX,
// asi que solo cubre el camino "empty body -> ErrEmptySubmission" que
// corta antes de db.Begin.

func newSubmissionSvcReadOnly(
	subRepo *fakes.RepoSubmission,
	photoRepo *fakes.RepoPhoto,
	photoSvc *fakes.PhotoSvc,
) services.SubmissionService {
	return services.NewSubmissionService(nil, subRepo, photoRepo, photoSvc)
}

// --- GetByPlace -----------------------------------------------------------

func TestSubmissionService_GetByPlace_EmptyResultsReturnsEmpty(t *testing.T) {
	subRepo := &fakes.RepoSubmission{}
	subRepo.FindByPlaceFn = func(_ context.Context, _ int64) ([]models.SubmissionWithDetails, error) {
		return nil, nil
	}
	photoRepo := &fakes.RepoPhoto{}
	photoRepo.FindBySubmissionIDsFn = func(_ context.Context, _ []int64) ([]models.Photo, error) {
		return nil, nil
	}
	svc := newSubmissionSvcReadOnly(subRepo, photoRepo, &fakes.PhotoSvc{})

	got, err := svc.GetByPlace(context.Background(), 1)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestSubmissionService_GetByPlace_AttachesPhotos(t *testing.T) {
	subRepo := &fakes.RepoSubmission{}
	subRepo.FindByPlaceFn = func(_ context.Context, _ int64) ([]models.SubmissionWithDetails, error) {
		return []models.SubmissionWithDetails{
			{Submission: models.Submission{ID: 10, PlaceID: 1, UserID: 1}, Username: "juan"},
			{Submission: models.Submission{ID: 20, PlaceID: 1, UserID: 2}, Username: "maria"},
		}, nil
	}
	photoRepo := &fakes.RepoPhoto{}
	photoRepo.FindBySubmissionIDsFn = func(_ context.Context, ids []int64) ([]models.Photo, error) {
		assert.ElementsMatch(t, []int64{10, 20}, ids)
		return []models.Photo{
			{ID: 100, SubmissionID: 10, URL: "u1"},
			{ID: 200, SubmissionID: 20, URL: "u2"},
		}, nil
	}
	svc := newSubmissionSvcReadOnly(subRepo, photoRepo, &fakes.PhotoSvc{})

	got, err := svc.GetByPlace(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Len(t, got[0].Photos, 1)
	assert.Len(t, got[1].Photos, 1)
	assert.Equal(t, "u1", got[0].Photos[0].URL)
	assert.Equal(t, "u2", got[1].Photos[0].URL)
}

// --- Save: ErrEmptySubmission (corta antes de TX) ------------------------

func TestSubmissionService_Save_EmptyBodyReturnsSentinel(t *testing.T) {
	svc := newSubmissionSvcReadOnly(&fakes.RepoSubmission{}, &fakes.RepoPhoto{}, &fakes.PhotoSvc{})

	_, err := svc.Save(context.Background(), 1, models.SubmissionRequest{
		PlaceID: 1, Comment: nil, Photos: nil,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, services.ErrEmptySubmission)
}

func TestSubmissionService_Save_WhitespaceCommentStillEmpty(t *testing.T) {
	svc := newSubmissionSvcReadOnly(&fakes.RepoSubmission{}, &fakes.RepoPhoto{}, &fakes.PhotoSvc{})

	ws := "   "
	_, err := svc.Save(context.Background(), 1, models.SubmissionRequest{
		PlaceID: 1, Comment: &ws, Photos: nil,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, services.ErrEmptySubmission)
}

func TestErrEmptySubmissionMessage(t *testing.T) {
	assert.Equal(t,
		"submission requires a comment or at least one photo",
		services.ErrEmptySubmission.Error())
}
