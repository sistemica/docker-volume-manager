package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// HealthHandler handles health check requests
type HealthHandler struct{}

// NewHealthHandler creates a new health handler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// HandleHealth handles GET /health
func (h *HealthHandler) HandleHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "healthy",
		"service": "volume-manager",
	})
}

// HandleReady handles GET /ready
func (h *HealthHandler) HandleReady(c echo.Context) error {
	// TODO: Check dependencies (etcd, storage backends)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "ready",
	})
}
