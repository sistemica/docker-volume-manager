package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/sistemica/docker-volume-manager/pkg/storage"
	"github.com/sistemica/docker-volume-manager/pkg/store"
	"github.com/sistemica/docker-volume-manager/pkg/types"
)

// VolumeHandler handles volume-related requests
type VolumeHandler struct {
	store  store.Store
	logger *slog.Logger
}

// NewVolumeHandler creates a new volume handler
func NewVolumeHandler(store store.Store, logger *slog.Logger) *VolumeHandler {
	return &VolumeHandler{
		store:  store,
		logger: logger.With("handler", "volume"),
	}
}

// HandleCreate handles POST /api/v1/volumes
func (h *VolumeHandler) HandleCreate(c echo.Context) error {
	var req types.CreateVolumeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	// Validate request
	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "validation_error",
			Message: "Volume name is required",
		})
	}

	if req.Backend == "" {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "validation_error",
			Message: "Backend is required",
		})
	}

	// Get backend
	backend, err := storage.GetBackend(req.Backend)
	if err != nil {
		if errors.Is(err, storage.ErrBackendNotFound) {
			return c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "invalid_backend",
				Message: "Backend not found: " + req.Backend,
			})
		}
		h.logger.Error("failed to get backend", "error", err)
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get backend",
		})
	}

	// Validate backend parameters
	if err := backend.Validate(req.Parameters); err != nil {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
	}

	// Create volume
	volume := &types.Volume{
		ID:         uuid.New().String(),
		Name:       req.Name,
		Backend:    req.Backend,
		Parameters: req.Parameters,
		Status:     types.VolumeStatusCreated,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Store volume
	if err := h.store.CreateVolume(c.Request().Context(), volume); err != nil {
		if errors.Is(err, store.ErrAlreadyExists) {
			return c.JSON(http.StatusConflict, types.ErrorResponse{
				Error:   "already_exists",
				Message: "Volume with this name already exists",
			})
		}
		h.logger.Error("failed to create volume", "error", err)
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create volume",
		})
	}

	h.logger.Info("volume created", "volume_id", volume.ID, "name", volume.Name)

	return c.JSON(http.StatusCreated, volume)
}

// HandleList handles GET /api/v1/volumes
func (h *VolumeHandler) HandleList(c echo.Context) error {
	volumes, err := h.store.ListVolumes(c.Request().Context())
	if err != nil {
		h.logger.Error("failed to list volumes", "error", err)
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list volumes",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"volumes": volumes,
		"count":   len(volumes),
	})
}

// HandleGet handles GET /api/v1/volumes/:id
func (h *VolumeHandler) HandleGet(c echo.Context) error {
	id := c.Param("id")

	volume, err := h.store.GetVolume(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return c.JSON(http.StatusNotFound, types.ErrorResponse{
				Error:   "not_found",
				Message: "Volume not found",
			})
		}
		h.logger.Error("failed to get volume", "error", err, "volume_id", id)
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get volume",
		})
	}

	return c.JSON(http.StatusOK, volume)
}

// HandleDelete handles DELETE /api/v1/volumes/:id
func (h *VolumeHandler) HandleDelete(c echo.Context) error {
	id := c.Param("id")

	// Get volume first
	volume, err := h.store.GetVolume(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return c.JSON(http.StatusNotFound, types.ErrorResponse{
				Error:   "not_found",
				Message: "Volume not found",
			})
		}
		h.logger.Error("failed to get volume", "error", err, "volume_id", id)
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get volume",
		})
	}

	// Check if volume is still published
	if len(volume.PublishedOn) > 0 {
		return c.JSON(http.StatusConflict, types.ErrorResponse{
			Error:   "volume_in_use",
			Message: "Volume is still published, unpublish first",
		})
	}

	// Delete volume
	if err := h.store.DeleteVolume(c.Request().Context(), id); err != nil {
		h.logger.Error("failed to delete volume", "error", err, "volume_id", id)
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete volume",
		})
	}

	h.logger.Info("volume deleted", "volume_id", id)

	return c.JSON(http.StatusOK, types.SuccessResponse{
		Message: "Volume deleted successfully",
	})
}

// HandleStage handles POST /api/v1/volumes/:id/stage
func (h *VolumeHandler) HandleStage(c echo.Context) error {
	id := c.Param("id")

	var req types.StageVolumeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	req.VolumeID = id

	// Get volume
	volume, err := h.store.GetVolume(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return c.JSON(http.StatusNotFound, types.ErrorResponse{
				Error:   "not_found",
				Message: "Volume not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get volume",
		})
	}

	// Get backend
	backend, err := storage.GetBackend(volume.Backend)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get backend",
		})
	}

	// Stage volume
	if err := backend.Stage(c.Request().Context(), volume, req.StagingPath); err != nil {
		h.logger.Error("failed to stage volume", "error", err, "volume_id", id)
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "stage_failed",
			Message: err.Error(),
		})
	}

	// Update volume status
	volume.Status = types.VolumeStatusStaged
	volume.StagedOn = append(volume.StagedOn, req.NodeID)
	volume.UpdatedAt = time.Now()

	if err := h.store.UpdateVolume(c.Request().Context(), volume); err != nil {
		h.logger.Error("failed to update volume", "error", err)
	}

	h.logger.Info("volume staged", "volume_id", id, "node_id", req.NodeID)

	return c.JSON(http.StatusOK, volume)
}

// HandlePublish handles POST /api/v1/volumes/:id/publish
func (h *VolumeHandler) HandlePublish(c echo.Context) error {
	id := c.Param("id")

	var req types.PublishVolumeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	req.VolumeID = id

	// Get volume
	volume, err := h.store.GetVolume(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return c.JSON(http.StatusNotFound, types.ErrorResponse{
				Error:   "not_found",
				Message: "Volume not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get volume",
		})
	}

	// Get backend
	backend, err := storage.GetBackend(volume.Backend)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get backend",
		})
	}

	// Publish volume (staging path not used for local backend)
	stagingPath := ""
	if err := backend.Publish(c.Request().Context(), volume, stagingPath, req.TargetPath, req.ReadOnly); err != nil {
		h.logger.Error("failed to publish volume", "error", err, "volume_id", id)
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "publish_failed",
			Message: err.Error(),
		})
	}

	// Update volume status
	volume.Status = types.VolumeStatusPublished
	volume.PublishedOn = append(volume.PublishedOn, req.NodeID)
	volume.UpdatedAt = time.Now()

	if err := h.store.UpdateVolume(c.Request().Context(), volume); err != nil {
		h.logger.Error("failed to update volume", "error", err)
	}

	h.logger.Info("volume published", "volume_id", id, "node_id", req.NodeID)

	return c.JSON(http.StatusOK, volume)
}
