package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger emite una linea estructurada por peticion con method, path, status,
// latencia y request id (si el middleware RequestID() se registro antes). El
// nivel sigue el codigo HTTP: 5xx=ERROR, 4xx=WARN, resto=INFO.
//
// Importante: NO loguea headers ni body (Authorization, payloads, etc).
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		slog.LogAttrs(c.Request.Context(), levelForStatus(status), "http_request",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", status),
			slog.Int64("latency_ms", latency.Milliseconds()),
			slog.String("client_ip", c.ClientIP()),
			slog.String("request_id", c.GetString("request_id")),
		)
	}
}

func levelForStatus(status int) slog.Level {
	if status >= 500 {
		return slog.LevelError
	}
	if status >= 400 {
		return slog.LevelWarn
	}
	return slog.LevelInfo
}