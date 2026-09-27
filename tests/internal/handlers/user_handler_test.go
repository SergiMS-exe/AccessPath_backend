package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"accesspath/internal/handlers"
	"accesspath/internal/models"
	"accesspath/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"accesspath/tests/internal/fakes"
)

const testJWTSecret = "test-secret-for-handler-tests-32+chars"

func userDo(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
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

// --- Register -------------------------------------------------------------

func TestUserHandler_Register_InvalidJSONReturns400(t *testing.T) {
	h := handlers.NewUserHandler(&fakes.SvcUser{}, testJWTSecret)
	r := gin.New()
	r.POST("/auth/register", h.Register)

	w := userDo(r, http.MethodPost, "/auth/register", `{`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_Register_EmailRequiredReturns400(t *testing.T) {
	h := handlers.NewUserHandler(&fakes.SvcUser{}, testJWTSecret)
	r := gin.New()
	r.POST("/auth/register", h.Register)

	w := userDo(r, http.MethodPost, "/auth/register",
		`{"username":"u","password":"abcdef"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_Register_ShortPasswordReturns400(t *testing.T) {
	h := handlers.NewUserHandler(&fakes.SvcUser{}, testJWTSecret)
	r := gin.New()
	r.POST("/auth/register", h.Register)

	w := userDo(r, http.MethodPost, "/auth/register",
		`{"email":"a@b.com","password":"123"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_Register_EmailTakenReturns400(t *testing.T) {
	svc := &fakes.SvcUser{}
	svc.RegisterFn = func(_ context.Context, _ models.CreateUserRequest) (*models.User, error) {
		return nil, services.ErrEmailAlreadyUsed
	}
	h := handlers.NewUserHandler(svc, testJWTSecret)
	r := gin.New()
	r.POST("/auth/register", h.Register)

	w := userDo(r, http.MethodPost, "/auth/register",
		`{"email":"juan@example.com","password":"abcdef"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	env := decodeEnvelope(t, w.Body.Bytes())
	assert.Contains(t, env.Error, "email")
}

func TestUserHandler_Register_HappyPathReturns201WithTokens(t *testing.T) {
	svc := &fakes.SvcUser{}
	svc.RegisterFn = func(_ context.Context, req models.CreateUserRequest) (*models.User, error) {
		return &models.User{ID: 7, Username: req.Username, Email: req.Email}, nil
	}
	h := handlers.NewUserHandler(svc, testJWTSecret)
	r := gin.New()
	r.POST("/auth/register", h.Register)

	w := userDo(r, http.MethodPost, "/auth/register",
		`{"username":"juan","email":"juan@example.com","password":"abcdef"}`)

	assert.Equal(t, http.StatusCreated, w.Code)
	var body struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
		User         models.User
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.NotEmpty(t, body.Token)
	assert.NotEmpty(t, body.RefreshToken)
}

// --- Login ----------------------------------------------------------------

func TestUserHandler_Login_InvalidJSONReturns400(t *testing.T) {
	h := handlers.NewUserHandler(&fakes.SvcUser{}, testJWTSecret)
	r := gin.New()
	r.POST("/auth/login", h.Login)

	w := userDo(r, http.MethodPost, "/auth/login", `not-json`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_Login_EmailRequiredReturns400(t *testing.T) {
	h := handlers.NewUserHandler(&fakes.SvcUser{}, testJWTSecret)
	r := gin.New()
	r.POST("/auth/login", h.Login)

	w := userDo(r, http.MethodPost, "/auth/login", `{"password":"abcdef"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_Login_BadCredentialsReturns401(t *testing.T) {
	svc := &fakes.SvcUser{}
	svc.LoginFn = func(_ context.Context, _ models.LoginRequest) (*models.User, error) {
		return nil, services.ErrInvalidCredentials
	}
	h := handlers.NewUserHandler(svc, testJWTSecret)
	r := gin.New()
	r.POST("/auth/login", h.Login)

	w := userDo(r, http.MethodPost, "/auth/login",
		`{"email":"a@b.com","password":"abcdef"}`)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUserHandler_Login_HappyPathReturns200WithTokens(t *testing.T) {
	svc := &fakes.SvcUser{}
	svc.LoginFn = func(_ context.Context, _ models.LoginRequest) (*models.User, error) {
		return &models.User{ID: 7, Username: "juan", Email: "a@b.com"}, nil
	}
	h := handlers.NewUserHandler(svc, testJWTSecret)
	r := gin.New()
	r.POST("/auth/login", h.Login)

	w := userDo(r, http.MethodPost, "/auth/login",
		`{"email":"a@b.com","password":"abcdef"}`)

	assert.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.NotEmpty(t, body.Token)
	assert.NotEmpty(t, body.RefreshToken)
}

// --- Refresh --------------------------------------------------------------

func TestUserHandler_Refresh_InvalidJSONReturns400(t *testing.T) {
	h := handlers.NewUserHandler(&fakes.SvcUser{}, testJWTSecret)
	r := gin.New()
	r.POST("/auth/refresh", h.Refresh)

	w := userDo(r, http.MethodPost, "/auth/refresh", `not-json`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_Refresh_TokenMissingReturns400(t *testing.T) {
	h := handlers.NewUserHandler(&fakes.SvcUser{}, testJWTSecret)
	r := gin.New()
	r.POST("/auth/refresh", h.Refresh)

	w := userDo(r, http.MethodPost, "/auth/refresh", `{}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_Refresh_MalformedTokenReturns401(t *testing.T) {
	h := handlers.NewUserHandler(&fakes.SvcUser{}, testJWTSecret)
	r := gin.New()
	r.POST("/auth/refresh", h.Refresh)

	w := userDo(r, http.MethodPost, "/auth/refresh",
		`{"refresh_token":"not.a.jwt"}`)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	env := decodeEnvelope(t, w.Body.Bytes())
	assert.Contains(t, env.Error, "refresh token invalido")
}

func TestUserHandler_Refresh_AccessTokenRejectedAs401(t *testing.T) {
	tok := signTestToken(t, testJWTSecret, 42, time.Now().Add(time.Hour), "")

	h := handlers.NewUserHandler(&fakes.SvcUser{}, testJWTSecret)
	r := gin.New()
	r.POST("/auth/refresh", h.Refresh)

	w := userDo(r, http.MethodPost, "/auth/refresh",
		`{"refresh_token":"`+tok+`"}`)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUserHandler_Refresh_DifferentSecretReturns401(t *testing.T) {
	tok := signTestToken(t, "OTHER-SECRET-DIFFERENT-32-CHARS-XYZ",
		42, time.Now().Add(time.Hour), "refresh")

	h := handlers.NewUserHandler(&fakes.SvcUser{}, testJWTSecret)
	r := gin.New()
	r.POST("/auth/refresh", h.Refresh)

	w := userDo(r, http.MethodPost, "/auth/refresh",
		`{"refresh_token":"`+tok+`"}`)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUserHandler_Refresh_ValidRefreshTokenReturns200(t *testing.T) {
	tok := signTestToken(t, testJWTSecret, 42, time.Now().Add(time.Hour), "refresh")

	h := handlers.NewUserHandler(&fakes.SvcUser{}, testJWTSecret)
	r := gin.New()
	r.POST("/auth/refresh", h.Refresh)

	w := userDo(r, http.MethodPost, "/auth/refresh",
		`{"refresh_token":"`+tok+`"}`)

	assert.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	assert.NotEmpty(t, body.Token, "se emite nuevo access token")
	assert.NotEmpty(t, body.RefreshToken, "se emite nuevo refresh token")
	assert.NotEqual(t, tok, body.RefreshToken, "refresh token es rotatorio")
}

// --- GetProfile -----------------------------------------------------------

func TestUserHandler_GetProfile_InvalidIDReturns400(t *testing.T) {
	h := handlers.NewUserHandler(&fakes.SvcUser{}, testJWTSecret)
	r := gin.New()
	r.GET("/users/:id", h.GetProfile)

	w := userDo(r, http.MethodGet, "/users/abc", "")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	env := decodeEnvelope(t, w.Body.Bytes())
	assert.Contains(t, env.Error, "Invalid user ID")
}

func TestUserHandler_GetProfile_HappyPathReturns200(t *testing.T) {
	svc := &fakes.SvcUser{}
	svc.GetByIDFn = func(_ context.Context, id int64) (*models.User, error) {
		return &models.User{ID: id, Username: "x"}, nil
	}
	h := handlers.NewUserHandler(svc, testJWTSecret)
	r := gin.New()
	r.GET("/users/:id", h.GetProfile)

	w := userDo(r, http.MethodGet, "/users/7", "")

	assert.Equal(t, http.StatusOK, w.Code)
}

// --- helpers --------------------------------------------------------------

func signTestToken(t *testing.T, secret string, userID int64, exp time.Time, withType string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     exp.Unix(),
	}
	if withType != "" {
		claims["type"] = withType
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(secret))
	require.NoError(t, err)
	return signed
}
