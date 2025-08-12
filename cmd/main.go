package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gomcpgo/mcp/pkg/handler"
	"github.com/gomcpgo/mcp/pkg/protocol"
	"github.com/gomcpgo/mcp/pkg/server"
	"github.com/gomcpgo/replicate_image_ai/pkg/client"
	"github.com/gomcpgo/replicate_image_ai/pkg/config"
	"github.com/gomcpgo/replicate_image_ai/pkg/storage"
	"github.com/gomcpgo/replicate_image_ai/pkg/types"
)

// Version information (set by build script)
var (
	Version   = "1.0.0"
	BuildTime = "unknown"
)

// ReplicateImageMCPServer implements the MCP server for Replicate Image AI
type ReplicateImageMCPServer struct {
	config    *config.Config
	client    *client.ReplicateClient
	storage   *storage.Storage
}

// NewReplicateImageMCPServer creates a new Replicate Image AI MCP server
func NewReplicateImageMCPServer() (*ReplicateImageMCPServer, error) {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &ReplicateImageMCPServer{
		config:  cfg,
		client:  client.NewReplicateClient(cfg.ReplicateAPIToken),
		storage: storage.NewStorage(cfg.ReplicateImagesRoot),
	}, nil
}

// ListTools returns the available tools
func (s *ReplicateImageMCPServer) ListTools(ctx context.Context) (*protocol.ListToolsResponse, error) {
	tools := []protocol.Tool{
		{
			Name:        "generate_image",
			Description: "Generate an AI image from a text prompt. Waits up to 30 seconds for completion. If generation takes longer, returns a prediction_id to check status with continue_operation.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"prompt": {
						"type": "string",
						"description": "Text description of the desired image"
					},
					"model": {
						"type": "string",
						"description": "Model to use: flux-schnell (default, fast), flux-pro (best quality), flux-dev, seedream-3, sdxl, ideogram-turbo (for text in images)",
						"enum": ["flux-schnell", "flux-pro", "flux-dev", "seedream-3", "sdxl", "ideogram-turbo"],
						"default": "flux-schnell"
					},
					"width": {
						"type": "integer",
						"description": "Image width in pixels (default: 1024)",
						"default": 1024
					},
					"height": {
						"type": "integer",
						"description": "Image height in pixels (default: 1024)",
						"default": 1024
					},
					"filename": {
						"type": "string",
						"description": "Optional filename for the generated image"
					},
					"seed": {
						"type": "integer",
						"description": "Seed for reproducible generation"
					},
					"guidance_scale": {
						"type": "number",
						"description": "How closely to follow the prompt (1-20, default: 7.5)",
						"default": 7.5
					},
					"negative_prompt": {
						"type": "string",
						"description": "What to avoid in the image"
					}
				},
				"required": ["prompt"]
			}`),
		},
		{
			Name:        "continue_operation",
			Description: "Continue waiting for an in-progress image operation. Use when a previous operation returned a prediction_id.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"prediction_id": {
						"type": "string",
						"description": "The prediction ID from a previous operation"
					},
					"wait_time": {
						"type": "integer",
						"description": "How many seconds to wait (max 30)",
						"default": 30
					}
				},
				"required": ["prediction_id"]
			}`),
		},
		{
			Name:        "list_images",
			Description: "List all generated/processed images",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {}
			}`),
		},
		{
			Name:        "get_image",
			Description: "Get details about a specific image including metadata",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"id": {
						"type": "string",
						"description": "The image ID"
					}
				},
				"required": ["id"]
			}`),
		},
		{
			Name:        "remove_background",
			Description: "Remove background from an image. Supports multiple models for different quality/speed tradeoffs.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"file_path": {
						"type": "string",
						"description": "Path to the local image file"
					},
					"model": {
						"type": "string",
						"description": "Model to use: remove-bg (fast), rembg (balanced), dis (high accuracy)",
						"enum": ["remove-bg", "rembg", "dis"],
						"default": "remove-bg"
					},
					"output_format": {
						"type": "string",
						"description": "Output format: png or webp",
						"enum": ["png", "webp"],
						"default": "png"
					},
					"filename": {
						"type": "string",
						"description": "Optional output filename"
					}
				},
				"required": ["file_path"]
			}`),
		},
		{
			Name:        "upscale_image",
			Description: "Upscale an image to higher resolution using AI models.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"file_path": {
						"type": "string",
						"description": "Path to the local image file"
					},
					"scale": {
						"type": "integer",
						"description": "Upscale factor: 2, 4, or 8",
						"enum": [2, 4, 8],
						"default": 2
					},
					"model": {
						"type": "string",
						"description": "Model to use: realesrgan (general), clarity (advanced)",
						"enum": ["realesrgan", "clarity"],
						"default": "realesrgan"
					},
					"face_enhance": {
						"type": "boolean",
						"description": "Enhance faces during upscaling",
						"default": false
					},
					"filename": {
						"type": "string",
						"description": "Optional output filename"
					}
				},
				"required": ["file_path"]
			}`),
		},
		{
			Name:        "enhance_face",
			Description: "Enhance and restore faces in images using AI models.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"file_path": {
						"type": "string",
						"description": "Path to the local image file"
					},
					"enhancement_model": {
						"type": "string",
						"description": "Model to use: gfpgan (fast) or codeformer (high quality)",
						"enum": ["gfpgan", "codeformer"],
						"default": "gfpgan"
					},
					"upscale": {
						"type": "integer",
						"description": "Upscale factor: 1, 2, or 4",
						"enum": [1, 2, 4],
						"default": 2
					},
					"fidelity": {
						"type": "number",
						"description": "Fidelity for codeformer (0.0-1.0, lower = better quality)",
						"default": 0.5
					},
					"background_enhance": {
						"type": "boolean",
						"description": "Enhance background as well",
						"default": false
					},
					"filename": {
						"type": "string",
						"description": "Optional output filename"
					}
				},
				"required": ["file_path"]
			}`),
		},
		{
			Name:        "restore_photo",
			Description: "Restore old or damaged photos using AI.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"file_path": {
						"type": "string",
						"description": "Path to the old/damaged photo"
					},
					"with_scratch": {
						"type": "boolean",
						"description": "Process scratches and tears",
						"default": true
					},
					"high_resolution": {
						"type": "boolean",
						"description": "Use high resolution mode",
						"default": false
					},
					"filename": {
						"type": "string",
						"description": "Optional output filename"
					}
				},
				"required": ["file_path"]
			}`),
		},
		{
			Name:        "edit_image",
			Description: "Edit specific parts of an image using AI inpainting. You can provide a mask to specify the area to edit, or describe what to change.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"file_path": {
						"type": "string",
						"description": "Path to the image to edit"
					},
					"edit_prompt": {
						"type": "string",
						"description": "What to generate in the edited area"
					},
					"mask_path": {
						"type": "string",
						"description": "Optional: Path to mask image (white=edit, black=keep)"
					},
					"selection_prompt": {
						"type": "string",
						"description": "Optional: Text description of what to select for editing"
					},
					"strength": {
						"type": "number",
						"description": "Edit strength (0-1, default 0.8)",
						"default": 0.8
					},
					"guidance_scale": {
						"type": "number",
						"description": "How closely to follow the prompt (1-20, default 7.5)",
						"default": 7.5
					},
					"negative_prompt": {
						"type": "string",
						"description": "What to avoid in the edited area"
					},
					"filename": {
						"type": "string",
						"description": "Optional output filename"
					}
				},
				"required": ["file_path", "edit_prompt"]
			}`),
		},
	}

	return &protocol.ListToolsResponse{Tools: tools}, nil
}

// CallTool executes a tool
func (s *ReplicateImageMCPServer) CallTool(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResponse, error) {
	switch req.Name {
	case "generate_image":
		result, err := s.generateImage(ctx, req.Arguments)
		if err != nil {
			return &protocol.CallToolResponse{
				Content: []protocol.ToolContent{{
					Type: "text",
					Text: fmt.Sprintf("Error: %v", err),
				}},
				IsError: true,
			}, nil
		}
		return &protocol.CallToolResponse{
			Content: []protocol.ToolContent{{
				Type: "text",
				Text: result,
			}},
		}, nil

	case "continue_operation":
		result, err := s.continueOperation(ctx, req.Arguments)
		if err != nil {
			return &protocol.CallToolResponse{
				Content: []protocol.ToolContent{{
					Type: "text",
					Text: fmt.Sprintf("Error: %v", err),
				}},
				IsError: true,
			}, nil
		}
		return &protocol.CallToolResponse{
			Content: []protocol.ToolContent{{
				Type: "text",
				Text: result,
			}},
		}, nil

	case "list_images":
		result, err := s.listImages(ctx)
		if err != nil {
			return &protocol.CallToolResponse{
				Content: []protocol.ToolContent{{
					Type: "text",
					Text: fmt.Sprintf("Error: %v", err),
				}},
				IsError: true,
			}, nil
		}
		return &protocol.CallToolResponse{
			Content: []protocol.ToolContent{{
				Type: "text",
				Text: result,
			}},
		}, nil

	case "get_image":
		result, err := s.getImage(ctx, req.Arguments)
		if err != nil {
			return &protocol.CallToolResponse{
				Content: []protocol.ToolContent{{
					Type: "text",
					Text: fmt.Sprintf("Error: %v", err),
				}},
				IsError: true,
			}, nil
		}
		return &protocol.CallToolResponse{
			Content: []protocol.ToolContent{{
				Type: "text",
				Text: result,
			}},
		}, nil

	case "remove_background":
		result, err := s.removeBackground(ctx, req.Arguments)
		if err != nil {
			return &protocol.CallToolResponse{
				Content: []protocol.ToolContent{{
					Type: "text",
					Text: fmt.Sprintf("Error: %v", err),
				}},
				IsError: true,
			}, nil
		}
		return &protocol.CallToolResponse{
			Content: []protocol.ToolContent{{
				Type: "text",
				Text: result,
			}},
		}, nil

	case "upscale_image":
		result, err := s.upscaleImage(ctx, req.Arguments)
		if err != nil {
			return &protocol.CallToolResponse{
				Content: []protocol.ToolContent{{
					Type: "text",
					Text: fmt.Sprintf("Error: %v", err),
				}},
				IsError: true,
			}, nil
		}
		return &protocol.CallToolResponse{
			Content: []protocol.ToolContent{{
				Type: "text",
				Text: result,
			}},
		}, nil

	case "enhance_face":
		result, err := s.enhanceFace(ctx, req.Arguments)
		if err != nil {
			return &protocol.CallToolResponse{
				Content: []protocol.ToolContent{{
					Type: "text",
					Text: fmt.Sprintf("Error: %v", err),
				}},
				IsError: true,
			}, nil
		}
		return &protocol.CallToolResponse{
			Content: []protocol.ToolContent{{
				Type: "text",
				Text: result,
			}},
		}, nil

	case "restore_photo":
		result, err := s.restorePhoto(ctx, req.Arguments)
		if err != nil {
			return &protocol.CallToolResponse{
				Content: []protocol.ToolContent{{
					Type: "text",
					Text: fmt.Sprintf("Error: %v", err),
				}},
				IsError: true,
			}, nil
		}
		return &protocol.CallToolResponse{
			Content: []protocol.ToolContent{{
				Type: "text",
				Text: result,
			}},
		}, nil

	case "edit_image":
		result, err := s.editImage(ctx, req.Arguments)
		if err != nil {
			return &protocol.CallToolResponse{
				Content: []protocol.ToolContent{{
					Type: "text",
					Text: fmt.Sprintf("Error: %v", err),
				}},
				IsError: true,
			}, nil
		}
		return &protocol.CallToolResponse{
			Content: []protocol.ToolContent{{
				Type: "text",
				Text: result,
			}},
		}, nil

	default:
		return &protocol.CallToolResponse{
			Content: []protocol.ToolContent{{
				Type: "text",
				Text: fmt.Sprintf("Unknown tool: %s", req.Name),
			}},
			IsError: true,
		}, nil
	}
}

// generateImage handles the generate_image tool
func (s *ReplicateImageMCPServer) generateImage(ctx context.Context, params map[string]interface{}) (string, error) {
	// Log incoming parameters
	log.Printf("\n=== DEBUG: generateImage called ===")
	log.Printf("Incoming params: %+v", params)

	// Parse parameters
	prompt, ok := params["prompt"].(string)
	if !ok || prompt == "" {
		return "", fmt.Errorf("prompt parameter is required")
	}
	log.Printf("Prompt: %s", prompt)

	// Get model selection - default to flux-schnell which works reliably
	model := "flux-schnell"
	if m, ok := params["model"].(string); ok && m != "" {
		model = m
	}

	// Map model name to Replicate model ID
	var modelID string
	switch model {
	case "flux-schnell":
		modelID = types.ModelFluxSchnell
	case "flux-pro":
		modelID = types.ModelFluxPro
	case "flux-dev":
		modelID = types.ModelFluxDev
	case "seedream-3", "seedream":
		modelID = types.ModelSeedream3
	case "sdxl":
		modelID = types.ModelSDXL
	case "sdxl-lightning":
		modelID = types.ModelSDXLLightning
	case "ideogram-turbo", "ideogram":
		modelID = types.ModelIdeogramTurbo
	case "recraft":
		modelID = types.ModelRecraft
	case "recraft-svg":
		modelID = types.ModelRecraftSVG
	default:
		modelID = types.ModelFluxSchnell  // Default to flux-schnell
	}

	log.Printf("Model selected: %s", model)
	log.Printf("Model ID: %s", modelID)

	// Build input parameters - simplified for better compatibility
	input := map[string]interface{}{
		"prompt": prompt,
	}

	// Add dimensions
	if width, ok := params["width"].(float64); ok {
		input["width"] = int(width)
	} else {
		input["width"] = 768  // Use 768 as safer default
	}

	if height, ok := params["height"].(float64); ok {
		input["height"] = int(height)
	} else {
		input["height"] = 768  // Use 768 as safer default
	}

	// Add optional parameters that most models support
	if seed, ok := params["seed"].(float64); ok {
		input["seed"] = int(seed)
	}

	// SDXL and most models use these parameters
	if model == "sdxl" || model == "seedream-3" {
		if guidanceScale, ok := params["guidance_scale"].(float64); ok {
			input["guidance_scale"] = guidanceScale
		} else {
			input["guidance_scale"] = 7.5
		}

		if negativePrompt, ok := params["negative_prompt"].(string); ok && negativePrompt != "" {
			input["negative_prompt"] = negativePrompt
		}
		
		// SDXL specific
		input["num_inference_steps"] = 25  // Good balance for SDXL
	} else if strings.HasPrefix(model, "flux") {
		// Flux models have simpler parameters
		input["num_inference_steps"] = 4  // Flux schnell default
		input["output_format"] = "webp"
	}

	log.Printf("Final input parameters: %+v", input)

	// Get filename if provided
	filename, _ := params["filename"].(string)

	// Generate unique ID for this operation
	id, err := s.storage.GenerateID()
	if err != nil {
		return "", fmt.Errorf("failed to generate ID: %w", err)
	}

	// Create prediction
	startTime := time.Now()
	log.Printf("Creating prediction with Replicate API...")
	prediction, err := s.client.CreatePrediction(ctx, modelID, input)
	if err != nil {
		log.Printf("ERROR: Failed to create prediction: %v", err)
		return "", fmt.Errorf("failed to create prediction: %w", err)
	}
	log.Printf("Prediction created successfully: ID = %s", prediction.ID)

	// Wait for completion (up to 30 seconds)
	log.Printf("Waiting for completion (timeout: %v)...", s.config.OperationTimeout)
	result, waitErr := s.client.WaitForCompletion(ctx, prediction.ID, s.config.OperationTimeout)
	if waitErr != nil {
		log.Printf("Wait error: %v", waitErr)
	}
	if result != nil {
		log.Printf("Result status: %s", result.Status)
	}
	
	// Check if completed successfully
	if waitErr == nil && result.Status == types.StatusSucceeded {
		// Save the image
		outputURL := ""
		switch output := result.Output.(type) {
		case string:
			outputURL = output
		case []interface{}:
			if len(output) > 0 {
				if url, ok := output[0].(string); ok {
					outputURL = url
				}
			}
		}

		if outputURL == "" {
			return "", fmt.Errorf("no output URL in prediction result")
		}

		imagePath, err := s.storage.SaveImage(id, outputURL, filename)
		if err != nil {
			return "", fmt.Errorf("failed to save image: %w", err)
		}

		// Save metadata
		metadata := &types.ImageMetadata{
			Version:   "1.0",
			ID:        id,
			Operation: "generate_image",
			Timestamp: time.Now(),
			Model:     modelID,
			Parameters: map[string]interface{}{
				"prompt":          prompt,
				"width":           input["width"],
				"height":          input["height"],
				"guidance_scale":  input["guidance_scale"],
				"negative_prompt": input["negative_prompt"],
			},
			Result: &types.OperationResult{
				Filename:       filepath.Base(imagePath),
				GenerationTime: time.Since(startTime).Seconds(),
				PredictionID:   prediction.ID,
				Width:          input["width"].(int),
				Height:         input["height"].(int),
			},
		}

		if err := s.storage.SaveMetadata(id, metadata); err != nil {
			// Log error but don't fail
			if s.config.DebugMode {
				log.Printf("Failed to save metadata: %v", err)
			}
		}

		return fmt.Sprintf("Image generated successfully and saved to: %s (ID: %s)", imagePath, id), nil
	}

	// If timed out or still processing
	if result != nil && (result.Status == types.StatusProcessing || result.Status == types.StatusStarting) {
		// Save partial metadata with prediction ID
		metadata := &types.ImageMetadata{
			Version:   "1.0",
			ID:        id,
			Operation: "generate_image",
			Timestamp: time.Now(),
			Model:     modelID,
			Parameters: map[string]interface{}{
				"prompt":          prompt,
				"width":           input["width"],
				"height":          input["height"],
				"guidance_scale":  input["guidance_scale"],
				"negative_prompt": input["negative_prompt"],
			},
			Result: &types.OperationResult{
				PredictionID: prediction.ID,
			},
		}
		s.storage.SaveMetadata(id, metadata)

		return fmt.Sprintf("Image generation in progress (prediction_id: %s, storage_id: %s). Please call continue_operation with prediction_id='%s' and wait_time=30 to check status.", 
			prediction.ID, id, prediction.ID), nil
	}

	// If failed
	if waitErr != nil {
		return "", fmt.Errorf("generation failed: %w", waitErr)
	}

	return "", fmt.Errorf("unexpected prediction status: %s", result.Status)
}

// continueOperation handles the continue_operation tool
func (s *ReplicateImageMCPServer) continueOperation(ctx context.Context, params map[string]interface{}) (string, error) {
	predictionID, ok := params["prediction_id"].(string)
	if !ok || predictionID == "" {
		return "", fmt.Errorf("prediction_id parameter is required")
	}

	waitTime := 30
	if wt, ok := params["wait_time"].(float64); ok {
		waitTime = int(wt)
		if waitTime > 30 {
			waitTime = 30
		}
	}

	// Wait for completion
	result, err := s.client.WaitForCompletion(ctx, predictionID, time.Duration(waitTime)*time.Second)
	
	if err == nil && result.Status == types.StatusSucceeded {
		// Get output URL
		outputURL := ""
		switch output := result.Output.(type) {
		case string:
			outputURL = output
		case []interface{}:
			if len(output) > 0 {
				if url, ok := output[0].(string); ok {
					outputURL = url
				}
			}
		}

		if outputURL == "" {
			return "", fmt.Errorf("no output URL in prediction result")
		}

		// Find the storage ID for this prediction
		// For now, generate a new ID (in production, we'd track this mapping)
		id, err := s.storage.GenerateID()
		if err != nil {
			return "", fmt.Errorf("failed to generate ID: %w", err)
		}

		// Save the image
		imagePath, err := s.storage.SaveImage(id, outputURL, "")
		if err != nil {
			return "", fmt.Errorf("failed to save image: %w", err)
		}

		return fmt.Sprintf("Image generated successfully and saved to: %s (ID: %s)", imagePath, id), nil
	}

	// If still processing
	if result != nil && (result.Status == types.StatusProcessing || result.Status == types.StatusStarting) {
		return fmt.Sprintf("Still processing (prediction_id: %s). Please call continue_operation again with prediction_id='%s' and wait_time=30 to continue checking.", 
			predictionID, predictionID), nil
	}

	// If failed
	if err != nil {
		return "", fmt.Errorf("operation failed: %w", err)
	}

	return "", fmt.Errorf("unexpected prediction status: %s", result.Status)
}

// listImages handles the list_images tool
func (s *ReplicateImageMCPServer) listImages(ctx context.Context) (string, error) {
	images, err := s.storage.ListImages()
	if err != nil {
		return "", fmt.Errorf("failed to list images: %w", err)
	}

	response := types.ListImagesResponse{
		Images: images,
		Total:  len(images),
	}

	jsonBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format response: %w", err)
	}

	return string(jsonBytes), nil
}

// getImage handles the get_image tool
func (s *ReplicateImageMCPServer) getImage(ctx context.Context, params map[string]interface{}) (string, error) {
	id, ok := params["id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("id parameter is required")
	}

	metadata, err := s.storage.LoadMetadata(id)
	if err != nil {
		return "", fmt.Errorf("failed to load image metadata: %w", err)
	}

	// Get the image file path
	imagePath := ""
	if metadata.Result != nil && metadata.Result.Filename != "" {
		imagePath = s.storage.GetImagePath(id, metadata.Result.Filename)
	}

	response := types.GetImageResponse{
		ID:       id,
		FilePath: imagePath,
		Metadata: metadata,
	}

	jsonBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format response: %w", err)
	}

	return string(jsonBytes), nil
}

// Model shortcuts for CLI
var modelShortcuts = map[string]string{
	"flux-schnell":   "flux-schnell",
	"flux-dev":       "flux-dev",
	"flux-pro":       "flux-pro",
	"sdxl":           "sdxl",
	"sdxl-lightning": "sdxl-lightning",
	"seedream":       "seedream-3",
	"ideogram":       "ideogram-turbo",
	"recraft":        "recraft",
	"recraft-svg":    "recraft-svg",
}

// Default test prompt
const defaultTestPrompt = "A futuristic city skyline at sunset with flying cars, neon lights, and a large moon in the sky, cyberpunk style, highly detailed"

func listAvailableModels() {
	fmt.Println("\n=== Available Models ===")
	fmt.Println("\nGeneration Models:")
	fmt.Println("  flux-schnell    - Fast generation (default)")
	fmt.Println("  flux-dev        - Development version")
	fmt.Println("  flux-pro        - High quality (paid)")
	fmt.Println("  sdxl            - Stable Diffusion XL")
	fmt.Println("  sdxl-lightning  - Fast SDXL variant")
	fmt.Println("  seedream        - High quality generation")
	fmt.Println("  ideogram        - Text in images")
	fmt.Println("  recraft         - Raster images")
	fmt.Println("  recraft-svg     - SVG generation")
	fmt.Println("\nUsage: ./replicate_image_ai -g <model> [-p \"custom prompt\"]")
}

func testSingleModel(server *ReplicateImageMCPServer, model, prompt string) error {
	ctx := context.Background()
	
	// Resolve model shortcut
	modelName := model
	if mapped, ok := modelShortcuts[model]; ok {
		modelName = mapped
	}
	
	fmt.Printf("\nTesting model: %s\n", modelName)
	fmt.Printf("Prompt: %s\n", prompt)
	fmt.Println("---")
	
	startTime := time.Now()
	
	// Call the same generateImage function used by MCP server
	result, err := server.generateImage(ctx, map[string]interface{}{
		"prompt": prompt,
		"model":  modelName,
		"width":  1024.0,
		"height": 1024.0,
	})
	
	elapsed := time.Since(startTime)
	
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return err
	}
	
	fmt.Printf("✅ Success! Time: %v\n", elapsed)
	fmt.Printf("Result: %s\n", result)
	
	// Check if we need to continue operation
	if strings.Contains(result, "prediction_id:") {
		// Extract prediction ID from result
		start := strings.Index(result, "prediction_id: ") + len("prediction_id: ")
		end := strings.Index(result[start:], ",")
		if end == -1 {
			end = strings.Index(result[start:], ")")
		}
		if end != -1 {
			predictionID := result[start : start+end]
			fmt.Printf("\nContinuing operation for prediction_id: %s\n", predictionID)
			
			// Wait for completion
			continueResult, err := server.continueOperation(ctx, map[string]interface{}{
				"prediction_id": predictionID,
				"wait_time":     30.0,
			})
			
			if err != nil {
				fmt.Printf("❌ Continue error: %v\n", err)
				return err
			}
			
			fmt.Printf("Result: %s\n", continueResult)
		}
	}
	
	return nil
}

func testAllModels(server *ReplicateImageMCPServer) {
	fmt.Println("\n=== Testing All Generation Models ===")
	
	models := []struct {
		name   string
		prompt string
	}{
		{"flux-schnell", defaultTestPrompt},
		{"flux-dev", defaultTestPrompt},
		{"sdxl", defaultTestPrompt},
		{"sdxl-lightning", defaultTestPrompt},
		{"seedream", defaultTestPrompt},
		{"ideogram", "The word 'REPLICATE' in bold futuristic letters with neon glow effect"},
		{"recraft", defaultTestPrompt},
		{"recraft-svg", "Simple geometric logo design with circles and triangles"},
	}
	
	successCount := 0
	for i, test := range models {
		fmt.Printf("\n[%d/%d] %s\n", i+1, len(models), test.name)
		
		err := testSingleModel(server, test.name, test.prompt)
		if err == nil {
			successCount++
		}
		
		// Add delay between tests to avoid rate limiting
		if i < len(models)-1 {
			fmt.Println("\nWaiting 2 seconds before next test...")
			time.Sleep(2 * time.Second)
		}
	}
	
	fmt.Printf("\n=== Test Summary ===")
	fmt.Printf("\nTotal: %d/%d models succeeded\n", successCount, len(models))
}

func main() {
	// Parse command line flags
	var (
		generateModel string
		listModels    bool
		testAll       bool
		prompt        string
		versionFlag   bool
		testModelID   string
		// Enhancement testing flags
		testEnhance      string
		inputImage       string
		enhanceModel     string
		outputFile       string
	)
	
	flag.StringVar(&generateModel, "g", "", "Generate an image using specified model (e.g., -g flux-schnell)")
	flag.BoolVar(&listModels, "list", false, "List all available models")
	flag.BoolVar(&testAll, "test", false, "Test all models")
	flag.StringVar(&prompt, "p", defaultTestPrompt, "Custom prompt for generation")
	flag.BoolVar(&versionFlag, "version", false, "Show version information")
	flag.StringVar(&testModelID, "test-id", "", "Test a specific model ID directly (e.g., -test-id stability-ai/stable-diffusion)")
	// Enhancement testing flags
	flag.StringVar(&testEnhance, "enhance", "", "Test enhancement tool: remove-bg, upscale, face, restore, edit")
	flag.StringVar(&inputImage, "input", "", "Input image path for enhancement tests")
	flag.StringVar(&enhanceModel, "model", "", "Model to use for enhancement (e.g., gfpgan, codeformer)")
	flag.StringVar(&outputFile, "output", "", "Output filename for enhanced image")
	flag.Parse()
	
	if versionFlag {
		fmt.Printf("Replicate Image AI MCP Server\n")
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		return
	}
	
	// Handle command-line testing options
	if listModels {
		listAvailableModels()
		return
	}
	
	// Handle enhancement testing
	if testEnhance != "" {
		if inputImage == "" {
			fmt.Println("Error: -input flag is required when using -enhance")
			fmt.Println("Usage: replicate_image_ai -enhance <tool> -input <image_path>")
			fmt.Println("Tools: remove-bg, upscale, face, restore, edit")
			os.Exit(1)
		}
		
		// Create server instance
		server, err := NewReplicateImageMCPServer()
		if err != nil {
			log.Fatalf("Failed to create server: %v", err)
		}
		
		ctx := context.Background()
		var result string
		
		switch testEnhance {
		case "remove-bg", "remove", "bg":
			fmt.Printf("Removing background from: %s\n", inputImage)
			params := map[string]interface{}{
				"file_path": inputImage,
			}
			if enhanceModel != "" {
				params["model"] = enhanceModel // remove-bg, rembg, dis
			}
			if outputFile != "" {
				params["filename"] = outputFile
			}
			result, err = server.removeBackground(ctx, params)
			
		case "upscale", "up":
			fmt.Printf("Upscaling image: %s\n", inputImage)
			params := map[string]interface{}{
				"file_path": inputImage,
				"scale": 2.0,
			}
			if enhanceModel != "" {
				params["model"] = enhanceModel // realesrgan, clarity
			}
			if outputFile != "" {
				params["filename"] = outputFile
			}
			result, err = server.upscaleImage(ctx, params)
			
		case "face", "enhance-face":
			fmt.Printf("Enhancing faces in: %s\n", inputImage)
			params := map[string]interface{}{
				"file_path": inputImage,
				"upscale": 2.0,
			}
			if enhanceModel != "" {
				params["enhancement_model"] = enhanceModel // gfpgan, codeformer
			}
			if outputFile != "" {
				params["filename"] = outputFile
			}
			result, err = server.enhanceFace(ctx, params)
			
		case "restore", "photo":
			fmt.Printf("Restoring photo: %s\n", inputImage)
			params := map[string]interface{}{
				"file_path": inputImage,
				"with_scratch": true,
			}
			if outputFile != "" {
				params["filename"] = outputFile
			}
			result, err = server.restorePhoto(ctx, params)
			
		case "edit", "inpaint":
			fmt.Printf("Editing image: %s\n", inputImage)
			editPrompt := "Replace with beautiful flowers"
			if prompt != "" && prompt != defaultTestPrompt {
				editPrompt = prompt
			}
			params := map[string]interface{}{
				"file_path": inputImage,
				"edit_prompt": editPrompt,
				"strength": 0.8,
			}
			// Check if mask image is provided via -model flag
			if enhanceModel != "" && enhanceModel != "inpainting" {
				params["mask_path"] = enhanceModel
				fmt.Printf("Using mask: %s\n", enhanceModel)
			}
			if outputFile != "" {
				params["filename"] = outputFile
			}
			result, err = server.editImage(ctx, params)
			
		default:
			fmt.Printf("Unknown enhancement tool: %s\n", testEnhance)
			fmt.Println("Available tools: remove-bg, upscale, face, restore, edit")
			os.Exit(1)
		}
		
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Printf("✅ Success!\n")
		fmt.Printf("Result:\n%s\n", result)
		return
	}
	
	if generateModel != "" || testAll || testModelID != "" {
		// Create server instance for testing
		server, err := NewReplicateImageMCPServer()
		if err != nil {
			log.Fatalf("Failed to create server: %v", err)
		}
		
		if testModelID != "" {
			// Test a raw model ID directly
			fmt.Printf("Testing raw model ID: %s\n", testModelID)
			ctx := context.Background()
			input := map[string]interface{}{
				"prompt": "A simple test image",
				"width":  512.0,
				"height": 512.0,
			}
			
			prediction, err := server.client.CreatePrediction(ctx, testModelID, input)
			if err != nil {
				fmt.Printf("❌ Failed: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Printf("✅ SUCCESS! Model ID works: %s\n", testModelID)
			fmt.Printf("   Prediction ID: %s\n", prediction.ID)
			fmt.Printf("   Status: %s\n", prediction.Status)
			
			// Cancel to save resources
			server.client.CancelPrediction(ctx, prediction.ID)
			return
		}
		
		if generateModel != "" {
			// Test single model
			if err := testSingleModel(server, generateModel, prompt); err != nil {
				os.Exit(1)
			}
		} else if testAll {
			// Test all models
			testAllModels(server)
		}
		return
	}

	// Create server
	replicateServer, err := NewReplicateImageMCPServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Create handler registry
	registry := handler.NewHandlerRegistry()

	// Register Replicate server as a tool handler
	registry.RegisterToolHandler(replicateServer)

	// Create and run MCP server
	mcpServer := server.New(server.Options{
		Name:     "Replicate Image AI",
		Version:  Version,
		Registry: registry,
	})

	if err := mcpServer.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}