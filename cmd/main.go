package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"path/filepath"
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
	// Parse parameters
	prompt, ok := params["prompt"].(string)
	if !ok || prompt == "" {
		return "", fmt.Errorf("prompt parameter is required")
	}

	// Get model selection
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
	case "seedream-3":
		modelID = types.ModelSeedream3
	case "sdxl":
		modelID = types.ModelSDXL
	case "ideogram-turbo":
		modelID = types.ModelIdeogramTurbo
	default:
		modelID = types.ModelFluxSchnell
	}

	// Build input parameters
	input := map[string]interface{}{
		"prompt": prompt,
	}

	// Add optional parameters
	if width, ok := params["width"].(float64); ok {
		input["width"] = int(width)
	} else {
		input["width"] = 1024
	}

	if height, ok := params["height"].(float64); ok {
		input["height"] = int(height)
	} else {
		input["height"] = 1024
	}

	if seed, ok := params["seed"].(float64); ok {
		input["seed"] = int(seed)
	}

	if guidanceScale, ok := params["guidance_scale"].(float64); ok {
		input["guidance_scale"] = guidanceScale
	} else {
		input["guidance_scale"] = 7.5
	}

	if negativePrompt, ok := params["negative_prompt"].(string); ok && negativePrompt != "" {
		input["negative_prompt"] = negativePrompt
	}

	// Get filename if provided
	filename, _ := params["filename"].(string)

	// Generate unique ID for this operation
	id, err := s.storage.GenerateID()
	if err != nil {
		return "", fmt.Errorf("failed to generate ID: %w", err)
	}

	// Create prediction
	startTime := time.Now()
	prediction, err := s.client.CreatePrediction(ctx, modelID, input)
	if err != nil {
		return "", fmt.Errorf("failed to create prediction: %w", err)
	}

	// Wait for completion (up to 30 seconds)
	result, waitErr := s.client.WaitForCompletion(ctx, prediction.ID, s.config.OperationTimeout)
	
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

// Test mode for the server
func runTestMode() {
	fmt.Println("Replicate Image AI MCP Server - Test Mode")
	fmt.Println("=========================================")

	server, err := NewReplicateImageMCPServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	ctx := context.Background()

	// Test image generation
	fmt.Println("\nTest: Generate Image")
	fmt.Println("-------------------")
	
	result, err := server.generateImage(ctx, map[string]interface{}{
		"prompt": "A beautiful sunset over mountains, digital art style",
		"model":  "flux-schnell",
		"width":  1024.0,
		"height": 1024.0,
	})
	
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result: %s\n", result)
	}

	// Test list images
	fmt.Println("\nTest: List Images")
	fmt.Println("----------------")
	
	listResult, err := server.listImages(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result: %s\n", listResult)
	}
}

func main() {
	// Parse command line flags
	testMode := flag.Bool("test", false, "Run in test mode")
	versionFlag := flag.Bool("version", false, "Show version information")
	flag.Parse()
	
	if *versionFlag {
		fmt.Printf("Replicate Image AI MCP Server\n")
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		return
	}

	if *testMode {
		runTestMode()
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