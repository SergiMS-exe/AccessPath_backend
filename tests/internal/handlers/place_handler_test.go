package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"accesspath/internal/handlers"
	"accesspath/internal/models"
	"accesspath/internal/services"
	"accesspath/pkg/apperr"
	"accesspath/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"accesspath/tests/internal/fakes"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// withUserID inyecta un user_id en el contexto igual que hace el middleware
// Auth tras validar el JWT. La clave "user_id" es la que usa middleware.Auth.
func withUserID(userID int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
	}
}

func newPlaceRouter(method, path string, useAuth bool, h gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	if useAuth {
		r.Use(withUserID(42))
	}
	switch method {
	case http.MethodGet:
		r.GET(path, h)
	case http.MethodPost:
		r.POST(path, h)
	case http.MethodPut:
		r.PUT(path, h)
	case http.MethodDelete:
		r.DELETE(path, h)
	default:
		panic("metodo no soportado: " + method)
	}
	return r
}

func do(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
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

func decodeEnvelope(t *testing.T, body []byte) response.Envelope {
	t.Helper()
	var env response.Envelope
	require.NoError(t, json.Unmarshal(body, &env))
	return env
}

// --- GetAll: parser y delegacion al servicio ------------------------------

func TestPlaceHandler_GetAll_PassesThroughFilters(t *testing.T) {
	svc := &fakes.SvcPlace{}
	svc.GetAllFn = func(_ context.Context, f models.PlaceFilters) (*models.PlaceListResult, error) {
		return &models.PlaceListResult{
			Places: []models.Place{{ID: 1, Name: "x"}},
			Total:  1, Limit: f.Limit, Offset: f.Offset,
		}, nil
	}
	h := handlers.NewPlaceHandler(svc)

	r := newPlaceRouter(http.MethodGet, "/places", false, h.GetAll)
	w := do(r, http.MethodGet, "/places?limit=10&offset=5", "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"data":{"places":[{"id":1,"code":"","name":"x","latitude":0,"longitude":0,"source":"","published":false,"moderation_status":"","created_by":0,"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"}],"total":1,"limit":10,"offset":5}}`, w.Body.String())
}

func TestPlaceHandler_GetAll_InvalidLimitFallsBackToDefault(t *testing.T) {
	var gotFilters models.PlaceFilters
	svc := &fakes.SvcPlace{}
	svc.GetAllFn = func(_ context.Context, f models.PlaceFilters) (*models.PlaceListResult, error) {
		gotFilters = f
		return &models.PlaceListResult{}, nil
	}
	h := handlers.NewPlaceHandler(svc)

	r := newPlaceRouter(http.MethodGet, "/places", false, h.GetAll)
	w := do(r, http.MethodGet, "/places?limit=abc", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 20, gotFilters.Limit, "limit no numerico -> default 20")
}

func TestPlaceHandler_GetAll_ServiceErrorMapsToInternal(t *testing.T) {
	svc := &fakes.SvcPlace{}
	svc.GetAllFn = func(_ context.Context, _ models.PlaceFilters) (*models.PlaceListResult, error) {
		return nil, errors.New("db down")
	}
	h := handlers.NewPlaceHandler(svc)

	r := newPlaceRouter(http.MethodGet, "/places", false, h.GetAll)
	w := do(r, http.MethodGet, "/places", "")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- GetByBounds: cadena de validacion ------------------------------------

func TestPlaceHandler_GetByBounds_MissingAnyQueryReturns400(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"no min_lat", "/places/map?max_lat=1&min_lng=1&max_lng=2"},
		{"no max_lat", "/places/map?min_lat=1&min_lng=1&max_lng=2"},
		{"no min_lng", "/places/map?min_lat=1&max_lat=2&max_lng=2"},
		{"no max_lng", "/places/map?min_lat=1&max_lat=2&min_lng=1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := handlers.NewPlaceHandler(&fakes.SvcPlace{})
			r := newPlaceRouter(http.MethodGet, "/places/map", false, h.GetByBounds)
			w := do(r, http.MethodGet, tt.path, "")
			assert.Equal(t, http.StatusBadRequest, w.Code)
			env := decodeEnvelope(t, w.Body.Bytes())
			assert.Contains(t, env.Error, "min_lat, max_lat, min_lng and max_lng are required")
		})
	}
}

