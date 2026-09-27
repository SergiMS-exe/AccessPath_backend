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

func contribDo(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
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

// --- NextQuestion ---------------------------------------------------------

func TestContributionHandler_NextQuestion_MissingUserIDReturns401(t *testing.T) {
	h := handlers.NewContributionHandler(&fakes.SvcContribution{}, &fakes.SvcQuestion{})
	r := gin.New()
	r.GET("/places/:id/next-question", h.NextQuestion)

	w := contribDo(r, http.MethodGet, "/places/1/next-question", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestContributionHandler_NextQuestion_InvalidPlaceIDReturns400(t *testing.T) {
	h := handlers.NewContributionHandler(&fakes.SvcContribution{}, &fakes.SvcQuestion{})
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int64(7)) })
	r.GET("/places/:id/next-question", h.NextQuestion)

	w := contribDo(r, http.MethodGet, "/places/abc/next-question", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestContributionHandler_NextQuestion_HappyPathReturns200(t *testing.T) {
	qsvc := &fakes.SvcQuestion{}
	qsvc.NextFn = func(_ context.Context, _, _ int64) (*models.NextQuestionResponse, error) {
		return &models.NextQuestionResponse{Criterion: &models.Criterion{ID: 1}}, nil
	}
	h := handlers.NewContributionHandler(&fakes.SvcContribution{}, qsvc)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int64(7)) })
	r.GET("/places/:id/next-question", h.NextQuestion)

	w := contribDo(r, http.MethodGet, "/places/1/next-question", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Create ---------------------------------------------------------------

func TestContributionHandler_Create_MissingUserIDReturns401(t *testing.T) {
	h := handlers.NewContributionHandler(&fakes.SvcContribution{}, &fakes.SvcQuestion{})
	r := gin.New()
	r.POST("/contributions", h.Create)

	w := contribDo(r, http.MethodPost, "/contributions",
		`{"place_id":1,"criterion_id":1,"answer_option_id":1}`)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestContributionHandler_Create_InvalidJSONReturns400(t *testing.T) {
	h := handlers.NewContributionHandler(&fakes.SvcContribution{}, &fakes.SvcQuestion{})
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int64(7)) })
	r.POST("/contributions", h.Create)

	w := contribDo(r, http.MethodPost, "/contributions", `{`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestContributionHandler_Create_MissingFieldsReturns400(t *testing.T) {
	h := handlers.NewContributionHandler(&fakes.SvcContribution{}, &fakes.SvcQuestion{})
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int64(7)) })
	r.POST("/contributions", h.Create)

	w := contribDo(r, http.MethodPost, "/contributions", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestContributionHandler_Create_OptionMismatchMapsTo400(t *testing.T) {
	svc := &fakes.SvcContribution{}
	svc.CreateFn = func(_ context.Context, _ int64, _ models.ContributionRequest) (*models.ContributionResult, error) {
		return nil, services.ErrOptionMismatch
	}
	h := handlers.NewContributionHandler(svc, &fakes.SvcQuestion{})
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int64(7)) })
	r.POST("/contributions", h.Create)

	w := contribDo(r, http.MethodPost, "/contributions",
		`{"place_id":1,"criterion_id":1,"answer_option_id":1}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestContributionHandler_Create_HappyPathReturns201(t *testing.T) {
	svc := &fakes.SvcContribution{}
	svc.CreateFn = func(_ context.Context, userID int64, _ models.ContributionRequest) (*models.ContributionResult, error) {
		id := int64(99)
		return &models.ContributionResult{ContributionID: &id, Criterion: models.CriterionScore{
			CriterionID: 1, State: models.StateGreen,
		}}, nil
	}
	h := handlers.NewContributionHandler(svc, &fakes.SvcQuestion{})
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int64(7)) })
	r.POST("/contributions", h.Create)

	w := contribDo(r, http.MethodPost, "/contributions",
		`{"place_id":1,"criterion_id":1,"answer_option_id":1}`)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestContributionHandler_Create_ServiceErrorMapsToInternal(t *testing.T) {
	svc := &fakes.SvcContribution{}
	svc.CreateFn = func(_ context.Context, _ int64, _ models.ContributionRequest) (*models.ContributionResult, error) {
		return nil, errors.New("db boom")
	}
	h := handlers.NewContributionHandler(svc, &fakes.SvcQuestion{})
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int64(7)) })
	r.POST("/contributions", h.Create)

	w := contribDo(r, http.MethodPost, "/contributions",
		`{"place_id":1,"criterion_id":1,"answer_option_id":1}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- Delete ---------------------------------------------------------------

func TestContributionHandler_Delete_MissingUserIDReturns401(t *testing.T) {
	h := handlers.NewContributionHandler(&fakes.SvcContribution{}, &fakes.SvcQuestion{})
	r := gin.New()
	r.DELETE("/contributions/:id", h.Delete)

	w := contribDo(r, http.MethodDelete, "/contributions/1", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestContributionHandler_Delete_InvalidIDReturns400(t *testing.T) {
	h := handlers.NewContributionHandler(&fakes.SvcContribution{}, &fakes.SvcQuestion{})
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int64(7)) })
	r.DELETE("/contributions/:id", h.Delete)

	w := contribDo(r, http.MethodDelete, "/contributions/abc", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestContributionHandler_Delete_NotOwnerMapsTo401(t *testing.T) {
	svc := &fakes.SvcContribution{}
	svc.DeleteFn = func(_ context.Context, _, _ int64) (*models.ContributionResult, error) {
		return nil, services.ErrNotOwner
	}
	h := handlers.NewContributionHandler(svc, &fakes.SvcQuestion{})
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int64(7)) })
	r.DELETE("/contributions/:id", h.Delete)

	w := contribDo(r, http.MethodDelete, "/contributions/1", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestContributionHandler_Delete_NotFoundMapsTo404(t *testing.T) {
	svc := &fakes.SvcContribution{}
	svc.DeleteFn = func(_ context.Context, _, _ int64) (*models.ContributionResult, error) {
		return nil, services.ErrContributionNotFound
	}
	h := handlers.NewContributionHandler(svc, &fakes.SvcQuestion{})
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int64(7)) })
	r.DELETE("/contributions/:id", h.Delete)

	w := contribDo(r, http.MethodDelete, "/contributions/1", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestContributionHandler_Delete_HappyPathReturns200(t *testing.T) {
	svc := &fakes.SvcContribution{}
	svc.DeleteFn = func(_ context.Context, _, _ int64) (*models.ContributionResult, error) {
		return &models.ContributionResult{Criterion: models.CriterionScore{State: models.StateGreen}}, nil
	}
	h := handlers.NewContributionHandler(svc, &fakes.SvcQuestion{})
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int64(7)) })
	r.DELETE("/contributions/:id", h.Delete)

	w := contribDo(r, http.MethodDelete, "/contributions/1", "")
	assert.Equal(t, http.StatusOK, w.Code)
}
