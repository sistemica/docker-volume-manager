package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sistemica/docker-volume-manager/pkg/api"
	"github.com/sistemica/docker-volume-manager/pkg/config"
	"github.com/sistemica/docker-volume-manager/pkg/store"

	// Import backends to register them
	_ "github.com/sistemica/docker-volume-manager/pkg/storage/local"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup logger
	logger := setupLogger(cfg)
	logger.Info("starting volume manager",
		"version", "0.1.0",
		"environment", cfg.Environment,
		"port", cfg.Port,
	)

	// Create metadata store (etcd or memory)
	var metaStore store.Store

	if cfg.EtcdEnabled {
		etcdCfg := store.EtcdConfig{
			DataDir:      cfg.DataDir,
			Name:         cfg.ServiceName,
			ClusterSize:  cfg.ClusterSize,
			ServiceName:  cfg.ServiceName,
			TaskSlot:     cfg.TaskSlot,
			ClientPort:   cfg.EtcdClientPort,
			PeerPort:     cfg.EtcdPeerPort,
		}
		metaStore, err = store.NewEtcdStore(etcdCfg, logger)
		if err != nil {
			logger.Error("failed to create etcd store", "error", err)
			os.Exit(1)
		}
		logger.Info("initialized embedded etcd metadata store",
			"cluster_size", cfg.ClusterSize,
			"task_slot", cfg.TaskSlot,
		)
	} else {
		metaStore = store.NewMemoryStore()
		logger.Info("initialized in-memory metadata store")
	}
	defer metaStore.Close()

	// Create API server
	server := api.NewServer(cfg, metaStore, logger)

	// Start server in goroutine
	go func() {
		if err := server.Start(); err != nil {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	logger.Info("volume manager started successfully")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down gracefully")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "error", err)
		os.Exit(1)
	}

	logger.Info("volume manager stopped")
}

// setupLogger configures the structured logger
func setupLogger(cfg *config.Config) *slog.Logger {
	var level slog.Level

	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if cfg.IsDevelopment() {
		// Text format for development
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		// JSON format for production
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
