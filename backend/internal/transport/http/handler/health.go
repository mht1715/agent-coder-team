package handler

import (
	"net/http"

	"backend/internal/health"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	health      *health.Service
	appName     string
	appVersion  string
	environment string
}

func NewHealthHandler(healthSvc *health.Service, appName, appVersion, environment string) *HealthHandler {
	return &HealthHandler{
		health:      healthSvc,
		appName:     appName,
		appVersion:  appVersion,
		environment: environment,
	}
}

func (h *HealthHandler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *HealthHandler) Readyz(c *gin.Context) {
	result := h.health.Ready(c.Request.Context())
	status := http.StatusOK
	if result.Status != "ok" {
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, result)
}

func (h *HealthHandler) Meta(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name":        h.appName,
		"version":     h.appVersion,
		"environment": h.environment,
	})
}
