package store

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/server/v3/embed"

	"github.com/sistemica/docker-volume-manager/pkg/types"
)

const (
	volumePrefix = "/volumes/"
	namePrefix   = "/volume-names/"
)

// EtcdStore implements a store backed by embedded etcd
type EtcdStore struct {
	etcd   *embed.Etcd
	client *clientv3.Client
	logger *slog.Logger
}

// EtcdConfig holds configuration for embedded etcd
type EtcdConfig struct {
	DataDir      string
	Name         string
	ClusterSize  int
	ServiceName  string
	TaskSlot     int
	ClientPort   int
	PeerPort     int
}

// NewEtcdStore creates a new etcd-backed store with embedded server
func NewEtcdStore(cfg EtcdConfig, logger *slog.Logger) (*EtcdStore, error) {
	// Create etcd configuration
	etcdCfg := embed.NewConfig()
	etcdCfg.Dir = filepath.Join(cfg.DataDir, "etcd")
	etcdCfg.Name = fmt.Sprintf("%s-%d", cfg.ServiceName, cfg.TaskSlot)

	// Listen URLs
	etcdCfg.ListenClientUrls = []url.URL{{Scheme: "http", Host: fmt.Sprintf("0.0.0.0:%d", cfg.ClientPort)}}
	etcdCfg.ListenPeerUrls = []url.URL{{Scheme: "http", Host: fmt.Sprintf("0.0.0.0:%d", cfg.PeerPort)}}

	// Advertise URLs (use hostname or localhost for single node)
	if cfg.ClusterSize > 1 {
		// Multi-node cluster - use Docker Swarm DNS
		hostname := fmt.Sprintf("%s.%d", cfg.ServiceName, cfg.TaskSlot)
		etcdCfg.AdvertiseClientUrls = []url.URL{{Scheme: "http", Host: fmt.Sprintf("%s:%d", hostname, cfg.ClientPort)}}
		etcdCfg.AdvertisePeerUrls = []url.URL{{Scheme: "http", Host: fmt.Sprintf("%s:%d", hostname, cfg.PeerPort)}}

		// Build initial cluster string
		initialCluster := make([]string, cfg.ClusterSize)
		for i := 1; i <= cfg.ClusterSize; i++ {
			memberName := fmt.Sprintf("%s-%d", cfg.ServiceName, i)
			peerURL := fmt.Sprintf("http://%s.%d:%d", cfg.ServiceName, i, cfg.PeerPort)
			initialCluster[i-1] = fmt.Sprintf("%s=%s", memberName, peerURL)
		}

		etcdCfg.InitialCluster = initialClusterFromSlice(initialCluster)
		etcdCfg.ClusterState = embed.ClusterStateFlagNew
	} else {
		// Single node - use default configuration
		etcdCfg.AdvertiseClientUrls = []url.URL{{Scheme: "http", Host: fmt.Sprintf("localhost:%d", cfg.ClientPort)}}
		etcdCfg.AdvertisePeerUrls = []url.URL{{Scheme: "http", Host: fmt.Sprintf("localhost:%d", cfg.PeerPort)}}
		// For single node, set initial cluster to itself
		etcdCfg.InitialCluster = fmt.Sprintf("%s=http://localhost:%d", etcdCfg.Name, cfg.PeerPort)
	}

	// Logging
	etcdCfg.LogLevel = "warn"

	logger.Info("starting embedded etcd",
		"name", etcdCfg.Name,
		"data_dir", etcdCfg.Dir,
		"cluster_size", cfg.ClusterSize,
	)

	// Start embedded etcd
	e, err := embed.StartEtcd(etcdCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to start etcd: %w", err)
	}

	// Wait for etcd to be ready
	select {
	case <-e.Server.ReadyNotify():
		logger.Info("etcd is ready")
	case <-time.After(60 * time.Second):
		e.Server.Stop()
		return nil, fmt.Errorf("etcd took too long to start")
	}

	// Create etcd client
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{fmt.Sprintf("localhost:%d", cfg.ClientPort)},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		e.Close()
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	return &EtcdStore{
		etcd:   e,
		client: client,
		logger: logger.With("store", "etcd"),
	}, nil
}

