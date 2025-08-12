package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gomcpgo/replicate_image_ai/pkg/responses"
	"github.com/gomcpgo/replicate_image_ai/pkg/storage"
	"github.com/gomcpgo/replicate_image_ai/pkg/types"
)

// removeBackground handles the remove_background tool
func (s *ReplicateImageMCPServer) removeBackground(ctx context.Context, params map[string]interface{}) (string, error) {
	// Parse parameters
	filePath, ok := params["file_path"].(string)
	if !ok || filePath == "" {
		return responses.BuildErrorResponse("remove_background", "invalid_parameters", 
			"file_path parameter is required", nil), nil
	}

	// Get model selection
	model := "remove-bg"
	if m, ok := params["model"].(string); ok && m != "" {
		model = m
	}

	// Map model name to Replicate model ID
	var modelID string
	switch model {
	case "remove-bg":
		modelID = types.ModelRemoveBG
	case "rembg":
		modelID = types.ModelRembg
	case "dis":
		modelID = types.ModelDISBGRemoval
	default:
		modelID = types.ModelRemoveBG
	}

	// Convert image to base64 data URL
	dataURL, err := storage.ImageToBase64(filePath)
	if err != nil {
		return responses.BuildErrorResponse("remove_background", "file_error", 
			fmt.Sprintf("Failed to load image: %v", err), map[string]interface{}{
				"file_path": filePath,
			}), nil
	}

	// Build input parameters
	input := map[string]interface{}{
		"image": dataURL,
	}

	// Generate unique ID for this operation
	id, err := s.storage.GenerateID()
	if err != nil {
		return responses.BuildErrorResponse("remove_background", "storage_error",
			fmt.Sprintf("Failed to generate ID: %v", err), nil), nil
	}

	// Save original image
	originalPath := s.storage.GetImagePath(id, "original"+filepath.Ext(filePath))
	if err := copyFile(filePath, originalPath); err != nil {
		log.Printf("Failed to copy original: %v", err)
	}

	// Create prediction
	startTime := time.Now()
	prediction, err := s.client.CreatePrediction(ctx, modelID, input)
	if err != nil {
		return responses.BuildErrorResponse("remove_background", "api_error",
			fmt.Sprintf("Failed to create prediction: %v", err), nil), nil
	}

	// Wait for completion (up to 30 seconds)
	result, waitErr := s.client.WaitForCompletion(ctx, prediction.ID, s.config.OperationTimeout)
	
	// Check if completed successfully
	if waitErr == nil && result.Status == types.StatusSucceeded {
		// Extract output URL
		outputURL := extractOutputURL(result.Output)
		if outputURL == "" {
			return responses.BuildErrorResponse("remove_background", "processing_error",
				"No output URL in prediction result", nil), nil
		}

		// Determine output format
		outputFormat := "png"
		if of, ok := params["output_format"].(string); ok && of != "" {
			outputFormat = of
		}
		
		// Determine filename
		filename := fmt.Sprintf("no_bg.%s", outputFormat)
		if fn, ok := params["filename"].(string); ok && fn != "" {
			filename = fn
		}

		// Save the processed image
		processedPath, err := s.storage.SaveImage(id, outputURL, filename)
		if err != nil {
			return responses.BuildErrorResponse("remove_background", "storage_error",
				fmt.Sprintf("Failed to save image: %v", err), nil), nil
		}

		// Save metadata
		metadata := &types.ImageMetadata{
			Version:   "1.0",
			ID:        id,
			Operation: "remove_background",
			Timestamp: time.Now(),
			Model:     modelID,
			Parameters: map[string]interface{}{
				"model":         model,
				"output_format": outputFormat,
			},
			Result: &types.OperationResult{
				Filename:       filename,
				GenerationTime: time.Since(startTime).Seconds(),
				PredictionID:   prediction.ID,
			},
		}

		if err := s.storage.SaveMetadata(id, metadata); err != nil {
			log.Printf("Failed to save metadata: %v", err)
		}

		// Build success response
		return responses.BuildSuccessResponse(
			"remove_background",
			id,
			map[string]string{
				"original":  originalPath,
				"processed": processedPath,
			},
			map[string]string{
				"id":   modelID,
				"name": responses.ExtractModelName(modelID),
			},
			map[string]interface{}{
				"model":         model,
				"output_format": outputFormat,
			},
			map[string]interface{}{
				"processing_time": time.Since(startTime).Seconds(),
				"file_size_original": responses.GetFileSize(filePath),
				"file_size_processed": responses.GetFileSize(processedPath),
			},
			prediction.ID,
		), nil
	}

	// If timed out or still processing
	if result != nil && (result.Status == types.StatusProcessing || result.Status == types.StatusStarting) {
		return responses.BuildProcessingResponse("remove_background", prediction.ID, id, 15), nil
	}

	// If failed
	if waitErr != nil {
		return responses.BuildErrorResponse("remove_background", "processing_failed",
			fmt.Sprintf("Processing failed: %v", waitErr), nil), nil
	}

	return responses.BuildErrorResponse("remove_background", "unknown_error",
		fmt.Sprintf("Unexpected status: %s", result.Status), nil), nil
}

