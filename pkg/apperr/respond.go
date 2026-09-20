package apperr

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"accesspath/pkg/response"

	"github.com/gin-gonic/gin"
)

// RequestIDKey es la clave usada en gin.Context para el request id. La
// exportamos para que tests y otros middleware puedan leerla.
const RequestIDKey = "request_id"

// headerRequestID es la cabecera HTTP con el id. La exportamos para que el
// middleware de request id la use sin duplicar la constante.
const HeaderRequestID = "X-Request-Id"

// defaultMessagesByStatus asocia codigos HTTP concretos con un mensaje
// generico user-friendly. Codigos 5xx sin entrada caen al fallback de "error
// interno" (cubierto mas abajo).
var defaultMessagesByStatus = map[int]string{
	http.StatusBadRequest:         "Solicitud invalida.",
	http.StatusUnauthorized:       "No autorizado.",
	http.StatusNotFound:           "Recurso no encontrado.",
	http.StatusTooManyRequests:    "Demasiadas solicitudes. Vuelve a intentarlo en unos minutos.",
	http.StatusBadGateway:         "Servicio externo no disponible.",
	http.StatusServiceUnavailable: "Servicio no disponible. Vuelve a intentarlo.",
}

func defaultMessageForStatus(status int) string {
	if msg, ok := defaultMessagesByStatus[status]; ok {
		return msg
	}
	if status >= 500 {
		return "Error interno del servidor. Intentalo de nuevo."
	}
	return "Error desconocido."
}

// levelForStatus mapea codigos HTTP a niveles de slog siguiendo la convencion
// estandar: 5xx es error operacional, 4xx es peticion del cliente que no se
// deberia repetir igual.
func levelForStatus(status int) slog.Level {
	if status >= 500 {
		return slog.LevelError
	}
	if status >= 400 {
		return slog.LevelWarn
	}
	return slog.LevelInfo
}

// RespondInternal loguea un *AppError ya construido y escribe la respuesta
// HTTP. Es la primitiva de bajo nivel; el helper que usan los handlers
// (handlers.Respond) la envuelve para clasificar primero errores genericos.
func RespondInternal(c *gin.Context, ae *AppError) {
	requestID, _ := c.Get(RequestIDKey)
	rid, _ := requestID.(string)

	attrs := []any{
		slog.String("op", ae.Op),
		slog.String("code", ae.Code),
		slog.Int("http_status", ae.HTTPStatus),
		slog.String("user_message", ae.UserMessage),
	}
	if rid != "" {
		attrs = append(attrs, slog.String("request_id", rid))
	}
	if ae.Cause != nil {
		attrs = append(attrs, slog.String("cause", ae.Cause.Error()))
	}
	for k, v := range ae.Meta {
		attrs = append(attrs, slog.Any(k, v))
	}

	slog.LogAttrs(c.Request.Context(), levelForStatus(ae.HTTPStatus), "app_error",
		attrsToSlogAttrs(attrs)...)

	userMsg := ae.UserMessage
	if userMsg == "" {
		userMsg = defaultMessageForStatus(ae.HTTPStatus)
	}

	c.AbortWithStatusJSON(ae.HTTPStatus, response.Envelope{Error: userMsg})
}

// Respond clasifica `err` y lo registra. Pensado como punto de entrada unico
// desde los handlers:
//
//	if apperr.Respond(c, err) { return }
//
// Si err es *AppError, se usa tal cual. Si no, se trata como Internal 500
// (logeando la causa). Para que el handler clasifique errores de dominio
// (sentinels de services), usar `handlers.Respond` (ver internal/handlers).
func Respond(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}

	var ae *AppError
	if !errors.As(err, &ae) {
		ae = Internal(opFromContext(c), err)
	}
	if ae.Op == "" {
		ae.Op = opFromContext(c)
	}

	RespondInternal(c, ae)
	return true
}

// opFromContext deduce un op estable del path cuando el error no trae uno.
// Sirve para que los logs agrupen por endpoint.
func opFromContext(c *gin.Context) string {
	if p := strings.Trim(c.FullPath(), "/"); p != "" {
		return "http." + p
	}
	return "http"
}

func attrsToSlogAttrs(pairs []any) []slog.Attr {
	out := make([]slog.Attr, 0, len(pairs))
	for i := 0; i+1 < len(pairs); i += 2 {
		key, ok := pairs[i].(string)
		if !ok {
			continue
		}
		if attr, ok := pairs[i+1].(slog.Attr); ok {
			out = append(out, attr)
			continue
		}
		out = append(out, slog.Any(key, pairs[i+1]))
	}
	return out
}

// RequestID extrae el request id del contexto. Helper de conveniencia.
func RequestID(c *gin.Context) string {
	v, _ := c.Get(RequestIDKey)
	s, _ := v.(string)
	return s
}