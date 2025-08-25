package handler

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gomcpgo/mcp/pkg/protocol"
	"github.com/gomcpgo/replicate_image_ai/pkg/responses"
	"github.com/gomcpgo/replicate_image_ai/pkg/types"
)

// handleContinueOperation handles the continue_operation tool
func (h *ReplicateImageHandler) handleContinueOperation(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	// Extract parameters
	predictionID, ok := args["prediction_id"].(string)
	if !ok || predictionID == "" {
		return nil, fmt.Errorf("prediction_id is required")
	}
	
	waitTime := 30
	if wt, ok := args["wait_time"].(float64); ok {
		waitTime = int(wt)
		if waitTime < 5 {
			waitTime = 5
		}
		if waitTime > 30 {
			waitTime = 30
		}
	}
	
	// Get pending operation info
	log.Printf("DEBUG: Looking for pending operation with prediction_id=%s", predictionID)
	pendingOp, exists := h.pendingOps.Get(predictionID)
	if !exists {
		log.Printf("DEBUG: Pending operation NOT FOUND for prediction_id=%s", predictionID)
		// If not in pending ops, still try to get the prediction
		// (in case of server restart or manual continuation)
		// For generate_with_visual_context, we should still download the image
		pendingOp = &PendingOperation{
			PredictionID: predictionID,
			Operation:    "generate_with_visual_context", // Assume this for now
			StartTime:    time.Now(),
			StorageID:    fmt.Sprintf("cont_%s", predictionID[:8]), // Generate a storage ID
		}
	} else {
		log.Printf("DEBUG: Found pending operation: storage_id=%s, operation=%s", pendingOp.StorageID, pendingOp.Operation)
	}
	
	if h.debug {
		log.Printf("Continuing operation %s (type: %s)", predictionID, pendingOp.Operation)
	}
	
	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, time.Duration(waitTime)*time.Second)
	defer cancel()
	
	// Poll for completion
	startTime := time.Now()
	result, err := h.client.WaitForCompletion(ctx, predictionID, time.Duration(waitTime)*time.Second)
	
	// Check if it's still processing
	if err != nil && ctx.Err() == context.DeadlineExceeded {
		// Still processing - return processing response
		elapsed := time.Since(pendingOp.StartTime)
		estimated := EstimateRemainingTime(pendingOp.Operation, elapsed)
		
		response := responses.BuildProcessingResponse(
			pendingOp.Operation,
			predictionID,
			pendingOp.StorageID,
			estimated,
		)
		
		return &protocol.CallToolResponse{
			Content: []protocol.ToolContent{
				{Type: "text", Text: response},
			},
		}, nil
	}
	
	if err != nil {
		// Operation failed
		h.pendingOps.Remove(predictionID)
		return nil, fmt.Errorf("operation failed: %w", err)
	}
	
	// Operation completed - process the result
	h.pendingOps.Remove(predictionID)
	
	// Extract output URL
	outputURL := ""
	if output, ok := result.Output.([]interface{}); ok && len(output) > 0 {
		if url, ok := output[0].(string); ok {
			outputURL = url
		}
	} else if url, ok := result.Output.(string); ok {
		outputURL = url
	}
	
	if outputURL == "" {
		return nil, fmt.Errorf("no output URL in completed prediction")
	}
	
	// Download and save the result
	filename := fmt.Sprintf("continued_%s", predictionID)
	log.Printf("DEBUG: About to save image - StorageID: %s, URL: %s", pendingOp.StorageID, outputURL)
	if pendingOp.StorageID != "" {
		log.Printf("DEBUG: Saving image with storage ID %s", pendingOp.StorageID)
		imagePath, err := h.storage.SaveImage(pendingOp.StorageID, outputURL, filename)
		if err != nil {
			return nil, fmt.Errorf("failed to save image: %w", err)
		}
		
		// Save metadata
		metadata := &types.ImageMetadata{
			Version:   "1.0",
			ID:        pendingOp.StorageID,
			Operation: pendingOp.Operation,
			Timestamp: time.Now(),
			Model:     pendingOp.Model,
			Parameters: pendingOp.Params,
			Result: &types.OperationResult{
				Filename:       filename,
				GenerationTime: time.Since(pendingOp.StartTime).Seconds(),
				PredictionID:   predictionID,
			},
		}
		
		if err := h.storage.SaveMetadata(pendingOp.StorageID, metadata); err != nil {
			log.Printf("Warning: failed to save metadata: %v", err)
		}
		
		// Build success response
		fileInfo, _ := os.Stat(imagePath)
		response := responses.BuildSuccessResponse(
			pendingOp.Operation,
			pendingOp.StorageID,
			map[string]string{
				"output": imagePath,
				"url":    outputURL,
			},
			map[string]string{
				"name": pendingOp.Model,
			},
			pendingOp.Params,
			map[string]interface{}{
				"generation_time": time.Since(pendingOp.StartTime).Seconds(),
				"file_size":       fileInfo.Size(),
			},
			predictionID,
		)
		
		return &protocol.CallToolResponse{
			Content: []protocol.ToolContent{
				{Type: "text", Text: response},
			},
		}, nil
	}
	
	// No storage ID - return simple success
	log.Printf("DEBUG: No storage ID found, returning URL only")
	response := responses.BuildSimpleSuccessResponse(
		"continue_operation",
		fmt.Sprintf("Operation completed. Output: %s", outputURL),
		map[string]interface{}{
			"prediction_id": predictionID,
			"output_url":    outputURL,
			"elapsed_time":  time.Since(startTime).Seconds(),
		},
	)
	
	return &protocol.CallToolResponse{
		Content: []protocol.ToolContent{
			{Type: "text", Text: response},
		},
	}, nil
}