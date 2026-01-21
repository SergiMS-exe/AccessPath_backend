package handlers

import (
	"net/http"

	"accesspath/internal/models"
	"accesspath/internal/services"

	"github.com/gin-gonic/gin"
)

type ReviewHandler struct {
	service *services.ReviewService
}

func NewReviewHandler(service *services.ReviewService) *ReviewHandler {
	return &ReviewHandler{service: service}
}

func (h *ReviewHandler) GetByPlace(c *gin.Context) {
	placeID := c.Param("id")

	reviews, err := h.service.GetByPlace(c.Request.Context(), placeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reviews"})
		return
	}

	c.JSON(http.StatusOK, reviews)
}

func (h *ReviewHandler) GetPlaceAccessibility(c *gin.Context) {
	placeID := c.Param("id")

	averages, err := h.service.GetPlaceAccessibility(c.Request.Context(), placeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch accessibility info"})
		return
	}

	c.JSON(http.StatusOK, averages)
}

func (h *ReviewHandler) Create(c *gin.Context) {
	placeID := c.Param("id")

	var req models.CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	review, err := h.service.Create(c.Request.Context(), placeID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, review)
}
