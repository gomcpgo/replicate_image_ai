package enhancement

import (
	"context"
	"fmt"
	"time"

	"github.com/gomcpgo/replicate_image_ai/pkg/storage"
)

// EnhanceFace enhances faces in an image
func (e *Enhancer) EnhanceFace(ctx context.Context, params EnhanceFaceParams) (*EnhancementResult, error) {
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
		params.Fidelity = 0.5 // Balance between quality and faithfulness
	}
	
	// Get model ID from alias if needed
	modelID := GetModelFromAlias("enhance_face", params.Model)
	
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
	input := e.buildFaceEnhanceInput(modelID, dataURL, params)
	
	e.logDebug("Enhancing faces with model %s, fidelity %.2f", modelID, params.Fidelity)
	
	// Create prediction
	prediction, err := e.client.CreatePrediction(ctx, modelID, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create prediction: %w", err)
	}
	
	// Poll for completion
	result, err := e.pollForCompletion(ctx, prediction.ID, 45, 2*time.Second)
	if err != nil {
		return nil, err
	}
	
	// Extract output URL
	outputURL, err := e.extractOutputURL(result)
	if err != nil {
		return nil, err
	}
	
	// Download and save image
	filename := e.generateFilename(params.Filename, params.ImagePath, "enhanced_face")
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
		Operation: "enhance_face",
		Timestamp: time.Now(),
		Model:     modelID,
		Parameters: map[string]interface{}{
			"input_path":         params.ImagePath,
			"model":              params.Model,
			"fidelity":           params.Fidelity,
			"only_center":        params.OnlyCenter,
			"has_aligned":        params.HasAligned,
			"background_enhance": params.BackgroundEnhance,
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
		Operation:    "enhance_face",
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

// buildFaceEnhanceInput builds input parameters for face enhancement
func (e *Enhancer) buildFaceEnhanceInput(modelID string, dataURL string, params EnhanceFaceParams) map[string]interface{} {
	switch modelID {
	case ModelGFPGAN:
		input := map[string]interface{}{
			"img":     dataURL,
			"version": "v1.4",
			"scale":   2,
		}
		return input
		
	case ModelCodeFormer:
		input := map[string]interface{}{
			"image":           dataURL,
			"codeformer_fidelity": params.Fidelity,
			"upscale":         2,
		}
		if params.BackgroundEnhance {
			input["background_enhance"] = true
		}
		if params.OnlyCenter {
			input["face_upsample"] = true
		}
		return input
		
	case ModelRestoreFormer:
		return map[string]interface{}{
			"image": dataURL,
		}
		
	default:
		return map[string]interface{}{
			"image": dataURL,
		}
	}
}