package csi

import (
	"context"
	"log/slog"
	"os"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sistemica/docker-volume-manager/pkg/driver/client"
)

// NodeServer implements the CSI Node service
type NodeServer struct {
	csi.UnimplementedNodeServer
	nodeID    string
	client    *client.VolumeManagerClient
	logger    *slog.Logger
}

// NewNodeServer creates a new Node service
func NewNodeServer(nodeID, managerURL string, logger *slog.Logger) (*NodeServer, error) {
	client := client.NewVolumeManagerClient(managerURL, logger)

	return &NodeServer{
		nodeID: nodeID,
		client: client,
		logger: logger.With("service", "csi-node"),
	}, nil
}

// NodeStageVolume prepares the volume (e.g., extract zip, setup filesystem)
func (s *NodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	// Validate request
	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID is required")
	}

	stagingPath := req.GetStagingTargetPath()
	if stagingPath == "" {
		return nil, status.Error(codes.InvalidArgument, "staging target path is required")
	}

	s.logger.Info("staging volume", "volume_id", volumeID, "staging_path", stagingPath)

	// Ensure staging path exists
	if err := os.MkdirAll(stagingPath, 0755); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create staging path: %v", err)
	}

	// Call Volume Manager to stage the volume
	if err := s.client.StageVolume(ctx, volumeID, stagingPath, s.nodeID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to stage volume: %v", err)
	}

	s.logger.Info("volume staged successfully", "volume_id", volumeID)
	return &csi.NodeStageVolumeResponse{}, nil
}

// NodeUnstageVolume cleans up the staged volume
func (s *NodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID is required")
	}

	stagingPath := req.GetStagingTargetPath()
	if stagingPath == "" {
		return nil, status.Error(codes.InvalidArgument, "staging target path is required")
	}

	s.logger.Info("unstaging volume", "volume_id", volumeID, "staging_path", stagingPath)

	// Call Volume Manager to unstage the volume
	if err := s.client.UnstageVolume(ctx, volumeID, stagingPath); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unstage volume: %v", err)
	}

	s.logger.Info("volume unstaged successfully", "volume_id", volumeID)
	return &csi.NodeUnstageVolumeResponse{}, nil
}

// NodePublishVolume mounts the volume at the target path (bind mount)
func (s *NodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID is required")
	}

	stagingPath := req.GetStagingTargetPath()
	if stagingPath == "" {
		return nil, status.Error(codes.InvalidArgument, "staging target path is required")
	}

	targetPath := req.GetTargetPath()
	if targetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "target path is required")
	}

	readOnly := req.GetReadonly()

	s.logger.Info("publishing volume",
		"volume_id", volumeID,
		"staging_path", stagingPath,
		"target_path", targetPath,
		"read_only", readOnly,
	)

	// Ensure target path exists
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create target path: %v", err)
	}

	// Call Volume Manager to publish (bind mount) the volume
	if err := s.client.PublishVolume(ctx, volumeID, stagingPath, targetPath, readOnly); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to publish volume: %v", err)
	}

	s.logger.Info("volume published successfully", "volume_id", volumeID, "target_path", targetPath)
	return &csi.NodePublishVolumeResponse{}, nil
}

// NodeUnpublishVolume unmounts the volume from the target path
func (s *NodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID is required")
	}

	targetPath := req.GetTargetPath()
	if targetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "target path is required")
	}

	s.logger.Info("unpublishing volume", "volume_id", volumeID, "target_path", targetPath)

	// Call Volume Manager to unpublish (unmount) the volume
	if err := s.client.UnpublishVolume(ctx, volumeID, targetPath); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unpublish volume: %v", err)
	}

	s.logger.Info("volume unpublished successfully", "volume_id", volumeID)
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

// NodeGetCapabilities returns node capabilities
func (s *NodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
					},
				},
			},
		},
	}, nil
}

// NodeGetInfo returns node information
func (s *NodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return &csi.NodeGetInfoResponse{
		NodeId: s.nodeID,
		// For local filesystem, no topology constraints - any node can access volumes
	}, nil
}

// NodeGetVolumeStats returns volume statistics (optional)
func (s *NodeServer) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "NodeGetVolumeStats is not implemented")
}

// NodeExpandVolume expands the volume (optional)
func (s *NodeServer) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "NodeExpandVolume is not implemented")
}