func TestPlaceHandler_GetByBounds_NonNumericReturns400(t *testing.T) {
	h := handlers.NewPlaceHandler(&fakes.SvcPlace{})
	r := newPlaceRouter(http.MethodGet, "/places/map", false, h.GetByBounds)
	w := do(r, http.MethodGet, "/places/map?min_lat=abc&max_lat=1&min_lng=1&max_lng=2", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	env := decodeEnvelope(t, w.Body.Bytes())
	assert.Contains(t, env.Error, "min_lat")
}

func TestPlaceHandler_GetByBounds_LatOutOfRangeReturns400(t *testing.T) {
	h := handlers.NewPlaceHandler(&fakes.SvcPlace{})
	r := newPlaceRouter(http.MethodGet, "/places/map", false, h.GetByBounds)
	w := do(r, http.MethodGet, "/places/map?min_lat=-91&max_lat=1&min_lng=1&max_lng=2", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPlaceHandler_GetByBounds_LngOutOfRangeReturns400(t *testing.T) {
	h := handlers.NewPlaceHandler(&fakes.SvcPlace{})
	r := newPlaceRouter(http.MethodGet, "/places/map", false, h.GetByBounds)
	w := do(r, http.MethodGet, "/places/map?min_lat=0&max_lat=1&min_lng=181&max_lng=2", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPlaceHandler_GetByBounds_InvertedBoundsReturns400(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"min_lat >= max_lat", "/places/map?min_lat=10&max_lat=10&min_lng=1&max_lng=2"},
		{"min_lng >= max_lng", "/places/map?min_lat=1&max_lat=2&min_lng=10&max_lng=10"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := handlers.NewPlaceHandler(&fakes.SvcPlace{})
			r := newPlaceRouter(http.MethodGet, "/places/map", false, h.GetByBounds)
			w := do(r, http.MethodGet, tt.path, "")
			assert.Equal(t, http.StatusBadRequest, w.Code)
			env := decodeEnvelope(t, w.Body.Bytes())
			assert.Contains(t, env.Error, "min_lat must be less than max_lat")
		})
	}
}

func TestPlaceHandler_GetByBounds_HappyPathDelegates(t *testing.T) {
	svc := &fakes.SvcPlace{}
	svc.GetByBoundsFn = func(_ context.Context, _ models.BoundsFilter) ([]models.PlaceMapItem, error) {
		return []models.PlaceMapItem{}, nil
	}
	h := handlers.NewPlaceHandler(svc)
	r := newPlaceRouter(http.MethodGet, "/places/map", false, h.GetByBounds)
	w := do(r, http.MethodGet, "/places/map?min_lat=0&max_lat=1&min_lng=0&max_lng=1", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- GetNearby ------------------------------------------------------------

func TestPlaceHandler_GetNearby_MissingLatReturns400(t *testing.T) {
	h := handlers.NewPlaceHandler(&fakes.SvcPlace{})
	r := newPlaceRouter(http.MethodGet, "/places/nearby", false, h.GetNearby)
	w := do(r, http.MethodGet, "/places/nearby?lng=1", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPlaceHandler_GetNearby_LatOutOfRangeReturns400(t *testing.T) {
	h := handlers.NewPlaceHandler(&fakes.SvcPlace{})
	r := newPlaceRouter(http.MethodGet, "/places/nearby", false, h.GetNearby)
	w := do(r, http.MethodGet, "/places/nearby?lat=200&lng=1", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPlaceHandler_GetNearby_NonNumericRadiusFallsBackToDefault(t *testing.T) {
	svc := &fakes.SvcPlace{}
	var got models.NearbyFilter
	svc.GetNearbyFn = func(_ context.Context, f models.NearbyFilter) ([]models.PlaceWithDistance, error) {
		got = f
		return nil, nil
	}
	h := handlers.NewPlaceHandler(svc)
	r := newPlaceRouter(http.MethodGet, "/places/nearby", false, h.GetNearby)
	w := do(r, http.MethodGet, "/places/nearby?lat=1&lng=2&radius=abc", "")
	require.Equal(t, http.StatusOK, w.Code)
	assert.InDelta(t, 5.0, got.Radius, 0.001)
}

func TestPlaceHandler_GetNearby_HappyPathDelegates(t *testing.T) {
	svc := &fakes.SvcPlace{}
	svc.GetNearbyFn = func(_ context.Context, _ models.NearbyFilter) ([]models.PlaceWithDistance, error) {
		return []models.PlaceWithDistance{}, nil
	}
	h := handlers.NewPlaceHandler(svc)
	r := newPlaceRouter(http.MethodGet, "/places/nearby", false, h.GetNearby)
	w := do(r, http.MethodGet, "/places/nearby?lat=1&lng=2", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- GetByID --------------------------------------------------------------

func TestPlaceHandler_GetByID_InvalidIDReturns400(t *testing.T) {
	h := handlers.NewPlaceHandler(&fakes.SvcPlace{})
	r := newPlaceRouter(http.MethodGet, "/places/:id", false, h.GetByID)
	w := do(r, http.MethodGet, "/places/abc", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPlaceHandler_GetByID_NotFoundReturns404(t *testing.T) {
	svc := &fakes.SvcPlace{}
	svc.GetByIDFn = func(_ context.Context, _ int64) (*models.PlaceDetail, error) {
		return nil, apperr.NotFound("places.detail", "Place")
	}
	h := handlers.NewPlaceHandler(svc)
	r := newPlaceRouter(http.MethodGet, "/places/:id", false, h.GetByID)
	w := do(r, http.MethodGet, "/places/99", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPlaceHandler_GetByID_HappyPathReturns200(t *testing.T) {
	svc := &fakes.SvcPlace{}
	svc.GetByIDFn = func(_ context.Context, id int64) (*models.PlaceDetail, error) {
		return &models.PlaceDetail{Place: models.Place{ID: id, Name: "x"}}, nil
	}
	h := handlers.NewPlaceHandler(svc)
	r := newPlaceRouter(http.MethodGet, "/places/:id", false, h.GetByID)
	w := do(r, http.MethodGet, "/places/42", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Search ---------------------------------------------------------------

func TestPlaceHandler_Search_MissingQueryReturns400(t *testing.T) {
	h := handlers.NewPlaceHandler(&fakes.SvcPlace{})
	r := newPlaceRouter(http.MethodGet, "/places/search", false, h.Search)
	w := do(r, http.MethodGet, "/places/search", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPlaceHandler_Search_CatalogUnavailablePropagates503(t *testing.T) {
	svc := &fakes.SvcPlace{}
	svc.SearchFn = func(_ context.Context, _ models.SearchOptions) ([]models.PlaceSearchItem, error) {
		return nil, services.ErrCatalogUnavailable
	}
	h := handlers.NewPlaceHandler(svc)
	r := newPlaceRouter(http.MethodGet, "/places/search", false, h.Search)
	w := do(r, http.MethodGet, "/places/search?q=cafe", "")
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestPlaceHandler_Search_HappyPathReturns200(t *testing.T) {
	svc := &fakes.SvcPlace{}
	svc.SearchFn = func(_ context.Context, _ models.SearchOptions) ([]models.PlaceSearchItem, error) {
		return []models.PlaceSearchItem{{PlaceID: "p1"}}, nil
	}
	h := handlers.NewPlaceHandler(svc)
	r := newPlaceRouter(http.MethodGet, "/places/search", false, h.Search)
	w := do(r, http.MethodGet, "/places/search?q=cafe", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Create ---------------------------------------------------------------

func TestPlaceHandler_Create_MissingUserIDReturns401(t *testing.T) {
	h := handlers.NewPlaceHandler(&fakes.SvcPlace{})
	r := newPlaceRouter(http.MethodPost, "/places", false, h.Create)
	body := `{"name":"x","latitude":1,"longitude":2}`
	w := do(r, http.MethodPost, "/places", body)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPlaceHandler_Create_InvalidJSONReturns400(t *testing.T) {
	h := handlers.NewPlaceHandler(&fakes.SvcPlace{})
	r := newPlaceRouter(http.MethodPost, "/places", true, h.Create)
	w := do(r, http.MethodPost, "/places", `{"name":`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPlaceHandler_Create_InjectsCreatedByFromToken(t *testing.T) {
	svc := &fakes.SvcPlace{}
	var gotReq models.CreatePlaceRequest
	svc.CreateFn = func(_ context.Context, req models.CreatePlaceRequest) (*models.Place, error) {
		gotReq = req
		return &models.Place{ID: 1, Name: req.Name}, nil
	}
	h := handlers.NewPlaceHandler(svc)
	r := newPlaceRouter(http.MethodPost, "/places", true, h.Create)
	body := `{"name":"x","latitude":1,"longitude":2}`
	w := do(r, http.MethodPost, "/places", body)
	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, int64(42), gotReq.CreatedBy, "el handler fija CreatedBy desde el token")
}

func TestPlaceHandler_Create_HappyPathReturns201(t *testing.T) {
	svc := &fakes.SvcPlace{}
	svc.CreateFn = func(_ context.Context, req models.CreatePlaceRequest) (*models.Place, error) {
		return &models.Place{ID: 9, Name: req.Name}, nil
	}
	h := handlers.NewPlaceHandler(svc)
	r := newPlaceRouter(http.MethodPost, "/places", true, h.Create)
	body := `{"name":"x","latitude":1,"longitude":2}`
	w := do(r, http.MethodPost, "/places", body)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// --- Update ---------------------------------------------------------------

func TestPlaceHandler_Update_InvalidIDReturns400(t *testing.T) {
	h := handlers.NewPlaceHandler(&fakes.SvcPlace{})
	r := newPlaceRouter(http.MethodPut, "/places/:id", true, h.Update)
	w := do(r, http.MethodPut, "/places/abc", `{"name":"x","latitude":1,"longitude":2}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPlaceHandler_Update_MissingUserIDReturns401(t *testing.T) {
	h := handlers.NewPlaceHandler(&fakes.SvcPlace{})
	r := newPlaceRouter(http.MethodPut, "/places/:id", false, h.Update)
	body := `{"name":"x","latitude":1,"longitude":2}`
	w := do(r, http.MethodPut, "/places/1", body)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPlaceHandler_Update_LatOutOfRangeReturns400(t *testing.T) {
	h := handlers.NewPlaceHandler(&fakes.SvcPlace{})
	r := newPlaceRouter(http.MethodPut, "/places/:id", true, h.Update)
	body := `{"name":"x","latitude":200,"longitude":2}`
	w := do(r, http.MethodPut, "/places/1", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPlaceHandler_Update_DescriptionTooLongReturns400(t *testing.T) {
	h := handlers.NewPlaceHandler(&fakes.SvcPlace{})
	r := newPlaceRouter(http.MethodPut, "/places/:id", true, h.Update)
	long := strings.Repeat("a", 2001)
	body := `{"name":"x","latitude":1,"longitude":2,"description":"` + long + `"}`
	w := do(r, http.MethodPut, "/places/1", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPlaceHandler_Update_NotOwnerMapsTo401(t *testing.T) {
	svc := &fakes.SvcPlace{}
	svc.UpdateFn = func(_ context.Context, _, _ int64, _ models.UpdatePlaceRequest) (*models.Place, error) {
		return nil, services.ErrNotOwner
	}
	h := handlers.NewPlaceHandler(svc)
	r := newPlaceRouter(http.MethodPut, "/places/:id", true, h.Update)
	w := do(r, http.MethodPut, "/places/1", `{"name":"x","latitude":1,"longitude":2}`)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPlaceHandler_Update_HappyPathReturns200(t *testing.T) {
	svc := &fakes.SvcPlace{}
	svc.UpdateFn = func(_ context.Context, id, userID int64, req models.UpdatePlaceRequest) (*models.Place, error) {
		return &models.Place{ID: id, CreatedBy: userID, Name: req.Name}, nil
	}
	h := handlers.NewPlaceHandler(svc)
	r := newPlaceRouter(http.MethodPut, "/places/:id", true, h.Update)
	w := do(r, http.MethodPut, "/places/7", `{"name":"x","latitude":1,"longitude":2}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Delete ---------------------------------------------------------------

func TestPlaceHandler_Delete_InvalidIDReturns400(t *testing.T) {
	h := handlers.NewPlaceHandler(&fakes.SvcPlace{})
	r := newPlaceRouter(http.MethodDelete, "/places/:id", true, h.Delete)
	w := do(r, http.MethodDelete, "/places/abc", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPlaceHandler_Delete_MissingUserIDReturns401(t *testing.T) {
	h := handlers.NewPlaceHandler(&fakes.SvcPlace{})
	r := newPlaceRouter(http.MethodDelete, "/places/:id", false, h.Delete)
	w := do(r, http.MethodDelete, "/places/1", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPlaceHandler_Delete_NotOwnerMapsTo401(t *testing.T) {
	svc := &fakes.SvcPlace{}
	svc.DeleteFn = func(_ context.Context, _, _ int64) error { return services.ErrNotOwner }
	h := handlers.NewPlaceHandler(svc)
	r := newPlaceRouter(http.MethodDelete, "/places/:id", true, h.Delete)
	w := do(r, http.MethodDelete, "/places/1", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPlaceHandler_Delete_HappyPathReturns204(t *testing.T) {
	svc := &fakes.SvcPlace{}
	svc.DeleteFn = func(_ context.Context, _, _ int64) error { return nil }
	h := handlers.NewPlaceHandler(svc)
	r := newPlaceRouter(http.MethodDelete, "/places/:id", true, h.Delete)
	w := do(r, http.MethodDelete, "/places/1", "")
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// --- ImportFromCatalog ----------------------------------------------------

func TestPlaceHandler_ImportFromCatalog_InvalidJSONReturns400(t *testing.T) {
	h := handlers.NewPlaceHandler(&fakes.SvcPlace{})
	r := newPlaceRouter(http.MethodPost, "/places/from-catalog", true, h.ImportFromCatalog)
	w := do(r, http.MethodPost, "/places/from-catalog", `{"external_id":`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPlaceHandler_ImportFromCatalog_MissingUserIDReturns401(t *testing.T) {
	h := handlers.NewPlaceHandler(&fakes.SvcPlace{})
	r := newPlaceRouter(http.MethodPost, "/places/from-catalog", false, h.ImportFromCatalog)
	w := do(r, http.MethodPost, "/places/from-catalog", `{"external_id":"gers-1"}`)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPlaceHandler_ImportFromCatalog_HappyPathReturns201(t *testing.T) {
	svc := &fakes.SvcPlace{}
	svc.ImportFromCatalogFn = func(_ context.Context, source, externalID string, placeID, userID int64) (*models.Place, error) {
		assert.Equal(t, "overture", source)
		assert.Equal(t, "gers-1", externalID)
		assert.Equal(t, int64(0), placeID)
		assert.Equal(t, int64(42), userID)
		return &models.Place{ID: 5, Name: "X", Source: "overture"}, nil
	}
	h := handlers.NewPlaceHandler(svc)
	r := newPlaceRouter(http.MethodPost, "/places/from-catalog", true, h.ImportFromCatalog)
	w := do(r, http.MethodPost, "/places/from-catalog", `{"source":"overture","external_id":"gers-1"}`)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestPlaceHandler_ImportFromCatalog_NotFoundMapsTo404(t *testing.T) {
	svc := &fakes.SvcPlace{}
	svc.ImportFromCatalogFn = func(_ context.Context, _, _ string, _, _ int64) (*models.Place, error) {
		return nil, services.ErrCatalogItemNotFound
	}
	h := handlers.NewPlaceHandler(svc)
	r := newPlaceRouter(http.MethodPost, "/places/from-catalog", true, h.ImportFromCatalog)
	w := do(r, http.MethodPost, "/places/from-catalog", `{"external_id":"nope"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPlaceHandler_ImportFromCatalog_MissingReferenceReturns400(t *testing.T) {
	h := handlers.NewPlaceHandler(&fakes.SvcPlace{})
	r := newPlaceRouter(http.MethodPost, "/places/from-catalog", true, h.ImportFromCatalog)
	w := do(r, http.MethodPost, "/places/from-catalog", `{"source":"overture"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPlaceHandler_ImportFromCatalog_ByPlaceIDReturns201(t *testing.T) {
	svc := &fakes.SvcPlace{}
	svc.ImportFromCatalogFn = func(_ context.Context, source, externalID string, placeID, userID int64) (*models.Place, error) {
		assert.Empty(t, source)
		assert.Empty(t, externalID)
		assert.Equal(t, int64(12), placeID)
		return &models.Place{ID: 12, Name: "Local"}, nil
	}
	h := handlers.NewPlaceHandler(svc)
	r := newPlaceRouter(http.MethodPost, "/places/from-catalog", true, h.ImportFromCatalog)
	w := do(r, http.MethodPost, "/places/from-catalog", `{"place_id":12}`)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// --- ResolveLink ----------------------------------------------------------

func TestPlaceHandler_ResolveLink_MissingUserIDReturns401(t *testing.T) {
	h := handlers.NewPlaceHandler(&fakes.SvcPlace{})
	r := newPlaceRouter(http.MethodPost, "/places/resolve-link", false, h.ResolveLink)
	w := do(r, http.MethodPost, "/places/resolve-link", `{"url":"https://maps.app.goo.gl/x"}`)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPlaceHandler_ResolveLink_InvalidBodyReturns400(t *testing.T) {
	h := handlers.NewPlaceHandler(&fakes.SvcPlace{})
	r := newPlaceRouter(http.MethodPost, "/places/resolve-link", true, h.ResolveLink)
	w := do(r, http.MethodPost, "/places/resolve-link", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPlaceHandler_ResolveLink_HappyPathReturns200(t *testing.T) {
	svc := &fakes.SvcPlace{}
	svc.ResolveLinkFn = func(_ context.Context, rawURL string) (*models.ResolvedLink, error) {
		assert.Contains(t, rawURL, "maps.app.goo.gl")
		return &models.ResolvedLink{Matched: true}, nil
	}
	h := handlers.NewPlaceHandler(svc)
	r := newPlaceRouter(http.MethodPost, "/places/resolve-link", true, h.ResolveLink)
	w := do(r, http.MethodPost, "/places/resolve-link", `{"url":"https://maps.app.goo.gl/abc"}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"data":{"matched":true}}`, w.Body.String())
}

// --- CreateCommunity ------------------------------------------------------

func TestPlaceHandler_CreateCommunity_MissingUserIDReturns401(t *testing.T) {
	h := handlers.NewPlaceHandler(&fakes.SvcPlace{})
	r := newPlaceRouter(http.MethodPost, "/places/community", false, h.CreateCommunity)
	w := do(r, http.MethodPost, "/places/community", `{"name":"X","address":"Y","latitude":1,"longitude":2}`)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPlaceHandler_CreateCommunity_EmptyAddressReturns400(t *testing.T) {
	h := handlers.NewPlaceHandler(&fakes.SvcPlace{})
	r := newPlaceRouter(http.MethodPost, "/places/community", true, h.CreateCommunity)
	w := do(r, http.MethodPost, "/places/community", `{"name":"X","address":"  ","latitude":43.4,"longitude":-5.8}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPlaceHandler_CreateCommunity_InvalidLatitudeReturns400(t *testing.T) {
	h := handlers.NewPlaceHandler(&fakes.SvcPlace{})
	r := newPlaceRouter(http.MethodPost, "/places/community", true, h.CreateCommunity)
	w := do(r, http.MethodPost, "/places/community", `{"name":"X","address":"Y","latitude":200,"longitude":-5.8}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPlaceHandler_CreateCommunity_CreatedReturns201(t *testing.T) {
	svc := &fakes.SvcPlace{}
	svc.CreateCommunityFn = func(_ context.Context, req models.CreateCommunityPlaceRequest, userID int64) (*models.Place, bool, error) {
		assert.Equal(t, "Bar Nuevo", req.Name)
		assert.Equal(t, int64(42), userID)
		return &models.Place{ID: 9, Name: req.Name, ModerationStatus: "pending", Source: "community"}, true, nil
	}
	h := handlers.NewPlaceHandler(svc)
	r := newPlaceRouter(http.MethodPost, "/places/community", true, h.CreateCommunity)
	w := do(r, http.MethodPost, "/places/community",
		`{"name":"Bar Nuevo","address":"Calle X, 1","latitude":43.4,"longitude":-5.8}`)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestPlaceHandler_CreateCommunity_DuplicateReturns200(t *testing.T) {
	svc := &fakes.SvcPlace{}
	svc.CreateCommunityFn = func(_ context.Context, _ models.CreateCommunityPlaceRequest, _ int64) (*models.Place, bool, error) {
		return &models.Place{ID: 9, Name: "Bar Nuevo"}, false, nil
	}
	h := handlers.NewPlaceHandler(svc)
	r := newPlaceRouter(http.MethodPost, "/places/community", true, h.CreateCommunity)
	w := do(r, http.MethodPost, "/places/community",
		`{"name":"Bar Nuevo","address":"Calle X, 1","latitude":43.4,"longitude":-5.8}`)
	assert.Equal(t, http.StatusOK, w.Code)
}
