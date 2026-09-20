package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Envelope es la respuesta canonica de la API: o data, o error, nunca ambos.
// Es publico para que paquetes como apperr puedan construir respuestas
// consistentes sin duplicar la estructura.
type Envelope struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func Wrap(data any) Envelope {
	return Envelope{Data: data}
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{Data: data})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Envelope{Data: data})
}

func BadRequest(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusBadRequest, Envelope{Error: msg})
}

func NotFound(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusNotFound, Envelope{Error: msg})
}

func Unauthorized(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, Envelope{Error: msg})
}

func InternalError(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, Envelope{Error: msg})
}

func TooManyRequests(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusTooManyRequests, Envelope{Error: msg})
}
