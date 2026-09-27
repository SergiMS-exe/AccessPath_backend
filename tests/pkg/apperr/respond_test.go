package apperr_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"accesspath/pkg/apperr"
	"accesspath/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newCtx(requestID string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	if requestID != "" {
		c.Set(apperr.RequestIDKey, requestID)
	}
	return c, w
}

func TestRespondInternal_WritesStatusAndEnvelope(t *testing.T) {
	c, w := newCtx("req-abc-123")

	ae := apperr.New("x.create", "x.failed", http.StatusBadRequest, "datos invalidos", nil)
	apperr.RespondInternal(c, ae)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var env response.Envelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, "datos invalidos", env.Error)
	assert.Nil(t, env.Data)
}

func TestRespondInternal_FallsBackToDefaultMessage(t *testing.T) {
	c, w := newCtx("")

	ae := apperr.New("x.y", "x.z", http.StatusBadRequest, "", nil)
	apperr.RespondInternal(c, ae)

	var env response.Envelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.NotEmpty(t, env.Error)
}

func TestRespondInternal_UnknownStatusFallback(t *testing.T) {
	c, w := newCtx("")

	ae := apperr.New("x.y", "x.z", 418, "", nil)
	apperr.RespondInternal(c, ae)

	var env response.Envelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, "Error desconocido.", env.Error)
}

func TestRespondInternal_5xxDefaultMessage(t *testing.T) {
	c, w := newCtx("")

	ae := apperr.New("x.y", "x.z", http.StatusInternalServerError, "", nil)
	apperr.RespondInternal(c, ae)

	var env response.Envelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, "Error interno del servidor. Intentalo de nuevo.", env.Error)
}

func TestRespond_NilDoesNothing(t *testing.T) {
	c, w := newCtx("")

	got := apperr.Respond(c, nil)

	assert.False(t, got)
	assert.Equal(t, 200, w.Code)
	assert.Empty(t, w.Body.String())
}

func TestRespond_PreservesAppError(t *testing.T) {
	c, w := newCtx("req-1")

	ae := apperr.BadRequest("places.create", "places.invalid", "body invalido")
	got := apperr.Respond(c, ae)

	assert.True(t, got)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var env response.Envelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, "body invalido", env.Error)
}

func TestRespond_GenericErrorBecomesInternal(t *testing.T) {
	c, w := newCtx("req-2")

	got := apperr.Respond(c, errors.New("something exploded"))

	assert.True(t, got)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var env response.Envelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.NotEmpty(t, env.Error)
}

func TestRespond_FillsEmptyOpFromContext(t *testing.T) {
	w := httptest.NewRecorder()
	r := gin.New()
	r.GET("/api/v1/places/:id", func(c *gin.Context) {
		ae := apperr.New("", "x.code", http.StatusNotFound, "no encontrado", nil)
		apperr.Respond(c, ae)
		assert.NotEmpty(t, ae.Op)
		assert.Contains(t, ae.Op, "places")
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/places/123", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDefaultMessageForStatus(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"bad request", http.StatusBadRequest},
		{"unauthorized", http.StatusUnauthorized},
		{"not found", http.StatusNotFound},
		{"too many requests", http.StatusTooManyRequests},
		{"bad gateway", http.StatusBadGateway},
		{"service unavailable", http.StatusServiceUnavailable},
		{"unknown 4xx", http.StatusTeapot},
		{"unknown 5xx", http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := apperr.DefaultMessageForStatus(tt.status)
			assert.NotEmpty(t, msg)
		})
	}
}

func TestLevelForStatus(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   string
	}{
		{"2xx is info", http.StatusOK, "INFO"},
		{"3xx is info", http.StatusFound, "INFO"},
		{"4xx is warn", http.StatusBadRequest, "WARN"},
		{"401 is warn", http.StatusUnauthorized, "WARN"},
		{"5xx is error", http.StatusInternalServerError, "ERROR"},
		{"502 is error", http.StatusBadGateway, "ERROR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := apperr.LevelForStatus(tt.status)
			assert.Equal(t, tt.want, got.String())
		})
	}
}

func TestOpFromContext(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"normal path", "/api/v1/places", "http.api/v1/places"},
		{"path with id", "/api/v1/places/123", "http.api/v1/places/123"},
	}

	r := gin.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r.GET(tt.path, func(c *gin.Context) {
				assert.Equal(t, tt.want, apperr.OpFromContext(c))
			})
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			r.ServeHTTP(w, req)
		})
	}
}

func TestOpFromContext_NoRoute(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	assert.Equal(t, "http", apperr.OpFromContext(c))
}

func TestRequestID(t *testing.T) {
	c, _ := newCtx("req-deadbeef")

	assert.Equal(t, "req-deadbeef", apperr.RequestID(c))
}

func TestRequestID_Empty(t *testing.T) {
	c, _ := newCtx("")

	assert.Equal(t, "", apperr.RequestID(c))
}
