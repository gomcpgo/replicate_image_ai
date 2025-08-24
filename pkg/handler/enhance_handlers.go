package handler

import (
	"context"

	"github.com/gomcpgo/mcp/pkg/protocol"
	"github.com/gomcpgo/replicate_image_ai/pkg/enhancement"
	"github.com/gomcpgo/replicate_image_ai/pkg/responses"
)

// handleRemoveBackground handles the remove_background tool
func (h *ReplicateImageHandler) handleRemoveBackground(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	// Extract and validate parameters
	filePath, ok := args["file_path"].(string)
	if !ok || filePath == "" {
		return h.errorResponse("remove_background", "invalid_parameters", "file_path parameter is required", nil)
	}
	
	// Build parameters
	params := enhancement.RemoveBackgroundParams{
		ImagePath: filePath,
	}
	
	// Extract optional parameters
	if model, ok := args["model"].(string); ok {
		params.Model = model
	} else {
		params.Model = "remove-bg" // Default
	}
	
	if filename, ok := args["filename"].(string); ok {
		params.Filename = filename
	}
	
	// Call core function
	result, err := h.enhancer.RemoveBackground(ctx, params)
	if err != nil {
		if enhErr, ok := err.(enhancement.EnhancementError); ok {
			return h.errorResponse("remove_background", enhErr.Code, enhErr.Message, enhErr.Details)
		}
		return h.errorResponse("remove_background", "processing_error", err.Error(), nil)
	}
	
	// Build success response
	response := h.buildEnhancementResponse(result)
	return h.successResponse(response)
}

// handleUpscaleImage handles the upscale_image tool
func (h *ReplicateImageHandler) handleUpscaleImage(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	// Extract and validate parameters
	filePath, ok := args["file_path"].(string)
	if !ok || filePath == "" {
		return h.errorResponse("upscale_image", "invalid_parameters", "file_path parameter is required", nil)
	}
	
	// Build parameters
	params := enhancement.UpscaleParams{
		ImagePath: filePath,
	}
	
	// Extract optional parameters
	if scale, ok := args["scale"].(float64); ok {
		params.Scale = int(scale)
	} else {
		params.Scale = 4 // Default
	}
	
	if model, ok := args["model"].(string); ok {
		params.Model = model
	} else {
		params.Model = "realesrgan" // Default
	}
	
	if faceEnhance, ok := args["face_enhance"].(bool); ok {
		params.FaceEnhance = faceEnhance
	}
	
	if filename, ok := args["filename"].(string); ok {
		params.Filename = filename
	}
	
	// Call core function
	result, err := h.enhancer.UpscaleImage(ctx, params)
	if err != nil {
		if enhErr, ok := err.(enhancement.EnhancementError); ok {
			return h.errorResponse("upscale_image", enhErr.Code, enhErr.Message, enhErr.Details)
		}
		return h.errorResponse("upscale_image", "processing_error", err.Error(), nil)
	}
	
	// Build success response
	response := h.buildEnhancementResponse(result)
	return h.successResponse(response)
}

// handleEnhanceFace handles the enhance_face tool
func (h *ReplicateImageHandler) handleEnhanceFace(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	// Extract and validate parameters
	filePath, ok := args["file_path"].(string)
	if !ok || filePath == "" {
		return h.errorResponse("enhance_face", "invalid_parameters", "file_path parameter is required", nil)
	}
	
	// Build parameters
	params := enhancement.EnhanceFaceParams{
		ImagePath: filePath,
	}
	
	// Extract optional parameters
	if model, ok := args["model"].(string); ok {
		params.Model = model
	} else {
		params.Model = "gfpgan" // Default
	}
	
	if fidelity, ok := args["fidelity"].(float64); ok {
		params.Fidelity = fidelity
	} else {
		params.Fidelity = 0.5 // Default
	}
	
	if onlyCenter, ok := args["only_center"].(bool); ok {
		params.OnlyCenter = onlyCenter
	}
	
	if backgroundEnhance, ok := args["background_enhance"].(bool); ok {
		params.BackgroundEnhance = backgroundEnhance
	}
	
	if filename, ok := args["filename"].(string); ok {
		params.Filename = filename
	}
	
	// Call core function
	result, err := h.enhancer.EnhanceFace(ctx, params)
	if err != nil {
		if enhErr, ok := err.(enhancement.EnhancementError); ok {
			return h.errorResponse("enhance_face", enhErr.Code, enhErr.Message, enhErr.Details)
		}
		return h.errorResponse("enhance_face", "processing_error", err.Error(), nil)
	}
	
	// Build success response
	response := h.buildEnhancementResponse(result)
	return h.successResponse(response)
}

// handleRestorePhoto handles the restore_photo tool
func (h *ReplicateImageHandler) handleRestorePhoto(ctx context.Context, args map[string]interface{}) (*protocol.CallToolResponse, error) {
	// Extract and validate parameters
	filePath, ok := args["file_path"].(string)
	if !ok || filePath == "" {
		return h.errorResponse("restore_photo", "invalid_parameters", "file_path parameter is required", nil)
	}
	
	// Build parameters
	params := enhancement.RestorePhotoParams{
		ImagePath: filePath,
	}
	
	// Extract optional parameters
	if model, ok := args["model"].(string); ok {
		params.Model = model
	} else {
		params.Model = "bopbtl" // Default
	}
	
	if faceEnhance, ok := args["face_enhance"].(bool); ok {
		params.FaceEnhance = faceEnhance
	} else {
		params.FaceEnhance = true // Default
	}
	
	if scratchRemoval, ok := args["scratch_removal"].(bool); ok {
		params.ScratchRemoval = scratchRemoval
	} else {
		params.ScratchRemoval = true // Default
	}
	
	if colorize, ok := args["colorize"].(bool); ok {
		params.Colorize = colorize
	}
	
	if filename, ok := args["filename"].(string); ok {
		params.Filename = filename
	}
	
	// Call core function
	result, err := h.enhancer.RestorePhoto(ctx, params)
	if err != nil {
		if enhErr, ok := err.(enhancement.EnhancementError); ok {
			return h.errorResponse("restore_photo", enhErr.Code, enhErr.Message, enhErr.Details)
		}
		return h.errorResponse("restore_photo", "processing_error", err.Error(), nil)
	}
	
	// Build success response
	response := h.buildEnhancementResponse(result)
	return h.successResponse(response)
}

// buildEnhancementResponse builds a structured response for enhancement results
func (h *ReplicateImageHandler) buildEnhancementResponse(result *enhancement.EnhancementResult) string {
	paths := map[string]string{
		"input_path": result.InputPath,
		"file_path":  result.OutputPath,
		"url":        result.OutputURL,
	}
	
	modelInfo := map[string]string{
		"id":   result.Model,
		"name": result.ModelName,
	}
	
	metrics := map[string]interface{}{
		"processing_time": result.Metrics.ProcessingTime,
		"input_size":      result.Metrics.InputSize,
		"output_size":     result.Metrics.OutputSize,
	}
	
	if result.Metrics.ScaleFactor > 0 {
		metrics["scale_factor"] = result.Metrics.ScaleFactor
	}
	
	return responses.BuildSuccessResponse(result.Operation, result.ID, paths, modelInfo, result.Parameters, metrics, result.PredictionID)
}