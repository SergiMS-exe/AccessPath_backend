package middleware

import (
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// sensitiveQueryKeys son query params que NUNCA deben aparecer en logs. Sus
// valores se reemplazan por "[REDACTED]" en la URL que se loguea.
var sensitiveQueryKeys = map[string]bool{
	"password":     true,
	"token":        true,
	"refresh":      true,
	"session":      true,
	"sessiontoken": true,
	"apikey":       true,
	"api_key":      true,
	"secret":       true,
}

// Logger emite una linea estructurada por peticion con method, path, status,
// latencia y request id (si el middleware RequestID() se registro antes). El
// nivel sigue el codigo HTTP: 5xx=ERROR, 4xx=WARN, resto=INFO.
//
// Importante: NO loguea headers ni body (Authorization, payloads, etc).
// Los query params sensibles (password, token, etc.) se redactan a [REDACTED].
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		slog.LogAttrs(c.Request.Context(), levelForStatus(status), "http_request",
			slog.String("method", c.Request.Method),
			slog.String("path", sanitizePath(c.Request.URL)),
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

// sanitizePath devuelve path + query con los valores sensibles redactados.
func sanitizePath(u *url.URL) string {
	if u.RawQuery == "" {
		return u.Path
	}
	values := u.Query()
	redacted := false
	for k, vs := range values {
		if sensitiveQueryKeys[strings.ToLower(k)] {
			values[k] = []string{"[REDACTED]"}
			redacted = true
		}
		_ = vs
	}
	if !redacted {
		return u.Path + "?" + u.RawQuery
	}
	return u.Path + "?" + values.Encode()
}