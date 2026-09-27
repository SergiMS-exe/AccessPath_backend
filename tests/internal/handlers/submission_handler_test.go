package handlers_test

import (
	"bytes"
	"context"
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

func subDo(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
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

// --- GetByPlace -----------------------------------------------------------

func TestSubmissionHandler_GetByPlace_InvalidIDReturns400(t *testing.T) {
	h := handlers.NewSubmissionHandler(&fakes.SvcSubmission{})
	r := gin.New()
	r.GET("/places/:id/submissions", h.GetByPlace)

	w := subDo(r, http.MethodGet, "/places/abc/submissions", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubmissionHandler_GetByPlace_HappyPathReturns200(t *testing.T) {
	svc := &fakes.SvcSubmission{}
	svc.GetByPlaceFn = func(_ context.Context, _ int64) ([]models.SubmissionWithDetails, error) {
		return []models.SubmissionWithDetails{}, nil
	}
	h := handlers.NewSubmissionHandler(svc)
	r := gin.New()
	r.GET("/places/:id/submissions", h.GetByPlace)

	w := subDo(r, http.MethodGet, "/places/1/submissions", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Save -----------------------------------------------------------------

func TestSubmissionHandler_Save_MissingUserIDReturns401(t *testing.T) {
	h := handlers.NewSubmissionHandler(&fakes.SvcSubmission{})
	r := gin.New()
	r.PUT("/submissions", h.Save)

	w := subDo(r, http.MethodPut, "/submissions",
		`{"place_id":1,"comment":"ok"}`)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSubmissionHandler_Save_PlaceIDRequiredReturns400(t *testing.T) {
	h := handlers.NewSubmissionHandler(&fakes.SvcSubmission{})
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int64(7)) })
	r.PUT("/submissions", h.Save)

	w := subDo(r, http.MethodPut, "/submissions", `{"comment":"x"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubmissionHandler_Save_InvalidJSONReturns400(t *testing.T) {
	h := handlers.NewSubmissionHandler(&fakes.SvcSubmission{})
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int64(7)) })
	r.PUT("/submissions", h.Save)

	w := subDo(r, http.MethodPut, "/submissions", `{"place_id":1,`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubmissionHandler_Save_EmptyMapsTo400(t *testing.T) {
	// La validacion de "comment vacio + sin fotos" vive en el servicio
	// (sentinel ErrEmptySubmission). Aqui verificamos que el handler lo
	// mapea a 400 correctamente via serviceErrorMappings.
	svc := &fakes.SvcSubmission{}
	svc.SaveFn = func(_ context.Context, _ int64, _ models.SubmissionRequest) (*models.Submission, error) {
		return nil, services.ErrEmptySubmission
	}
	h := handlers.NewSubmissionHandler(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int64(7)) })
	r.PUT("/submissions", h.Save)

	w := subDo(r, http.MethodPut, "/submissions",
		`{"place_id":1,"comment":"x"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubmissionHandler_Save_HappyPathReturns200(t *testing.T) {
	svc := &fakes.SvcSubmission{}
	svc.SaveFn = func(_ context.Context, userID int64, req models.SubmissionRequest) (*models.Submission, error) {
		return &models.Submission{ID: 1, UserID: userID, PlaceID: req.PlaceID}, nil
	}
	h := handlers.NewSubmissionHandler(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int64(7)) })
	r.PUT("/submissions", h.Save)

	w := subDo(r, http.MethodPut, "/submissions",
		`{"place_id":1,"comment":"ok"}`)
	assert.Equal(t, http.StatusOK, w.Code)
}
