package config

import (
	"os"
	"strconv"
	"time"
)

// TimeoutConfig holds all configurable timeout values
type TimeoutConfig struct {
	// InitialWait is how long to wait before returning a processing status
	InitialWait time.Duration
	
	// ContinueWait is how long continue_operation waits for completion
	ContinueWait time.Duration
	
	// MaxOperationTime is when to clean up pending operations
	MaxOperationTime time.Duration
	
	// PollInterval is how often to check prediction status
	PollInterval time.Duration
}

// DefaultTimeouts returns the default timeout configuration
func DefaultTimeouts() TimeoutConfig {
	return TimeoutConfig{
		InitialWait:      15 * time.Second,
		ContinueWait:     30 * time.Second,
		MaxOperationTime: 10 * time.Minute,
		PollInterval:     2 * time.Second,
	}
}

// LoadTimeouts loads timeout configuration from environment variables
func LoadTimeouts() TimeoutConfig {
	config := DefaultTimeouts()
	
	// Override with environment variables if set
	if val := os.Getenv("REPLICATE_INITIAL_WAIT"); val != "" {
		if seconds, err := strconv.Atoi(val); err == nil && seconds > 0 {
			config.InitialWait = time.Duration(seconds) * time.Second
		}
	}
	
	if val := os.Getenv("REPLICATE_CONTINUE_WAIT"); val != "" {
		if seconds, err := strconv.Atoi(val); err == nil && seconds > 0 {
			config.ContinueWait = time.Duration(seconds) * time.Second
		}
	}
	
	if val := os.Getenv("REPLICATE_MAX_OPERATION_TIME"); val != "" {
		if minutes, err := strconv.Atoi(val); err == nil && minutes > 0 {
			config.MaxOperationTime = time.Duration(minutes) * time.Minute
		}
	}
	
	if val := os.Getenv("REPLICATE_POLL_INTERVAL"); val != "" {
		if seconds, err := strconv.Atoi(val); err == nil && seconds > 0 {
			config.PollInterval = time.Duration(seconds) * time.Second
		}
	}
	
	return config
}

// TestTimeouts returns timeout configuration suitable for testing
func TestTimeouts() TimeoutConfig {
	return TimeoutConfig{
		InitialWait:      1 * time.Second,
		ContinueWait:     2 * time.Second,
		MaxOperationTime: 1 * time.Minute,
		PollInterval:     100 * time.Millisecond,
	}
}