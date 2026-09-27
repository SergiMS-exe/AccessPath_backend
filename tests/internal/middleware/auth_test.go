package middleware_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"accesspath/internal/middleware"
	"accesspath/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

const testSecret = "test-secret-32-chars-long-1234567890"

func signWithSecret(t *testing.T, secret string, userID int64, exp time.Time, withType string) string {
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

func newAuthRouter() *gin.Engine {
	r := gin.New()
	r.GET("/protected", middleware.Auth(testSecret), func(c *gin.Context) {
		uid, ok := middleware.UserID(c)
		if !ok {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.Header("X-Test-UserID", strconv.FormatInt(uid, 10))
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestAuth_NoHeaderReturns401(t *testing.T) {
	r := newAuthRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Empty(t, w.Header().Get("X-Test-UserID"))
}

func TestAuth_EmptyHeaderReturns401(t *testing.T) {
	r := newAuthRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_NonBearerSchemeReturns401(t *testing.T) {
	r := newAuthRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_BearerWithoutTokenReturns401(t *testing.T) {
	r := newAuthRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer ")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_TamperedTokenReturns401(t *testing.T) {
	r := newAuthRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer not.a.valid.jwt")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_AlgNoneReturns401(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString(
		[]byte(`{"user_id":123,"exp":9999999999}`),
	)
	noneToken := header + "." + payload + "."

	r := newAuthRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+noneToken)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"tokens con alg=none deben rechazarse (algoritmo confusion attack)")
}

func TestAuth_TokenSignedWithDifferentSecretReturns401(t *testing.T) {
	wrongSecretToken := signWithSecret(t, "OTHER-SECRET-32-CHARS-LONG-XYZXY", 42, time.Now().Add(time.Hour), "")

	r := newAuthRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+wrongSecretToken)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"token firmado con otro secreto debe rechazarse")
}

func TestAuth_ExpiredTokenReturns401(t *testing.T) {
	expired := signWithSecret(t, testSecret, 42, time.Now().Add(-time.Hour), "")

	r := newAuthRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+expired)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"token expirado debe rechazarse")
}

func TestAuth_RefreshTokenReturns401(t *testing.T) {
	refreshToken := signWithSecret(t, testSecret, 42, time.Now().Add(time.Hour), "refresh")

	r := newAuthRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+refreshToken)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"refresh tokens no pueden pasar por rutas protegidas (solo /auth/refresh)")
}

func TestAuth_ValidTokenInjectsUserID(t *testing.T) {
	token := signWithSecret(t, testSecret, 42, time.Now().Add(time.Hour), "")

	r := newAuthRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "42", w.Header().Get("X-Test-UserID"))
}

func TestAuth_ResponseEnvelopeOnFailure(t *testing.T) {
	r := newAuthRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var env response.Envelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.NotEmpty(t, env.Error)
}

func TestUserID_ExtractsFromFloat64(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("user_id", float64(42))

	uid, ok := middleware.UserID(c)
	assert.True(t, ok)
	assert.Equal(t, int64(42), uid)
}

func TestUserID_AcceptsInt64(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("user_id", int64(42))

	uid, ok := middleware.UserID(c)
	assert.True(t, ok)
	assert.Equal(t, int64(42), uid)
}

func TestUserID_AcceptsInt(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("user_id", 42)

	uid, ok := middleware.UserID(c)
	assert.True(t, ok)
	assert.Equal(t, int64(42), uid)
}

func TestUserID_MissingReturnsFalse(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	uid, ok := middleware.UserID(c)
	assert.False(t, ok)
	assert.Equal(t, int64(0), uid)
}

func TestUserID_WrongTypeReturnsFalse(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("user_id", "not a number")

	uid, ok := middleware.UserID(c)
	assert.False(t, ok)
	assert.Equal(t, int64(0), uid)
}
