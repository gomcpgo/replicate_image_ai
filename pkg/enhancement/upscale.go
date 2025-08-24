package enhancement

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gomcpgo/replicate_image_ai/pkg/storage"
	"github.com/gomcpgo/replicate_image_ai/pkg/types"
)

// UpscaleImage upscales an image to higher resolution
func (e *Enhancer) UpscaleImage(ctx context.Context, params UpscaleParams) (*EnhancementResult, error) {
	startTime := time.Now()
	
	// Validate parameters
	if params.ImagePath == "" {
		return nil, EnhancementError{
			Code:    "invalid_parameters",
			Message: "image path is required",
		}
	}
	
	if params.Scale <= 0 {
		params.Scale = 4 // Default scale
	}
	
	// Get model ID from alias if needed
	modelID := GetModelFromAlias("upscale", params.Model)
	
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
	input := e.buildUpscaleInput(modelID, dataURL, params)
	
	e.logDebug("Upscaling image with model %s, scale %dx", modelID, params.Scale)
	
	// Create prediction
	prediction, err := e.client.CreatePrediction(ctx, modelID, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create prediction: %w", err)
	}
	
	// Poll for completion (upscaling can take longer)
	result, err := e.pollForCompletion(ctx, prediction.ID, 60, 2*time.Second)
	if err != nil {
		return nil, err
	}
	
	// Extract output URL
	outputURL, err := e.extractOutputURL(result)
	if err != nil {
		return nil, err
	}
	
	// Download and save image
	filename := e.generateFilename(params.Filename, params.ImagePath, fmt.Sprintf("upscaled_%dx", params.Scale))
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
		ScaleFactor:    params.Scale,
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
		Operation: "upscale_image",
		Timestamp: time.Now(),
		Model:     modelID,
		Parameters: map[string]interface{}{
			"input_path":    params.ImagePath,
			"model":         params.Model,
			"scale":         params.Scale,
			"face_enhance":  params.FaceEnhance,
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
		Operation:    "upscale_image",
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

// buildUpscaleInput builds input parameters for upscaling
func (e *Enhancer) buildUpscaleInput(modelID string, dataURL string, params UpscaleParams) map[string]interface{} {
	switch modelID {
	case ModelRealESRGAN:
		input := map[string]interface{}{
			"img":   dataURL,
			"scale": params.Scale,
		}
		if params.FaceEnhance {
			input["face_enhance"] = true
		}
		return input
		
	case ModelESRGAN:
		return map[string]interface{}{
			"image": dataURL,
			"scale": params.Scale,
		}
		
	case ModelSwinIR:
		return map[string]interface{}{
			"image": dataURL,
			"task_type": "Real-World Image Super-Resolution",
			"scale": params.Scale,
		}
		
	default:
		return map[string]interface{}{
			"image": dataURL,
			"scale": params.Scale,
		}
	}
}