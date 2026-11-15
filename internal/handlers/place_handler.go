package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"accesspath/internal/middleware"
	"accesspath/internal/models"
	"accesspath/internal/services"
	"accesspath/pkg/apperr"
	"accesspath/pkg/response"
	"accesspath/pkg/validate"

	"github.com/gin-gonic/gin"
)

type PlaceHandler struct {
	service services.PlaceService
}

func NewPlaceHandler(service services.PlaceService) *PlaceHandler {
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

	minLat, errLat := parseStrictFloat(c, "min_lat")
	if errLat != nil {
		return
	}
	maxLat, errLat := parseStrictFloat(c, "max_lat")
	if errLat != nil {
		return
	}
	minLng, errLng := parseStrictFloat(c, "min_lng")
	if errLng != nil {
		return
	}
	maxLng, errLng := parseStrictFloat(c, "max_lng")
	if errLng != nil {
		return
	}

	if err := validate.Lat(minLat); err != nil {
		Respond(c, apperr.Validation("places.map", "min_lat", err.Error()))
		return
	}
	if err := validate.Lat(maxLat); err != nil {
		Respond(c, apperr.Validation("places.map", "max_lat", err.Error()))
		return
	}
	if err := validate.Lng(minLng); err != nil {
		Respond(c, apperr.Validation("places.map", "min_lng", err.Error()))
		return
	}
	if err := validate.Lng(maxLng); err != nil {
		Respond(c, apperr.Validation("places.map", "max_lng", err.Error()))
		return
	}

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
	lat, errLat := parseStrictFloat(c, "lat")
	if errLat != nil {
		return
	}
	lng, errLng := parseStrictFloat(c, "lng")
	if errLng != nil {
		return
	}
	if err := validate.Lat(lat); err != nil {
		Respond(c, apperr.Validation("places.nearby", "lat", err.Error()))
		return
	}
	if err := validate.Lng(lng); err != nil {
		Respond(c, apperr.Validation("places.nearby", "lng", err.Error()))
		return
	}

	filters := models.NearbyFilter{
		Lat:    lat,
		Lng:    lng,
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

// Search godoc
// @Summary      Busca lugares por texto
// @Description  Busca en el catalogo local (Overture Maps) y en los lugares ya verificados.
// @Tags         places
// @Produce      json
// @Param        q       query  string  true   "Texto a buscar (minimo 3 caracteres)"
// @Param        lat     query  number  false  "Latitud del usuario (opcional, anade distancia)"
// @Param        lng     query  number  false  "Longitud del usuario (opcional, anade distancia)"
// @Param        limit   query  int     false  "Maximo de resultados"  default(10)
// @Success      200  {object}  map[string][]models.PlaceSearchItem
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /places/search [get]
func (h *PlaceHandler) Search(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		Respond(c, apperr.BadRequest("places.search", "places.missing_query",
			"q is required"))
		return
	}

	limit := parseIntOrDefault(c.Query("limit"), 10)
	if limit < 1 {
		limit = 10
	}
	if limit > 20 {
		limit = 20
	}

	opts := models.SearchOptions{
		Query: q,
		Limit: limit,
	}

	if raw := c.Query("lat"); raw != "" {
		lat, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			Respond(c, apperr.Validation("places.search", "lat", "debe ser un numero"))
			return
		}
		if err := validate.Lat(lat); err != nil {
			Respond(c, apperr.Validation("places.search", "lat", err.Error()))
			return
		}
		opts.Latitude = &lat
	}
	if raw := c.Query("lng"); raw != "" {
		lng, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			Respond(c, apperr.Validation("places.search", "lng", "debe ser un numero"))
			return
		}
		if err := validate.Lng(lng); err != nil {
			Respond(c, apperr.Validation("places.search", "lng", err.Error()))
			return
		}
		opts.Longitude = &lng
	}

	items, err := h.service.Search(c.Request.Context(), opts)
	if Respond(c, err) {
		return
	}
	response.OK(c, items)
}

