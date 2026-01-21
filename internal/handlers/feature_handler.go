package handlers

import (
	"net/http"
	"strconv"

	"accesspath/internal/services"

	"github.com/gin-gonic/gin"
)

type FeatureHandler struct {
	service *services.FeatureService
}

func NewFeatureHandler(service *services.FeatureService) *FeatureHandler {
	return &FeatureHandler{service: service}
}

func (h *FeatureHandler) GetAll(c *gin.Context) {
	categoryID := c.Query("category_id")

	if categoryID != "" {
		id, err := strconv.Atoi(categoryID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category_id"})
			return
		}

		features, err := h.service.GetByCategory(c.Request.Context(), int32(id))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch features"})
			return
		}

		c.JSON(http.StatusOK, features)
		return
	}

	features, err := h.service.GetAllFeatures(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch features"})
		return
	}

	c.JSON(http.StatusOK, features)
}

func (h *FeatureHandler) GetCategories(c *gin.Context) {
	categories, err := h.service.GetAllCategories(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
		return
	}

	c.JSON(http.StatusOK, categories)
}

func (h *FeatureHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid feature ID"})
		return
	}

	feature, err := h.service.GetByID(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Feature not found"})
		return
	}

	c.JSON(http.StatusOK, feature)
}
