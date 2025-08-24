package generation

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gomcpgo/replicate_image_ai/pkg/storage"
	"github.com/gomcpgo/replicate_image_ai/pkg/types"
)

// GenerateWithVisualContext generates images using RunwayML Gen-4 with reference images
func (g *Generator) GenerateWithVisualContext(ctx context.Context, params Gen4Params) (*ImageResult, error) {
	startTime := time.Now()
	
	// Validate parameters
	if err := g.validateGen4Params(params); err != nil {
		return nil, err
	}
	
	// Generate unique ID for this operation
	id, err := g.storage.GenerateID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID: %w", err)
	}
	
	// Convert local file paths to data URLs
	imageURLs, err := g.convertImagesToDataURLs(params.ReferenceImages)
	if err != nil {
		return nil, err
	}
	
	if g.debug {
		log.Printf("Generating with visual context: %d reference images", len(imageURLs))
		log.Printf("Reference tags: %v", params.ReferenceTags)
	}
	
	// Build input parameters for Gen-4
	input := map[string]interface{}{
		"prompt":           params.Prompt,
		"reference_images": imageURLs,
		"reference_tags":   params.ReferenceTags,
		"aspect_ratio":     params.AspectRatio,
		"resolution":       params.Resolution,
	}
	
	if params.Seed > 0 {
		input["seed"] = params.Seed
	}
	
	// Create prediction with Gen-4 model
	prediction, err := g.client.CreatePrediction(ctx, ModelGen4Image, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create prediction: %w", err)
	}
	
	// Wait for initial period (15 seconds) - Gen-4 can take longer
	// Use a separate context to ensure we return after 15 seconds
	// regardless of the original MCP timeout
	ctx15s, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	// Start a goroutine to wait for completion
	type waitResult struct {
		prediction *types.ReplicatePredictionResponse
		err        error
	}
	resultChan := make(chan waitResult, 1)
	
	go func() {
		pred, err := g.client.WaitForCompletion(ctx15s, prediction.ID, 15*time.Second)
		resultChan <- waitResult{prediction: pred, err: err}
	}()
	
	// Wait for either result or timeout
	var result *types.ReplicatePredictionResponse
	
	select {
	case res := <-resultChan:
		result = res.prediction
		err = res.err
	case <-time.After(15 * time.Second):
		// Timeout - return processing status
		result = nil
		err = fmt.Errorf("operation timed out after 15s")
	}
	
	// Check if it's still processing
	if err != nil && strings.Contains(err.Error(), "timed out") {
		// Return processing status
		return &ImageResult{
			ID:           id,
			Status:       "processing",
			PredictionID: prediction.ID,
			StorageID:    id,
			Model:        ModelGen4Image,
			ModelName:    "RunwayML Gen-4",
			Prompt:       params.Prompt,
			Parameters:   input,
		}, nil
	}
	
	if err != nil {
		return nil, GenerationError{
			Code:    "generation_failed",
			Message: fmt.Sprintf("Generation failed: %v", err),
			Details: map[string]interface{}{
				"prediction_id": prediction.ID,
			},
		}
	}
	
	// Process output
	outputURL := ""
	if output, ok := result.Output.([]interface{}); ok && len(output) > 0 {
		if url, ok := output[0].(string); ok {
			outputURL = url
		}
	} else if url, ok := result.Output.(string); ok {
		outputURL = url
	}
	
	if outputURL == "" {
		return nil, GenerationError{
			Code:    "no_output",
			Message: "No output URL in result",
		}
	}
	
	// Download and save image
	filename := g.generateFilename(params.Filename, params.Prompt, ModelGen4Image)
	imagePath, err := g.storage.SaveImage(id, outputURL, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to save image: %w", err)
	}
	
	// Calculate metrics
	fileInfo, _ := os.Stat(imagePath)
	metrics := GenerationMetrics{
		GenerationTime: time.Since(startTime).Seconds(),
		FileSize:       fileInfo.Size(),
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
		Operation: "generate_with_visual_context",
		Timestamp: time.Now(),
		Model:     ModelGen4Image,
		Parameters: map[string]interface{}{
			"prompt":           params.Prompt,
			"reference_images": params.ReferenceImages, // Store original paths
			"reference_tags":   params.ReferenceTags,
			"aspect_ratio":     params.AspectRatio,
			"resolution":       params.Resolution,
		},
		Result: opResult,
	}
	
	if err := g.storage.SaveMetadata(id, metadata); err != nil && g.debug {
		log.Printf("Failed to save metadata: %v", err)
	}
	
	// Build result
	modelInfo := GetModelInfo(ModelGen4Image)
	return &ImageResult{
		ID:        id,
		FilePath:  imagePath,
		URL:       outputURL,
		Model:     ModelGen4Image,
		ModelName: modelInfo.Name,
		Prompt:    params.Prompt,
		Parameters: map[string]interface{}{
			"prompt":           params.Prompt,
			"reference_images": len(params.ReferenceImages),
			"reference_tags":   params.ReferenceTags,
			"aspect_ratio":     params.AspectRatio,
			"resolution":       params.Resolution,
		},
		Metrics:      metrics,
		PredictionID: prediction.ID,
		Status:       "completed",
		StorageID:    id,
	}, nil
}

