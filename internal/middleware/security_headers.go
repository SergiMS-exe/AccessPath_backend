package middleware

import (
	"accesspath/internal/config"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders anade cabeceras de seguridad a TODAS las respuestas:
//   - X-Content-Type-Options: nosniff (evita MIME sniffing del browser)
//   - X-Frame-Options: DENY (anti clickjacking)
//   - Referrer-Policy: strict-origin-when-cross-origin
//   - Permissions-Policy: deshabilita APIs que la app no usa (camera, mic, geo...)
//
// En produccion (cfg.Env == "production") se anade ademas HSTS para forzar
// HTTPS durante un ano en navegadores que lo respeten. NO se anade en dev
// porque romperia http://localhost y los tests.
func SecurityHeaders(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy",
			"camera=(), microphone=(), geolocation=(self), payment=()")
		if cfg != nil && cfg.Env == "production" {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}
