package handlers

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/sistemica/docker-volume-manager/pkg/store"
	"github.com/sistemica/docker-volume-manager/pkg/types"
)

// FileHandler handles file operations within volumes
type FileHandler struct {
	store  store.Store
	logger *slog.Logger
}

// NewFileHandler creates a new file handler
func NewFileHandler(store store.Store, logger *slog.Logger) *FileHandler {
	return &FileHandler{
		store:  store,
		logger: logger.With("handler", "file"),
	}
}

// FileInfo represents file/directory information
type FileInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	IsDir   bool   `json:"is_dir"`
	ModTime string `json:"mod_time"`
}

// ListFilesResponse represents the response for listing files
type ListFilesResponse struct {
	Path  string     `json:"path"`
	Files []FileInfo `json:"files"`
	Count int        `json:"count"`
}

// ReadFileResponse represents the response for reading a file
type ReadFileResponse struct {
	Path        string `json:"path"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	Content     string `json:"content"` // base64 encoded for binary, plain text for text files
	IsText      bool   `json:"is_text"`
}

// WriteFileRequest represents the request to write a file
type WriteFileRequest struct {
	Content string `json:"content" validate:"required"` // base64 encoded or plain text
	Mode    string `json:"mode,omitempty"`              // file permissions (e.g., "0644")
}

// HandleGet handles GET /api/v1/volumes/:id/files/*
// Returns file content if path is a file, lists directory if path is a directory
func (h *FileHandler) HandleGet(c echo.Context) error {
	volumeID := c.Param("id")
	// Get path from wildcard (everything after /files/)
	requestedPath := c.Param("*")
	if requestedPath == "" {
		requestedPath = "/"
	}
	if !strings.HasPrefix(requestedPath, "/") {
		requestedPath = "/" + requestedPath
	}

	// Get volume
	volume, err := h.store.GetVolume(c.Request().Context(), volumeID)
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

	// Get volume root path
	volumePath := volume.Parameters["path"]
	if volumePath == "" {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "invalid_volume",
			Message: "Volume has no path parameter",
		})
	}

	// Construct full path and validate it's within volume
	fullPath := filepath.Join(volumePath, requestedPath)
	if !strings.HasPrefix(fullPath, volumePath) {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "invalid_path",
			Message: "Path traversal not allowed",
		})
	}

	// Check if path exists
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return c.JSON(http.StatusNotFound, types.ErrorResponse{
				Error:   "not_found",
				Message: "Path not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "internal_error",
			Message: fmt.Sprintf("Failed to stat path: %v", err),
		})
	}

	// If directory, list contents
	if info.IsDir() {
		return h.listDirectory(c, volumeID, requestedPath, fullPath)
	}

	// If file, return content
	return h.readFile(c, volumeID, requestedPath, fullPath, info)
}

// HandlePut handles PUT /api/v1/volumes/:id/files/*
// Creates or updates a file
func (h *FileHandler) HandlePut(c echo.Context) error {
	volumeID := c.Param("id")
	requestedPath := c.Param("*")
	if requestedPath == "" || requestedPath == "/" {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "invalid_path",
			Message: "Cannot write to root directory",
		})
	}
	if !strings.HasPrefix(requestedPath, "/") {
		requestedPath = "/" + requestedPath
	}

	var req WriteFileRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	// Get volume
	volume, err := h.store.GetVolume(c.Request().Context(), volumeID)
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

	// Get volume root path
	volumePath := volume.Parameters["path"]
	if volumePath == "" {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "invalid_volume",
			Message: "Volume has no path parameter",
		})
	}

	// Construct full path and validate
	fullPath := filepath.Join(volumePath, requestedPath)
	if !strings.HasPrefix(fullPath, volumePath) {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "invalid_path",
			Message: "Path traversal not allowed",
		})
	}

	// Decode content (try base64 first, fallback to plain text)
	var content []byte
	decoded, err := base64.StdEncoding.DecodeString(req.Content)
	if err == nil {
		content = decoded
	} else {
		// Not base64, treat as plain text
		content = []byte(req.Content)
	}

	// Ensure parent directory exists
	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "internal_error",
			Message: fmt.Sprintf("Failed to create parent directory: %v", err),
		})
	}

	// Parse file mode
	mode := os.FileMode(0644)
	if req.Mode != "" {
		var modeInt uint32
		if _, err := fmt.Sscanf(req.Mode, "%o", &modeInt); err == nil {
			mode = os.FileMode(modeInt)
		}
	}

	// Write file
	if err := os.WriteFile(fullPath, content, mode); err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "internal_error",
			Message: fmt.Sprintf("Failed to write file: %v", err),
		})
	}

	h.logger.Info("wrote file",
		"volume_id", volumeID,
		"path", requestedPath,
		"size", len(content),
	)

	// Get file info
	info, _ := os.Stat(fullPath)

	return c.JSON(http.StatusCreated, FileInfo{
		Name:    filepath.Base(requestedPath),
		Path:    requestedPath,
		Size:    info.Size(),
		Mode:    info.Mode().String(),
		IsDir:   false,
		ModTime: info.ModTime().Format("2006-01-02T15:04:05Z"),
	})
}

// HandleDelete handles DELETE /api/v1/volumes/:id/files/*
func (h *FileHandler) HandleDelete(c echo.Context) error {
	volumeID := c.Param("id")
	requestedPath := c.Param("*")
	if requestedPath == "" || requestedPath == "/" {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "invalid_path",
			Message: "Cannot delete root directory",
		})
	}
	if !strings.HasPrefix(requestedPath, "/") {
		requestedPath = "/" + requestedPath
	}

	// Get volume
	volume, err := h.store.GetVolume(c.Request().Context(), volumeID)
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

	// Get volume root path
	volumePath := volume.Parameters["path"]
	if volumePath == "" {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "invalid_volume",
			Message: "Volume has no path parameter",
		})
	}

	// Construct full path and validate
	fullPath := filepath.Join(volumePath, requestedPath)
	if !strings.HasPrefix(fullPath, volumePath) {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "invalid_path",
			Message: "Path traversal not allowed",
		})
	}

	// Check if path exists
	if _, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return c.JSON(http.StatusNotFound, types.ErrorResponse{
				Error:   "not_found",
				Message: "File or directory not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "internal_error",
			Message: fmt.Sprintf("Failed to stat path: %v", err),
		})
	}

	// Delete file or directory
	if err := os.RemoveAll(fullPath); err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "internal_error",
			Message: fmt.Sprintf("Failed to delete: %v", err),
		})
	}

	h.logger.Info("deleted file/directory",
		"volume_id", volumeID,
		"path", requestedPath,
	)

	return c.JSON(http.StatusOK, types.SuccessResponse{
		Message: "File or directory deleted successfully",
	})
}

// listDirectory lists the contents of a directory
func (h *FileHandler) listDirectory(c echo.Context, volumeID, requestedPath, fullPath string) error {
	// Read directory
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "internal_error",
			Message: fmt.Sprintf("Failed to read directory: %v", err),
		})
	}

	// Build response
	files := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			h.logger.Warn("failed to get file info", "entry", entry.Name(), "error", err)
			continue
		}

		filePath := filepath.Join(requestedPath, entry.Name())
		if requestedPath == "/" {
			filePath = "/" + entry.Name()
		}

		files = append(files, FileInfo{
			Name:    entry.Name(),
			Path:    filePath,
			Size:    info.Size(),
			Mode:    info.Mode().String(),
			IsDir:   entry.IsDir(),
			ModTime: info.ModTime().Format("2006-01-02T15:04:05Z"),
		})
	}

	h.logger.Info("listed directory",
		"volume_id", volumeID,
		"path", requestedPath,
		"file_count", len(files),
	)

	return c.JSON(http.StatusOK, ListFilesResponse{
		Path:  requestedPath,
		Files: files,
		Count: len(files),
	})
}

// readFile reads a file and returns its content
func (h *FileHandler) readFile(c echo.Context, volumeID, requestedPath, fullPath string, info os.FileInfo) error {
	// Read file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "internal_error",
			Message: fmt.Sprintf("Failed to read file: %v", err),
		})
	}

	// Detect if content is text or binary
	isText := isTextContent(content)
	var contentStr string
	if isText {
		contentStr = string(content)
	} else {
		contentStr = base64.StdEncoding.EncodeToString(content)
	}

	h.logger.Info("read file",
		"volume_id", volumeID,
		"path", requestedPath,
		"size", info.Size(),
		"is_text", isText,
	)

	return c.JSON(http.StatusOK, ReadFileResponse{
		Path:        requestedPath,
		Size:        info.Size(),
		ContentType: detectContentType(fullPath, content),
		Content:     contentStr,
		IsText:      isText,
	})
}

// isTextContent checks if content is likely text
func isTextContent(content []byte) bool {
	if len(content) == 0 {
		return true
	}

	// Check for null bytes (binary indicator)
	for i := 0; i < len(content) && i < 512; i++ {
		if content[i] == 0 {
			return false
		}
	}

	return true
}

// detectContentType detects the MIME type of a file
func detectContentType(path string, content []byte) string {
	// Try to detect from extension first
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt":
		return "text/plain"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".html":
		return "text/html"
	case ".md":
		return "text/markdown"
	case ".yaml", ".yml":
		return "application/yaml"
	}

	// Fallback to content detection
	if len(content) > 0 {
		return http.DetectContentType(content)
	}

	return "application/octet-stream"
}
