package handlers

import (
	"net/http"
	"strconv"

	"accesspath/internal/middleware"
	"accesspath/internal/models"
	"accesspath/internal/services"
	"accesspath/pkg/apperr"
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
		Search: c.Query("search"),
		Limit:  parseIntOrDefault(c.Query("limit"), 20),
		Offset: parseIntOrDefault(c.Query("offset"), 0),
	}

	result, err := h.service.GetAll(c.Request.Context(), filters)
	if Respond(c, err) {
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
// @Param        limit       query   int     false  "Máximo de resultados"  default(100)
// @Success      200  {object}  map[string][]models.PlaceMapItem
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /places/map [get]
func (h *PlaceHandler) GetByBounds(c *gin.Context) {
	if c.Query("min_lat") == "" || c.Query("max_lat") == "" ||
		c.Query("min_lng") == "" || c.Query("max_lng") == "" {
		Respond(c, apperr.BadRequest("places.map", "places.missing_bounds",
			"min_lat, max_lat, min_lng and max_lng are required"))
		return
	}

	minLat := parseFloatOrDefault(c.Query("min_lat"), 0)
	maxLat := parseFloatOrDefault(c.Query("max_lat"), 0)
	minLng := parseFloatOrDefault(c.Query("min_lng"), 0)
	maxLng := parseFloatOrDefault(c.Query("max_lng"), 0)

	if minLat >= maxLat || minLng >= maxLng {
		Respond(c, apperr.BadRequest("places.map", "places.invalid_bounds",
			"min_lat must be less than max_lat and min_lng less than max_lng"))
		return
	}

	filters := models.BoundsFilter{
		MinLat: minLat,
		MaxLat: maxLat,
		MinLng: minLng,
		MaxLng: maxLng,
		Limit:  parseIntOrDefault(c.Query("limit"), 100),
	}

	places, err := h.service.GetByBounds(c.Request.Context(), filters)
	if Respond(c, err) {
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
	if Respond(c, err) {
		return
	}
	response.OK(c, places)
}

// GetByID returns the place detail including its rating cache.
func (h *PlaceHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Respond(c, apperr.BadRequest("places.detail", "places.invalid_id",
			"Invalid place ID"))
		return
	}

	detail, err := h.service.GetByID(c.Request.Context(), id)
	if Respond(c, err) {
		return
	}
	response.OK(c, detail)
}

func (h *PlaceHandler) Search(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		Respond(c, apperr.BadRequest("places.search", "places.missing_query",
			"q is required"))
		return
	}
	session := c.Query("session")

	items, err := h.service.Search(c.Request.Context(), q, session)
	if Respond(c, err) {
		return
	}
	response.OK(c, items)
}

func (h *PlaceHandler) ImportFromGoogle(c *gin.Context) {
	var req models.ImportFromGoogleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Respond(c, apperr.BadRequest("places.import", "places.invalid_body", err.Error()))
		return
	}

	userID, ok := middleware.UserID(c)
	if !ok {
		Respond(c, apperr.Unauthorized("places.import", "token requerido"))
		return
	}

	place, err := h.service.ImportFromGoogle(c.Request.Context(), req.GooglePlaceID, req.SessionToken, userID)
	if Respond(c, err) {
		return
	}

	c.JSON(http.StatusCreated, response.Wrap(place))
}

func (h *PlaceHandler) Create(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		Respond(c, apperr.Unauthorized("places.create", "token requerido"))
		return
	}
	var req models.CreatePlaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Respond(c, apperr.BadRequest("places.create", "places.invalid_body", err.Error()))
		return
	}
	req.CreatedBy = userID

	place, err := h.service.Create(c.Request.Context(), req)
	if Respond(c, err) {
		return
	}

	c.JSON(http.StatusCreated, response.Wrap(place))
}

func (h *PlaceHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Respond(c, apperr.BadRequest("places.update", "places.invalid_id",
			"Invalid place ID"))
		return
	}

	var req models.UpdatePlaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Respond(c, apperr.BadRequest("places.update", "places.invalid_body", err.Error()))
		return
	}

	place, err := h.service.Update(c.Request.Context(), id, req)
	if Respond(c, err) {
		return
	}

	response.OK(c, place)
}

func (h *PlaceHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Respond(c, apperr.BadRequest("places.delete", "places.invalid_id",
			"Invalid place ID"))
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); Respond(c, err) {
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