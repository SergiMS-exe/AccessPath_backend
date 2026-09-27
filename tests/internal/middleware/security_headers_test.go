package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"accesspath/internal/config"
	"accesspath/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func newRouterWithSecurityHeaders(env string) *gin.Engine {
	r := gin.New()
	cfg := &config.Config{Env: env}
	r.Use(middleware.SecurityHeaders(cfg))
	r.GET("/x", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestSecurityHeaders_AlwaysPresent(t *testing.T) {
	tests := []struct {
		env string
	}{
		{"development"},
		{"production"},
		{"test"},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			r := newRouterWithSecurityHeaders(tt.env)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
			assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
			assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
			assert.Contains(t, w.Header().Get("Permissions-Policy"), "camera=()")
		})
	}
}

func TestSecurityHeaders_HSTSOnlyInProduction(t *testing.T) {
	tests := []struct {
		env      string
		wantHSTS string
	}{
		{"production", "max-age=31536000; includeSubDomains"},
		{"development", ""},
		{"test", ""},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			r := newRouterWithSecurityHeaders(tt.env)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantHSTS, w.Header().Get("Strict-Transport-Security"))
		})
	}
}

func TestSecurityHeaders_NilConfigDoesNotPanic(t *testing.T) {
	r := gin.New()
	r.Use(middleware.SecurityHeaders(nil))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)

	assert.NotPanics(t, func() {
		r.ServeHTTP(w, req)
	})
}
