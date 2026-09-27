package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"accesspath/internal/models"
	"accesspath/internal/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"accesspath/tests/internal/fakes"
)

// newProfileSvc es inalcanzable: ProfileService necesita *pgxpool.Pool real
// para abrir TX. Asi que los tests de ProfileService viven en una capa
// mas arriba (handlers via ErrInvalidNeedKey) o cubren solo el camino
// "no llegamos a la TX" (Get). Aqui cubrimos ese camino y la API publica.

// Para los tests de Set/Delete necesitamos un pool real. Como no tenemos,
// testeamos solo Get y ErrInvalidNeedKey (que corta antes de TX).

func newProfileSvcReadOnly(t *testing.T, repo *fakes.RepoProfile) services.ProfileService {
	t.Helper()
	// Pasamos nil al pool: Get no toca la TX, asi que es seguro.
	return services.NewProfileService(nil, repo)
}

func TestProfileService_Get_ReturnsNeedsAndConsent(t *testing.T) {
	now := time.Now()
	repo := &fakes.RepoProfile{}
	repo.GetNeedsFn = func(_ context.Context, _ int64) ([]string, error) {
		return []string{"silla", "baja_vision"}, nil
	}
	repo.GetConsentFn = func(_ context.Context, _ int64) (*time.Time, error) {
		return &now, nil
	}
	svc := newProfileSvcReadOnly(t, repo)

	got, err := svc.Get(context.Background(), 7)
	require.NoError(t, err)
	assert.Equal(t, []string{"silla", "baja_vision"}, got.Needs)
	assert.True(t, got.HasConsent)
	require.NotNil(t, got.ConsentAt)
	assert.Equal(t, now, *got.ConsentAt)
}

func TestProfileService_Get_NoConsentYieldsHasConsentFalse(t *testing.T) {
	repo := &fakes.RepoProfile{}
	repo.GetNeedsFn = func(_ context.Context, _ int64) ([]string, error) {
		return []string{}, nil
	}
	repo.GetConsentFn = func(_ context.Context, _ int64) (*time.Time, error) {
		return nil, nil
	}
	svc := newProfileSvcReadOnly(t, repo)

	got, err := svc.Get(context.Background(), 7)
	require.NoError(t, err)
	assert.False(t, got.HasConsent)
	assert.Nil(t, got.ConsentAt)
}

func TestProfileService_Get_NilNeedsReplacedByEmptySlice(t *testing.T) {
	repo := &fakes.RepoProfile{}
	repo.GetNeedsFn = func(_ context.Context, _ int64) ([]string, error) {
		return nil, nil // repo explicito nil
	}
	repo.GetConsentFn = func(_ context.Context, _ int64) (*time.Time, error) {
		return nil, nil
	}
	svc := newProfileSvcReadOnly(t, repo)

	got, err := svc.Get(context.Background(), 7)
	require.NoError(t, err)
	assert.NotNil(t, got.Needs, "nil del repo se traduce a [] vacio")
	assert.Len(t, got.Needs, 0)
}

func TestProfileService_Get_NeedsErrorPropagates(t *testing.T) {
	repo := &fakes.RepoProfile{}
	repo.GetNeedsFn = func(_ context.Context, _ int64) ([]string, error) {
		return nil, errors.New("db down")
	}
	svc := newProfileSvcReadOnly(t, repo)

	_, err := svc.Get(context.Background(), 7)
	require.Error(t, err)
}

func TestProfileService_Get_ConsentErrorPropagates(t *testing.T) {
	repo := &fakes.RepoProfile{}
	repo.GetNeedsFn = func(_ context.Context, _ int64) ([]string, error) {
		return []string{}, nil
	}
	repo.GetConsentFn = func(_ context.Context, _ int64) (*time.Time, error) {
		return nil, errors.New("consent read failed")
	}
	svc := newProfileSvcReadOnly(t, repo)

	_, err := svc.Get(context.Background(), 7)
	require.Error(t, err)
}

func TestErrInvalidNeedKey_ErrorMessage(t *testing.T) {
	// Set() valida los needs y devuelve ErrInvalidNeedKey; dado que Set
	// necesita una TX para ejecutarse, aqui solo validamos el mensaje del
	// sentinel (es el camino que el handler muestra al cliente).
	e := services.ErrInvalidNeedKey{Key: "mala"}
	assert.Equal(t, "invalid need_key: mala", e.Error())
}

func TestValidNeedKeysContainsAllDocumentedKeys(t *testing.T) {
	// Lista exhaustiva: cualquier cambio debe ser consciente.
	want := []string{
		"silla", "movilidad_reducida", "baja_vision",
		"ceguera", "auditiva", "cognitiva", "sensorial",
	}
	for _, k := range want {
		assert.True(t, models.ValidNeedKeys[k], "%s debe estar en ValidNeedKeys", k)
	}
}
