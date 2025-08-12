package responses

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// BuildSuccessResponse creates a standardized success response
func BuildSuccessResponse(operation string, id string, paths map[string]string, modelInfo map[string]string, params map[string]interface{}, metrics map[string]interface{}, predictionID string) string {
	response := map[string]interface{}{
		"success":    true,
		"operation":  operation,
		"id":         id,
		"paths":      paths,
		"model":      modelInfo,
		"parameters": params,
		"metrics":    metrics,
	}
	
	if predictionID != "" {
		response["prediction_id"] = predictionID
	}
	
	// Add cost estimate based on operation
	response["cost_estimate"] = EstimateCost(operation)
	
	jsonBytes, _ := json.MarshalIndent(response, "", "  ")
	return string(jsonBytes)
}

// BuildErrorResponse creates a standardized error response
func BuildErrorResponse(operation string, errorType string, message string, details map[string]interface{}) string {
	response := map[string]interface{}{
		"success":   false,
		"operation": operation,
		"error": map[string]interface{}{
			"type":       errorType,
			"message":    message,
			"details":    details,
			"suggestion": GetSuggestion(errorType),
		},
	}
	
	jsonBytes, _ := json.MarshalIndent(response, "", "  ")
	return string(jsonBytes)
}

// BuildProcessingResponse creates a response for operations still in progress
func BuildProcessingResponse(operation string, predictionID string, storageID string, estimatedRemaining int) string {
	response := map[string]interface{}{
		"success":       false,
		"operation":     operation,
		"status":        "processing",
		"prediction_id": predictionID,
		"storage_id":    storageID,
		"message":       fmt.Sprintf("Operation still in progress. Use continue_operation with prediction_id='%s' to check status.", predictionID),
	}
	
	if estimatedRemaining > 0 {
		response["estimated_remaining"] = estimatedRemaining
	}
	
	jsonBytes, _ := json.MarshalIndent(response, "", "  ")
	return string(jsonBytes)
}

// BuildSimpleSuccessResponse creates a simple success response with just a message
func BuildSimpleSuccessResponse(operation string, message string, data map[string]interface{}) string {
	response := map[string]interface{}{
		"success":   true,
		"operation": operation,
		"message":   message,
	}
	
	// Merge additional data if provided
	for k, v := range data {
		response[k] = v
	}
	
	jsonBytes, _ := json.MarshalIndent(response, "", "  ")
	return string(jsonBytes)
}

// GetImageDimensions attempts to get basic image dimensions
func GetImageDimensions(filePath string) map[string]int {
	// This is a simplified version - in production you'd use an image library
	// For now, return estimated dimensions based on file size
	info, err := os.Stat(filePath)
	if err != nil {
		return map[string]int{"width": 0, "height": 0}
	}
	
	// Rough estimation - this should be replaced with actual image reading
	size := info.Size()
	if size < 100*1024 { // < 100KB
		return map[string]int{"width": 512, "height": 512}
	} else if size < 500*1024 { // < 500KB
		return map[string]int{"width": 1024, "height": 1024}
	} else {
		return map[string]int{"width": 2048, "height": 2048}
	}
}

// GetFileSize returns the size of a file in bytes
func GetFileSize(filePath string) int64 {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0
	}
	return info.Size()
}

// EstimateCost estimates the cost of an operation in USD
func EstimateCost(operation string) float64 {
	costs := map[string]float64{
		"generate_image":     0.003,
		"enhance_face":       0.005,
		"upscale_image":      0.007,
		"remove_background":  0.004,
		"edit_image":         0.006,
		"restore_photo":      0.005,
		"batch_process":      0.020,
	}
	
	if cost, ok := costs[operation]; ok {
		return cost
	}
	return 0.001 // Default minimal cost
}

// GetSuggestion provides helpful suggestions for different error types
func GetSuggestion(errorType string) string {
	suggestions := map[string]string{
		"file_not_found":     "Please check the file path and ensure the file exists",
		"file_too_large":     "Please compress or resize the image to under 5MB",
		"invalid_format":     "Please provide an image in JPEG, PNG, or WebP format",
		"model_unavailable":  "Try using a different model or wait and retry",
		"rate_limit":         "Wait a few seconds before retrying",
		"invalid_parameters": "Check the parameter values and ensure they meet the requirements",
		"timeout":            "The operation is taking longer than expected. Use continue_operation to check status",
		"api_error":          "Check your API key and network connection",
		"permission_denied":  "Ensure you have the necessary permissions for this operation",
	}
	
	if suggestion, ok := suggestions[errorType]; ok {
		return suggestion
	}
	return "Please check your input and try again"
}

// ExtractModelName extracts a friendly model name from the model ID
func ExtractModelName(modelID string) string {
	modelNames := map[string]string{
		"black-forest-labs/flux-schnell": "Flux Schnell",
		"black-forest-labs/flux-dev":     "Flux Dev",
		"black-forest-labs/flux-pro":     "Flux Pro",
		"stability-ai/sdxl":               "SDXL",
		"bytedance/sdxl-lightning-4step": "SDXL Lightning",
		"bytedance/seedream-3":            "Seedream 3",
		"ideogram-ai/ideogram-v3-turbo":  "Ideogram Turbo",
		"recraft-ai/recraft-v3":           "Recraft",
		"recraft-ai/recraft-v3-svg":       "Recraft SVG",
		"tencentarc/gfpgan":               "GFPGAN",
		"sczhou/codeformer":               "CodeFormer",
		"nightmareai/real-esrgan":         "RealESRGAN",
		"philz1337x/clarity-upscaler":    "Clarity Upscaler",
		"lucataco/remove-bg":              "RemoveBG",
		"cjwbw/rembg":                     "Rembg",
		"lucataco/dis-background-removal": "DIS Background Removal",
		"stability-ai/stable-diffusion-inpainting": "SD Inpainting",
		"microsoft/bringing-old-photos-back-to-life": "Old Photo Restoration",
	}
	
	// Remove version hash if present
	if idx := filepath.Base(modelID); idx != "" {
		modelID = idx
	}
	
	// Check for known model names
	for prefix, name := range modelNames {
		if len(modelID) >= len(prefix) && modelID[:len(prefix)] == prefix {
			return name
		}
	}
	
	// Return the base name if not found
	return filepath.Base(modelID)
}