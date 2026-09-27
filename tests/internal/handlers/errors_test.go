package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"accesspath/internal/handlers"
	"accesspath/internal/services"
	"accesspath/pkg/apperr"
	"accesspath/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func runWithContext(t *testing.T, fn func(c *gin.Context)) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r := gin.New()
	r.GET("/api/v1/x", fn)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/x", nil)
	r.ServeHTTP(w, req)
	return w
}

func TestRespond_NilDoesNothing(t *testing.T) {
	w := runWithContext(t, func(c *gin.Context) {
		got := handlers.Respond(c, nil)
		assert.False(t, got)
	})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Body.String())
}

func TestRespond_AppErrorPassesThrough(t *testing.T) {
	w := runWithContext(t, func(c *gin.Context) {
		ae := apperr.BadRequest("places.create", "places.invalid", "datos invalidos")
		got := handlers.Respond(c, ae)
		assert.True(t, got)
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var env response.Envelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, "datos invalidos", env.Error)
}

func TestRespond_AppErrorPreservesCodeAndStatus(t *testing.T) {
	w := runWithContext(t, func(c *gin.Context) {
		ae := apperr.New("x.y", "x.custom_code", http.StatusTeapot, "soy una tetera", nil)
		handlers.Respond(c, ae)
	})

	assert.Equal(t, http.StatusTeapot, w.Code)
}

func TestRespond_GenericErrorBecomesInternal500(t *testing.T) {
	w := runWithContext(t, func(c *gin.Context) {
		got := handlers.Respond(c, errors.New("boom"))
		assert.True(t, got)
	})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRespond_SentinelMappingToCorrectStatus(t *testing.T) {
	tests := []struct {
		name       string
		sentinel   error
		wantStatus int
	}{
		{"ErrInvalidCredentials -> 401", services.ErrInvalidCredentials, http.StatusUnauthorized},
		{"ErrEmailAlreadyUsed -> 400", services.ErrEmailAlreadyUsed, http.StatusBadRequest},
		{"ErrNotOwner -> 401", services.ErrNotOwner, http.StatusUnauthorized},
		{"ErrEmptySubmission -> 400", services.ErrEmptySubmission, http.StatusBadRequest},
		{"ErrContributionNotFound -> 404", services.ErrContributionNotFound, http.StatusNotFound},
		{"ErrGmapsQuotaExceeded -> 429", services.ErrGmapsQuotaExceeded, http.StatusTooManyRequests},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := runWithContext(t, func(c *gin.Context) {
				got := handlers.Respond(c, tt.sentinel)
				assert.True(t, got)
			})
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestRespond_OpFilledFromFullPathWhenEmpty(t *testing.T) {
	w := runWithContext(t, func(c *gin.Context) {
		ae := apperr.New("", "", http.StatusBadRequest, "msg", nil)
		handlers.Respond(c, ae)
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMatchServiceError_NoMatchReturnsFalse(t *testing.T) {
	_, ok := handlers.MatchServiceError(errors.New("unknown error"))
	assert.False(t, ok)
}

func TestMatchServiceError_FirstMatchWins(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"ErrInvalidCredentials matches", services.ErrInvalidCredentials, true},
		{"wrapped ErrInvalidCredentials matches", errors.Join(errors.New("ctx"), services.ErrInvalidCredentials), true},
		{"generic error does not match", errors.New("random"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := handlers.MatchServiceError(tt.err)
			assert.Equal(t, tt.want, ok)
		})
	}
}
