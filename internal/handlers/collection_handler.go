package handlers

import (
	"net/http"
	"strconv"

	"accesspath/internal/models"
	"accesspath/internal/services"

	"github.com/gin-gonic/gin"
)

type CollectionHandler struct {
	service *services.CollectionService
}

func NewCollectionHandler(service *services.CollectionService) *CollectionHandler {
	return &CollectionHandler{service: service}
}

func (h *CollectionHandler) GetByUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	cols, err := h.service.GetByUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch collections"})
		return
	}

	c.JSON(http.StatusOK, cols)
}

func (h *CollectionHandler) Create(c *gin.Context) {
	var req models.CreateCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	col, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create collection"})
		return
	}

	c.JSON(http.StatusCreated, col)
}

func (h *CollectionHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"})
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete collection"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *CollectionHandler) GetPlaces(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"})
		return
	}

	places, err := h.service.GetPlaces(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch places"})
		return
	}

	c.JSON(http.StatusOK, places)
}

func (h *CollectionHandler) AddPlace(c *gin.Context) {
	collectionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"})
		return
	}

	placeID, err := strconv.ParseInt(c.Param("placeId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid place ID"})
		return
	}

	if err := h.service.AddPlace(c.Request.Context(), collectionID, placeID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add place"})
		return
	}

	c.Status(http.StatusCreated)
}

func (h *CollectionHandler) RemovePlace(c *gin.Context) {
	collectionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"})
		return
	}

	placeID, err := strconv.ParseInt(c.Param("placeId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid place ID"})
		return
	}

	if err := h.service.RemovePlace(c.Request.Context(), collectionID, placeID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove place"})
		return
	}

	c.Status(http.StatusNoContent)
}