// ImportFromCatalog godoc
// @Summary      Importa un lugar del catalogo externo
// @Description  Copia a la base local una ficha del catalogo (Overture Maps). Idempotente por (source, external_id). Si se envia place_id devuelve ese lugar local ya existente.
// @Tags         places
// @Accept       json
// @Produce      json
// @Param        body  body  models.ImportFromCatalogRequest  true  "source + external_id, o place_id"
// @Success      201  {object}  map[string]models.Place
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Router       /places/from-catalog [post]
func (h *PlaceHandler) ImportFromCatalog(c *gin.Context) {
	var req models.ImportFromCatalogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Respond(c, apperr.BadRequest("places.import", "places.invalid_body", err.Error()))
		return
	}

	userID, ok := middleware.UserID(c)
	if !ok {
		Respond(c, apperr.Unauthorized("places.import", "token requerido"))
		return
	}

	if req.PlaceID <= 0 && req.ExternalID == "" {
		Respond(c, apperr.BadRequest("places.import", "places.missing_reference",
			"external_id o place_id es obligatorio"))
		return
	}

	place, err := h.service.ImportFromCatalog(c.Request.Context(), req.Source, req.ExternalID, req.PlaceID, userID)
	if Respond(c, err) {
		return
	}

	c.JSON(http.StatusCreated, response.Wrap(place))
}

// ResolveLink godoc
// @Summary      Lee un enlace de Google Maps
// @Description  Extrae nombre, coordenadas y ftid de un enlace compartido por el usuario y busca el sitio en el catalogo. No descarga la ficha de Google ni usa su API.
// @Tags         places
// @Accept       json
// @Produce      json
// @Param        body  body  models.ResolveLinkRequest  true  "url del enlace"
// @Success      200  {object}  map[string]models.ResolvedLink
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Router       /places/resolve-link [post]
func (h *PlaceHandler) ResolveLink(c *gin.Context) {
	var req models.ResolveLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Respond(c, apperr.BadRequest("places.resolve_link", "places.invalid_body", err.Error()))
		return
	}

	// El endpoint es autenticado (para poder limitar el uso por usuario),
	// aunque el enlace no se asocia a ninguna fila.
	if _, ok := middleware.UserID(c); !ok {
		Respond(c, apperr.Unauthorized("places.resolve_link", "token requerido"))
		return
	}

	if err := validate.MaxLen(req.URL, "url", 2048); err != nil {
		Respond(c, apperr.Validation("places.resolve_link", "url", err.Error()))
		return
	}

	resolved, err := h.service.ResolveLink(c.Request.Context(), req.URL)
	if Respond(c, err) {
		return
	}

	response.OK(c, resolved)
}

// CreateCommunity godoc
// @Summary      Crea un lugar comunitario pendiente de moderacion
// @Description  Crea el sitio que el usuario no encontro en el catalogo. Nace con moderation_status=pending: no aparece en el mapa ni en el buscador hasta que se verifica. Si ya existe un lugar parecido a menos de 50 m devuelve el existente con 200.
// @Tags         places
// @Accept       json
// @Produce      json
// @Param        body  body  models.CreateCommunityPlaceRequest  true  "datos del sitio"
// @Success      201  {object}  map[string]models.Place
// @Success      200  {object}  map[string]models.Place
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Router       /places/community [post]
func (h *PlaceHandler) CreateCommunity(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		Respond(c, apperr.Unauthorized("places.community", "token requerido"))
		return
	}

	var req models.CreateCommunityPlaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Respond(c, apperr.BadRequest("places.community", "places.invalid_body", err.Error()))
		return
	}

	if err := validateCommunityRequest(req); err != nil {
		Respond(c, err)
		return
	}

	place, created, err := h.service.CreateCommunity(c.Request.Context(), req, userID)
	if Respond(c, err) {
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	c.JSON(status, response.Wrap(place))
}

