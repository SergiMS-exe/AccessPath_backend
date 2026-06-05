package handlers

import (
	"net/http"
	"strconv"

	"accesspath/internal/models"
	"accesspath/internal/services"
	"accesspath/pkg/response"

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
		Search:     c.Query("search"),
		CategoryID: parseInt64OrDefault(c.Query("category_id"), 0),
		MinRating:  parseFloatOrDefault(c.Query("min_rating"), 0),
		Limit:      parseIntOrDefault(c.Query("limit"), 20),
		Offset:     parseIntOrDefault(c.Query("offset"), 0),
	}

	result, err := h.service.GetAll(c.Request.Context(), filters)
	if err != nil {
		response.InternalError(c, "Failed to fetch places")
		return
	}

	response.OK(c, result)
}

// GetByBounds godoc
// @Summary      Lugares en el mapa
// @Description  Retorna los lugares dentro de un bounding box definido por esquina superior-izquierda y esquina inferior-derecha
// @Tags         places
// @Produce      json
// @Param        min_lat     query   number  true   "Latitud de la esquina inferior-izquierda"
// @Param        max_lat     query   number  true   "Latitud de la esquina superior-derecha"
// @Param        min_lng     query   number  true   "Longitud de la esquina inferior-izquierda"
// @Param        max_lng     query   number  true   "Longitud de la esquina superior-derecha"
// @Param        category_id query   int     false  "Filtrar por categoría"
// @Param        limit       query   int     false  "Máximo de resultados"  default(100)
// @Success      200  {object}  map[string][]models.Place
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /places/map [get]
func (h *PlaceHandler) GetByBounds(c *gin.Context) {
	minLat := parseFloatOrDefault(c.Query("min_lat"), 0)
	maxLat := parseFloatOrDefault(c.Query("max_lat"), 0)
	minLng := parseFloatOrDefault(c.Query("min_lng"), 0)
	maxLng := parseFloatOrDefault(c.Query("max_lng"), 0)

	if c.Query("min_lat") == "" || c.Query("max_lat") == "" ||
		c.Query("min_lng") == "" || c.Query("max_lng") == "" {
		response.BadRequest(c, "min_lat, max_lat, min_lng and max_lng are required")
		return
	}
	if minLat >= maxLat || minLng >= maxLng {
		response.BadRequest(c, "min_lat must be less than max_lat and min_lng less than max_lng")
		return
	}

	filters := models.BoundsFilter{
		MinLat:     minLat,
		MaxLat:     maxLat,
		MinLng:     minLng,
		MaxLng:     maxLng,
		CategoryID: parseInt64OrDefault(c.Query("category_id"), 0),
		Limit:      parseIntOrDefault(c.Query("limit"), 100),
	}

	places, err := h.service.GetByBounds(c.Request.Context(), filters)
	if err != nil {
		response.InternalError(c, "Failed to fetch places")
		return
	}

	response.OK(c, places)
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
		response.InternalError(c, "Failed to fetch places")
		return
	}

	response.OK(c, places)
}

// GetByID returns the place detail including its rating cache.
func (h *PlaceHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid place ID")
		return
	}

	detail, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "Place not found")
		return
	}

	response.OK(c, detail)
}

func (h *PlaceHandler) Create(c *gin.Context) {
	var req models.CreatePlaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	place, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		response.InternalError(c, "Failed to create place")
		return
	}

	c.JSON(http.StatusCreated, response.Wrap(place))
}

func (h *PlaceHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid place ID")
		return
	}

	var req models.UpdatePlaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	place, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		response.InternalError(c, "Failed to update place")
		return
	}

	response.OK(c, place)
}

func (h *PlaceHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid place ID")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.InternalError(c, "Failed to delete place")
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

func parseInt64OrDefault(s string, def int64) int64 {
	if v, err := strconv.ParseInt(s, 10, 64); err == nil {
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
