package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds the configuration for the Replicate Image AI MCP server
type Config struct {
	// Required
	ReplicateAPIToken     string
	ReplicateImagesRoot   string
	
	// Optional with defaults
	MaxImageSizeMB        int
	MaxBatchSize          int
	OperationTimeout      time.Duration
	DebugMode            bool
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{
		// Set defaults
		MaxImageSizeMB:   5,
		MaxBatchSize:     10,
		OperationTimeout: 30 * time.Second,
		DebugMode:        false,
	}

	// Required fields
	cfg.ReplicateAPIToken = os.Getenv("REPLICATE_API_TOKEN")
	if cfg.ReplicateAPIToken == "" {
		return nil, fmt.Errorf("REPLICATE_API_TOKEN environment variable is required")
	}

	cfg.ReplicateImagesRoot = os.Getenv("REPLICATE_IMAGES_ROOT_FOLDER")
	if cfg.ReplicateImagesRoot == "" {
		// Default to current directory + /replicate_images
		cfg.ReplicateImagesRoot = "./replicate_images"
	}

	// Optional fields
	if maxSize := os.Getenv("MAX_IMAGE_SIZE_MB"); maxSize != "" {
		val, err := strconv.Atoi(maxSize)
		if err != nil {
			return nil, fmt.Errorf("invalid MAX_IMAGE_SIZE_MB: %w", err)
		}
		cfg.MaxImageSizeMB = val
	}

	if maxBatch := os.Getenv("MAX_BATCH_SIZE"); maxBatch != "" {
		val, err := strconv.Atoi(maxBatch)
		if err != nil {
			return nil, fmt.Errorf("invalid MAX_BATCH_SIZE: %w", err)
		}
		cfg.MaxBatchSize = val
	}

	if timeout := os.Getenv("OPERATION_TIMEOUT_SECONDS"); timeout != "" {
		val, err := strconv.Atoi(timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid OPERATION_TIMEOUT_SECONDS: %w", err)
		}
		cfg.OperationTimeout = time.Duration(val) * time.Second
	}

	if debug := os.Getenv("DEBUG_MODE"); debug != "" {
		val, err := strconv.ParseBool(debug)
		if err != nil {
			return nil, fmt.Errorf("invalid DEBUG_MODE: %w", err)
		}
		cfg.DebugMode = val
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.ReplicateAPIToken == "" {
		return fmt.Errorf("Replicate API token is required")
	}
	if c.MaxImageSizeMB <= 0 {
		return fmt.Errorf("max image size must be positive")
	}
	if c.MaxBatchSize <= 0 {
		return fmt.Errorf("max batch size must be positive")
	}
	if c.OperationTimeout <= 0 {
		return fmt.Errorf("operation timeout must be positive")
	}
	
	// Create images root folder if it doesn't exist
	if err := os.MkdirAll(c.ReplicateImagesRoot, 0755); err != nil {
		return fmt.Errorf("failed to create images root folder: %w", err)
	}
	
	return nil
}