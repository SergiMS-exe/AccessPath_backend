package handlers

import (
	"net/http"
	"strconv"

	"accesspath/internal/models"
	"accesspath/internal/services"

	"github.com/gin-gonic/gin"
)

type PlaceHandler struct {
	service *services.PlaceService
}

func NewPlaceHandler(service *services.PlaceService) *PlaceHandler {
	return &PlaceHandler{service: service}
}

func (h *PlaceHandler) GetAll(c *gin.Context) {
	filters := models.PlaceFilters{
		City:   c.Query("city"),
		Limit:  parseIntOrDefault(c.Query("limit"), 20),
		Offset: parseIntOrDefault(c.Query("offset"), 0),
	}

	places, err := h.service.GetAll(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch places"})
		return
	}

	c.JSON(http.StatusOK, places)
}

func (h *PlaceHandler) GetByBounds(c *gin.Context) {
	filters := models.BoundsFilter{
		MinLat: parseFloatOrDefault(c.Query("min_lat"), 0),
		MaxLat: parseFloatOrDefault(c.Query("max_lat"), 0),
		MinLng: parseFloatOrDefault(c.Query("min_lng"), 0),
		MaxLng: parseFloatOrDefault(c.Query("max_lng"), 0),
		Limit:  parseIntOrDefault(c.Query("limit"), 100),
	}

	places, err := h.service.GetByBounds(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch places"})
		return
	}

	c.JSON(http.StatusOK, places)
}

func (h *PlaceHandler) GetNearby(c *gin.Context) {
	filters := models.NearbyFilter{
		Lat:    parseFloatOrDefault(c.Query("lat"), 0),
		Lng:    parseFloatOrDefault(c.Query("lng"), 0),
		Radius: parseFloatOrDefault(c.Query("radius"), 5),
		Limit:  parseIntOrDefault(c.Query("limit"), 20),
		Offset: parseIntOrDefault(c.Query("offset"), 0),
	}

	places, err := h.service.GetNearby(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch places"})
		return
	}

	c.JSON(http.StatusOK, places)
}

func (h *PlaceHandler) GetByID(c *gin.Context) {
	id := c.Param("id")

	place, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Place not found"})
		return
	}

	c.JSON(http.StatusOK, place)
}

func (h *PlaceHandler) Create(c *gin.Context) {
	var req models.CreatePlaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	place, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create place"})
		return
	}

	c.JSON(http.StatusCreated, place)
}

func (h *PlaceHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req models.CreatePlaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	place, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update place"})
		return
	}

	c.JSON(http.StatusOK, place)
}

func (h *PlaceHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete place"})
		return
	}

	c.Status(http.StatusNoContent)
}

func parseIntOrDefault(s string, def int) int {
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return def
}

func parseFloatOrDefault(s string, def float64) float64 {
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return v
	}
	return def
}