// upscaleImage handles the upscale_image tool
func (s *ReplicateImageMCPServer) upscaleImage(ctx context.Context, params map[string]interface{}) (string, error) {
	// Parse parameters
	filePath, ok := params["file_path"].(string)
	if !ok || filePath == "" {
		return responses.BuildErrorResponse("upscale_image", "invalid_parameters",
			"file_path parameter is required", nil), nil
	}

	// Get scale factor
	scale := 2
	if sc, ok := params["scale"].(float64); ok {
		scale = int(sc)
	}

	// Get model selection
	model := "realesrgan"
	if m, ok := params["model"].(string); ok && m != "" {
		model = m
	}

	// Map model name to Replicate model ID
	var modelID string
	switch model {
	case "realesrgan":
		modelID = types.ModelRealESRGAN
	case "clarity":
		modelID = types.ModelClarityUpscaler
	default:
		modelID = types.ModelRealESRGAN
	}

	// Convert image to base64 data URL
	dataURL, err := storage.ImageToBase64(filePath)
	if err != nil {
		return responses.BuildErrorResponse("upscale_image", "file_error",
			fmt.Sprintf("Failed to load image: %v", err), map[string]interface{}{
				"file_path": filePath,
			}), nil
	}

	// Build input parameters
	input := map[string]interface{}{
		"image": dataURL,
		"scale": scale,
	}

	// Add model-specific parameters
	if model == "realesrgan" {
		if faceEnhance, ok := params["face_enhance"].(bool); ok {
			input["face_enhance"] = faceEnhance
		}
	} else if model == "clarity" {
		// Clarity has many parameters, use sensible defaults
		input["prompt"] = "masterpiece, best quality, highres"
		input["negative_prompt"] = "worst quality, low quality, normal quality"
		input["dynamic"] = 6
		input["creativity"] = 0.35
		input["resemblance"] = 0.6
		input["scale_factor"] = scale
		input["num_inference_steps"] = 18
	}

	// Generate unique ID
	id, err := s.storage.GenerateID()
	if err != nil {
		return responses.BuildErrorResponse("upscale_image", "storage_error",
			fmt.Sprintf("Failed to generate ID: %v", err), nil), nil
	}

	// Save original
	originalPath := s.storage.GetImagePath(id, "original"+filepath.Ext(filePath))
	if err := copyFile(filePath, originalPath); err != nil {
		log.Printf("Failed to copy original: %v", err)
	}

	// Create prediction
	startTime := time.Now()
	prediction, err := s.client.CreatePrediction(ctx, modelID, input)
	if err != nil {
		return responses.BuildErrorResponse("upscale_image", "api_error",
			fmt.Sprintf("Failed to create prediction: %v", err), nil), nil
	}

	// Wait for completion
	result, waitErr := s.client.WaitForCompletion(ctx, prediction.ID, s.config.OperationTimeout)
	
	if waitErr == nil && result.Status == types.StatusSucceeded {
		outputURL := extractOutputURL(result.Output)
		if outputURL == "" {
			return responses.BuildErrorResponse("upscale_image", "processing_error",
				"No output URL in prediction result", nil), nil
		}

		// Determine filename
		filename := fmt.Sprintf("upscaled_%dx.png", scale)
		if fn, ok := params["filename"].(string); ok && fn != "" {
			filename = fn
		}

		// Save processed image
		processedPath, err := s.storage.SaveImage(id, outputURL, filename)
		if err != nil {
			return responses.BuildErrorResponse("upscale_image", "storage_error",
				fmt.Sprintf("Failed to save image: %v", err), nil), nil
		}

		// Save metadata
		metadata := &types.ImageMetadata{
			Version:   "1.0",
			ID:        id,
			Operation: "upscale_image",
			Timestamp: time.Now(),
			Model:     modelID,
			Parameters: map[string]interface{}{
				"model": model,
				"scale": scale,
			},
			Result: &types.OperationResult{
				Filename:       filename,
				GenerationTime: time.Since(startTime).Seconds(),
				PredictionID:   prediction.ID,
			},
		}

		if err := s.storage.SaveMetadata(id, metadata); err != nil {
			log.Printf("Failed to save metadata: %v", err)
		}

		// Build success response
		originalDims := responses.GetImageDimensions(filePath)
		processedDims := map[string]int{
			"width":  originalDims["width"] * scale,
			"height": originalDims["height"] * scale,
		}

		return responses.BuildSuccessResponse(
			"upscale_image",
			id,
			map[string]string{
				"original":  originalPath,
				"processed": processedPath,
			},
			map[string]string{
				"id":   modelID,
				"name": responses.ExtractModelName(modelID),
			},
			map[string]interface{}{
				"model": model,
				"scale": scale,
			},
			map[string]interface{}{
				"processing_time":      time.Since(startTime).Seconds(),
				"file_size_original":   responses.GetFileSize(filePath),
				"file_size_processed":  responses.GetFileSize(processedPath),
				"dimensions_original":  originalDims,
				"dimensions_processed": processedDims,
			},
			prediction.ID,
		), nil
	}

	if result != nil && (result.Status == types.StatusProcessing || result.Status == types.StatusStarting) {
		return responses.BuildProcessingResponse("upscale_image", prediction.ID, id, 30), nil
	}

	if waitErr != nil {
		return responses.BuildErrorResponse("upscale_image", "processing_failed",
			fmt.Sprintf("Processing failed: %v", waitErr), nil), nil
	}

	return responses.BuildErrorResponse("upscale_image", "unknown_error",
		fmt.Sprintf("Unexpected status: %s", result.Status), nil), nil
}

