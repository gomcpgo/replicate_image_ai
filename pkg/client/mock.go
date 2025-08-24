package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gomcpgo/replicate_image_ai/pkg/types"
)

// MockClient is a mock implementation of the Client interface for testing
type MockClient struct {
	// Control behavior
	ResponseDelay time.Duration // How long operations take to complete
	ShouldFail    bool          // Whether operations should fail
	FailAfter     time.Duration // Fail after this duration
	FailMessage   string        // Custom failure message
	
	// Track calls for assertions
	CreateCalls []CreateCall
	GetCalls    []string
	CancelCalls []string
	
	// Predictions state
	predictions map[string]*MockPrediction
	mu          sync.RWMutex
	
	// Control completion timing
	completeAt map[string]time.Time // When each prediction should complete
}

// CreateCall records a call to CreatePrediction
type CreateCall struct {
	ModelVersion string
	Input        map[string]interface{}
	Timestamp    time.Time
}

// MockPrediction represents a mock prediction
type MockPrediction struct {
	ID         string
	Status     string
	StartTime  time.Time
	CompleteAt time.Duration // When to complete relative to start
	Output     interface{}
	Error      interface{}
}

// NewMockClient creates a new mock client
func NewMockClient() *MockClient {
	return &MockClient{
		ResponseDelay: 5 * time.Second, // Default 5 second completion
		predictions:   make(map[string]*MockPrediction),
		completeAt:    make(map[string]time.Time),
		CreateCalls:   []CreateCall{},
		GetCalls:      []string{},
		CancelCalls:   []string{},
	}
}

// CreatePrediction creates a mock prediction
func (m *MockClient) CreatePrediction(ctx context.Context, modelVersion string, input map[string]interface{}) (*types.ReplicatePredictionResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Record the call
	m.CreateCalls = append(m.CreateCalls, CreateCall{
		ModelVersion: modelVersion,
		Input:        input,
		Timestamp:    time.Now(),
	})
	
	// Check if we should fail immediately
	if m.ShouldFail && m.FailAfter == 0 {
		if m.FailMessage != "" {
			return nil, fmt.Errorf(m.FailMessage)
		}
		return nil, fmt.Errorf("mock client configured to fail")
	}
	
	// Create a mock prediction
	predID := fmt.Sprintf("mock-pred-%d", len(m.predictions)+1)
	
	pred := &MockPrediction{
		ID:         predID,
		Status:     types.StatusStarting,
		StartTime:  time.Now(),
		CompleteAt: m.ResponseDelay,
		// Use a simple base64 PNG data URL for testing (1x1 transparent pixel)
		Output:     []interface{}{"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="},
	}
	
	m.predictions[predID] = pred
	m.completeAt[predID] = time.Now().Add(m.ResponseDelay)
	
	return &types.ReplicatePredictionResponse{
		ID:        predID,
		Status:    types.StatusStarting,
		CreatedAt: time.Now().Format(time.RFC3339),
	}, nil
}

// GetPrediction gets the status of a mock prediction
func (m *MockClient) GetPrediction(ctx context.Context, predictionID string) (*types.ReplicatePredictionResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Record the call
	m.GetCalls = append(m.GetCalls, predictionID)
	
	pred, exists := m.predictions[predictionID]
	if !exists {
		return nil, fmt.Errorf("prediction not found: %s", predictionID)
	}
	
	// Calculate current status based on time
	elapsed := time.Since(pred.StartTime)
	completeTime := m.completeAt[predictionID]
	
	status := types.StatusProcessing
	var output interface{}
	var predError interface{}
	
	if m.ShouldFail && m.FailAfter > 0 && elapsed >= m.FailAfter {
		status = types.StatusFailed
		predError = m.FailMessage
		if predError == "" {
			predError = "mock failure"
		}
	} else if time.Now().After(completeTime) {
		status = types.StatusSucceeded
		output = pred.Output
	} else if elapsed < 2*time.Second {
		status = types.StatusStarting
	}
	
	return &types.ReplicatePredictionResponse{
		ID:        predictionID,
		Status:    status,
		Output:    output,
		Error:     predError,
		CreatedAt: pred.StartTime.Format(time.RFC3339),
	}, nil
}

// WaitForCompletion waits for a mock prediction to complete
func (m *MockClient) WaitForCompletion(ctx context.Context, predictionID string, timeout time.Duration) (*types.ReplicatePredictionResponse, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond) // Poll faster in tests
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				// Get last status before timeout
				pred, _ := m.GetPrediction(ctx, predictionID)
				return pred, fmt.Errorf("operation timed out after %v", timeout)
			}
			
			pred, err := m.GetPrediction(ctx, predictionID)
			if err != nil {
				return nil, err
			}
			
			switch pred.Status {
			case types.StatusSucceeded:
				return pred, nil
			case types.StatusFailed:
				return pred, fmt.Errorf("prediction failed: %v", pred.Error)
			case types.StatusCanceled:
				return pred, fmt.Errorf("prediction was canceled")
			}
		}
	}
}

// CancelPrediction cancels a mock prediction
func (m *MockClient) CancelPrediction(ctx context.Context, predictionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Record the call
	m.CancelCalls = append(m.CancelCalls, predictionID)
	
	pred, exists := m.predictions[predictionID]
	if !exists {
		return fmt.Errorf("prediction not found: %s", predictionID)
	}
	
	pred.Status = types.StatusCanceled
	return nil
}

// Helper methods for testing

// SetPredictionComplete marks a prediction as complete immediately
func (m *MockClient) SetPredictionComplete(predictionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if pred, exists := m.predictions[predictionID]; exists {
		pred.Status = types.StatusSucceeded
		m.completeAt[predictionID] = time.Now()
	}
}

// SetPredictionFailed marks a prediction as failed
func (m *MockClient) SetPredictionFailed(predictionID string, errorMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if pred, exists := m.predictions[predictionID]; exists {
		pred.Status = types.StatusFailed
		pred.Error = errorMsg
	}
}

// SetResponseDelay changes the response delay for future predictions
func (m *MockClient) SetResponseDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ResponseDelay = delay
}

// Reset clears all state for a fresh test
func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.predictions = make(map[string]*MockPrediction)
	m.completeAt = make(map[string]time.Time)
	m.CreateCalls = []CreateCall{}
	m.GetCalls = []string{}
	m.CancelCalls = []string{}
	m.ShouldFail = false
	m.FailAfter = 0
	m.FailMessage = ""
}