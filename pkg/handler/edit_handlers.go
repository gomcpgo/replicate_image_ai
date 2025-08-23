package handler

import (
	"context"

	"github.com/gomcpgo/mcp/pkg/protocol"
	"github.com/gomcpgo/replicate_image_ai/pkg/editing"
	"github.com/gomcpgo/replicate_image_ai/pkg/responses"
)

// handleEditImage handles the edit_image tool
func (h *ReplicateImageHandler) handleEditImage(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	// Extract and validate parameters
	filePath, ok := args["file_path"].(string)
	if !ok || filePath == "" {
		return h.errorResponse("edit_image", "invalid_parameters", "file_path parameter is required", nil)
	}
	
	prompt, ok := args["prompt"].(string)
	if !ok || prompt == "" {
		return h.errorResponse("edit_image", "invalid_parameters", "prompt parameter is required", nil)
	}
	
	// Build parameters
	params := editing.EditParams{
		ImagePath: filePath,
		Prompt:    prompt,
	}
	
	// Extract optional parameters
	if model, ok := args["model"].(string); ok {
		params.Model = model
	} else {
		params.Model = "pro" // Default to FLUX Kontext Pro
	}
	
	if strength, ok := args["strength"].(float64); ok {
		params.Strength = strength
	} else {
		params.Strength = 0.8 // Default
	}
	
	if guidanceScale, ok := args["guidance_scale"].(float64); ok {
		params.GuidanceScale = guidanceScale
	} else {
		params.GuidanceScale = 7.5 // Default
	}
	
	if seed, ok := args["seed"].(float64); ok {
		params.Seed = int(seed)
	}
	
	if filename, ok := args["filename"].(string); ok {
		params.Filename = filename
	}
	
	// Call core function
	result, err := h.editor.EditImage(ctx, params)
	if err != nil {
		if editErr, ok := err.(editing.EditError); ok {
			return h.errorResponse("edit_image", editErr.Code, editErr.Message, editErr.Details)
		}
		return h.errorResponse("edit_image", "editing_error", err.Error(), nil)
	}
	
	// Build success response
	response := h.buildEditResponse(result)
	return h.successResponse(response)
}

// buildEditResponse builds a structured response for edit results
func (h *ReplicateImageHandler) buildEditResponse(result *editing.EditResult) string {
	paths := map[string]string{
		"input_path": result.InputPath,
		"file_path":  result.OutputPath,
		"url":        result.OutputURL,
	}
	
	modelInfo := map[string]string{
		"id":   result.Model,
		"name": result.ModelName,
	}
	
	parameters := map[string]interface{}{
		"prompt": result.EditPrompt,
	}
	// Add other parameters from result.Parameters if needed
	for k, v := range result.Parameters {
		if k != "image" { // Don't include the data URL
			parameters[k] = v
		}
	}
	
	metrics := map[string]interface{}{
		"processing_time": result.Metrics.ProcessingTime,
		"input_size":      result.Metrics.InputSize,
		"output_size":     result.Metrics.OutputSize,
	}
	
	return responses.BuildSuccessResponse(result.Operation, result.ID, paths, modelInfo, parameters, metrics, result.PredictionID)
}