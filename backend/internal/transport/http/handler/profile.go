package handler

import (
	"errors"
	"net/http"

	profileapp "backend/internal/application/profile"
	"backend/internal/domain"
	"backend/internal/transport/http/middleware"
	"backend/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

type ProfileHandler struct {
	profiles *profileapp.Service
}

func NewProfileHandler(profiles *profileapp.Service) *ProfileHandler {
	return &ProfileHandler{profiles: profiles}
}

func (h *ProfileHandler) Get(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "auth.unauthorized", "missing user")
		return
	}
	projectID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	projectProfile, err := h.profiles.Get(c.Request.Context(), user.ID, projectID)
	if err != nil {
		writeProfileError(c, err)
		return
	}
	c.JSON(http.StatusOK, projectProfile)
}

func (h *ProfileHandler) Scan(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "auth.unauthorized", "missing user")
		return
	}
	projectID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	projectProfile, err := h.profiles.Scan(c.Request.Context(), user.ID, projectID)
	if err != nil {
		writeProfileError(c, err)
		return
	}
	c.JSON(http.StatusOK, projectProfile)
}

func writeProfileError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, profileapp.ErrValidation):
		response.Error(c, http.StatusUnprocessableEntity, "profile.scan_failed", err.Error())
	case errors.Is(err, domain.ErrNotFound):
		response.Error(c, http.StatusNotFound, "profile.not_found", "project or project profile not found")
	default:
		response.Error(c, http.StatusInternalServerError, "internal_error", "project profile operation failed")
	}
}
