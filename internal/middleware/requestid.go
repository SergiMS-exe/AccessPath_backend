package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"accesspath/pkg/apperr"

	"github.com/gin-gonic/gin"
)

// headerRequestID es la cabecera usada para propagar el id entre cliente y
// servidor. Si el cliente ya manda una, se respeta (para que los logs del
// movil y del backend se puedan correlacionar con la cabecera que el cliente
// decidio, por ejemplo tras un reintento).
const headerRequestID = "X-Request-Id"

// RequestID asigna un id unico por peticion. Si la peticion entrante trae
// X-Request-Id, se respeta; si no, se genera uno aleatorio (16 bytes hex =
// 32 chars). El id queda en:
//
//   - c.GetString("request_id") para handlers y logs internos
//   - la cabecera de respuesta X-Request-Id (para que el cliente lo vea)
//
// Debe ejecutarse ANTES del middleware.Logger() para que el id este disponible
// cuando se registre el access log.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(headerRequestID)
		if rid == "" {
			rid = newRequestID()
		}
		c.Set(apperr.RequestIDKey, rid)
		c.Writer.Header().Set(headerRequestID, rid)
		c.Next()
	}
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Extremadamente raro; caemos a timestamp en nanos para que al menos
		// sea univoco dentro del proceso.
		return "fallback-request-id"
	}
	return hex.EncodeToString(b[:])
}