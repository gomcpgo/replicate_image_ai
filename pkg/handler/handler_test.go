package handler

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gomcpgo/mcp/pkg/protocol"
	"github.com/gomcpgo/replicate_image_ai/pkg/client"
	"github.com/gomcpgo/replicate_image_ai/pkg/config"
	"github.com/gomcpgo/replicate_image_ai/pkg/editing"
	"github.com/gomcpgo/replicate_image_ai/pkg/enhancement"
	"github.com/gomcpgo/replicate_image_ai/pkg/generation"
	"github.com/gomcpgo/replicate_image_ai/pkg/storage"
)

// TestAsyncFlow_Complete tests the complete async flow from initial request to completion
func TestAsyncFlow_Complete(t *testing.T) {
	// Setup mock client with 3 second response time
	mockClient := client.NewMockClient()
	mockClient.SetResponseDelay(3 * time.Second)
	
	// Setup storage
	tempDir := t.TempDir()
	store := storage.NewStorage(tempDir)
	
	// Create generator with short initial wait
	timeouts := config.TimeoutConfig{
		InitialWait:  1 * time.Second,
		ContinueWait: 2 * time.Second,
		PollInterval: 100 * time.Millisecond,
	}
	gen := generation.NewGeneratorWithTimeouts(mockClient, store, timeouts, false)
	enh := enhancement.NewEnhancer(mockClient, store, false)
	edit := editing.NewEditor(mockClient, store, false)
	
	// Create handler
	h := &ReplicateImageHandler{
		generator:  gen,
		enhancer:   enh,
		editor:     edit,
		storage:    store,
		client:     mockClient,
		pendingOps: NewPendingOperationsManager(),
		debug:      false,
	}
	
	ctx := context.Background()
	
	// Step 1: Initial call to generate_image
	req1 := &protocol.CallToolRequest{
		Name: "generate_image",
		Arguments: map[string]interface{}{
			"prompt": "test image",
			"model":  "flux-schnell",
		},
	}
	
	resp1, err := h.CallTool(ctx, req1)
	if err != nil {
		t.Fatalf("Initial call failed: %v", err)
	}
	
	// Should get processing response
	if len(resp1.Content) == 0 {
		t.Fatal("Expected content in response")
	}
	
	respText := resp1.Content[0].Text
	if !strings.Contains(respText, "processing") {
		t.Errorf("Expected processing status, got: %s", respText)
	}
	
	// Extract prediction_id
	var respData map[string]interface{}
	if err := json.Unmarshal([]byte(respText), &respData); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	
	predictionID, ok := respData["prediction_id"].(string)
	if !ok || predictionID == "" {
		t.Fatal("Expected prediction_id in response")
	}
	
	// Verify operation is in pending operations
	if _, exists := h.pendingOps.Get(predictionID); !exists {
		t.Error("Expected operation to be in pending operations")
	}
	
	// Step 2: Call continue_operation while still processing
	req2 := &protocol.CallToolRequest{
		Name: "continue_operation",
		Arguments: map[string]interface{}{
			"prediction_id": predictionID,
			"wait_time":     1, // Short wait
		},
	}
	
	resp2, err := h.CallTool(ctx, req2)
	if err != nil {
		t.Fatalf("Continue call failed: %v", err)
	}
	
	// Could be either processing or completed depending on timing
	respText2 := resp2.Content[0].Text
	// Just verify it's a valid response
	if !strings.Contains(respText2, "processing") && !strings.Contains(respText2, "success") {
		t.Errorf("Expected valid response, got: %s", respText2)
	}
	
	// Step 3: Mark prediction as complete in mock
	mockClient.SetPredictionComplete(predictionID)
	
	// Step 4: Call continue_operation again
	req3 := &protocol.CallToolRequest{
		Name: "continue_operation",
		Arguments: map[string]interface{}{
			"prediction_id": predictionID,
			"wait_time":     5,
		},
	}
	
	resp3, err := h.CallTool(ctx, req3)
	if err != nil {
		t.Fatalf("Final continue call failed: %v", err)
	}
	
	// Should now be completed
	respText3 := resp3.Content[0].Text
	if !strings.Contains(respText3, "success") {
		t.Errorf("Expected success, got: %s", respText3)
	}
	
	// Verify operation removed from pending
	if _, exists := h.pendingOps.Get(predictionID); exists {
		t.Error("Expected operation to be removed from pending operations")
	}
}

