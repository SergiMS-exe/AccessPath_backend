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

type ContributionHandler struct {
	service     *services.ContributionService
	questionSvc *services.QuestionService
}

func NewContributionHandler(service *services.ContributionService, questionSvc *services.QuestionService) *ContributionHandler {
	return &ContributionHandler{service: service, questionSvc: questionSvc}
}

// NextQuestion godoc
// @Summary      Siguiente pregunta
// @Description  Devuelve el criterio de mayor valor para (place, user); vacio si nada util.
// @Tags         contributions
// @Produce      json
// @Security     BearerAuth
// @Param        id   path   int  true  "ID del lugar"
// @Success      200  {object}  map[string]models.NextQuestionResponse
// @Router       /places/{id}/next-question [get]
func (h *ContributionHandler) NextQuestion(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		Respond(c, apperr.Unauthorized("contributions.next", "token requerido"))
		return
	}
	placeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Respond(c, apperr.BadRequest("contributions.next", "places.invalid_id",
			"Invalid place ID"))
		return
	}

	next, err := h.questionSvc.Next(c.Request.Context(), userID, placeID)
	if Respond(c, err) {
		return
	}
	response.OK(c, next)
}

// Create godoc
// @Summary      Crear contribucion
// @Description  Contribucion atomica; user_id del token. Devuelve el semaforo en vivo.
// @Tags         contributions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  models.ContributionRequest  true  "Contribucion"
// @Success      201  {object}  map[string]models.ContributionResult
// @Router       /contributions [post]
func (h *ContributionHandler) Create(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		Respond(c, apperr.Unauthorized("contributions.create", "token requerido"))
		return
	}
	var req models.ContributionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Respond(c, apperr.BadRequest("contributions.create", "contributions.invalid_body", err.Error()))
		return
	}

	result, err := h.service.Create(c.Request.Context(), userID, req)
	if Respond(c, err) {
		return
	}
	response.Created(c, result)
}

// Delete godoc
// @Summary      Deshacer contribucion
// @Description  Soft delete + recalculo. Solo el propietario.
// @Tags         contributions
// @Produce      json
// @Security     BearerAuth
// @Param        id   path   int  true  "ID de la contribucion"
// @Success      200  {object}  map[string]models.ContributionResult
// @Router       /contributions/{id} [delete]
func (h *ContributionHandler) Delete(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		Respond(c, apperr.Unauthorized("contributions.delete", "token requerido"))
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Respond(c, apperr.BadRequest("contributions.delete", "contributions.invalid_id",
			"Invalid contribution ID"))
		return
	}

	result, err := h.service.Delete(c.Request.Context(), userID, id)
	if Respond(c, err) {
		return
	}
	response.OK(c, result)
}