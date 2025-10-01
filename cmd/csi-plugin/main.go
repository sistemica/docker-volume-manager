package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc"

	csipkg "github.com/sistemica/docker-volume-manager/pkg/driver/csi"
)

const (
	version = "0.1.0"
)

func main() {
	// Configure logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("starting CSI plugin", "version", version)

	// Get configuration from environment
	endpoint := os.Getenv("CSI_ENDPOINT")
	if endpoint == "" {
		endpoint = "unix:///csi/csi.sock"
	}

	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			logger.Error("failed to get hostname", "error", err)
			os.Exit(1)
		}
		nodeID = hostname
	}

	managerURL := os.Getenv("MANAGER_URL")
	if managerURL == "" {
		managerURL = "http://volume-manager:9789"
	}

	logger.Info("CSI plugin configuration",
		"endpoint", endpoint,
		"node_id", nodeID,
		"manager_url", managerURL,
	)

	// Create CSI services
	identityServer := csipkg.NewIdentityServer()

	controllerServer, err := csipkg.NewControllerServer(managerURL, logger)
	if err != nil {
		logger.Error("failed to create controller server", "error", err)
		os.Exit(1)
	}

	nodeServer, err := csipkg.NewNodeServer(nodeID, managerURL, logger)
	if err != nil {
		logger.Error("failed to create node server", "error", err)
		os.Exit(1)
	}

	// Parse endpoint
	scheme, addr, err := parseEndpoint(endpoint)
	if err != nil {
		logger.Error("failed to parse endpoint", "error", err)
		os.Exit(1)
	}

	// Remove existing socket file
	if scheme == "unix" {
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			logger.Error("failed to remove socket file", "error", err)
			os.Exit(1)
		}
	}

	// Create listener
	listener, err := net.Listen(scheme, addr)
	if err != nil {
		logger.Error("failed to listen", "error", err)
		os.Exit(1)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(logGRPC(logger)),
	)

	// Register CSI services
	csi.RegisterIdentityServer(grpcServer, identityServer)
	csi.RegisterControllerServer(grpcServer, controllerServer)
	csi.RegisterNodeServer(grpcServer, nodeServer)

	logger.Info("CSI plugin started", "endpoint", endpoint)

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("shutting down CSI plugin")
		grpcServer.GracefulStop()
	}()

	// Start server
	if err := grpcServer.Serve(listener); err != nil {
		logger.Error("failed to serve", "error", err)
		os.Exit(1)
	}

	logger.Info("CSI plugin stopped")
}

// parseEndpoint parses endpoint string (e.g., "unix:///csi/csi.sock")
func parseEndpoint(endpoint string) (string, string, error) {
	scheme := "unix"
	addr := endpoint

	// Parse scheme if present
	if len(endpoint) > 7 && endpoint[:7] == "unix://" {
		addr = endpoint[7:]
	} else if len(endpoint) > 6 && endpoint[:6] == "tcp://" {
		scheme = "tcp"
		addr = endpoint[6:]
	}

	return scheme, addr, nil
}

// logGRPC is a gRPC interceptor for logging
func logGRPC(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		logger.Debug("grpc request", "method", info.FullMethod)
		resp, err := handler(ctx, req)
		if err != nil {
			logger.Error("grpc request failed", "method", info.FullMethod, "error", err)
		}
		return resp, err
	}
}
