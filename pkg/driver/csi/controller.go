package csi

import (
	"context"
	"log/slog"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sistemica/docker-volume-manager/pkg/driver/client"
)

// ControllerServer implements the CSI Controller service
type ControllerServer struct {
	csi.UnimplementedControllerServer
	client *client.VolumeManagerClient
	logger *slog.Logger
}

// NewControllerServer creates a new Controller service
func NewControllerServer(managerURL string, logger *slog.Logger) (*ControllerServer, error) {
	client := client.NewVolumeManagerClient(managerURL, logger)

	return &ControllerServer{
		client: client,
		logger: logger.With("service", "csi-controller"),
	}, nil
}

// CreateVolume creates a new volume
func (s *ControllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	volumeName := req.GetName()
	if volumeName == "" {
		return nil, status.Error(codes.InvalidArgument, "volume name is required")
	}

	// Extract parameters
	parameters := req.GetParameters()
	backend := parameters["backend"]
	if backend == "" {
		backend = "local" // Default to local backend
	}

	s.logger.Info("creating volume", "name", volumeName, "backend", backend)

	// Call Volume Manager to create the volume
	volume, err := s.client.CreateVolume(ctx, volumeName, backend, parameters)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create volume: %v", err)
	}

	s.logger.Info("volume created", "volume_id", volume.ID, "name", volumeName)

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      volume.ID,
			CapacityBytes: 0, // Not applicable for local filesystem
			VolumeContext: parameters,
		},
	}, nil
}

// DeleteVolume deletes a volume
func (s *ControllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID is required")
	}

	s.logger.Info("deleting volume", "volume_id", volumeID)

	// Call Volume Manager to delete the volume
	if err := s.client.DeleteVolume(ctx, volumeID); err != nil {
		// If volume not found, consider it a success (idempotent delete)
		if err.Error() == "volume not found" {
			s.logger.Info("volume already deleted", "volume_id", volumeID)
			return &csi.DeleteVolumeResponse{}, nil
		}
		return nil, status.Errorf(codes.Internal, "failed to delete volume: %v", err)
	}

	s.logger.Info("volume deleted", "volume_id", volumeID)
	return &csi.DeleteVolumeResponse{}, nil
}

// ControllerPublishVolume attaches the volume to a node (not used for local filesystem)
func (s *ControllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ControllerPublishVolume is not supported")
}

// ControllerUnpublishVolume detaches the volume from a node (not used for local filesystem)
func (s *ControllerServer) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ControllerUnpublishVolume is not supported")
}

// ValidateVolumeCapabilities validates volume capabilities
func (s *ControllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID is required")
	}

	// Check if volume exists
	_, err := s.client.GetVolume(ctx, volumeID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "volume not found: %v", err)
	}

	// All capabilities are supported for local filesystem
	return &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeCapabilities: req.GetVolumeCapabilities(),
			VolumeContext:      req.GetVolumeContext(),
		},
	}, nil
}

// ListVolumes lists all volumes
func (s *ControllerServer) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	volumes, err := s.client.ListVolumes(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list volumes: %v", err)
	}

	entries := make([]*csi.ListVolumesResponse_Entry, len(volumes))
	for i, vol := range volumes {
		entries[i] = &csi.ListVolumesResponse_Entry{
			Volume: &csi.Volume{
				VolumeId:      vol.ID,
				CapacityBytes: 0,
			},
		}
	}

	return &csi.ListVolumesResponse{
		Entries: entries,
	}, nil
}

// GetCapacity returns available capacity (not applicable for local filesystem)
func (s *ControllerServer) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return &csi.GetCapacityResponse{
		AvailableCapacity: 0, // Unlimited for local filesystem
	}, nil
}

// ControllerGetCapabilities returns controller capabilities
func (s *ControllerServer) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: []*csi.ControllerServiceCapability{
			{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{
						Type: csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
					},
				},
			},
			{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{
						Type: csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
					},
				},
			},
		},
	}, nil
}

// CreateSnapshot creates a snapshot (not implemented)
func (s *ControllerServer) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "CreateSnapshot is not implemented")
}

// DeleteSnapshot deletes a snapshot (not implemented)
func (s *ControllerServer) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "DeleteSnapshot is not implemented")
}

// ListSnapshots lists snapshots (not implemented)
func (s *ControllerServer) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ListSnapshots is not implemented")
}

// ControllerExpandVolume expands a volume (not implemented)
func (s *ControllerServer) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ControllerExpandVolume is not implemented")
}

// ControllerGetVolume gets volume information (not implemented)
func (s *ControllerServer) ControllerGetVolume(ctx context.Context, req *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ControllerGetVolume is not implemented")
}

// ControllerModifyVolume modifies a volume (not implemented)
func (s *ControllerServer) ControllerModifyVolume(ctx context.Context, req *csi.ControllerModifyVolumeRequest) (*csi.ControllerModifyVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ControllerModifyVolume is not implemented")
}
