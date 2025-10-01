package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sistemica/docker-volume-manager/pkg/api/handlers"
	custommw "github.com/sistemica/docker-volume-manager/pkg/api/middleware"
	"github.com/sistemica/docker-volume-manager/pkg/config"
	"github.com/sistemica/docker-volume-manager/pkg/store"
)

// Server represents the HTTP API server
type Server struct {
	echo   *echo.Echo
	config *config.Config
	logger *slog.Logger
	store  store.Store
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, store store.Store, logger *slog.Logger) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	s := &Server{
		echo:   e,
		config: cfg,
		logger: logger,
		store:  store,
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

// setupMiddleware configures middleware
func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.echo.Use(custommw.Recovery(s.logger))

	// Logger middleware
	s.echo.Use(custommw.Logger(s.logger))

	// CORS middleware
	s.echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
	}))

	// Request ID middleware
	s.echo.Use(middleware.RequestID())

	// Timeout middleware
	s.echo.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: 30 * time.Second,
	}))
}

// setupRoutes configures API routes
func (s *Server) setupRoutes() {
	// Health checks
	healthHandler := handlers.NewHealthHandler()
	s.echo.GET("/health", healthHandler.HandleHealth)
	s.echo.GET("/ready", healthHandler.HandleReady)

	// API v1
	v1 := s.echo.Group("/api/v1")

	// Volume routes
	volumeHandler := handlers.NewVolumeHandler(s.store, s.logger)
	v1.POST("/volumes", volumeHandler.HandleCreate)
	v1.GET("/volumes", volumeHandler.HandleList)
	v1.GET("/volumes/:id", volumeHandler.HandleGet)
	v1.DELETE("/volumes/:id", volumeHandler.HandleDelete)
	v1.POST("/volumes/:id/stage", volumeHandler.HandleStage)
	v1.POST("/volumes/:id/publish", volumeHandler.HandlePublish)

	// File operations routes (RESTful - files as resources)
	fileHandler := handlers.NewFileHandler(s.store, s.logger)
	v1.GET("/volumes/:id/files/*", fileHandler.HandleGet)         // Read file or list directory
	v1.PUT("/volumes/:id/files/*", fileHandler.HandlePut)         // Create/update file
	v1.DELETE("/volumes/:id/files/*", fileHandler.HandleDelete)   // Delete file/directory

	// Backend routes
	backendHandler := handlers.NewBackendHandler(s.logger)
	v1.GET("/backends", backendHandler.HandleList)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.logger.Info("starting server", "address", addr)
	return s.echo.Start(addr)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down server")
	return s.echo.Shutdown(ctx)
}
