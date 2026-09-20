package handlers

import (
	"strconv"

	"accesspath/internal/models"
	"accesspath/internal/services"
	"accesspath/pkg/apperr"
	"accesspath/pkg/response"

	"github.com/gin-gonic/gin"
)

type CollectionHandler struct {
	service *services.CollectionService
}

func NewCollectionHandler(service *services.CollectionService) *CollectionHandler {
	return &CollectionHandler{service: service}
}

func parseID(c *gin.Context, param, what string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(param), 10, 64)
	if err != nil {
		Respond(c, apperr.BadRequest("collections", "collections.invalid_id",
			"Invalid "+what+" ID"))
		return 0, false
	}
	return id, true
}

func (h *CollectionHandler) GetByUser(c *gin.Context) {
	id, ok := parseID(c, "id", "user")
	if !ok {
		return
	}

	cols, err := h.service.GetByUser(c.Request.Context(), id)
	if Respond(c, err) {
		return
	}

	response.OK(c, cols)
}

func (h *CollectionHandler) Create(c *gin.Context) {
	var req models.CreateCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Respond(c, apperr.BadRequest("collections.create", "collections.invalid_body", err.Error()))
		return
	}

	col, err := h.service.Create(c.Request.Context(), req)
	if Respond(c, err) {
		return
	}

	response.Created(c, col)
}

func (h *CollectionHandler) Delete(c *gin.Context) {
	id, ok := parseID(c, "id", "collection")
	if !ok {
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); Respond(c, err) {
		return
	}

	c.Status(204)
}

func (h *CollectionHandler) GetPlaces(c *gin.Context) {
	id, ok := parseID(c, "id", "collection")
	if !ok {
		return
	}

	places, err := h.service.GetPlaces(c.Request.Context(), id)
	if Respond(c, err) {
		return
	}

	response.OK(c, places)
}

func (h *CollectionHandler) AddPlace(c *gin.Context) {
	collectionID, ok := parseID(c, "id", "collection")
	if !ok {
		return
	}
	placeID, ok := parseID(c, "placeId", "place")
	if !ok {
		return
	}

	if err := h.service.AddPlace(c.Request.Context(), collectionID, placeID); Respond(c, err) {
		return
	}

	c.Status(201)
}

func (h *CollectionHandler) RemovePlace(c *gin.Context) {
	collectionID, ok := parseID(c, "id", "collection")
	if !ok {
		return
	}
	placeID, ok := parseID(c, "placeId", "place")
	if !ok {
		return
	}

	if err := h.service.RemovePlace(c.Request.Context(), collectionID, placeID); Respond(c, err) {
		return
	}

	c.Status(204)
}