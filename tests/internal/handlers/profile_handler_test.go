package handlers_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"accesspath/internal/handlers"
	"accesspath/internal/models"
	"accesspath/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"accesspath/tests/internal/fakes"
)

func profDo(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// --- Get ------------------------------------------------------------------

func TestProfileHandler_Get_MissingUserIDReturns401(t *testing.T) {
	h := handlers.NewProfileHandler(&fakes.SvcProfile{})
	r := gin.New()
	r.GET("/me/profile", h.Get)

	w := profDo(r, http.MethodGet, "/me/profile", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestProfileHandler_Get_HappyPathReturns200(t *testing.T) {
	svc := &fakes.SvcProfile{}
	svc.GetFn = func(_ context.Context, _ int64) (*models.ProfileResponse, error) {
		return &models.ProfileResponse{
			Needs: []string{"silla"}, HasConsent: true,
		}, nil
	}
	h := handlers.NewProfileHandler(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int64(7)) })
	r.GET("/me/profile", h.Get)

	w := profDo(r, http.MethodGet, "/me/profile", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Set ------------------------------------------------------------------

func TestProfileHandler_Set_MissingUserIDReturns401(t *testing.T) {
	h := handlers.NewProfileHandler(&fakes.SvcProfile{})
	r := gin.New()
	r.PUT("/me/profile", h.Set)

	w := profDo(r, http.MethodPut, "/me/profile", `{"needs":[],"consent":true}`)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestProfileHandler_Set_ConsentRequiredReturns400(t *testing.T) {
	h := handlers.NewProfileHandler(&fakes.SvcProfile{})
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int64(7)) })
	r.PUT("/me/profile", h.Set)

	w := profDo(r, http.MethodPut, "/me/profile",
		`{"needs":["silla"]}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_Set_InvalidJSONReturns400(t *testing.T) {
	h := handlers.NewProfileHandler(&fakes.SvcProfile{})
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int64(7)) })
	r.PUT("/me/profile", h.Set)

	w := profDo(r, http.MethodPut, "/me/profile", `{`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_Set_InvalidNeedKeyMapsTo400(t *testing.T) {
	svc := &fakes.SvcProfile{}
	svc.SetFn = func(_ context.Context, _ int64, _ models.ProfileRequest) (*models.ProfileResponse, error) {
		return nil, services.ErrInvalidNeedKey{Key: "mala"}
	}
	h := handlers.NewProfileHandler(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int64(7)) })
	r.PUT("/me/profile", h.Set)

	w := profDo(r, http.MethodPut, "/me/profile",
		`{"needs":["mala"],"consent":true}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_Set_HappyPathReturns200(t *testing.T) {
	svc := &fakes.SvcProfile{}
	svc.SetFn = func(_ context.Context, _ int64, req models.ProfileRequest) (*models.ProfileResponse, error) {
		return &models.ProfileResponse{Needs: req.Needs, HasConsent: true}, nil
	}
	h := handlers.NewProfileHandler(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int64(7)) })
	r.PUT("/me/profile", h.Set)

	w := profDo(r, http.MethodPut, "/me/profile",
		`{"needs":["silla"],"consent":true}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Delete ---------------------------------------------------------------

func TestProfileHandler_Delete_MissingUserIDReturns401(t *testing.T) {
	h := handlers.NewProfileHandler(&fakes.SvcProfile{})
	r := gin.New()
	r.DELETE("/me/profile", h.Delete)

	w := profDo(r, http.MethodDelete, "/me/profile", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestProfileHandler_Delete_HappyPathReturns204(t *testing.T) {
	svc := &fakes.SvcProfile{}
	svc.DeleteFn = func(_ context.Context, _ int64) error { return nil }
	h := handlers.NewProfileHandler(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int64(7)) })
	r.DELETE("/me/profile", h.Delete)

	w := profDo(r, http.MethodDelete, "/me/profile", "")
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestProfileHandler_Delete_ErrorMapsToInternal(t *testing.T) {
	svc := &fakes.SvcProfile{}
	svc.DeleteFn = func(_ context.Context, _ int64) error { return errors.New("boom") }
	h := handlers.NewProfileHandler(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int64(7)) })
	r.DELETE("/me/profile", h.Delete)

	w := profDo(r, http.MethodDelete, "/me/profile", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
