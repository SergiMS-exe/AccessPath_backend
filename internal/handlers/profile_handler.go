package handlers

import (
	"errors"

	"accesspath/internal/middleware"
	"accesspath/internal/models"
	"accesspath/internal/services"
	"accesspath/pkg/response"

	"github.com/gin-gonic/gin"
)

type ProfileHandler struct {
	service *services.ProfileService
}

func NewProfileHandler(service *services.ProfileService) *ProfileHandler {
	return &ProfileHandler{service: service}
}

// Get godoc
// @Summary      Perfil funcional
// @Description  Necesidades funcionales + estado de consentimiento del usuario.
// @Tags         profile
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]models.ProfileResponse
// @Router       /me/profile [get]
func (h *ProfileHandler) Get(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		response.Unauthorized(c, "token requerido")
		return
	}
	profile, err := h.service.Get(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, "Failed to fetch profile")
		return
	}
	response.OK(c, profile)
}

// Set godoc
// @Summary      Fijar perfil funcional
// @Description  Set de necesidades + consentimiento explicito (opt-in). Requiere consent=true.
// @Tags         profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  models.ProfileRequest  true  "Perfil"
// @Success      200  {object}  map[string]models.ProfileResponse
// @Router       /me/profile [put]
func (h *ProfileHandler) Set(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		response.Unauthorized(c, "token requerido")
		return
	}
	var req models.ProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "se requiere consentimiento explicito")
		return
	}

	profile, err := h.service.Set(c.Request.Context(), userID, req)
	if err != nil {
		var invalid services.ErrInvalidNeedKey
		if errors.As(err, &invalid) {
			response.BadRequest(c, invalid.Error())
			return
		}
		response.InternalError(c, "Failed to save profile")
		return
	}
	response.OK(c, profile)
}

// Delete godoc
// @Summary      Borrar perfil funcional
// @Description  Borra necesidades y retira el consentimiento.
// @Tags         profile
// @Security     BearerAuth
// @Success      204
// @Router       /me/profile [delete]
func (h *ProfileHandler) Delete(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		response.Unauthorized(c, "token requerido")
		return
	}
	if err := h.service.Delete(c.Request.Context(), userID); err != nil {
		response.InternalError(c, "Failed to delete profile")
		return
	}
	c.Status(204)
}
