package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type envelope struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func Wrap(data any) envelope {
	return envelope{Data: data}
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, envelope{Data: data})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, envelope{Data: data})
}

func BadRequest(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusBadRequest, envelope{Error: msg})
}

func NotFound(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusNotFound, envelope{Error: msg})
}

func Unauthorized(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, envelope{Error: msg})
}

func InternalError(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, envelope{Error: msg})
}
