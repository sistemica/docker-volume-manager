package handlers

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sistemica/docker-volume-manager/pkg/storage"
	"github.com/sistemica/docker-volume-manager/pkg/types"
)

// BackendHandler handles backend-related requests
type BackendHandler struct {
	logger *slog.Logger
}

// NewBackendHandler creates a new backend handler
func NewBackendHandler(logger *slog.Logger) *BackendHandler {
	return &BackendHandler{
		logger: logger.With("handler", "backend"),
	}
}

// HandleList handles GET /api/v1/backends
func (h *BackendHandler) HandleList(c echo.Context) error {
	backends, err := storage.ListBackends()
	if err != nil {
		h.logger.Error("failed to list backends", "error", err)
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list backends",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"backends": backends,
		"count":    len(backends),
	})
}
