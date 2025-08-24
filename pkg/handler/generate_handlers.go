package handler

import (
	"context"
	"log"
	"time"

	"github.com/gomcpgo/mcp/pkg/protocol"
	"github.com/gomcpgo/replicate_image_ai/pkg/generation"
	"github.com/gomcpgo/replicate_image_ai/pkg/responses"
)

// handleGenerateImage handles the generate_image tool
func (h *ReplicateImageHandler) handleGenerateImage(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	// Extract and validate parameters
	prompt, ok := args["prompt"].(string)
	if !ok || prompt == "" {
		return h.errorResponse("generate_image", "invalid_parameters", "prompt parameter is required", nil)
	}
	
	// Build generation parameters
	params := generation.GenerateParams{
		Prompt: prompt,
	}
	
	// Extract optional parameters
	if model, ok := args["model"].(string); ok {
		params.Model = model
	} else {
		params.Model = "flux-schnell" // Default
	}
	
	if width, ok := args["width"].(float64); ok {
		params.Width = int(width)
	}
	
	if height, ok := args["height"].(float64); ok {
		params.Height = int(height)
	}
	
	if aspectRatio, ok := args["aspect_ratio"].(string); ok {
		params.AspectRatio = aspectRatio
	}
	
	if resolution, ok := args["resolution"].(string); ok {
		params.Resolution = resolution
	}
	
	if seed, ok := args["seed"].(float64); ok {
		params.Seed = int(seed)
	}
	
	if guidanceScale, ok := args["guidance_scale"].(float64); ok {
		params.GuidanceScale = guidanceScale
	}
	
	if negativePrompt, ok := args["negative_prompt"].(string); ok {
		params.NegativePrompt = negativePrompt
	}
	
	if numOutputs, ok := args["num_outputs"].(float64); ok {
		params.NumOutputs = int(numOutputs)
	}
	
	if safetyFilter, ok := args["safety_filter_level"].(string); ok {
		params.SafetyFilter = safetyFilter
	}
	
	if outputFormat, ok := args["output_format"].(string); ok {
		params.OutputFormat = outputFormat
	}
	
	if filename, ok := args["filename"].(string); ok {
		params.Filename = filename
	}
	
	// Call core generation function
	result, err := h.generator.GenerateImage(ctx, params)
	if err != nil {
		if genErr, ok := err.(generation.GenerationError); ok {
			return h.errorResponse("generate_image", genErr.Code, genErr.Message, genErr.Details)
		}
		return h.errorResponse("generate_image", "generation_error", err.Error(), nil)
	}
	
	// Check if operation is still processing
	if result.Status == "processing" {
		// Store in pending operations
		h.pendingOps.Add(result.PredictionID, &PendingOperation{
			PredictionID: result.PredictionID,
			StorageID:    result.StorageID,
			Operation:    "generate_image",
			StartTime:    time.Now(),
			Model:        result.Model,
			Params:       result.Parameters,
		})
		
		// Return processing response
		response := responses.BuildProcessingResponse(
			"generate_image",
			result.PredictionID,
			result.StorageID,
			30, // Initial estimate
		)
		return h.successResponse(response)
	}
	
	// Build success response
	response := h.buildGenerationResponse("generate_image", result)
	return h.successResponse(response)
}

