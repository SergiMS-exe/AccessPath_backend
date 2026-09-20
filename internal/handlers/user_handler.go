package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"accesspath/internal/models"
	"accesspath/internal/services"
	"accesspath/pkg/apperr"
	"accesspath/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type UserHandler struct {
	service   *services.UserService
	jwtSecret []byte
}

func NewUserHandler(service *services.UserService, jwtSecret string) *UserHandler {
	return &UserHandler{service: service, jwtSecret: []byte(jwtSecret)}
}

// issueTokens genera el par (access + refresh) para un usuario. Usado por
// Login y Register para que ambos endpoints devuelvan exactamente la misma
// forma de respuesta y el cliente no necesite una llamada extra tras registrarse.
func (h *UserHandler) issueTokens(user *models.User) (*models.LoginResponse, *apperr.AppError) {
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	accessTokenString, err := accessToken.SignedString(h.jwtSecret)
	if err != nil {
		return nil, apperr.Internal("users.tokens", err)
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"type":    "refresh",
		"exp":     time.Now().Add(30 * 24 * time.Hour).Unix(),
	})
	refreshTokenString, err := refreshToken.SignedString(h.jwtSecret)
	if err != nil {
		return nil, apperr.Internal("users.tokens", err)
	}

	return &models.LoginResponse{
		Token:        accessTokenString,
		RefreshToken: refreshTokenString,
		User:         *user,
	}, nil
}

func (h *UserHandler) Register(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Respond(c, apperr.BadRequest("users.register", "users.invalid_body", err.Error()))
		return
	}

	user, err := h.service.Register(c.Request.Context(), req)
	if Respond(c, err) {
		return
	}

	// Devolvemos el mismo LoginResponse que Login (access + refresh + user)
	// directamente, sin envelope {data:...}, igual que el endpoint Login. Asi el
	// cliente puede usar el mismo codigo de deserializacion y se evita la
	// doble llamada (register + login) que tenia antes.
	resp, apierr := h.issueTokens(user)
	if apierr != nil {
		Respond(c, apierr)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *UserHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Respond(c, apperr.BadRequest("users.login", "users.invalid_body", err.Error()))
		return
	}

	user, err := h.service.Login(c.Request.Context(), req)
	if Respond(c, err) {
		return
	}

	resp, apierr := h.issueTokens(user)
	if apierr != nil {
		Respond(c, apierr)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *UserHandler) Refresh(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Respond(c, apperr.BadRequest("users.refresh", "users.invalid_body", err.Error()))
		return
	}

	token, err := jwt.Parse(req.RefreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("metodo de firma inesperado")
		}
		return h.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		Respond(c, apperr.Unauthorized("users.refresh", "refresh token invalido"))
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["type"] != "refresh" {
		Respond(c, apperr.Unauthorized("users.refresh", "token invalido"))
		return
	}

	userID := int64(claims["user_id"].(float64))

	newAccessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	newAccessTokenString, err := newAccessToken.SignedString(h.jwtSecret)
	if err != nil {
		Respond(c, apperr.Internal("users.refresh", err))
		return
	}

	newRefreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"type":    "refresh",
		"exp":     time.Now().Add(30 * 24 * time.Hour).Unix(),
	})
	newRefreshTokenString, err := newRefreshToken.SignedString(h.jwtSecret)
	if err != nil {
		Respond(c, apperr.Internal("users.refresh", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":         newAccessTokenString,
		"refresh_token": newRefreshTokenString,
	})
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Respond(c, apperr.BadRequest("users.profile", "users.invalid_id",
			"Invalid user ID"))
		return
	}

	user, err := h.service.GetByID(c.Request.Context(), id)
	if Respond(c, err) {
		return
	}

	response.OK(c, user)
}