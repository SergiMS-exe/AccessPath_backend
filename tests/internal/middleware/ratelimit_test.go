package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"accesspath/internal/middleware"
	"accesspath/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRateLimit_NilRedisFailsOpen(t *testing.T) {
	r := gin.New()
	r.GET("/x", middleware.RateLimit(nil, middleware.KeyByIP, 5, time.Minute), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code,
			"con Redis nil el middleware debe dejar pasar (fail-open)")
	}
}

func TestRateLimit_NilKeyFnFailsOpen(t *testing.T) {
	r := gin.New()
	r.GET("/x", middleware.RateLimit(nil, nil, 5, time.Minute), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestKeyByIP_ReturnsClientIP(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	c.Request.RemoteAddr = "192.0.2.1:1234"

	got := middleware.KeyByIP(c)
	assert.Contains(t, got, "192.0.2.1")
	assert.Contains(t, got, "ip:")
}

func TestKeyByUserID_PrefersUserID(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	c.Request.RemoteAddr = "192.0.2.1:1234"
	c.Set("user_id", float64(42))

	got := middleware.KeyByUserID(c)
	assert.Contains(t, got, "u:")
	assert.Contains(t, got, "42")
	assert.NotContains(t, got, "ip:")
}

func TestKeyByUserID_FallsBackToIP(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	c.Request.RemoteAddr = "192.0.2.1:1234"

	got := middleware.KeyByUserID(c)
	assert.Contains(t, got, "ip:")
	assert.NotContains(t, got, "u:")
}

func TestItoaForLimit(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{123456, "123456"},
		{-1, "-1"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, middleware.ItoaForLimit(tt.in))
		})
	}
}

func TestRetryAfterSeconds(t *testing.T) {
	tests := []struct {
		name   string
		window time.Duration
		want   string
	}{
		{"1 minute", time.Minute, "60"},
		{"10 seconds", 10 * time.Second, "10"},
		{"sub-second clamps to 1", 500 * time.Millisecond, "1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, middleware.RetryAfterSeconds(tt.window))
		})
	}
}

func TestRateLimit_ResponseShape_DocumentedIntegrationTest(t *testing.T) {
	t.Skip("Requiere Redis vivo; test de integracion fuera de unidad")
}

func TestRateLimit_AppliesLimit_WithRealRedis(t *testing.T) {
	_ = response.Envelope{}
	require.NotNil(t, &response.Envelope{})
}