// enhanceFace handles the enhance_face tool
func (s *ReplicateImageMCPServer) enhanceFace(ctx context.Context, params map[string]interface{}) (string, error) {
	// Parse parameters
	filePath, ok := params["file_path"].(string)
	if !ok || filePath == "" {
		return responses.BuildErrorResponse("enhance_face", "invalid_parameters",
			"file_path parameter is required", nil), nil
	}

	// Get model selection
	model := "gfpgan"
	if m, ok := params["enhancement_model"].(string); ok && m != "" {
		model = m
	}

	// Get upscale factor
	upscale := 2
	if u, ok := params["upscale"].(float64); ok {
		upscale = int(u)
	}

	// Map model to ID
	var modelID string
	switch model {
	case "gfpgan":
		modelID = types.ModelGFPGAN
	case "codeformer":
		modelID = types.ModelCodeFormer
	default:
		modelID = types.ModelGFPGAN
	}

	// Convert image to base64
	dataURL, err := storage.ImageToBase64(filePath)
	if err != nil {
		return responses.BuildErrorResponse("enhance_face", "file_error",
			fmt.Sprintf("Failed to load image: %v", err), map[string]interface{}{
				"file_path": filePath,
			}), nil
	}

	// Build input parameters
	input := make(map[string]interface{})
	
	if model == "gfpgan" {
		input["img"] = dataURL  // GFPGAN uses "img"
		input["scale"] = upscale
		input["version"] = "v1.4"
	} else if model == "codeformer" {
		input["image"] = dataURL  // CodeFormer uses "image"
		input["upscale"] = upscale
		
		// Get fidelity
		fidelity := 0.5
		if f, ok := params["fidelity"].(float64); ok {
			fidelity = f
		}
		input["codeformer_fidelity"] = fidelity
		
		// Background enhance
		if bg, ok := params["background_enhance"].(bool); ok {
			input["background_enhance"] = bg
			input["face_upsample"] = true
		}
	}

	// Generate unique ID
	id, err := s.storage.GenerateID()
	if err != nil {
		return responses.BuildErrorResponse("enhance_face", "storage_error",
			fmt.Sprintf("Failed to generate ID: %v", err), nil), nil
	}

	// Save original
	originalPath := s.storage.GetImagePath(id, "original"+filepath.Ext(filePath))
	if err := copyFile(filePath, originalPath); err != nil {
		log.Printf("Failed to copy original: %v", err)
	}

	// Create prediction
	startTime := time.Now()
	prediction, err := s.client.CreatePrediction(ctx, modelID, input)
	if err != nil {
		return responses.BuildErrorResponse("enhance_face", "api_error",
			fmt.Sprintf("Failed to create prediction: %v", err), nil), nil
	}

	// Wait for completion
	result, waitErr := s.client.WaitForCompletion(ctx, prediction.ID, s.config.OperationTimeout)
	
	if waitErr == nil && result.Status == types.StatusSucceeded {
		outputURL := extractOutputURL(result.Output)
		if outputURL == "" {
			return responses.BuildErrorResponse("enhance_face", "processing_error",
				"No output URL in prediction result", nil), nil
		}

		// Determine filename
		filename := "enhanced.png"
		if fn, ok := params["filename"].(string); ok && fn != "" {
			filename = fn
		}

		// Save processed image
		processedPath, err := s.storage.SaveImage(id, outputURL, filename)
		if err != nil {
			return responses.BuildErrorResponse("enhance_face", "storage_error",
				fmt.Sprintf("Failed to save image: %v", err), nil), nil
		}

		// Save metadata
		metadata := &types.ImageMetadata{
			Version:   "1.0",
			ID:        id,
			Operation: "enhance_face",
			Timestamp: time.Now(),
			Model:     modelID,
			Parameters: map[string]interface{}{
				"enhancement_model": model,
				"upscale":          upscale,
			},
			Result: &types.OperationResult{
				Filename:       filename,
				GenerationTime: time.Since(startTime).Seconds(),
				PredictionID:   prediction.ID,
			},
		}

		if err := s.storage.SaveMetadata(id, metadata); err != nil {
			log.Printf("Failed to save metadata: %v", err)
		}

		// Build success response
		return responses.BuildSuccessResponse(
			"enhance_face",
			id,
			map[string]string{
				"original":  originalPath,
				"processed": processedPath,
			},
			map[string]string{
				"id":   modelID,
				"name": responses.ExtractModelName(modelID),
			},
			map[string]interface{}{
				"enhancement_model": model,
				"upscale":          upscale,
			},
			map[string]interface{}{
				"processing_time":     time.Since(startTime).Seconds(),
				"file_size_original":  responses.GetFileSize(filePath),
				"file_size_processed": responses.GetFileSize(processedPath),
			},
			prediction.ID,
		), nil
	}

	if result != nil && (result.Status == types.StatusProcessing || result.Status == types.StatusStarting) {
		return responses.BuildProcessingResponse("enhance_face", prediction.ID, id, 20), nil
	}

	if waitErr != nil {
		return responses.BuildErrorResponse("enhance_face", "processing_failed",
			fmt.Sprintf("Processing failed: %v", waitErr), nil), nil
	}

	return responses.BuildErrorResponse("enhance_face", "unknown_error",
		fmt.Sprintf("Unexpected status: %s", result.Status), nil), nil
}

