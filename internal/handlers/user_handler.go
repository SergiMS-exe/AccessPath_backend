package handlers

import (
	"net/http"

	"accesspath/internal/models"
	"accesspath/internal/services"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service *services.UserService
}

func NewUserHandler(service *services.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) Register(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *UserHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// TODO: Generate JWT token here
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := c.Param("id")

	user, err := h.service.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) GetSavedPlaces(c *gin.Context) {
	userID := c.Param("id")

	places, err := h.service.GetSavedPlaces(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch saved places"})
		return
	}

	c.JSON(http.StatusOK, places)
}

func (h *UserHandler) SavePlace(c *gin.Context) {
	userID := c.Param("id")
	placeID := c.Param("placeId")

	if err := h.service.SavePlace(c.Request.Context(), userID, placeID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save place"})
		return
	}

	c.Status(http.StatusCreated)
}

func (h *UserHandler) UnsavePlace(c *gin.Context) {
	userID := c.Param("id")
	placeID := c.Param("placeId")

	if err := h.service.UnsavePlace(c.Request.Context(), userID, placeID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unsave place"})
		return
	}

	c.Status(http.StatusNoContent)
}
