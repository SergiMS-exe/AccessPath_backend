package handlers

import (
	"accesspath/internal/middleware"
	"accesspath/internal/models"
	"accesspath/internal/services"
	"accesspath/pkg/apperr"
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
		Respond(c, apperr.Unauthorized("profile.get", "token requerido"))
		return
	}
	profile, err := h.service.Get(c.Request.Context(), userID)
	if Respond(c, err) {
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
		Respond(c, apperr.Unauthorized("profile.set", "token requerido"))
		return
	}
	var req models.ProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Respond(c, apperr.BadRequest("profile.set", "profile.invalid_body",
			"se requiere consentimiento explicito"))
		return
	}

	profile, err := h.service.Set(c.Request.Context(), userID, req)
	if Respond(c, err) {
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
		Respond(c, apperr.Unauthorized("profile.delete", "token requerido"))
		return
	}
	if err := h.service.Delete(c.Request.Context(), userID); Respond(c, err) {
		return
	}
	c.Status(204)
}