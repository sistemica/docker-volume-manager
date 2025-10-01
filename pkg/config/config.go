package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds the application configuration
type Config struct {
	// Server configuration
	Port        int    `json:"port"`
	Host        string `json:"host"`
	Environment string `json:"environment"`
	LogLevel    string `json:"log_level"`

	// Storage configuration
	DataDir string `json:"data_dir"`

	// Etcd configuration
	EtcdEnabled bool   `json:"etcd_enabled"`
	ClusterSize int    `json:"cluster_size"`
	ServiceName string `json:"service_name"`
	TaskSlot    int    `json:"task_slot"`
	EtcdClientPort int `json:"etcd_client_port"`
	EtcdPeerPort   int `json:"etcd_peer_port"`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Try to load .env file (ignore error if not exists)
	_ = godotenv.Load()

	cfg := &Config{
		Port:           getEnvInt("PORT", 9789),
		Host:           getEnv("HOST", "0.0.0.0"),
		Environment:    getEnv("ENVIRONMENT", "development"),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
		DataDir:        getEnv("DATA_DIR", "/var/lib/volume-manager"),
		EtcdEnabled:    getEnvBool("ETCD_ENABLED", false),
		ClusterSize:    getEnvInt("CLUSTER_SIZE", 1),
		ServiceName:    getEnv("SERVICE_NAME", "volume-manager"),
		TaskSlot:       getEnvInt("TASK_SLOT", 1),
		EtcdClientPort: getEnvInt("ETCD_CLIENT_PORT", 2379),
		EtcdPeerPort:   getEnvInt("ETCD_PEER_PORT", 2380),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}

	if c.LogLevel != "debug" && c.LogLevel != "info" && c.LogLevel != "warn" && c.LogLevel != "error" {
		return fmt.Errorf("invalid log level: %s", c.LogLevel)
	}

	return nil
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an integer environment variable or returns a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvBool gets a boolean environment variable or returns a default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