// validateGen4Params validates the parameters for Gen-4 generation
func (g *Generator) validateGen4Params(params Gen4Params) error {
	if params.Prompt == "" {
		return GenerationError{
			Code:    "invalid_parameters",
			Message: "prompt is required",
		}
	}
	
	if len(params.ReferenceImages) == 0 || len(params.ReferenceImages) > 3 {
		return GenerationError{
			Code:    "invalid_parameters",
			Message: "reference_images must contain 1-3 image paths",
		}
	}
	
	if len(params.ReferenceTags) != len(params.ReferenceImages) {
		return GenerationError{
			Code:    "invalid_parameters",
			Message: "reference_tags must match the number of reference_images",
		}
	}
	
	// Validate tags format (3-15 alphanumeric characters)
	for _, tag := range params.ReferenceTags {
		if len(tag) < 3 || len(tag) > 15 {
			return GenerationError{
				Code:    "invalid_parameters",
				Message: fmt.Sprintf("reference_tag '%s' must be 3-15 characters", tag),
			}
		}
		
		for _, ch := range tag {
			if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')) {
				return GenerationError{
					Code:    "invalid_parameters",
					Message: fmt.Sprintf("reference_tag '%s' must contain only alphanumeric characters", tag),
				}
			}
		}
	}
	
	// Set defaults if not provided
	if params.AspectRatio == "" {
		params.AspectRatio = "16:9"
	}
	
	if params.Resolution == "" {
		params.Resolution = "1080p"
	}
	
	return nil
}

// convertImagesToDataURLs converts local file paths to data URLs
func (g *Generator) convertImagesToDataURLs(imagePaths []string) ([]string, error) {
	imageURLs := make([]string, 0, len(imagePaths))
	
	for _, imagePath := range imagePaths {
		// Check if file exists
		if _, err := os.Stat(imagePath); os.IsNotExist(err) {
			return nil, GenerationError{
				Code:    "file_not_found",
				Message: fmt.Sprintf("reference image not found: %s", imagePath),
			}
		}
		
		// Convert to data URL
		dataURL, err := storage.ImageToBase64(imagePath)
		if err != nil {
			return nil, GenerationError{
				Code:    "file_error",
				Message: fmt.Sprintf("failed to read reference image: %v", err),
			}
		}
		
		if g.debug {
			log.Printf("Converted reference image: %s -> data URL (length: %d)", 
				imagePath, len(dataURL))
		}
		
		imageURLs = append(imageURLs, dataURL)
	}
	
	return imageURLs, nil
}