// restorePhoto handles the restore_photo tool
func (s *ReplicateImageMCPServer) restorePhoto(ctx context.Context, params map[string]interface{}) (string, error) {
	// Parse parameters
	filePath, ok := params["file_path"].(string)
	if !ok || filePath == "" {
		return responses.BuildErrorResponse("restore_photo", "invalid_parameters",
			"file_path parameter is required", nil), nil
	}

	// Get parameters
	withScratch := true
	if ws, ok := params["with_scratch"].(bool); ok {
		withScratch = ws
	}

	highResolution := false
	if hr, ok := params["high_resolution"].(bool); ok {
		highResolution = hr
	}

	// Convert image to base64
	dataURL, err := storage.ImageToBase64(filePath)
	if err != nil {
		return responses.BuildErrorResponse("restore_photo", "file_error",
			fmt.Sprintf("Failed to load image: %v", err), map[string]interface{}{
				"file_path": filePath,
			}), nil
	}

	// Build input parameters
	input := map[string]interface{}{
		"image":        dataURL,
		"with_scratch": withScratch,
		"HR":          highResolution,
	}

	// Generate unique ID
	id, err := s.storage.GenerateID()
	if err != nil {
		return responses.BuildErrorResponse("restore_photo", "storage_error",
			fmt.Sprintf("Failed to generate ID: %v", err), nil), nil
	}

	// Save original
	originalPath := s.storage.GetImagePath(id, "original"+filepath.Ext(filePath))
	if err := copyFile(filePath, originalPath); err != nil {
		log.Printf("Failed to copy original: %v", err)
	}

	// Create prediction
	startTime := time.Now()
	modelID := types.ModelOldPhotoRestore
	prediction, err := s.client.CreatePrediction(ctx, modelID, input)
	if err != nil {
		return responses.BuildErrorResponse("restore_photo", "api_error",
			fmt.Sprintf("Failed to create prediction: %v", err), nil), nil
	}

	// Wait for completion
	result, waitErr := s.client.WaitForCompletion(ctx, prediction.ID, s.config.OperationTimeout)
	
	if waitErr == nil && result.Status == types.StatusSucceeded {
		outputURL := extractOutputURL(result.Output)
		if outputURL == "" {
			return responses.BuildErrorResponse("restore_photo", "processing_error",
				"No output URL in prediction result", nil), nil
		}

		// Determine filename
		filename := "restored.png"
		if fn, ok := params["filename"].(string); ok && fn != "" {
			filename = fn
		}

		// Save processed image
		processedPath, err := s.storage.SaveImage(id, outputURL, filename)
		if err != nil {
			return responses.BuildErrorResponse("restore_photo", "storage_error",
				fmt.Sprintf("Failed to save image: %v", err), nil), nil
		}

		// Save metadata
		metadata := &types.ImageMetadata{
			Version:   "1.0",
			ID:        id,
			Operation: "restore_photo",
			Timestamp: time.Now(),
			Model:     modelID,
			Parameters: map[string]interface{}{
				"with_scratch":     withScratch,
				"high_resolution": highResolution,
			},
			Result: &types.OperationResult{
				Filename:       filename,
				GenerationTime: time.Since(startTime).Seconds(),
				PredictionID:   prediction.ID,
			},
		}

		if err := s.storage.SaveMetadata(id, metadata); err != nil {
			log.Printf("Failed to save metadata: %v", err)
		}

		// Build success response
		return responses.BuildSuccessResponse(
			"restore_photo",
			id,
			map[string]string{
				"original":  originalPath,
				"processed": processedPath,
			},
			map[string]string{
				"id":   modelID,
				"name": "Old Photo Restoration",
			},
			map[string]interface{}{
				"with_scratch":     withScratch,
				"high_resolution": highResolution,
			},
			map[string]interface{}{
				"processing_time":     time.Since(startTime).Seconds(),
				"file_size_original":  responses.GetFileSize(filePath),
				"file_size_processed": responses.GetFileSize(processedPath),
			},
			prediction.ID,
		), nil
	}

	if result != nil && (result.Status == types.StatusProcessing || result.Status == types.StatusStarting) {
		return responses.BuildProcessingResponse("restore_photo", prediction.ID, id, 25), nil
	}

	if waitErr != nil {
		return responses.BuildErrorResponse("restore_photo", "processing_failed",
			fmt.Sprintf("Processing failed: %v", waitErr), nil), nil
	}

	return responses.BuildErrorResponse("restore_photo", "unknown_error",
		fmt.Sprintf("Unexpected status: %s", result.Status), nil), nil
}

