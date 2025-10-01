package store

import (
	"context"
	"sync"

	"github.com/sistemica/docker-volume-manager/pkg/types"
)

// MemoryStore implements an in-memory store for development
type MemoryStore struct {
	mu      sync.RWMutex
	volumes map[string]*types.Volume // indexed by ID
	names   map[string]string        // name -> ID mapping
}

// NewMemoryStore creates a new in-memory store
func NewMemoryStore() Store {
	return &MemoryStore{
		volumes: make(map[string]*types.Volume),
		names:   make(map[string]string),
	}
}

// CreateVolume creates a new volume
func (s *MemoryStore) CreateVolume(ctx context.Context, volume *types.Volume) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if volume with same ID exists
	if _, exists := s.volumes[volume.ID]; exists {
		return ErrAlreadyExists
	}

	// Check if volume with same name exists
	if _, exists := s.names[volume.Name]; exists {
		return ErrAlreadyExists
	}

	// Store volume
	s.volumes[volume.ID] = volume
	s.names[volume.Name] = volume.ID

	return nil
}

// GetVolume retrieves a volume by ID
func (s *MemoryStore) GetVolume(ctx context.Context, id string) (*types.Volume, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	volume, exists := s.volumes[id]
	if !exists {
		return nil, ErrNotFound
	}

	return volume, nil
}

// GetVolumeByName retrieves a volume by name
func (s *MemoryStore) GetVolumeByName(ctx context.Context, name string) (*types.Volume, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, exists := s.names[name]
	if !exists {
		return nil, ErrNotFound
	}

	volume, exists := s.volumes[id]
	if !exists {
		return nil, ErrNotFound
	}

	return volume, nil
}

// ListVolumes lists all volumes
func (s *MemoryStore) ListVolumes(ctx context.Context) ([]*types.Volume, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	volumes := make([]*types.Volume, 0, len(s.volumes))
	for _, volume := range s.volumes {
		volumes = append(volumes, volume)
	}

	return volumes, nil
}

// UpdateVolume updates an existing volume
func (s *MemoryStore) UpdateVolume(ctx context.Context, volume *types.Volume) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.volumes[volume.ID]; !exists {
		return ErrNotFound
	}

	s.volumes[volume.ID] = volume
	return nil
}

// DeleteVolume deletes a volume by ID
func (s *MemoryStore) DeleteVolume(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	volume, exists := s.volumes[id]
	if !exists {
		return ErrNotFound
	}

	delete(s.volumes, id)
	delete(s.names, volume.Name)

	return nil
}

// Close closes the store
func (s *MemoryStore) Close() error {
	// Nothing to close for memory store
	return nil
}
