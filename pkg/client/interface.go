package client

import (
	"context"
	"time"

	"github.com/gomcpgo/replicate_image_ai/pkg/types"
)

// Client defines the interface for interacting with the Replicate API
type Client interface {
	// CreatePrediction creates a new prediction on Replicate
	CreatePrediction(ctx context.Context, modelVersion string, input map[string]interface{}) (*types.ReplicatePredictionResponse, error)
	
	// GetPrediction gets the status of a prediction
	GetPrediction(ctx context.Context, predictionID string) (*types.ReplicatePredictionResponse, error)
	
	// WaitForCompletion waits for a prediction to complete or timeout
	WaitForCompletion(ctx context.Context, predictionID string, timeout time.Duration) (*types.ReplicatePredictionResponse, error)
	
	// CancelPrediction cancels a running prediction
	CancelPrediction(ctx context.Context, predictionID string) error
}

// Ensure ReplicateClient implements the Client interface
var _ Client = (*ReplicateClient)(nil)