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

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"accesspath/tests/internal/fakes"
)

func colDo(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
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

// --- GetByUser ------------------------------------------------------------

func TestCollectionHandler_GetByUser_InvalidIDReturns400(t *testing.T) {
	h := handlers.NewCollectionHandler(&fakes.SvcCollection{})
	r := gin.New()
	r.GET("/users/:id/collections", h.GetByUser)

	w := colDo(r, http.MethodGet, "/users/abc/collections", "")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCollectionHandler_GetByUser_HappyPathReturns200(t *testing.T) {
	svc := &fakes.SvcCollection{}
	svc.GetByUserFn = func(_ context.Context, _ int64) ([]models.Collection, error) {
		return []models.Collection{}, nil
	}
	h := handlers.NewCollectionHandler(svc)
	r := gin.New()
	r.GET("/users/:id/collections", h.GetByUser)

	w := colDo(r, http.MethodGet, "/users/7/collections", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Create ---------------------------------------------------------------

func TestCollectionHandler_Create_InvalidJSONReturns400(t *testing.T) {
	h := handlers.NewCollectionHandler(&fakes.SvcCollection{})
	r := gin.New()
	r.POST("/collections", h.Create)

	w := colDo(r, http.MethodPost, "/collections", `{`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCollectionHandler_Create_MissingUserIDReturns400(t *testing.T) {
	h := handlers.NewCollectionHandler(&fakes.SvcCollection{})
	r := gin.New()
	r.POST("/collections", h.Create)

	w := colDo(r, http.MethodPost, "/collections", `{"name":"favoritos"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCollectionHandler_Create_HappyPathReturns201(t *testing.T) {
	svc := &fakes.SvcCollection{}
	svc.CreateFn = func(_ context.Context, req models.CreateCollectionRequest) (*models.Collection, error) {
		return &models.Collection{ID: 1, UserID: req.UserID, Name: req.Name}, nil
	}
	h := handlers.NewCollectionHandler(svc)
	r := gin.New()
	r.POST("/collections", h.Create)

	w := colDo(r, http.MethodPost, "/collections",
		`{"user_id":42,"name":"favoritos"}`)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCollectionHandler_Create_ServiceErrorMapsToInternal(t *testing.T) {
	svc := &fakes.SvcCollection{}
	svc.CreateFn = func(_ context.Context, _ models.CreateCollectionRequest) (*models.Collection, error) {
		return nil, errors.New("db boom")
	}
	h := handlers.NewCollectionHandler(svc)
	r := gin.New()
	r.POST("/collections", h.Create)

	w := colDo(r, http.MethodPost, "/collections",
		`{"user_id":42,"name":"favoritos"}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- Delete ---------------------------------------------------------------

func TestCollectionHandler_Delete_InvalidIDReturns400(t *testing.T) {
	h := handlers.NewCollectionHandler(&fakes.SvcCollection{})
	r := gin.New()
	r.DELETE("/collections/:id", h.Delete)

	w := colDo(r, http.MethodDelete, "/collections/abc", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCollectionHandler_Delete_HappyPathReturns204(t *testing.T) {
	svc := &fakes.SvcCollection{}
	svc.DeleteFn = func(_ context.Context, _ int64) error { return nil }
	h := handlers.NewCollectionHandler(svc)
	r := gin.New()
	r.DELETE("/collections/:id", h.Delete)

	w := colDo(r, http.MethodDelete, "/collections/1", "")
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// --- GetPlaces ------------------------------------------------------------

func TestCollectionHandler_GetPlaces_InvalidIDReturns400(t *testing.T) {
	h := handlers.NewCollectionHandler(&fakes.SvcCollection{})
	r := gin.New()
	r.GET("/collections/:id/places", h.GetPlaces)

	w := colDo(r, http.MethodGet, "/collections/abc/places", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCollectionHandler_GetPlaces_HappyPathReturns200(t *testing.T) {
	svc := &fakes.SvcCollection{}
	svc.GetPlacesFn = func(_ context.Context, _ int64) ([]models.Place, error) {
		return []models.Place{}, nil
	}
	h := handlers.NewCollectionHandler(svc)
	r := gin.New()
	r.GET("/collections/:id/places", h.GetPlaces)

	w := colDo(r, http.MethodGet, "/collections/1/places", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- AddPlace / RemovePlace ----------------------------------------------

func TestCollectionHandler_AddPlace_InvalidCollectionIDReturns400(t *testing.T) {
	h := handlers.NewCollectionHandler(&fakes.SvcCollection{})
	r := gin.New()
	r.POST("/collections/:id/places/:placeId", h.AddPlace)

	w := colDo(r, http.MethodPost, "/collections/abc/places/1", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCollectionHandler_AddPlace_InvalidPlaceIDReturns400(t *testing.T) {
	h := handlers.NewCollectionHandler(&fakes.SvcCollection{})
	r := gin.New()
	r.POST("/collections/:id/places/:placeId", h.AddPlace)

	w := colDo(r, http.MethodPost, "/collections/1/places/abc", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCollectionHandler_AddPlace_HappyPathReturns201(t *testing.T) {
	svc := &fakes.SvcCollection{}
	svc.AddPlaceFn = func(_ context.Context, _, _ int64) error { return nil }
	h := handlers.NewCollectionHandler(svc)
	r := gin.New()
	r.POST("/collections/:id/places/:placeId", h.AddPlace)

	w := colDo(r, http.MethodPost, "/collections/1/places/2", "")
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCollectionHandler_RemovePlace_InvalidCollectionIDReturns400(t *testing.T) {
	h := handlers.NewCollectionHandler(&fakes.SvcCollection{})
	r := gin.New()
	r.DELETE("/collections/:id/places/:placeId", h.RemovePlace)

	w := colDo(r, http.MethodDelete, "/collections/abc/places/1", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCollectionHandler_RemovePlace_InvalidPlaceIDReturns400(t *testing.T) {
	h := handlers.NewCollectionHandler(&fakes.SvcCollection{})
	r := gin.New()
	r.DELETE("/collections/:id/places/:placeId", h.RemovePlace)

	w := colDo(r, http.MethodDelete, "/collections/1/places/abc", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCollectionHandler_RemovePlace_HappyPathReturns204(t *testing.T) {
	svc := &fakes.SvcCollection{}
	svc.RemovePlaceFn = func(_ context.Context, _, _ int64) error { return nil }
	h := handlers.NewCollectionHandler(svc)
	r := gin.New()
	r.DELETE("/collections/:id/places/:placeId", h.RemovePlace)

	w := colDo(r, http.MethodDelete, "/collections/1/places/2", "")
	assert.Equal(t, http.StatusNoContent, w.Code)
}