// validateCommunityRequest aplica las mismas reglas que el resto de altas:
// nombre, direccion, coordenadas y longitudes maximas.
func validateCommunityRequest(req models.CreateCommunityPlaceRequest) *apperr.AppError {
	op := "places.community"

	if err := validate.NonEmpty(req.Name, "name"); err != nil {
		return apperr.Validation(op, "name", err.Error())
	}
	if err := validate.MaxLen(req.Name, "name", 255); err != nil {
		return apperr.Validation(op, "name", err.Error())
	}
	if err := validate.NonEmpty(req.Address, "address"); err != nil {
		return apperr.Validation(op, "address", err.Error())
	}
	if err := validate.MaxLen(req.Address, "address", 500); err != nil {
		return apperr.Validation(op, "address", err.Error())
	}
	if err := validate.Lat(req.Latitude); err != nil {
		return apperr.Validation(op, "latitude", err.Error())
	}
	if err := validate.Lng(req.Longitude); err != nil {
		return apperr.Validation(op, "longitude", err.Error())
	}
	if req.Category != nil {
		if err := validate.MaxLen(*req.Category, "category", 100); err != nil {
			return apperr.Validation(op, "category", err.Error())
		}
	}
	if req.Description != nil {
		if err := validate.MaxLen(*req.Description, "description", 2000); err != nil {
			return apperr.Validation(op, "description", err.Error())
		}
	}
	if req.SourceURL != nil {
		if err := validate.MaxLen(*req.SourceURL, "source_url", 2048); err != nil {
			return apperr.Validation(op, "source_url", err.Error())
		}
	}
	if req.GoogleFTID != nil {
		if err := validate.MaxLen(*req.GoogleFTID, "google_ftid", 128); err != nil {
			return apperr.Validation(op, "google_ftid", err.Error())
		}
	}
	return nil
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

	userID, ok := middleware.UserID(c)
	if !ok {
		Respond(c, apperr.Unauthorized("places.update", "token requerido"))
		return
	}

	var req models.UpdatePlaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Respond(c, apperr.BadRequest("places.update", "places.invalid_body", err.Error()))
		return
	}
	if err := validate.Lat(req.Latitude); err != nil {
		Respond(c, apperr.Validation("places.update", "latitude", err.Error()))
		return
	}
	if err := validate.Lng(req.Longitude); err != nil {
		Respond(c, apperr.Validation("places.update", "longitude", err.Error()))
		return
	}
	if err := validate.MaxLen(req.Name, "name", 255); err != nil {
		Respond(c, apperr.Validation("places.update", "name", err.Error()))
		return
	}
	if req.Address != nil {
		if err := validate.MaxLen(*req.Address, "address", 500); err != nil {
			Respond(c, apperr.Validation("places.update", "address", err.Error()))
			return
		}
	}
	if req.Description != nil {
		if err := validate.MaxLen(*req.Description, "description", 2000); err != nil {
			Respond(c, apperr.Validation("places.update", "description", err.Error()))
			return
		}
	}

	place, err := h.service.Update(c.Request.Context(), id, userID, req)
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

	userID, ok := middleware.UserID(c)
	if !ok {
		Respond(c, apperr.Unauthorized("places.delete", "token requerido"))
		return
	}

	if err := h.service.Delete(c.Request.Context(), id, userID); Respond(c, err) {
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

// parseStrictFloat parsea un query param como float64 obligatorio. Si el
// param falta o no es un numero valido, escribe un BadRequest en el contexto
// via Respond y devuelve error para que el handler haga return. Asi no se
// enmascaran inputs malformados con 0 como hacia parseFloatOrDefault.
func parseStrictFloat(c *gin.Context, field string) (float64, error) {
	raw := c.Query(field)
	if raw == "" {
		Respond(c, apperr.BadRequest(opFromRequest(c),
			"validation.missing", field+" es obligatorio"))
		return 0, errors.New("missing")
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		Respond(c, apperr.Validation(opFromRequest(c), field,
			"debe ser un numero; recibido: "+raw))
		return 0, err
	}
	return v, nil
}
