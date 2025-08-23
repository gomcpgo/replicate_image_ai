package enhancement

import (
	"context"
	"fmt"
	"time"

	"github.com/gomcpgo/replicate_image_ai/pkg/storage"
)

// RestorePhoto restores old or damaged photos
func (e *Enhancer) RestorePhoto(ctx context.Context, params RestorePhotoParams) (*EnhancementResult, error) {
	startTime := time.Now()
	
	// Validate parameters
	if params.ImagePath == "" {
		return nil, EnhancementError{
			Code:    "invalid_parameters",
			Message: "image path is required",
		}
	}
	
	// Set default fidelity if not specified
	if params.Fidelity == 0 {
		params.Fidelity = 0.5
	}
	
	// Get model ID from alias if needed
	modelID := GetModelFromAlias("restore_photo", params.Model)
	
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
	input := e.buildRestoreInput(modelID, dataURL, params)
	
	e.logDebug("Restoring photo with model %s", modelID)
	
	// Create prediction
	prediction, err := e.client.CreatePrediction(ctx, modelID, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create prediction: %w", err)
	}
	
	// Poll for completion (restoration can take longer)
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
	filename := e.generateFilename(params.Filename, params.ImagePath, "restored")
	outputPath, err := e.storage.DownloadAndSaveImage(outputURL, id, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to save image: %w", err)
	}
	
	// Calculate metrics
	inputInfo, _ := e.storage.GetFileInfo(params.ImagePath)
	outputInfo, _ := e.storage.GetFileInfo(outputPath)
	
	metrics := EnhancementMetrics{
		ProcessingTime: time.Since(startTime).Seconds(),
		InputSize:      inputInfo.Size(),
		OutputSize:     outputInfo.Size(),
	}
	
	// Save metadata
	metadata := &EnhancementMetadata{
		Version:   "1.0",
		ID:        id,
		Operation: "restore_photo",
		Timestamp: time.Now(),
		Model:     modelID,
		Parameters: map[string]interface{}{
			"input_path":      params.ImagePath,
			"model":           params.Model,
			"fidelity":        params.Fidelity,
			"face_enhance":    params.FaceEnhance,
			"colorize":        params.Colorize,
			"scratch_removal": params.ScratchRemoval,
		},
		Result: result,
	}
	
	if err := e.storage.SaveMetadata(id, metadata); err != nil {
		e.logDebug("Failed to save metadata: %v", err)
	}
	
	// Build result
	modelInfo := GetModelInfo(modelID)
	return &EnhancementResult{
		ID:           id,
		Operation:    "restore_photo",
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

// buildRestoreInput builds input parameters for photo restoration
func (e *Enhancer) buildRestoreInput(modelID string, dataURL string, params RestorePhotoParams) map[string]interface{} {
	switch modelID {
	case ModelBOPBTL:
		input := map[string]interface{}{
			"image":            dataURL,
			"HR":               true, // High resolution
			"with_scratch":     params.ScratchRemoval,
		}
		return input
		
	case ModelGFPGAN:
		// When used for restoration
		input := map[string]interface{}{
			"img":     dataURL,
			"version": "v1.4",
			"scale":   2,
		}
		return input
		
	case ModelCodeFormer:
		// When used for restoration
		input := map[string]interface{}{
			"image":               dataURL,
			"codeformer_fidelity": params.Fidelity,
			"upscale":             2,
			"background_enhance":  true,
		}
		if params.FaceEnhance {
			input["face_upsample"] = true
		}
		return input
		
	default:
		return map[string]interface{}{
			"image": dataURL,
		}
	}
}