// editImage handles the edit_image tool for inpainting and masked editing
func (s *ReplicateImageMCPServer) editImage(ctx context.Context, params map[string]interface{}) (string, error) {
	// Parse parameters
	filePath, ok := params["file_path"].(string)
	if !ok || filePath == "" {
		return responses.BuildErrorResponse("edit_image", "invalid_parameters",
			"file_path parameter is required", nil), nil
	}

	editPrompt, ok := params["edit_prompt"].(string)
	if !ok || editPrompt == "" {
		return responses.BuildErrorResponse("edit_image", "invalid_parameters",
			"edit_prompt parameter is required", nil), nil
	}

	// Convert image to base64 data URL
	imageDataURL, err := storage.ImageToBase64(filePath)
	if err != nil {
		return responses.BuildErrorResponse("edit_image", "file_error",
			fmt.Sprintf("Failed to load image: %v", err), map[string]interface{}{
				"file_path": filePath,
			}), nil
	}

	// Prepare input
	input := map[string]interface{}{
		"image":  imageDataURL,
		"prompt": editPrompt,
	}

	// Handle mask path if provided
	if maskPath, ok := params["mask_path"].(string); ok && maskPath != "" {
		maskDataURL, err := storage.ImageToBase64(maskPath)
		if err != nil {
			return responses.BuildErrorResponse("edit_image", "file_error",
				fmt.Sprintf("Failed to load mask: %v", err), map[string]interface{}{
					"mask_path": maskPath,
				}), nil
		}
		input["mask"] = maskDataURL
	} else if selectionPrompt, ok := params["selection_prompt"].(string); ok && selectionPrompt != "" {
		// Some models support automatic mask generation from text
		input["mask_prompt"] = selectionPrompt
	}

	// Add optional parameters
	if strength, ok := params["strength"].(float64); ok {
		input["strength"] = strength
	} else {
		input["strength"] = 0.8 // Default strength
	}

	if guidanceScale, ok := params["guidance_scale"].(float64); ok {
		input["guidance_scale"] = guidanceScale
	} else {
		input["guidance_scale"] = 7.5 // Default guidance scale
	}

	// Add negative prompt if needed
	if negativePrompt, ok := params["negative_prompt"].(string); ok && negativePrompt != "" {
		input["negative_prompt"] = negativePrompt
	}

	// Generate unique ID
	id, err := s.storage.GenerateID()
	if err != nil {
		return responses.BuildErrorResponse("edit_image", "storage_error",
			fmt.Sprintf("Failed to generate ID: %v", err), nil), nil
	}

	// Save original
	originalPath := s.storage.GetImagePath(id, "original"+filepath.Ext(filePath))
	if err := copyFile(filePath, originalPath); err != nil {
		log.Printf("Failed to copy original: %v", err)
	}

	// Save mask if provided
	var maskSavedPath string
	if maskPath, ok := params["mask_path"].(string); ok && maskPath != "" {
		maskSavedPath = s.storage.GetImagePath(id, "mask"+filepath.Ext(maskPath))
		if err := copyFile(maskPath, maskSavedPath); err != nil {
			log.Printf("Failed to copy mask: %v", err)
		}
	}

	// Create prediction
	startTime := time.Now()
	modelID := types.ModelInpainting
	prediction, err := s.client.CreatePrediction(ctx, modelID, input)
	if err != nil {
		return responses.BuildErrorResponse("edit_image", "api_error",
			fmt.Sprintf("Failed to create prediction: %v", err), nil), nil
	}

	// Wait for completion
	result, waitErr := s.client.WaitForCompletion(ctx, prediction.ID, s.config.OperationTimeout)

	if waitErr == nil && result.Status == types.StatusSucceeded {
		outputURL := extractOutputURL(result.Output)
		if outputURL == "" {
			return responses.BuildErrorResponse("edit_image", "processing_error",
				"No output URL in prediction result", nil), nil
		}

		// Determine filename
		filename := "edited.png"
		if fn, ok := params["filename"].(string); ok && fn != "" {
			filename = fn
		}

		// Save processed image
		processedPath, err := s.storage.SaveImage(id, outputURL, filename)
		if err != nil {
			return responses.BuildErrorResponse("edit_image", "storage_error",
				fmt.Sprintf("Failed to save image: %v", err), nil), nil
		}

		// Save metadata
		metadata := &types.ImageMetadata{
			Version:   "1.0",
			ID:        id,
			Operation: "edit_image",
			Timestamp: time.Now(),
			Model:     modelID,
			Parameters: map[string]interface{}{
				"edit_prompt":    editPrompt,
				"strength":       input["strength"],
				"guidance_scale": input["guidance_scale"],
				"has_mask":       maskSavedPath != "",
			},
			Result: &types.OperationResult{
				Filename:       filename,
				GenerationTime: time.Since(startTime).Seconds(),
				PredictionID:   prediction.ID,
			},
		}

		if err := s.storage.SaveMetadata(id, metadata); err != nil {
			log.Printf("Failed to save metadata: %v", err)
		}

		// Prepare paths
		paths := map[string]string{
			"original":  originalPath,
			"processed": processedPath,
		}
		if maskSavedPath != "" {
			paths["mask"] = maskSavedPath
		}

		// Build success response
		return responses.BuildSuccessResponse(
			"edit_image",
			id,
			paths,
			map[string]string{
				"id":   modelID,
				"name": "Stable Diffusion Inpainting",
			},
			map[string]interface{}{
				"edit_prompt":    editPrompt,
				"strength":       input["strength"],
				"guidance_scale": input["guidance_scale"],
				"has_mask":       maskSavedPath != "",
			},
			map[string]interface{}{
				"processing_time":     time.Since(startTime).Seconds(),
				"file_size_original":  responses.GetFileSize(filePath),
				"file_size_processed": responses.GetFileSize(processedPath),
			},
			prediction.ID,
		), nil
	}

	if result != nil && (result.Status == types.StatusProcessing || result.Status == types.StatusStarting) {
		return responses.BuildProcessingResponse("edit_image", prediction.ID, id, 25), nil
	}

	if waitErr != nil {
		return responses.BuildErrorResponse("edit_image", "processing_failed",
			fmt.Sprintf("Processing failed: %v", waitErr), nil), nil
	}

	return responses.BuildErrorResponse("edit_image", "unknown_error",
		fmt.Sprintf("Unexpected status: %s", result.Status), nil), nil
}

// Helper function to extract output URL from various response formats
func extractOutputURL(output interface{}) string {
	switch v := output.(type) {
	case string:
		return v
	case []interface{}:
		if len(v) > 0 {
			if url, ok := v[0].(string); ok {
				return url
			}
		}
	case map[string]interface{}:
		if url, ok := v["url"].(string); ok {
			return url
		}
		if images, ok := v["images"].([]interface{}); ok && len(images) > 0 {
			if url, ok := images[0].(string); ok {
				return url
			}
		}
	}
	return ""
}

// Helper function to copy a file
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}