// CreateVolume creates a new volume
func (s *EtcdStore) CreateVolume(ctx context.Context, volume *types.Volume) error {
	// Check if volume with same name exists
	nameKey := namePrefix + volume.Name
	getResp, err := s.client.Get(ctx, nameKey)
	if err != nil {
		return fmt.Errorf("failed to check volume name: %w", err)
	}
	if getResp.Count > 0 {
		return ErrAlreadyExists
	}

	// Serialize volume
	data, err := json.Marshal(volume)
	if err != nil {
		return fmt.Errorf("failed to marshal volume: %w", err)
	}

	// Store volume and name mapping in a transaction
	volumeKey := volumePrefix + volume.ID

	txn := s.client.Txn(ctx).
		If(clientv3.Compare(clientv3.Version(nameKey), "=", 0)).
		Then(
			clientv3.OpPut(volumeKey, string(data)),
			clientv3.OpPut(nameKey, volume.ID),
		)

	resp, err := txn.Commit()
	if err != nil {
		return fmt.Errorf("failed to create volume: %w", err)
	}

	if !resp.Succeeded {
		return ErrAlreadyExists
	}

	s.logger.Debug("volume created in etcd", "volume_id", volume.ID, "name", volume.Name)
	return nil
}

// GetVolume retrieves a volume by ID
func (s *EtcdStore) GetVolume(ctx context.Context, id string) (*types.Volume, error) {
	key := volumePrefix + id
	resp, err := s.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get volume: %w", err)
	}

	if resp.Count == 0 {
		return nil, ErrNotFound
	}

	var volume types.Volume
	if err := json.Unmarshal(resp.Kvs[0].Value, &volume); err != nil {
		return nil, fmt.Errorf("failed to unmarshal volume: %w", err)
	}

	return &volume, nil
}

// GetVolumeByName retrieves a volume by name
func (s *EtcdStore) GetVolumeByName(ctx context.Context, name string) (*types.Volume, error) {
	// Get volume ID from name mapping
	nameKey := namePrefix + name
	resp, err := s.client.Get(ctx, nameKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get volume name mapping: %w", err)
	}

	if resp.Count == 0 {
		return nil, ErrNotFound
	}

	volumeID := string(resp.Kvs[0].Value)
	return s.GetVolume(ctx, volumeID)
}

// ListVolumes lists all volumes
func (s *EtcdStore) ListVolumes(ctx context.Context) ([]*types.Volume, error) {
	resp, err := s.client.Get(ctx, volumePrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}

	volumes := make([]*types.Volume, 0, resp.Count)
	for _, kv := range resp.Kvs {
		var volume types.Volume
		if err := json.Unmarshal(kv.Value, &volume); err != nil {
			s.logger.Warn("failed to unmarshal volume", "error", err)
			continue
		}
		volumes = append(volumes, &volume)
	}

	return volumes, nil
}

// UpdateVolume updates an existing volume
func (s *EtcdStore) UpdateVolume(ctx context.Context, volume *types.Volume) error {
	// Check if volume exists
	key := volumePrefix + volume.ID
	getResp, err := s.client.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to check volume: %w", err)
	}

	if getResp.Count == 0 {
		return ErrNotFound
	}

	// Serialize volume
	data, err := json.Marshal(volume)
	if err != nil {
		return fmt.Errorf("failed to marshal volume: %w", err)
	}

	// Update volume
	if _, err := s.client.Put(ctx, key, string(data)); err != nil {
		return fmt.Errorf("failed to update volume: %w", err)
	}

	s.logger.Debug("volume updated in etcd", "volume_id", volume.ID)
	return nil
}

// DeleteVolume deletes a volume by ID
func (s *EtcdStore) DeleteVolume(ctx context.Context, id string) error {
	// Get volume to find name
	volume, err := s.GetVolume(ctx, id)
	if err != nil {
		return err
	}

	// Delete both volume and name mapping
	volumeKey := volumePrefix + id
	nameKey := namePrefix + volume.Name

	ops := []clientv3.Op{
		clientv3.OpDelete(volumeKey),
		clientv3.OpDelete(nameKey),
	}

	if _, err := s.client.Txn(ctx).Then(ops...).Commit(); err != nil {
		return fmt.Errorf("failed to delete volume: %w", err)
	}

	s.logger.Debug("volume deleted from etcd", "volume_id", id)
	return nil
}

// Close closes the store and stops etcd
func (s *EtcdStore) Close() error {
	s.logger.Info("closing etcd store")

	if s.client != nil {
		if err := s.client.Close(); err != nil {
			s.logger.Error("failed to close etcd client", "error", err)
		}
	}

	if s.etcd != nil {
		s.etcd.Close()
		<-s.etcd.Server.StopNotify()
		s.logger.Info("etcd stopped")
	}

	return nil
}

// initialClusterFromSlice converts a slice of member=url strings to comma-separated string
func initialClusterFromSlice(members []string) string {
	result := ""
	for i, member := range members {
		if i > 0 {
			result += ","
		}
		result += member
	}
	return result
}

// Ensure data directory exists
func ensureDir(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}
