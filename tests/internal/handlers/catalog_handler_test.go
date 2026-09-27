package handlers_test

import (
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

func TestCatalogHandler_GetDimensions_HappyPathReturns200(t *testing.T) {
	svc := &fakes.SvcCatalog{}
	svc.GetCatalogFn = func(_ context.Context) ([]models.DimensionDetail, error) {
		return []models.DimensionDetail{}, nil
	}
	h := handlers.NewCatalogHandler(svc)
	r := gin.New()
	r.GET("/dimensions", h.GetDimensions)

	req := httptest.NewRequest(http.MethodGet, "/dimensions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCatalogHandler_GetDimensions_ServiceErrorMapsToInternal(t *testing.T) {
	svc := &fakes.SvcCatalog{}
	svc.GetCatalogFn = func(_ context.Context) ([]models.DimensionDetail, error) {
		return nil, errors.New("catalog boom")
	}
	h := handlers.NewCatalogHandler(svc)
	r := gin.New()
	r.GET("/dimensions", h.GetDimensions)

	req := httptest.NewRequest(http.MethodGet, "/dimensions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
