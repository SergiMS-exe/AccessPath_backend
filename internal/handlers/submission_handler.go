package handlers

import (
	"strconv"

	"accesspath/internal/middleware"
	"accesspath/internal/models"
	"accesspath/internal/services"
	"accesspath/pkg/apperr"
	"accesspath/pkg/response"

	"github.com/gin-gonic/gin"
)

type SubmissionHandler struct {
	service *services.SubmissionService
}

func NewSubmissionHandler(service *services.SubmissionService) *SubmissionHandler {
	return &SubmissionHandler{service: service}
}

// GetByPlace godoc
// @Summary      Comentarios y fotos de un lugar
// @Description  "Que cuenta la gente": submissions (comentario + fotos).
// @Tags         submissions
// @Produce      json
// @Param        id   path   int  true  "ID del lugar"
// @Success      200  {object}  map[string][]models.SubmissionWithDetails
// @Router       /places/{id}/submissions [get]
func (h *SubmissionHandler) GetByPlace(c *gin.Context) {
	placeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Respond(c, apperr.BadRequest("submissions.get", "places.invalid_id",
			"Invalid place ID"))
		return
	}
	submissions, err := h.service.GetByPlace(c.Request.Context(), placeID)
	if Respond(c, err) {
		return
	}
	response.OK(c, submissions)
}

// Save godoc
// @Summary      Guardar valoracion (comentario/fotos)
// @Description  get-or-create de la valoracion viva del usuario para el lugar; fija comentario y/o adjunta fotos. user_id del token.
// @Tags         submissions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  models.SubmissionRequest  true  "Submission"
// @Success      200  {object}  map[string]models.Submission
// @Router       /submissions [put]
func (h *SubmissionHandler) Save(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		Respond(c, apperr.Unauthorized("submissions.save", "token requerido"))
		return
	}
	var req models.SubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Respond(c, apperr.BadRequest("submissions.save", "submissions.invalid_body", err.Error()))
		return
	}

	submission, err := h.service.Save(c.Request.Context(), userID, req)
	if Respond(c, err) {
		return
	}
	response.OK(c, submission)
}