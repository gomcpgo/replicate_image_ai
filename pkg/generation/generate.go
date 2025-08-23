package generation

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

// Generator handles image generation operations
type Generator struct {
	client  *client.ReplicateClient
	storage *storage.Storage
	debug   bool
}

// NewGenerator creates a new Generator instance
func NewGenerator(client *client.ReplicateClient, storage *storage.Storage, debug bool) *Generator {
	return &Generator{
		client:  client,
		storage: storage,
		debug:   debug,
	}
}

// GenerateImage generates an image using the specified model and parameters
func (g *Generator) GenerateImage(ctx context.Context, params GenerateParams) (*ImageResult, error) {
	startTime := time.Now()
	
	// Validate parameters
	if params.Prompt == "" {
		return nil, GenerationError{
			Code:    "invalid_parameters",
			Message: "prompt is required",
		}
	}
	
	// Get model ID from alias if needed
	modelID := GetModelFromAlias(params.Model)
	
	// Generate unique ID for this operation
	id, err := g.storage.GenerateID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID: %w", err)
	}
	
	// Build input parameters based on model type
	input := g.buildInputParams(params, modelID)
	
	if g.debug {
		log.Printf("Generating image with model %s", modelID)
		log.Printf("Parameters: %+v", input)
	}
	
	// Create prediction
	prediction, err := g.client.CreatePrediction(ctx, modelID, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create prediction: %w", err)
	}
	
	// Poll for completion
	const maxAttempts = 60
	const pollInterval = 2 * time.Second
	
	var result *types.ReplicatePredictionResponse
	for i := 0; i < maxAttempts; i++ {
		result, err = g.client.GetPrediction(ctx, prediction.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get prediction status: %w", err)
		}
		
		if result.Status == "succeeded" {
			break
		}
		
		if result.Status == "failed" || result.Status == "canceled" {
			return nil, GenerationError{
				Code:    "generation_failed",
				Message: fmt.Sprintf("Generation %s: %v", result.Status, result.Error),
				Details: map[string]interface{}{
					"prediction_id": prediction.ID,
					"status":        result.Status,
				},
			}
		}
		
		time.Sleep(pollInterval)
	}
	
	if result == nil || result.Status != "succeeded" {
		return nil, GenerationError{
			Code:    "timeout",
			Message: "Generation timed out",
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
	filename := g.generateFilename(params.Filename, params.Prompt, modelID)
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
		Operation: "generate_image",
		Timestamp: time.Now(),
		Model:     modelID,
		Parameters: map[string]interface{}{
			"prompt": params.Prompt,
			"model":  params.Model,
		},
		Result: opResult,
	}
	
	if err := g.storage.SaveMetadata(id, metadata); err != nil && g.debug {
		log.Printf("Failed to save metadata: %v", err)
	}
	
	// Build result
	modelInfo := GetModelInfo(modelID)
	return &ImageResult{
		ID:           id,
		FilePath:     imagePath,
		URL:          outputURL,
		Model:        modelID,
		ModelName:    modelInfo.Name,
		Prompt:       params.Prompt,
		Parameters:   input,
		Metrics:      metrics,
		PredictionID: prediction.ID,
	}, nil
}

// buildInputParams builds the input parameters for the API based on model type
func (g *Generator) buildInputParams(params GenerateParams, modelID string) map[string]interface{} {
	input := map[string]interface{}{
		"prompt": params.Prompt,
	}
	
	// Special handling for different models
	switch modelID {
	case ModelImagen4:
		// Imagen-4 uses aspect_ratio instead of width/height
		aspectRatio := params.AspectRatio
		if aspectRatio == "" {
			aspectRatio = g.inferAspectRatio(params.Width, params.Height)
		}
		input["aspect_ratio"] = aspectRatio
		
		if params.SafetyFilter != "" {
			input["safety_filter_level"] = params.SafetyFilter
		} else {
			input["safety_filter_level"] = "block_only_high"
		}
		
		if params.OutputFormat != "" {
			input["output_format"] = params.OutputFormat
		} else {
			input["output_format"] = "jpg"
		}
		
	case ModelGen4Image:
		// Gen-4 uses aspect_ratio and resolution
		aspectRatio := params.AspectRatio
		if aspectRatio == "" {
			aspectRatio = g.inferAspectRatio(params.Width, params.Height)
		}
		input["aspect_ratio"] = aspectRatio
		
		if params.Resolution != "" {
			input["resolution"] = params.Resolution
		} else {
			input["resolution"] = "1080p"
		}
		
	default:
		// Standard models use width/height
		width := params.Width
		height := params.Height
		if width <= 0 {
			width = 1024
		}
		if height <= 0 {
			height = 1024
		}
		input["width"] = width
		input["height"] = height
		
		if params.GuidanceScale > 0 {
			input["guidance_scale"] = params.GuidanceScale
		} else {
			input["guidance_scale"] = 7.5
		}
		
		if params.NegativePrompt != "" {
			input["negative_prompt"] = params.NegativePrompt
		}
		
		if params.NumOutputs > 0 {
			input["num_outputs"] = params.NumOutputs
		} else {
			input["num_outputs"] = 1
		}
	}
	
	// Add seed if specified
	if params.Seed > 0 {
		input["seed"] = params.Seed
	}
	
	return input
}

// inferAspectRatio infers aspect ratio from width and height
func (g *Generator) inferAspectRatio(width, height int) string {
	if width <= 0 || height <= 0 {
		return "1:1"
	}
	
	ratio := float64(width) / float64(height)
	
	if ratio > 1.7 { // ~16:9
		return "16:9"
	} else if ratio < 0.6 { // ~9:16
		return "9:16"
	} else if ratio > 1.2 && ratio < 1.4 { // ~4:3
		return "4:3"
	} else if ratio > 0.7 && ratio < 0.8 { // ~3:4
		return "3:4"
	}
	
	return "1:1"
}

// generateFilename generates a filename for the image
func (g *Generator) generateFilename(userFilename, prompt, modelID string) string {
	if userFilename != "" {
		// Ensure it has an extension
		if !strings.Contains(userFilename, ".") {
			userFilename += ".png"
		}
		return userFilename
	}
	
	// Generate from prompt
	cleanPrompt := strings.ToLower(prompt)
	cleanPrompt = strings.ReplaceAll(cleanPrompt, " ", "_")
	
	// Remove special characters
	var result []rune
	for _, r := range cleanPrompt {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			result = append(result, r)
		}
	}
	
	filename := string(result)
	if len(filename) > 50 {
		filename = filename[:50]
	}
	if filename == "" {
		filename = "generated"
	}
	
	// Add model suffix
	modelName := filepath.Base(modelID)
	modelName = strings.Split(modelName, ":")[0]
	
	return fmt.Sprintf("%s_%s.png", filename, modelName)
}