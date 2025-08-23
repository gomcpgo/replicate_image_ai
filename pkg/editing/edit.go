package editing

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gomcpgo/replicate_image_ai/pkg/client"
	"github.com/gomcpgo/replicate_image_ai/pkg/storage"
	"github.com/gomcpgo/replicate_image_ai/pkg/types"
)

// Editor handles image editing operations
type Editor struct {
	client  *client.ReplicateClient
	storage *storage.Storage
	debug   bool
}

// NewEditor creates a new Editor instance
func NewEditor(client *client.ReplicateClient, storage *storage.Storage, debug bool) *Editor {
	return &Editor{
		client:  client,
		storage: storage,
		debug:   debug,
	}
}

// EditImage performs text-based image editing using FLUX Kontext
func (e *Editor) EditImage(ctx context.Context, params EditParams) (*EditResult, error) {
	startTime := time.Now()
	
	// Validate parameters
	if err := e.validateEditParams(&params); err != nil {
		return nil, err
	}
	
	// Get model ID from alias if needed
	modelID := GetModelFromAlias(params.Model)
	
	// Generate unique ID for this operation
	id, err := e.storage.GenerateID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID: %w", err)
	}
	
	// Convert image to base64 data URL
	dataURL, err := storage.ImageToBase64(params.ImagePath)
	if err != nil {
		return nil, EditError{
			Code:    "file_error",
			Message: fmt.Sprintf("failed to load image: %v", err),
			Details: map[string]interface{}{
				"file_path": params.ImagePath,
			},
		}
	}
	
	// Build input parameters for FLUX Kontext
	input := e.buildEditInput(modelID, dataURL, params)
	
	if e.debug {
		log.Printf("Editing image with model %s", modelID)
		log.Printf("Edit prompt: %s", params.Prompt)
	}
	
	// Create prediction
	prediction, err := e.client.CreatePrediction(ctx, modelID, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create prediction: %w", err)
	}
	
	// Poll for completion (editing can take time)
	const maxAttempts = 60
	const pollInterval = 2 * time.Second
	
	var result *types.ReplicatePredictionResponse
	for i := 0; i < maxAttempts; i++ {
		result, err = e.client.GetPrediction(ctx, prediction.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get prediction status: %w", err)
		}
		
		if result.Status == "succeeded" {
			break
		}
		
		if result.Status == "failed" || result.Status == "canceled" {
			return nil, EditError{
				Code:    "editing_failed",
				Message: fmt.Sprintf("Editing %s: %v", result.Status, result.Error),
				Details: map[string]interface{}{
					"prediction_id": prediction.ID,
					"status":        result.Status,
				},
			}
		}
		
		time.Sleep(pollInterval)
	}
	
	if result == nil || result.Status != "succeeded" {
		return nil, EditError{
			Code:    "timeout",
			Message: "Editing timed out",
			Details: map[string]interface{}{
				"prediction_id": prediction.ID,
			},
		}
	}
	
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
		return nil, EditError{
			Code:    "no_output",
			Message: "No output URL in result",
		}
	}
	
	// Download and save image
	filename := e.generateFilename(params.Filename, params.ImagePath, "edited")
	outputPath, err := e.storage.SaveImage(id, outputURL, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to save image: %w", err)
	}
	
	// Calculate metrics
	inputInfo, _ := os.Stat(params.ImagePath)
	outputInfo, _ := os.Stat(outputPath)
	
	metrics := EditMetrics{
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
		Operation: "edit_image",
		Timestamp: time.Now(),
		Model:     modelID,
		Parameters: map[string]interface{}{
			"input_path":     params.ImagePath,
			"prompt":         params.Prompt,
			"model":          params.Model,
			"strength":       params.Strength,
			"guidance_scale": params.GuidanceScale,
		},
		Result: opResult,
	}
	
	if err := e.storage.SaveMetadata(id, metadata); err != nil && e.debug {
		log.Printf("Failed to save metadata: %v", err)
	}
	
	// Build result
	modelInfo := GetModelInfo(modelID)
	return &EditResult{
		ID:           id,
		Operation:    "edit_image",
		InputPath:    params.ImagePath,
		OutputPath:   outputPath,
		OutputURL:    outputURL,
		Model:        modelID,
		ModelName:    modelInfo.Name,
		EditPrompt:   params.Prompt,
		Parameters:   input,
		Metrics:      metrics,
		PredictionID: prediction.ID,
	}, nil
}

// validateEditParams validates and sets defaults for edit parameters
func (e *Editor) validateEditParams(params *EditParams) error {
	if params.ImagePath == "" {
		return EditError{
			Code:    "invalid_parameters",
			Message: "image path is required",
		}
	}
	
	if params.Prompt == "" {
		return EditError{
			Code:    "invalid_parameters",
			Message: "edit prompt is required",
		}
	}
	
	// Set defaults
	if params.Strength == 0 {
		params.Strength = 0.8 // Strong edit by default
	}
	
	if params.GuidanceScale == 0 {
		params.GuidanceScale = 7.5
	}
	
	if params.NumOutputs <= 0 {
		params.NumOutputs = 1
	}
	
	return nil
}

// buildEditInput builds input parameters for FLUX Kontext editing
func (e *Editor) buildEditInput(modelID string, dataURL string, params EditParams) map[string]interface{} {
	// FLUX Kontext models have similar input structure
	input := map[string]interface{}{
		"image":          dataURL,
		"prompt":         params.Prompt,
		"guidance_scale": params.GuidanceScale,
		"num_outputs":    params.NumOutputs,
	}
	
	// Add model-specific parameters
	switch modelID {
	case ModelFluxKontextPro:
		// Pro model - balanced settings
		if params.Strength > 0 {
			input["strength"] = params.Strength
		}
		
	case ModelFluxKontextMax:
		// Max model - highest quality settings
		if params.Strength > 0 {
			input["strength"] = params.Strength
		}
		input["quality"] = "max"
		
	case ModelFluxKontextDev:
		// Dev model - all controls exposed
		if params.Strength > 0 {
			input["strength"] = params.Strength
		}
		input["num_inference_steps"] = 50 // More steps for dev
	}
	
	// Add seed if specified
	if params.Seed > 0 {
		input["seed"] = params.Seed
	}
	
	return input
}

// generateFilename generates a filename for the edited image
func (e *Editor) generateFilename(userFilename, inputPath, suffix string) string {
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