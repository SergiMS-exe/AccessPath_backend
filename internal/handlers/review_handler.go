package handlers

import (
	"net/http"
	"strconv"

	"accesspath/internal/models"
	"accesspath/internal/services"
	"accesspath/pkg/response"

	"github.com/gin-gonic/gin"
)

type ReviewHandler struct {
	service *services.ReviewService
}

func NewReviewHandler(service *services.ReviewService) *ReviewHandler {
	return &ReviewHandler{service: service}
}

func (h *ReviewHandler) GetByPlace(c *gin.Context) {
	placeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid place ID")
		return
	}

	reviews, err := h.service.GetByPlace(c.Request.Context(), placeID)
	if err != nil {
		response.InternalError(c, "Failed to fetch reviews")
		return
	}

	response.OK(c, reviews)
}

func (h *ReviewHandler) Create(c *gin.Context) {
	var req models.CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	review, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		response.InternalError(c, "Failed to create review")
		return
	}

	c.JSON(http.StatusCreated, response.Wrap(review))
}

func (h *ReviewHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid review ID")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.InternalError(c, "Failed to delete review")
		return
	}

	c.Status(http.StatusNoContent)
}
