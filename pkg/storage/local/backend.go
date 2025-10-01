package local

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/sistemica/docker-volume-manager/pkg/storage"
	"github.com/sistemica/docker-volume-manager/pkg/types"
)

func init() {
	// Register the local backend on import
	if err := storage.RegisterBackend("local", NewBackend); err != nil {
		panic(fmt.Sprintf("failed to register local backend: %v", err))
	}
}

// Backend implements the local filesystem storage backend
type Backend struct {
	logger *slog.Logger
}

// NewBackend creates a new local filesystem backend
func NewBackend() (storage.Backend, error) {
	return &Backend{
		logger: slog.Default().With("backend", "local"),
	}, nil
}

// Name returns the backend name
func (b *Backend) Name() string {
	return "local"
}

// Capabilities returns the backend capabilities
func (b *Backend) Capabilities() types.BackendCapability {
	return types.BackendCapability{
		SupportsReadOnly:  true,
		SupportsReadWrite: true,
		SupportsSnapshot:  false,
		SupportsClone:     false,
	}
}

// Validate validates the volume parameters
func (b *Backend) Validate(params map[string]string) error {
	path, ok := params["path"]
	if !ok || path == "" {
		return errors.New("parameter 'path' is required")
	}

	// Validate path is absolute
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path must be absolute: %s", path)
	}

	return nil
}

// Stage prepares the volume on a node
func (b *Backend) Stage(ctx context.Context, volume *types.Volume, stagingPath string) error {
	b.logger.Info("staging volume",
		"volume_id", volume.ID,
		"staging_path", stagingPath,
	)

	sourcePath := volume.Parameters["path"]

	// Ensure source path exists
	if err := b.ensureDirectoryExists(sourcePath); err != nil {
		return fmt.Errorf("failed to ensure source path: %w", err)
	}

	// Ensure staging path exists
	if err := b.ensureDirectoryExists(stagingPath); err != nil {
		return fmt.Errorf("failed to ensure staging path: %w", err)
	}

	// For local backend, staging is just validation
	// The actual bind mount happens in Publish
	b.logger.Info("volume staged successfully",
		"volume_id", volume.ID,
		"source_path", sourcePath,
	)

	return nil
}

// Unstage cleans up the staged volume
func (b *Backend) Unstage(ctx context.Context, volume *types.Volume, stagingPath string) error {
	b.logger.Info("unstaging volume",
		"volume_id", volume.ID,
		"staging_path", stagingPath,
	)

	// For local backend, nothing to clean up during unstage
	// The directory is managed by the host

	return nil
}

// Publish makes the volume available at the target path
func (b *Backend) Publish(ctx context.Context, volume *types.Volume, stagingPath, targetPath string, readOnly bool) error {
	b.logger.Info("publishing volume",
		"volume_id", volume.ID,
		"staging_path", stagingPath,
		"target_path", targetPath,
		"read_only", readOnly,
	)

	sourcePath := volume.Parameters["path"]

	// Ensure target path exists
	if err := b.ensureDirectoryExists(targetPath); err != nil {
		return fmt.Errorf("failed to ensure target path: %w", err)
	}

	// For local backend, we use bind mount
	// Note: In a real implementation, this would use mount syscall
	// For now, we'll create a symlink as a placeholder
	if err := b.createBindMount(sourcePath, targetPath, readOnly); err != nil {
		return fmt.Errorf("failed to bind mount: %w", err)
	}

	b.logger.Info("volume published successfully",
		"volume_id", volume.ID,
		"target_path", targetPath,
	)

	return nil
}

// Unpublish removes the volume from the target path
func (b *Backend) Unpublish(ctx context.Context, volume *types.Volume, targetPath string) error {
	b.logger.Info("unpublishing volume",
		"volume_id", volume.ID,
		"target_path", targetPath,
	)

	// Remove the bind mount or symlink
	if err := b.removeBindMount(targetPath); err != nil {
		return fmt.Errorf("failed to remove bind mount: %w", err)
	}

	b.logger.Info("volume unpublished successfully",
		"volume_id", volume.ID,
	)

	return nil
}

// ensureDirectoryExists creates a directory if it doesn't exist
func (b *Backend) ensureDirectoryExists(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}

// createBindMount creates a bind mount (placeholder: creates symlink for now)
func (b *Backend) createBindMount(source, target string, readOnly bool) error {
	// TODO: Implement actual bind mount using unix.Mount
	// For development, we'll create a marker file
	markerFile := filepath.Join(target, ".volume-mount")
	content := fmt.Sprintf("source=%s\nreadonly=%t\n", source, readOnly)

	if err := os.WriteFile(markerFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create mount marker: %w", err)
	}

	b.logger.Debug("bind mount created (placeholder)",
		"source", source,
		"target", target,
	)

	return nil
}

// removeBindMount removes a bind mount
func (b *Backend) removeBindMount(target string) error {
	// TODO: Implement actual unmount using unix.Unmount
	// For development, we'll remove the marker file
	markerFile := filepath.Join(target, ".volume-mount")

	if err := os.Remove(markerFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove mount marker: %w", err)
	}

	b.logger.Debug("bind mount removed (placeholder)",
		"target", target,
	)

	return nil
}