// handleGenerateWithVisualContext handles the generate_with_visual_context tool
func (h *ReplicateImageHandler) handleGenerateWithVisualContext(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	log.Printf("DEBUG: ===== START handleGenerateWithVisualContext =====")
	startTime := time.Now()
	
	// Extract and validate parameters
	prompt, ok := args["prompt"].(string)
	if !ok || prompt == "" {
		return h.errorResponse("generate_with_visual_context", "invalid_parameters", "prompt parameter is required", nil)
	}
	
	// Extract reference images
	var referenceImages []string
	if refImagesRaw, ok := args["reference_images"].([]interface{}); ok {
		for _, img := range refImagesRaw {
			if imgStr, ok := img.(string); ok {
				referenceImages = append(referenceImages, imgStr)
			}
		}
	}
	
	if len(referenceImages) == 0 {
		return h.errorResponse("generate_with_visual_context", "invalid_parameters", 
			"reference_images parameter is required (1-3 images)", nil)
	}
	
	// Extract reference tags
	var referenceTags []string
	if refTagsRaw, ok := args["reference_tags"].([]interface{}); ok {
		for _, tag := range refTagsRaw {
			if tagStr, ok := tag.(string); ok {
				referenceTags = append(referenceTags, tagStr)
			}
		}
	}
	
	if len(referenceTags) != len(referenceImages) {
		return h.errorResponse("generate_with_visual_context", "invalid_parameters",
			"reference_tags must match the number of reference_images", nil)
	}
	
	// Build Gen4 parameters
	params := generation.Gen4Params{
		Prompt:          prompt,
		ReferenceImages: referenceImages,
		ReferenceTags:   referenceTags,
	}
	
	// Extract optional parameters
	if aspectRatio, ok := args["aspect_ratio"].(string); ok {
		params.AspectRatio = aspectRatio
	} else {
		params.AspectRatio = "16:9" // Default
	}
	
	if resolution, ok := args["resolution"].(string); ok {
		params.Resolution = resolution
	} else {
		params.Resolution = "1080p" // Default
	}
	
	if seed, ok := args["seed"].(float64); ok {
		params.Seed = int(seed)
	}
	
	if filename, ok := args["filename"].(string); ok {
		params.Filename = filename
	}
	
	// Call core generation function
	log.Printf("DEBUG: Handler calling GenerateWithVisualContext")
	result, err := h.generator.GenerateWithVisualContext(ctx, params)
	if result != nil {
		log.Printf("DEBUG: GenerateWithVisualContext returned: err=%v, status=%s", err, result.Status)
	} else {
		log.Printf("DEBUG: GenerateWithVisualContext returned: err=%v, result=nil", err)
	}
	if err != nil {
		if genErr, ok := err.(generation.GenerationError); ok {
			return h.errorResponse("generate_with_visual_context", genErr.Code, genErr.Message, genErr.Details)
		}
		return h.errorResponse("generate_with_visual_context", "generation_error", err.Error(), nil)
	}
	
	// Check if operation is still processing
	if result.Status == "processing" {
		log.Printf("DEBUG: Handling processing status")
		// Store in pending operations
		h.pendingOps.Add(result.PredictionID, &PendingOperation{
			PredictionID: result.PredictionID,
			StorageID:    result.StorageID,
			Operation:    "generate_with_visual_context",
			StartTime:    time.Now(),
			Model:        result.Model,
			Params:       result.Parameters,
		})
		
		// Return processing response
		response := responses.BuildProcessingResponse(
			"generate_with_visual_context",
			result.PredictionID,
			result.StorageID,
			45, // Initial estimate for Gen-4
		)
		log.Printf("DEBUG: Returning processing response")
		resp, err := h.successResponse(response)
		log.Printf("DEBUG: ===== END handleGenerateWithVisualContext (async) - took %v =====", time.Since(startTime))
		return resp, err
	}
	
	// Build success response
	response := h.buildGenerationResponse("generate_with_visual_context", result)
	resp, err := h.successResponse(response)
	log.Printf("DEBUG: ===== END handleGenerateWithVisualContext (complete) - took %v =====", time.Since(startTime))
	return resp, err
}

// buildGenerationResponse builds a structured response for generation results
func (h *ReplicateImageHandler) buildGenerationResponse(operation string, result *generation.ImageResult) string {
	paths := map[string]string{
		"file_path": result.FilePath,
		"url":       result.URL,
	}
	
	modelInfo := map[string]string{
		"id":   result.Model,
		"name": result.ModelName,
	}
	
	metrics := map[string]interface{}{
		"generation_time": result.Metrics.GenerationTime,
		"file_size":       result.Metrics.FileSize,
	}
	
	return responses.BuildSuccessResponse(operation, result.ID, paths, modelInfo, result.Parameters, metrics, result.PredictionID)
}

// errorResponse builds an error response
func (h *ReplicateImageHandler) errorResponse(operation, code, message string, details map[string]interface{}) (*protocol.CallToolResponse, error) {
	content := responses.BuildErrorResponse(operation, code, message, details)
	
	return &protocol.CallToolResponse{
		Content: []protocol.ToolContent{
			{
				Type: "text",
				Text: content,
			},
		},
	}, nil
}

// successResponse builds a success response
func (h *ReplicateImageHandler) successResponse(content string) (*protocol.CallToolResponse, error) {
	return &protocol.CallToolResponse{
		Content: []protocol.ToolContent{
			{
				Type: "text",
				Text: content,
			},
		},
	}, nil
}