// TestContinueOperation_InvalidPredictionID tests handling of invalid prediction ID
func TestContinueOperation_InvalidPredictionID(t *testing.T) {
	// Setup
	mockClient := client.NewMockClient()
	tempDir := t.TempDir()
	store := storage.NewStorage(tempDir)
	
	h := &ReplicateImageHandler{
		generator:  generation.NewGenerator(mockClient, store, false),
		enhancer:   enhancement.NewEnhancer(mockClient, store, false),
		editor:     editing.NewEditor(mockClient, store, false),
		storage:    store,
		client:     mockClient,
		pendingOps: NewPendingOperationsManager(),
		debug:      false,
	}
	
	ctx := context.Background()
	
	// Call continue_operation with invalid ID
	req := &protocol.CallToolRequest{
		Name: "continue_operation",
		Arguments: map[string]interface{}{
			"prediction_id": "invalid-id-12345",
			"wait_time":     5,
		},
	}
	
	resp, err := h.CallTool(ctx, req)
	
	// Should return error
	if err == nil {
		t.Error("Expected error for invalid prediction ID")
	}
	
	if resp != nil {
		t.Errorf("Expected nil response on error, got: %+v", resp)
	}
}

// TestAsyncFlow_MultipleOperations tests handling multiple concurrent async operations
func TestAsyncFlow_MultipleOperations(t *testing.T) {
	// Setup
	mockClient := client.NewMockClient()
	mockClient.SetResponseDelay(2 * time.Second)
	
	tempDir := t.TempDir()
	store := storage.NewStorage(tempDir)
	
	timeouts := config.TimeoutConfig{
		InitialWait:  500 * time.Millisecond,
		PollInterval: 100 * time.Millisecond,
	}
	gen := generation.NewGeneratorWithTimeouts(mockClient, store, timeouts, false)
	
	h := &ReplicateImageHandler{
		generator:  gen,
		enhancer:   enhancement.NewEnhancer(mockClient, store, false),
		editor:     editing.NewEditor(mockClient, store, false),
		storage:    store,
		client:     mockClient,
		pendingOps: NewPendingOperationsManager(),
		debug:      false,
	}
	
	ctx := context.Background()
	
	// Start multiple operations
	var predictionIDs []string
	
	for i := 0; i < 3; i++ {
		req := &protocol.CallToolRequest{
			Name: "generate_image",
			Arguments: map[string]interface{}{
				"prompt": "test image " + string(rune('A'+i)),
				"model":  "flux-schnell",
			},
		}
		
		resp, err := h.CallTool(ctx, req)
		if err != nil {
			t.Fatalf("Call %d failed: %v", i, err)
		}
		
		// Extract prediction ID
		var respData map[string]interface{}
		json.Unmarshal([]byte(resp.Content[0].Text), &respData)
		if predID, ok := respData["prediction_id"].(string); ok {
			predictionIDs = append(predictionIDs, predID)
		}
	}
	
	// Should have 3 pending operations
	if len(predictionIDs) != 3 {
		t.Fatalf("Expected 3 prediction IDs, got %d", len(predictionIDs))
	}
	
	// Each should be trackable independently
	for _, predID := range predictionIDs {
		if _, exists := h.pendingOps.Get(predID); !exists {
			t.Errorf("Expected prediction %s to be pending", predID)
		}
	}
}

// TestTimeoutConfiguration tests that timeout configuration is respected
func TestTimeoutConfiguration(t *testing.T) {
	// Test with very short timeout
	mockClient := client.NewMockClient()
	mockClient.SetResponseDelay(5 * time.Second)
	
	tempDir := t.TempDir()
	store := storage.NewStorage(tempDir)
	
	// Very short timeout - should immediately go async
	timeouts := config.TimeoutConfig{
		InitialWait:  100 * time.Millisecond,
		PollInterval: 50 * time.Millisecond,
	}
	gen := generation.NewGeneratorWithTimeouts(mockClient, store, timeouts, false)
	
	h := &ReplicateImageHandler{
		generator:  gen,
		enhancer:   enhancement.NewEnhancer(mockClient, store, false),
		editor:     editing.NewEditor(mockClient, store, false),
		storage:    store,
		client:     mockClient,
		pendingOps: NewPendingOperationsManager(),
		debug:      false,
	}
	
	ctx := context.Background()
	
	start := time.Now()
	req := &protocol.CallToolRequest{
		Name: "generate_image",
		Arguments: map[string]interface{}{
			"prompt": "test",
			"model":  "flux-schnell",
		},
	}
	
	resp, err := h.CallTool(ctx, req)
	elapsed := time.Since(start)
	
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	
	// Should return quickly (around 100ms, not 5s)
	if elapsed > 500*time.Millisecond {
		t.Errorf("Expected quick return, took %v", elapsed)
	}
	
	// Should be processing
	if !strings.Contains(resp.Content[0].Text, "processing") {
		t.Error("Expected processing status")
	}
}