package types

import "time"

// VolumeStatus represents the current state of a volume
type VolumeStatus string

const (
	VolumeStatusCreated   VolumeStatus = "created"
	VolumeStatusStaging   VolumeStatus = "staging"
	VolumeStatusStaged    VolumeStatus = "staged"
	VolumeStatusPublished VolumeStatus = "published"
	VolumeStatusFailed    VolumeStatus = "failed"
)

// Volume represents a storage volume
type Volume struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Backend    string                 `json:"backend"`
	Parameters map[string]string      `json:"parameters"`
	Status     VolumeStatus           `json:"status"`
	StagedOn   []string               `json:"staged_on,omitempty"`   // Node IDs where volume is staged
	PublishedOn []string              `json:"published_on,omitempty"` // Node IDs where volume is published
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// CreateVolumeRequest is the request to create a new volume
type CreateVolumeRequest struct {
	Name       string            `json:"name" validate:"required"`
	Backend    string            `json:"backend" validate:"required"`
	Parameters map[string]string `json:"parameters"`
}

// StageVolumeRequest is the request to stage a volume on a node
type StageVolumeRequest struct {
	VolumeID    string `json:"volume_id" validate:"required"`
	NodeID      string `json:"node_id" validate:"required"`
	StagingPath string `json:"staging_path" validate:"required"`
}

// PublishVolumeRequest is the request to publish a volume to a target path
type PublishVolumeRequest struct {
	VolumeID   string `json:"volume_id" validate:"required"`
	NodeID     string `json:"node_id" validate:"required"`
	TargetPath string `json:"target_path" validate:"required"`
	ReadOnly   bool   `json:"read_only"`
}

// Backend represents a storage backend
type Backend struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Capabilities BackendCapability `json:"capabilities"`
}

// BackendCapability describes what a backend supports
type BackendCapability struct {
	SupportsReadOnly  bool `json:"supports_read_only"`
	SupportsReadWrite bool `json:"supports_read_write"`
	SupportsSnapshot  bool `json:"supports_snapshot"`
	SupportsClone     bool `json:"supports_clone"`
}

// ErrorResponse is the standard error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// SuccessResponse is a generic success response
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
