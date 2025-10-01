package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/sistemica/docker-volume-manager/pkg/types"
)

// VolumeManagerClient is an HTTP client for the Volume Manager REST API
type VolumeManagerClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewVolumeManagerClient creates a new Volume Manager client
func NewVolumeManagerClient(baseURL string, logger *slog.Logger) *VolumeManagerClient {
	return &VolumeManagerClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger.With("client", "volume-manager"),
	}
}

// CreateVolume creates a new volume
func (c *VolumeManagerClient) CreateVolume(ctx context.Context, name, backend string, parameters map[string]string) (*types.Volume, error) {
	req := map[string]interface{}{
		"name":       name,
		"backend":    backend,
		"parameters": parameters,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/volumes", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var volume types.Volume
	if err := json.NewDecoder(resp.Body).Decode(&volume); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &volume, nil
}

// GetVolume retrieves a volume by ID
func (c *VolumeManagerClient) GetVolume(ctx context.Context, volumeID string) (*types.Volume, error) {
	url := fmt.Sprintf("%s/api/v1/volumes/%s", c.baseURL, volumeID)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("volume not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var volume types.Volume
	if err := json.NewDecoder(resp.Body).Decode(&volume); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &volume, nil
}

// ListVolumes lists all volumes
func (c *VolumeManagerClient) ListVolumes(ctx context.Context) ([]*types.Volume, error) {
	url := fmt.Sprintf("%s/api/v1/volumes", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Count   int              `json:"count"`
		Volumes []*types.Volume `json:"volumes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Volumes, nil
}

// DeleteVolume deletes a volume
func (c *VolumeManagerClient) DeleteVolume(ctx context.Context, volumeID string) error {
	url := fmt.Sprintf("%s/api/v1/volumes/%s", c.baseURL, volumeID)
	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("volume not found")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// StageVolume stages a volume on a node
func (c *VolumeManagerClient) StageVolume(ctx context.Context, volumeID, stagingPath, nodeID string) error {
	req := map[string]string{
		"staging_path": stagingPath,
		"node_id":      nodeID,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/volumes/%s/stage", c.baseURL, volumeID)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UnstageVolume unstages a volume from a node
func (c *VolumeManagerClient) UnstageVolume(ctx context.Context, volumeID, stagingPath string) error {
	req := map[string]string{
		"staging_path": stagingPath,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/volumes/%s/stage", c.baseURL, volumeID)
	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// PublishVolume publishes (mounts) a volume
func (c *VolumeManagerClient) PublishVolume(ctx context.Context, volumeID, stagingPath, targetPath string, readOnly bool) error {
	req := map[string]interface{}{
		"staging_path": stagingPath,
		"target_path":  targetPath,
		"read_only":    readOnly,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/volumes/%s/publish", c.baseURL, volumeID)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UnpublishVolume unpublishes (unmounts) a volume
func (c *VolumeManagerClient) UnpublishVolume(ctx context.Context, volumeID, targetPath string) error {
	req := map[string]string{
		"target_path": targetPath,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/volumes/%s/publish", c.baseURL, volumeID)
	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
