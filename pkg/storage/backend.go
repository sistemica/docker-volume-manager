package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/sistemica/docker-volume-manager/pkg/types"
)

var (
	// ErrBackendNotFound is returned when a backend is not found
	ErrBackendNotFound = errors.New("backend not found")

	// ErrBackendAlreadyRegistered is returned when trying to register a backend that already exists
	ErrBackendAlreadyRegistered = errors.New("backend already registered")

	// ErrVolumeNotStaged is returned when trying to publish a volume that is not staged
	ErrVolumeNotStaged = errors.New("volume not staged")
)

// Backend defines the interface that all storage backends must implement
type Backend interface {
	// Name returns the backend name
	Name() string

	// Capabilities returns the backend capabilities
	Capabilities() types.BackendCapability

	// Validate validates the volume parameters for this backend
	Validate(params map[string]string) error

	// Stage prepares the volume on a node (download, extract, etc.)
	Stage(ctx context.Context, volume *types.Volume, stagingPath string) error

	// Unstage cleans up the staged volume
	Unstage(ctx context.Context, volume *types.Volume, stagingPath string) error

	// Publish makes the volume available at the target path (mount, bind mount, etc.)
	Publish(ctx context.Context, volume *types.Volume, stagingPath, targetPath string, readOnly bool) error

	// Unpublish removes the volume from the target path
	Unpublish(ctx context.Context, volume *types.Volume, targetPath string) error
}

// Factory is a function that creates a new backend instance
type Factory func() (Backend, error)

// registry holds all registered backends
var registry = make(map[string]Factory)

// RegisterBackend registers a new backend
func RegisterBackend(name string, factory Factory) error {
	if _, exists := registry[name]; exists {
		return fmt.Errorf("%w: %s", ErrBackendAlreadyRegistered, name)
	}
	registry[name] = factory
	return nil
}

// GetBackend returns a backend by name
func GetBackend(name string) (Backend, error) {
	factory, exists := registry[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrBackendNotFound, name)
	}
	return factory()
}

// ListBackends returns a list of all registered backends
func ListBackends() ([]types.Backend, error) {
	backends := make([]types.Backend, 0, len(registry))

	for name := range registry {
		backend, err := GetBackend(name)
		if err != nil {
			continue
		}

		backends = append(backends, types.Backend{
			Name:         backend.Name(),
			Description:  fmt.Sprintf("%s storage backend", backend.Name()),
			Capabilities: backend.Capabilities(),
		})
	}

	return backends, nil
}
