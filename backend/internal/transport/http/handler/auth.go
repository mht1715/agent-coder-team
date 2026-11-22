package handler

import (
	"errors"
	"net/http"
	"time"

	"backend/internal/application/auth"
	"backend/internal/domain"
	"backend/internal/transport/http/middleware"
	"backend/internal/transport/http/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthHandler struct {
	auth *auth.Service
}

type registerRequest struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	DisplayName string `json:"display_name"`
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type userResponse struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type authResponse struct {
	User   userResponse   `json:"user"`
	Tokens auth.TokenPair `json:"tokens"`
}

func NewAuthHandler(authSvc *auth.Service) *AuthHandler {
	return &AuthHandler{auth: authSvc}
}

func (h *AuthHandler) BootstrapStatus(c *gin.Context) {
	open, err := h.auth.RegistrationOpen(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", "bootstrap status failed")
		return
	}
	c.JSON(http.StatusOK, gin.H{"registration_open": open})
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}

	user, tokens, err := h.auth.Register(c.Request.Context(), auth.RegisterInput{
		Username:    req.Username,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	}, clientMeta(c))
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrValidation):
			response.Validation(c, err.Error())
		case errors.Is(err, domain.ErrRegistrationClosed):
			response.Error(c, http.StatusForbidden, "auth.registration_closed", "local account already exists")
		case errors.Is(err, domain.ErrConflict):
			response.Error(c, http.StatusConflict, "auth.username_exists", "username already exists")
		default:
			response.Error(c, http.StatusInternalServerError, "internal_error", "registration failed")
		}
		return
	}

	c.JSON(http.StatusCreated, authResponse{User: toUserResponse(user), Tokens: *tokens})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}

	user, tokens, err := h.auth.Login(c.Request.Context(), auth.LoginInput{
		Username: req.Username,
		Password: req.Password,
	}, clientMeta(c))
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			response.Error(c, http.StatusUnauthorized, "auth.invalid_credentials", "invalid username or password")
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", "login failed")
		return
	}

	c.JSON(http.StatusOK, authResponse{User: toUserResponse(user), Tokens: *tokens})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}

	user, tokens, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken, clientMeta(c))
	if err != nil {
		if errors.Is(err, auth.ErrUnauthorized) {
			response.Error(c, http.StatusUnauthorized, "auth.unauthorized", "invalid or expired refresh token")
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", "refresh failed")
		return
	}

	c.JSON(http.StatusOK, authResponse{User: toUserResponse(user), Tokens: *tokens})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	session, ok := middleware.CurrentSession(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "auth.unauthorized", "missing session")
		return
	}
	if err := h.auth.Logout(c.Request.Context(), session.ID); err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", "logout failed")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) Me(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "auth.unauthorized", "missing user")
		return
	}
	c.JSON(http.StatusOK, toUserResponse(user))
}

func clientMeta(c *gin.Context) auth.ClientMeta {
	return auth.ClientMeta{
		UserAgent: c.Request.UserAgent(),
		ClientIP:  c.ClientIP(),
	}
}

func toUserResponse(user *domain.User) userResponse {
	return userResponse{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Status:      user.Status,
		CreatedAt:   user.CreatedAt,
	}
}
