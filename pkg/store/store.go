package store

import (
	"context"
	"errors"

	"github.com/sistemica/docker-volume-manager/pkg/types"
)

var (
	// ErrNotFound is returned when a volume is not found
	ErrNotFound = errors.New("volume not found")

	// ErrAlreadyExists is returned when a volume already exists
	ErrAlreadyExists = errors.New("volume already exists")
)

// Store defines the interface for metadata storage
type Store interface {
	// CreateVolume creates a new volume
	CreateVolume(ctx context.Context, volume *types.Volume) error

	// GetVolume retrieves a volume by ID
	GetVolume(ctx context.Context, id string) (*types.Volume, error)

	// GetVolumeByName retrieves a volume by name
	GetVolumeByName(ctx context.Context, name string) (*types.Volume, error)

	// ListVolumes lists all volumes
	ListVolumes(ctx context.Context) ([]*types.Volume, error)

	// UpdateVolume updates an existing volume
	UpdateVolume(ctx context.Context, volume *types.Volume) error

	// DeleteVolume deletes a volume by ID
	DeleteVolume(ctx context.Context, id string) error

	// Close closes the store
	Close() error
}
