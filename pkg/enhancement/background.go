package enhancement

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gomcpgo/replicate_image_ai/pkg/storage"
	"github.com/gomcpgo/replicate_image_ai/pkg/types"
)

// RemoveBackground removes the background from an image
func (e *Enhancer) RemoveBackground(ctx context.Context, params RemoveBackgroundParams) (*EnhancementResult, error) {
	startTime := time.Now()
	
	// Validate parameters
	if params.ImagePath == "" {
		return nil, EnhancementError{
			Code:    "invalid_parameters",
			Message: "image path is required",
		}
	}
	
	// Get model ID from alias if needed
	modelID := GetModelFromAlias("remove_background", params.Model)
	
	// Generate unique ID for this operation
	id, err := e.storage.GenerateID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID: %w", err)
	}
	
	// Convert image to base64 data URL
	dataURL, err := storage.ImageToBase64(params.ImagePath)
	if err != nil {
		return nil, EnhancementError{
			Code:    "file_error",
			Message: fmt.Sprintf("failed to load image: %v", err),
			Details: map[string]interface{}{
				"file_path": params.ImagePath,
			},
		}
	}
	
	// Build input parameters based on model
	input := e.buildRemoveBackgroundInput(modelID, dataURL)
	
	e.logDebug("Removing background with model %s", modelID)
	
	// Create prediction
	prediction, err := e.client.CreatePrediction(ctx, modelID, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create prediction: %w", err)
	}
	
	// Poll for completion
	result, err := e.pollForCompletion(ctx, prediction.ID, 30, 2*time.Second)
	if err != nil {
		return nil, err
	}
	
	// Extract output URL
	outputURL, err := e.extractOutputURL(result)
	if err != nil {
		return nil, err
	}
	
	// Download and save image
	filename := e.generateFilename(params.Filename, params.ImagePath, "no_bg")
	outputPath, err := e.storage.SaveImage(id, outputURL, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to save image: %w", err)
	}
	
	// Calculate metrics
	inputInfo, _ := os.Stat(params.ImagePath)
	outputInfo, _ := os.Stat(outputPath)
	
	metrics := EnhancementMetrics{
		ProcessingTime: time.Since(startTime).Seconds(),
		InputSize:      inputInfo.Size(),
		OutputSize:     outputInfo.Size(),
	}
	
	// Save metadata
	opResult := &types.OperationResult{
		Filename:       filename,
		GenerationTime: time.Since(startTime).Seconds(),
		PredictionID:   prediction.ID,
	}
	
	metadata := &types.ImageMetadata{
		Version:   "1.0",
		ID:        id,
		Operation: "remove_background",
		Timestamp: time.Now(),
		Model:     modelID,
		Parameters: map[string]interface{}{
			"input_path": params.ImagePath,
			"model":      params.Model,
			"alpha":      params.Alpha,
		},
		Result: opResult,
	}
	
	if err := e.storage.SaveMetadata(id, metadata); err != nil {
		e.logDebug("Failed to save metadata: %v", err)
	}
	
	// Build result
	modelInfo := GetModelInfo(modelID)
	return &EnhancementResult{
		ID:           id,
		Operation:    "remove_background",
		InputPath:    params.ImagePath,
		OutputPath:   outputPath,
		OutputURL:    outputURL,
		Model:        modelID,
		ModelName:    modelInfo.Name,
		Parameters:   input,
		Metrics:      metrics,
		PredictionID: prediction.ID,
	}, nil
}

// buildRemoveBackgroundInput builds input parameters for background removal
func (e *Enhancer) buildRemoveBackgroundInput(modelID, dataURL string) map[string]interface{} {
	switch modelID {
	case ModelRemoveBG:
		return map[string]interface{}{
			"image": dataURL,
		}
	case ModelRembg:
		return map[string]interface{}{
			"image": dataURL,
		}
	case ModelDISBGRemoval:
		return map[string]interface{}{
			"image": dataURL,
		}
	default:
		return map[string]interface{}{
			"image": dataURL,
		}
	}
}

// pollForCompletion polls the API until the prediction completes
func (e *Enhancer) pollForCompletion(ctx context.Context, predictionID string, maxAttempts int, interval time.Duration) (*types.ReplicatePredictionResponse, error) {
	for i := 0; i < maxAttempts; i++ {
		result, err := e.client.GetPrediction(ctx, predictionID)
		if err != nil {
			return nil, fmt.Errorf("failed to get prediction status: %w", err)
		}
		
		if result.Status == "succeeded" {
			return result, nil
		}
		
		if result.Status == "failed" || result.Status == "canceled" {
			return nil, EnhancementError{
				Code:    "processing_failed",
				Message: fmt.Sprintf("Processing %s: %v", result.Status, result.Error),
				Details: map[string]interface{}{
					"prediction_id": predictionID,
					"status":        result.Status,
				},
			}
		}
		
		time.Sleep(interval)
	}
	
	return nil, EnhancementError{
		Code:    "timeout",
		Message: "Processing timed out",
		Details: map[string]interface{}{
			"prediction_id": predictionID,
		},
	}
}

// extractOutputURL extracts the output URL from prediction result
func (e *Enhancer) extractOutputURL(result *types.ReplicatePredictionResponse) (string, error) {
	// Handle string output
	if url, ok := result.Output.(string); ok && url != "" {
		return url, nil
	}
	
	// Handle array output
	if outputs, ok := result.Output.([]interface{}); ok && len(outputs) > 0 {
		if url, ok := outputs[0].(string); ok && url != "" {
			return url, nil
		}
	}
	
	// Handle map output with specific keys
	if outputMap, ok := result.Output.(map[string]interface{}); ok {
		// Try common keys
		for _, key := range []string{"image", "output", "url", "file"} {
			if url, ok := outputMap[key].(string); ok && url != "" {
				return url, nil
			}
		}
	}
	
	return "", EnhancementError{
		Code:    "no_output",
		Message: "No output URL in result",
	}
}

// generateFilename generates a filename for the enhanced image
func (e *Enhancer) generateFilename(userFilename, inputPath, suffix string) string {
	if userFilename != "" {
		// Ensure it has an extension
		if !strings.Contains(userFilename, ".") {
			userFilename += ".png"
		}
		return userFilename
	}
	
	// Generate from input filename
	base := filepath.Base(inputPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	
	if suffix != "" {
		name = fmt.Sprintf("%s_%s", name, suffix)
	}
	
	if ext == "" {
		ext = ".png"
	}
	
	return name + ext
}