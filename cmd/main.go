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
	"github.com/gomcpgo/replicate_image_ai/pkg/responses"
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
						"description": "Model to use. Choose based on your needs:\n- flux-schnell: Fast, general purpose (default)\n- flux-pro: Premium artistic quality\n- flux-dev: Development version\n- imagen-4: Google's photorealistic model, best for lifelike images, superior text rendering\n- gen4-image: RunwayML Gen-4 for consistent characters (use generate_with_visual_context for reference images)\n- seedream-3: High quality artistic/stylized\n- sdxl: Versatile, good balance\n- ideogram-turbo: Best for images with text/logos",
						"enum": ["flux-schnell", "flux-pro", "flux-dev", "imagen-4", "gen4-image", "seedream-3", "sdxl", "ideogram-turbo"],
						"default": "flux-schnell"
					},
					"width": {
						"type": "integer",
						"description": "Image width in pixels (default: 1024). Note: imagen-4 uses aspect_ratio instead",
						"default": 1024
					},
					"height": {
						"type": "integer",
						"description": "Image height in pixels (default: 1024). Note: imagen-4 uses aspect_ratio instead",
						"default": 1024
					},
					"aspect_ratio": {
						"type": "string",
						"description": "Aspect ratio for imagen-4 model only",
						"enum": ["1:1", "9:16", "16:9", "3:4", "4:3"]
					},
					"safety_filter_level": {
						"type": "string",
						"description": "Safety filter level for imagen-4 model only",
						"enum": ["block_low_and_above", "block_medium_and_above", "block_only_high"],
						"default": "block_only_high"
					},
					"output_format": {
						"type": "string",
						"description": "Output format for imagen-4 model only",
						"enum": ["jpg", "png"],
						"default": "jpg"
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
						"description": "How closely to follow the prompt (1-20, default: 7.5). Not supported by imagen-4",
						"default": 7.5
					},
					"negative_prompt": {
						"type": "string",
						"description": "What to avoid in the image. Not supported by imagen-4"
					}
				},
				"required": ["prompt"]
			}`),
		},
		{
			Name:        "generate_with_visual_context",
			Description: `Generate images using RunwayML Gen-4 with visual reference images. This tool excels at maintaining visual consistency of people, objects, and locations across different scenes.

Use this when you need to:
- Keep a person's appearance consistent across different images
- Place specific products or objects in new settings
- Combine elements from multiple reference images
- Create variations while preserving visual identity

How it works: Provide 1-3 reference images with tags, then use @tag in your prompt to reference them.

Examples:
- "@woman and @robot are lounging on the sofa in @living_room, it is evening and low light"
- "@woman holds the @bottle up, the bottle is the subject, @woman is visible but blurred, product photo shoot"
- "close up portrait of @woman and @man standing in @park, hands in pockets, looking cool"
- "@product placed on marble surface with dramatic lighting, luxury advertisement style"

The model preserves visual characteristics from your reference images while following your text instructions for composition, lighting, and context.`,
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"prompt": {
						"type": "string",
						"description": "Describe the desired image using @tag to reference your images. Examples: '@woman sitting in @cafe', '@product on beach at sunset', '@person1 and @person2 shaking hands in @office'"
					},
					"reference_images": {
						"type": "array",
						"description": "Local file paths to 1-3 reference images that provide visual elements to preserve",
						"items": {"type": "string"},
						"minItems": 1,
						"maxItems": 3
					},
					"reference_tags": {
						"type": "array",
						"description": "Tags for each reference image (3-15 alphanumeric chars). Use descriptive names like 'woman', 'robot', 'bottle', 'office' rather than generic 'img1', 'obj2'. Must match reference_images count.",
						"items": {"type": "string", "pattern": "^[a-zA-Z0-9]{3,15}$"}
					},
					"aspect_ratio": {
						"type": "string",
						"description": "Output image dimensions",
						"enum": ["16:9", "9:16", "4:3", "3:4", "1:1", "21:9"],
						"default": "16:9"
					},
					"resolution": {
						"type": "string",
						"description": "Output quality - 1080p for high quality, 720p for faster generation",
						"enum": ["1080p", "720p"],
						"default": "1080p"
					},
					"filename": {
						"type": "string",
						"description": "Optional output filename"
					},
					"seed": {
						"type": "integer",
						"description": "Seed for reproducible generation"
					}
				},
				"required": ["prompt", "reference_images", "reference_tags"]
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
			Description: "Edit images using natural language instructions with FLUX Kontext. Transform entire images without masks. Examples: 'Make it a 90s cartoon', 'Change the car to red', 'Make it nighttime with rain', 'Convert to oil painting style', 'Add sunglasses to the person', 'Make the text 3D and glowing'.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"file_path": {
						"type": "string",
						"description": "Path to the local image file to edit"
					},
					"prompt": {
						"type": "string",
						"description": "Text instruction describing the desired changes. Be specific and clear."
					},
					"model": {
						"type": "string",
						"description": "Model variant: kontext-pro (recommended, balanced), kontext-max (highest quality, premium cost), kontext-dev (advanced controls)",
						"enum": ["kontext-pro", "kontext-max", "kontext-dev"],
						"default": "kontext-pro"
					},
					"aspect_ratio": {
						"type": "string",
						"description": "Output aspect ratio. Options: match_input_image (default), 1:1, 16:9, 9:16, 4:3, 3:4, 3:2, 2:3",
						"default": "match_input_image"
					},
					"prompt_upsampling": {
						"type": "boolean",
						"description": "Automatically enhance prompt for better results (Pro/Max only)",
						"default": false
					},
					"safety_tolerance": {
						"type": "integer",
						"description": "Content filter level: 0 (strictest) to 6 (most permissive). Max 2 with input images.",
						"minimum": 0,
						"maximum": 2,
						"default": 2
					},
					"output_format": {
						"type": "string",
						"description": "Output format: png, jpg, or webp",
						"enum": ["png", "jpg", "webp"],
						"default": "png"
					},
					"go_fast": {
						"type": "boolean",
						"description": "Speed up generation (Dev model only)",
						"default": false
					},
					"guidance": {
						"type": "number",
						"description": "Guidance strength 0-10 (Dev model only, default 2.5)",
						"default": 2.5
					},
					"num_inference_steps": {
						"type": "integer",
						"description": "Number of steps 1-50 (Dev model only, default 30)",
						"default": 30
					},
					"output_quality": {
						"type": "integer",
						"description": "JPEG quality 1-100 (Dev model only, default 80)",
						"default": 80
					},
					"seed": {
						"type": "integer",
						"description": "Seed for reproducible generation"
					},
					"filename": {
						"type": "string",
						"description": "Optional output filename"
					}
				},
				"required": ["file_path", "prompt"]
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

	case "generate_with_visual_context":
		result, err := s.generateWithVisualContext(ctx, req.Arguments)
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
		return responses.BuildErrorResponse("generate_image", "invalid_parameters", "prompt parameter is required", nil), nil
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
	case "imagen-4":
		modelID = types.ModelImagen4
	case "gen4-image":
		modelID = types.ModelGen4Image
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

	// Special handling for Imagen-4 and Gen-4
	if model == "imagen-4" {
		// Imagen-4 uses aspect_ratio instead of width/height
		aspectRatio := "1:1"  // Default
		if ar, ok := params["aspect_ratio"].(string); ok && ar != "" {
			aspectRatio = ar
		} else {
			// Try to infer aspect ratio from width/height if provided
			width, hasWidth := params["width"].(float64)
			height, hasHeight := params["height"].(float64)
			if hasWidth && hasHeight {
				ratio := width / height
				if ratio > 1.7 { // ~16:9
					aspectRatio = "16:9"
				} else if ratio < 0.6 { // ~9:16
					aspectRatio = "9:16"
				} else if ratio > 1.2 && ratio < 1.4 { // ~4:3
					aspectRatio = "4:3"
				} else if ratio > 0.7 && ratio < 0.8 { // ~3:4
					aspectRatio = "3:4"
				} else {
					aspectRatio = "1:1"
				}
			}
		}
		input["aspect_ratio"] = aspectRatio

		// Safety filter level
		if sfl, ok := params["safety_filter_level"].(string); ok && sfl != "" {
			input["safety_filter_level"] = sfl
		} else {
			input["safety_filter_level"] = "block_only_high"
		}

		// Output format
		if of, ok := params["output_format"].(string); ok && of != "" {
			input["output_format"] = of
		} else {
			input["output_format"] = "jpg"
		}

		log.Printf("Imagen-4 parameters: aspect_ratio=%s, safety_filter=%s, format=%s", 
			input["aspect_ratio"], input["safety_filter_level"], input["output_format"])
	} else if model == "gen4-image" {
		// Gen-4 Image uses aspect_ratio and resolution
		aspectRatio := "16:9"  // Default
		if ar, ok := params["aspect_ratio"].(string); ok && ar != "" {
			aspectRatio = ar
		} else {
			// Try to infer aspect ratio from width/height if provided
			width, hasWidth := params["width"].(float64)
			height, hasHeight := params["height"].(float64)
			if hasWidth && hasHeight {
				ratio := width / height
				if ratio > 1.7 { // ~16:9
					aspectRatio = "16:9"
				} else if ratio < 0.6 { // ~9:16
					aspectRatio = "9:16"
				} else if ratio > 1.2 && ratio < 1.4 { // ~4:3
					aspectRatio = "4:3"
				} else if ratio > 0.7 && ratio < 0.8 { // ~3:4
					aspectRatio = "3:4"
				} else if ratio > 2.0 { // ~21:9
					aspectRatio = "21:9"
				} else {
					aspectRatio = "1:1"
				}
			}
		}
		input["aspect_ratio"] = aspectRatio
		
		// Resolution
		resolution := "1080p"
		if res, ok := params["resolution"].(string); ok && res != "" {
			resolution = res
		}
		input["resolution"] = resolution
		
		log.Printf("Gen-4 parameters: aspect_ratio=%s, resolution=%s", aspectRatio, resolution)
	} else {
		// Standard models use width/height
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
	}

	// Add optional parameters that most models support (except Imagen-4)
	if seed, ok := params["seed"].(float64); ok {
		input["seed"] = int(seed)
	}

	// Model-specific parameters
	if model == "imagen-4" || model == "gen4-image" {
		// Imagen-4 and Gen-4 don't support guidance_scale or negative_prompt
		// Parameters already set above
	} else if model == "sdxl" || model == "seedream-3" {
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
		return responses.BuildErrorResponse("generate_image", "internal_error", fmt.Sprintf("failed to generate ID: %v", err), nil), nil
	}

	// Create prediction
	startTime := time.Now()
	log.Printf("Creating prediction with Replicate API...")
	prediction, err := s.client.CreatePrediction(ctx, modelID, input)
	if err != nil {
		log.Printf("ERROR: Failed to create prediction: %v", err)
		details := map[string]interface{}{
			"model": modelID,
			"input": input,
		}
		return responses.BuildErrorResponse("generate_image", "api_error", fmt.Sprintf("failed to create prediction: %v", err), details), nil
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
			details := map[string]interface{}{
				"prediction_id": prediction.ID,
				"output":        result.Output,
			}
			return responses.BuildErrorResponse("generate_image", "no_output", "no output URL in prediction result", details), nil
		}

		imagePath, err := s.storage.SaveImage(id, outputURL, filename)
		if err != nil {
			details := map[string]interface{}{
				"prediction_id": prediction.ID,
				"url":           outputURL,
			}
			return responses.BuildErrorResponse("generate_image", "save_failed", fmt.Sprintf("failed to save image: %v", err), details), nil
		}

		// Save metadata
		metadataParams := map[string]interface{}{
			"prompt": prompt,
		}
		resultObj := &types.OperationResult{
			Filename:       filepath.Base(imagePath),
			GenerationTime: time.Since(startTime).Seconds(),
			PredictionID:   prediction.ID,
		}
		
		// Add model-specific metadata
		if model == "imagen-4" {
			metadataParams["aspect_ratio"] = input["aspect_ratio"]
			metadataParams["safety_filter_level"] = input["safety_filter_level"]
			metadataParams["output_format"] = input["output_format"]
			// Imagen-4 doesn't return exact dimensions, estimate from aspect ratio
			switch input["aspect_ratio"] {
			case "16:9":
				resultObj.Width, resultObj.Height = 1024, 576
			case "9:16":
				resultObj.Width, resultObj.Height = 576, 1024
			case "4:3":
				resultObj.Width, resultObj.Height = 1024, 768
			case "3:4":
				resultObj.Width, resultObj.Height = 768, 1024
			default: // "1:1"
				resultObj.Width, resultObj.Height = 1024, 1024
			}
		} else if model == "gen4-image" {
			metadataParams["aspect_ratio"] = input["aspect_ratio"]
			metadataParams["resolution"] = input["resolution"]
			// Gen-4 resolution dimensions
			width, height := 1920, 1080 // Default for 1080p 16:9
			if input["resolution"] == "720p" {
				width, height = 1280, 720
			}
			// Adjust for aspect ratio
			switch input["aspect_ratio"] {
			case "9:16":
				width, height = height*9/16, height
			case "4:3":
				width, height = height*4/3, height
			case "3:4":
				width, height = height*3/4, height
			case "1:1":
				width, height = height, height
			case "21:9":
				width, height = height*21/9, height
			}
			resultObj.Width, resultObj.Height = width, height
		} else {
			metadataParams["width"] = input["width"]
			metadataParams["height"] = input["height"]
			metadataParams["guidance_scale"] = input["guidance_scale"]
			metadataParams["negative_prompt"] = input["negative_prompt"]
			if w, ok := input["width"].(int); ok {
				resultObj.Width = w
			}
			if h, ok := input["height"].(int); ok {
				resultObj.Height = h
			}
		}
		
		metadata := &types.ImageMetadata{
			Version:    "1.0",
			ID:         id,
			Operation:  "generate_image",
			Timestamp:  time.Now(),
			Model:      modelID,
			Parameters: metadataParams,
			Result:     resultObj,
		}

		if err := s.storage.SaveMetadata(id, metadata); err != nil {
			// Log error but don't fail
			if s.config.DebugMode {
				log.Printf("Failed to save metadata: %v", err)
			}
		}

		// Build structured success response
		paths := map[string]string{
			"file_path": imagePath,
			"url":       outputURL,
		}
		
		modelInfo := map[string]string{
			"id":   modelID,
			"name": responses.ExtractModelName(modelID),
			"type": model,
		}
		
		parameters := map[string]interface{}{
			"prompt": prompt,
		}
		
		// Add model-specific parameters to response
		if model == "imagen-4" {
			parameters["aspect_ratio"] = input["aspect_ratio"]
			parameters["safety_filter_level"] = input["safety_filter_level"]
			parameters["output_format"] = input["output_format"]
		} else if model == "gen4-image" {
			parameters["aspect_ratio"] = input["aspect_ratio"]
			parameters["resolution"] = input["resolution"]
		} else {
			parameters["width"] = input["width"]
			parameters["height"] = input["height"]
			parameters["guidance_scale"] = input["guidance_scale"]
			parameters["negative_prompt"] = input["negative_prompt"]
		}
		if seed, ok := input["seed"]; ok {
			parameters["seed"] = seed
		}
		
		metrics := map[string]interface{}{
			"generation_time": time.Since(startTime).Seconds(),
			"file_size":       responses.GetFileSize(imagePath),
		}
		
		return responses.BuildSuccessResponse("generate_image", id, paths, modelInfo, parameters, metrics, prediction.ID), nil
	}

	// If timed out or still processing
	if result != nil && (result.Status == types.StatusProcessing || result.Status == types.StatusStarting) {
		// Save partial metadata with prediction ID
		partialParams := map[string]interface{}{
			"prompt": prompt,
		}
		
		// Add model-specific parameters
		if model == "imagen-4" {
			partialParams["aspect_ratio"] = input["aspect_ratio"]
			partialParams["safety_filter_level"] = input["safety_filter_level"]
			partialParams["output_format"] = input["output_format"]
		} else if model == "gen4-image" {
			partialParams["aspect_ratio"] = input["aspect_ratio"]
			partialParams["resolution"] = input["resolution"]
		} else {
			partialParams["width"] = input["width"]
			partialParams["height"] = input["height"]
			partialParams["guidance_scale"] = input["guidance_scale"]
			partialParams["negative_prompt"] = input["negative_prompt"]
		}
		
		metadata := &types.ImageMetadata{
			Version:    "1.0",
			ID:         id,
			Operation:  "generate_image",
			Timestamp:  time.Now(),
			Model:      modelID,
			Parameters: partialParams,
			Result: &types.OperationResult{
				PredictionID: prediction.ID,
			},
		}
		s.storage.SaveMetadata(id, metadata)

		// Build processing response
		return responses.BuildProcessingResponse("generate_image", prediction.ID, id, 30), nil
	}

	// If failed
	if waitErr != nil {
		details := map[string]interface{}{
			"prediction_id": prediction.ID,
			"storage_id":    id,
		}
		return responses.BuildErrorResponse("generate_image", "generation_failed", waitErr.Error(), details), nil
	}

	details := map[string]interface{}{
		"prediction_id": prediction.ID,
		"status":        result.Status,
	}
	return responses.BuildErrorResponse("generate_image", "unexpected_status", fmt.Sprintf("Unexpected prediction status: %s", result.Status), details), nil
}

// generateWithVisualContext handles the generate_with_visual_context tool for RunwayML Gen-4
func (s *ReplicateImageMCPServer) generateWithVisualContext(ctx context.Context, params map[string]interface{}) (string, error) {
	// Parse parameters
	prompt, ok := params["prompt"].(string)
	if !ok || prompt == "" {
		return responses.BuildErrorResponse("generate_with_visual_context", "invalid_parameters", "prompt parameter is required", nil), nil
	}

	// Get reference images
	referenceImagesRaw, ok := params["reference_images"].([]interface{})
	if !ok || len(referenceImagesRaw) == 0 {
		return responses.BuildErrorResponse("generate_with_visual_context", "invalid_parameters", "reference_images parameter is required (1-3 images)", nil), nil
	}

	// Convert to string array
	referenceImages := make([]string, 0, len(referenceImagesRaw))
	for _, img := range referenceImagesRaw {
		if imgStr, ok := img.(string); ok {
			referenceImages = append(referenceImages, imgStr)
		}
	}

	if len(referenceImages) == 0 || len(referenceImages) > 3 {
		return responses.BuildErrorResponse("generate_with_visual_context", "invalid_parameters", "reference_images must contain 1-3 image paths", nil), nil
	}

	// Get reference tags
	referenceTagsRaw, ok := params["reference_tags"].([]interface{})
	if !ok || len(referenceTagsRaw) != len(referenceImages) {
		return responses.BuildErrorResponse("generate_with_visual_context", "invalid_parameters", "reference_tags must match the number of reference_images", nil), nil
	}

	// Convert to string array and validate
	referenceTags := make([]string, 0, len(referenceTagsRaw))
	for i, tag := range referenceTagsRaw {
		tagStr, ok := tag.(string)
		if !ok {
			return responses.BuildErrorResponse("generate_with_visual_context", "invalid_parameters", fmt.Sprintf("reference_tag at index %d must be a string", i), nil), nil
		}
		// Validate tag format (3-15 alphanumeric characters)
		if len(tagStr) < 3 || len(tagStr) > 15 {
			return responses.BuildErrorResponse("generate_with_visual_context", "invalid_parameters", fmt.Sprintf("reference_tag '%s' must be 3-15 characters", tagStr), nil), nil
		}
		for _, ch := range tagStr {
			if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')) {
				return responses.BuildErrorResponse("generate_with_visual_context", "invalid_parameters", fmt.Sprintf("reference_tag '%s' must contain only alphanumeric characters", tagStr), nil), nil
			}
		}
		referenceTags = append(referenceTags, tagStr)
	}

	// Convert local file paths to data URLs
	imageURLs := make([]string, 0, len(referenceImages))
	for i, imagePath := range referenceImages {
		// Check if file exists
		if _, err := os.Stat(imagePath); os.IsNotExist(err) {
			return responses.BuildErrorResponse("generate_with_visual_context", "file_not_found", fmt.Sprintf("reference image not found: %s", imagePath), nil), nil
		}

		// Convert to data URL
		dataURL, err := s.storage.FileToDataURL(imagePath)
		if err != nil {
			return responses.BuildErrorResponse("generate_with_visual_context", "file_error", fmt.Sprintf("failed to read reference image %d: %v", i+1, err), nil), nil
		}
		
		// Debug logging
		if s.config.DebugMode {
			log.Printf("[DEBUG] Converted reference image %d: %s -> data URL (length: %d)", i+1, imagePath, len(dataURL))
		}
		
		imageURLs = append(imageURLs, dataURL)
	}

	// Get optional parameters
	aspectRatio := "16:9"
	if ar, ok := params["aspect_ratio"].(string); ok && ar != "" {
		aspectRatio = ar
	}

	resolution := "1080p"
	if res, ok := params["resolution"].(string); ok && res != "" {
		resolution = res
	}

	// Build input parameters for Gen-4
	input := map[string]interface{}{
		"prompt":           prompt,
		"reference_images": imageURLs,  // These are now data URLs, not file paths
		"reference_tags":   referenceTags,
		"aspect_ratio":     aspectRatio,
		"resolution":       resolution,
	}

	// Add seed if provided
	if seed, ok := params["seed"].(float64); ok {
		input["seed"] = int(seed)
	}

	// Get filename if provided
	filename, _ := params["filename"].(string)

	// Generate unique ID for this operation
	id, err := s.storage.GenerateID()
	if err != nil {
		return responses.BuildErrorResponse("generate_with_visual_context", "internal_error", fmt.Sprintf("failed to generate ID: %v", err), nil), nil
	}

	// Create prediction with Gen-4 model
	modelID := types.ModelGen4Image
	startTime := time.Now()
	
	if s.config.DebugMode {
		log.Printf("Creating Gen-4 prediction with %d reference images and tags: %v", len(imageURLs), referenceTags)
	}

	prediction, err := s.client.CreatePrediction(ctx, modelID, input)
	if err != nil {
		details := map[string]interface{}{
			"model": modelID,
			"tags":  referenceTags,
		}
		return responses.BuildErrorResponse("generate_with_visual_context", "api_error", fmt.Sprintf("failed to create prediction: %v", err), details), nil
	}

	// Wait for completion (up to 30 seconds)
	result, waitErr := s.client.WaitForCompletion(ctx, prediction.ID, s.config.OperationTimeout)
	
	// Check if completed successfully
	if waitErr == nil && result.Status == types.StatusSucceeded {
		// Extract output URL
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
			details := map[string]interface{}{
				"prediction_id": prediction.ID,
				"output":        result.Output,
			}
			return responses.BuildErrorResponse("generate_with_visual_context", "no_output", "no output URL in prediction result", details), nil
		}

		// Save the generated image
		imagePath, err := s.storage.SaveImage(id, outputURL, filename)
		if err != nil {
			details := map[string]interface{}{
				"prediction_id": prediction.ID,
				"url":           outputURL,
			}
			return responses.BuildErrorResponse("generate_with_visual_context", "save_failed", fmt.Sprintf("failed to save image: %v", err), details), nil
		}

		// Save metadata
		metadataParams := map[string]interface{}{
			"prompt":           prompt,
			"reference_images": referenceImages,
			"reference_tags":   referenceTags,
			"aspect_ratio":     aspectRatio,
			"resolution":       resolution,
		}
		
		// Estimate dimensions based on aspect ratio and resolution
		width, height := 1920, 1080 // Default for 1080p 16:9
		if resolution == "720p" {
			width, height = 1280, 720
		}
		
		// Adjust for aspect ratio
		switch aspectRatio {
		case "9:16":
			width, height = height*9/16, height
		case "4:3":
			width, height = height*4/3, height
		case "3:4":
			width, height = height*3/4, height
		case "1:1":
			width, height = height, height
		case "21:9":
			width, height = height*21/9, height
		}
		
		resultObj := &types.OperationResult{
			Filename:       filepath.Base(imagePath),
			GenerationTime: time.Since(startTime).Seconds(),
			PredictionID:   prediction.ID,
			Width:          width,
			Height:         height,
		}
		
		metadata := &types.ImageMetadata{
			Version:    "1.0",
			ID:         id,
			Operation:  "generate_with_visual_context",
			Timestamp:  time.Now(),
			Model:      modelID,
			Parameters: metadataParams,
			Result:     resultObj,
		}

		if err := s.storage.SaveMetadata(id, metadata); err != nil {
			// Log error but don't fail
			if s.config.DebugMode {
				log.Printf("Failed to save metadata: %v", err)
			}
		}

		// Build success response
		paths := map[string]string{
			"file_path": imagePath,
			"url":       outputURL,
		}
		
		modelInfo := map[string]string{
			"id":   modelID,
			"name": "RunwayML Gen-4 Image",
			"type": "gen4-image",
		}
		
		parameters := map[string]interface{}{
			"prompt":           prompt,
			"reference_count":  len(referenceImages),
			"reference_tags":   referenceTags,
			"aspect_ratio":     aspectRatio,
			"resolution":       resolution,
		}
		if seed, ok := input["seed"]; ok {
			parameters["seed"] = seed
		}
		
		metrics := map[string]interface{}{
			"generation_time": time.Since(startTime).Seconds(),
			"file_size":       responses.GetFileSize(imagePath),
		}
		
		return responses.BuildSuccessResponse("generate_with_visual_context", id, paths, modelInfo, parameters, metrics, prediction.ID), nil
	}

	// If timed out or still processing
	if result != nil && (result.Status == types.StatusProcessing || result.Status == types.StatusStarting) {
		// Save partial metadata
		partialParams := map[string]interface{}{
			"prompt":           prompt,
			"reference_images": referenceImages,
			"reference_tags":   referenceTags,
			"aspect_ratio":     aspectRatio,
			"resolution":       resolution,
		}
		
		metadata := &types.ImageMetadata{
			Version:    "1.0",
			ID:         id,
			Operation:  "generate_with_visual_context",
			Timestamp:  time.Now(),
			Model:      modelID,
			Parameters: partialParams,
			Result: &types.OperationResult{
				PredictionID: prediction.ID,
			},
		}
		s.storage.SaveMetadata(id, metadata)

		// Build processing response
		return responses.BuildProcessingResponse("generate_with_visual_context", prediction.ID, id, 30), nil
	}

	// If failed
	if waitErr != nil {
		details := map[string]interface{}{
			"prediction_id": prediction.ID,
			"storage_id":    id,
		}
		return responses.BuildErrorResponse("generate_with_visual_context", "generation_failed", waitErr.Error(), details), nil
	}

	details := map[string]interface{}{
		"prediction_id": prediction.ID,
		"status":        result.Status,
	}
	return responses.BuildErrorResponse("generate_with_visual_context", "unexpected_status", fmt.Sprintf("Unexpected prediction status: %s", result.Status), details), nil
}

// continueOperation handles the continue_operation tool
func (s *ReplicateImageMCPServer) continueOperation(ctx context.Context, params map[string]interface{}) (string, error) {
	predictionID, ok := params["prediction_id"].(string)
	if !ok || predictionID == "" {
		return responses.BuildErrorResponse("continue_operation", "invalid_parameters", "prediction_id parameter is required", nil), nil
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
			details := map[string]interface{}{
				"prediction_id": predictionID,
				"output":        result.Output,
			}
			return responses.BuildErrorResponse("continue_operation", "no_output", "no output URL in prediction result", details), nil
		}

		// Find the storage ID for this prediction
		// For now, generate a new ID (in production, we'd track this mapping)
		id, err := s.storage.GenerateID()
		if err != nil {
			return responses.BuildErrorResponse("continue_operation", "internal_error", fmt.Sprintf("failed to generate ID: %v", err), nil), nil
		}

		// Save the image
		imagePath, err := s.storage.SaveImage(id, outputURL, "")
		if err != nil {
			details := map[string]interface{}{
				"prediction_id": predictionID,
				"url":           outputURL,
			}
			return responses.BuildErrorResponse("continue_operation", "save_failed", fmt.Sprintf("failed to save image: %v", err), details), nil
		}

		// Build success response
		paths := map[string]string{
			"file_path": imagePath,
			"url":       outputURL,
		}
		
		modelInfo := map[string]string{
			"prediction_id": predictionID,
		}
		
		metrics := map[string]interface{}{
			"file_size": responses.GetFileSize(imagePath),
		}
		
		return responses.BuildSuccessResponse("continue_operation", id, paths, modelInfo, nil, metrics, predictionID), nil
	}

	// If still processing
	if result != nil && (result.Status == types.StatusProcessing || result.Status == types.StatusStarting) {
		return responses.BuildProcessingResponse("continue_operation", predictionID, "", 30), nil
	}

	// If failed
	if err != nil {
		details := map[string]interface{}{
			"prediction_id": predictionID,
		}
		return responses.BuildErrorResponse("continue_operation", "operation_failed", err.Error(), details), nil
	}

	details := map[string]interface{}{
		"prediction_id": predictionID,
		"status":        result.Status,
	}
	return responses.BuildErrorResponse("continue_operation", "unexpected_status", fmt.Sprintf("Unexpected prediction status: %s", result.Status), details), nil
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
	fmt.Println("  imagen-4        - Google's photorealistic model, superior text rendering")
	fmt.Println("  sdxl            - Stable Diffusion XL")
	fmt.Println("  sdxl-lightning  - Fast SDXL variant")
	fmt.Println("  seedream        - High quality generation")
	fmt.Println("  ideogram        - Text in images")
	fmt.Println("  recraft         - Raster images")
	fmt.Println("  recraft-svg     - SVG generation")
	fmt.Println("\nFLUX Kontext Models (Text-based Image Editing):")
	fmt.Println("  kontext-pro     - Balanced speed/quality (recommended)")
	fmt.Println("  kontext-max     - Highest quality, premium tier")
	fmt.Println("  kontext-dev     - Advanced controls, more parameters")
	fmt.Println("\nUsage:")
	fmt.Println("  Generation: ./replicate_image_ai -g <model> [-p \"custom prompt\"]")
	fmt.Println("  Kontext Edit: ./replicate_image_ai -kontext <pro|max|dev> -input image.jpg [-kprompt \"edit prompt\"]")
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
		{"imagen-4", "A photorealistic close-up of a hummingbird hovering near a bright red flower, with iridescent feathers catching the sunlight"},
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
		// Image editing flags
		editModel     string
		editPrompt    string
		// Imagen-4 testing
		imagen4Flag   bool
		aspectRatio   string
		safetyFilter  string
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
	// Edit image flags
	flag.StringVar(&editModel, "edit", "", "Test image editing with FLUX Kontext: pro, max, or dev")
	flag.StringVar(&editPrompt, "eprompt", "", "Prompt for image editing (use with -edit and -input)")
	// Imagen-4 flags
	flag.BoolVar(&imagen4Flag, "imagen4", false, "Test Google Imagen-4 photorealistic generation")
	flag.StringVar(&aspectRatio, "aspect", "16:9", "Aspect ratio for Imagen-4 (1:1, 9:16, 16:9, 3:4, 4:3)")
	flag.StringVar(&safetyFilter, "safety", "block_only_high", "Safety filter for Imagen-4 (block_low_and_above, block_medium_and_above, block_only_high)")
	// Gen-4 flags
	var gen4Flag bool
	var refImages, refTags string
	var gen4Resolution string
	flag.BoolVar(&gen4Flag, "gen4", false, "Test RunwayML Gen-4 with reference images")
	flag.StringVar(&refImages, "ref-images", "", "Comma-separated paths to reference images (1-3)")
	flag.StringVar(&refTags, "ref-tags", "", "Comma-separated tags for reference images")
	flag.StringVar(&gen4Resolution, "resolution", "1080p", "Resolution for Gen-4 (720p, 1080p)")
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
	
	// Handle Imagen-4 testing
	if imagen4Flag {
		// Create server instance
		server, err := NewReplicateImageMCPServer()
		if err != nil {
			log.Fatalf("Failed to create server: %v", err)
		}
		
		ctx := context.Background()
		
		// Use custom prompt or default for Imagen-4
		testPrompt := prompt
		if prompt == defaultTestPrompt {
			testPrompt = "A photorealistic portrait of a young woman with vibrant red hair, looking directly at the camera with a warm smile, soft golden hour lighting streaming through a window, shallow depth of field with blurred background, shot on professional camera"
		}
		
		fmt.Println("\n=== Testing Google Imagen-4 ===")
		fmt.Printf("Prompt: %s\n", testPrompt)
		fmt.Printf("Aspect Ratio: %s\n", aspectRatio)
		fmt.Printf("Safety Filter: %s\n", safetyFilter)
		fmt.Println("---")
		
		startTime := time.Now()
		
		// Call generateImage with Imagen-4 specific parameters
		result, err := server.generateImage(ctx, map[string]interface{}{
			"prompt":               testPrompt,
			"model":                "imagen-4",
			"aspect_ratio":         aspectRatio,
			"safety_filter_level":  safetyFilter,
			"output_format":        "jpg",
		})
		
		elapsed := time.Since(startTime)
		
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Printf("✅ Success! Time: %v\n", elapsed)
		fmt.Printf("Result:\n%s\n", result)
		
		// Check if we need to continue operation
		if strings.Contains(result, "prediction_id:") && strings.Contains(result, "PROCESSING") {
			// Extract prediction ID from result
			lines := strings.Split(result, "\n")
			for _, line := range lines {
				if strings.Contains(line, "prediction_id:") {
					parts := strings.Split(line, ":")
					if len(parts) >= 2 {
						predictionID := strings.TrimSpace(parts[1])
						fmt.Printf("\nContinuing operation for prediction_id: %s\n", predictionID)
						
						// Wait for completion
						continueResult, err := server.continueOperation(ctx, map[string]interface{}{
							"prediction_id": predictionID,
							"wait_time":     30.0,
						})
						
						if err != nil {
							fmt.Printf("❌ Continue error: %v\n", err)
							os.Exit(1)
						}
						
						fmt.Printf("Final Result:\n%s\n", continueResult)
						break
					}
				}
			}
		}
		
		return
	}
	
	// Handle Gen-4 testing with reference images
	if gen4Flag {
		// Validate parameters
		if refImages == "" || refTags == "" {
			fmt.Println("Error: -ref-images and -ref-tags are required for Gen-4 testing")
			fmt.Println("Example: -gen4 -ref-images 'person.jpg,product.jpg' -ref-tags 'person,product' -p '@person holding @product'")
			os.Exit(1)
		}
		
		// Parse reference images and tags
		imagePaths := strings.Split(refImages, ",")
		tags := strings.Split(refTags, ",")
		
		if len(imagePaths) != len(tags) {
			fmt.Printf("Error: Number of reference images (%d) must match number of tags (%d)\n", len(imagePaths), len(tags))
			os.Exit(1)
		}
		
		if len(imagePaths) < 1 || len(imagePaths) > 3 {
			fmt.Println("Error: Must provide 1-3 reference images")
			os.Exit(1)
		}
		
		// Create server instance
		server, err := NewReplicateImageMCPServer()
		if err != nil {
			log.Fatalf("Failed to create server: %v", err)
		}
		
		ctx := context.Background()
		
		// Use custom prompt or default for Gen-4
		testPrompt := prompt
		if prompt == defaultTestPrompt {
			// Build a default prompt using the tags
			if len(tags) > 0 {
				testPrompt = fmt.Sprintf("@%s in a modern office setting with natural lighting", tags[0])
			}
		}
		
		fmt.Println("\n=== Testing RunwayML Gen-4 with Reference Images ===")
		fmt.Printf("Prompt: %s\n", testPrompt)
		fmt.Printf("Reference Images: %v\n", imagePaths)
		fmt.Printf("Reference Tags: %v\n", tags)
		fmt.Printf("Aspect Ratio: %s\n", aspectRatio)
		fmt.Printf("Resolution: %s\n", gen4Resolution)
		fmt.Println("---")
		
		startTime := time.Now()
		
		// Convert arrays to interface{} slices
		refImagesInterface := make([]interface{}, len(imagePaths))
		for i, img := range imagePaths {
			refImagesInterface[i] = strings.TrimSpace(img)
		}
		refTagsInterface := make([]interface{}, len(tags))
		for i, tag := range tags {
			refTagsInterface[i] = strings.TrimSpace(tag)
		}
		
		// Call generateWithVisualContext
		result, err := server.generateWithVisualContext(ctx, map[string]interface{}{
			"prompt":           testPrompt,
			"reference_images": refImagesInterface,
			"reference_tags":   refTagsInterface,
			"aspect_ratio":     aspectRatio,
			"resolution":       gen4Resolution,
		})
		
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Printf("Result:\n%s\n", result)
		
		// Check if we need to continue operation
		if strings.Contains(result, "prediction_id:") && strings.Contains(result, "PROCESSING") {
			// Extract prediction ID from result
			lines := strings.Split(result, "\n")
			for _, line := range lines {
				if strings.Contains(line, "prediction_id:") {
					parts := strings.Split(line, ":")
					if len(parts) >= 2 {
						predictionID := strings.TrimSpace(parts[1])
						fmt.Printf("\nContinuing operation for prediction_id: %s\n", predictionID)
						
						// Wait for completion
						continueResult, err := server.continueOperation(ctx, map[string]interface{}{
							"prediction_id": predictionID,
							"wait_time":     30.0,
						})
						
						if err != nil {
							fmt.Printf("❌ Continue error: %v\n", err)
							os.Exit(1)
						}
						
						fmt.Printf("Final Result:\n%s\n", continueResult)
						break
					}
				}
			}
		}
		
		fmt.Printf("\n✅ Operation completed in %.2f seconds\n", time.Since(startTime).Seconds())
		return
	}
	
	// Handle image editing testing
	if editModel != "" {
		if inputImage == "" {
			fmt.Println("Error: -input flag is required when using -edit")
			fmt.Println("Usage: replicate_image_ai -edit <model> -input <image_path> [-eprompt \"editing prompt\"]")
			fmt.Println("Models: pro (recommended), max (highest quality), dev (advanced controls)")
			os.Exit(1)
		}
		
		// Default prompt if not provided
		if editPrompt == "" {
			editPrompt = "Make it a vintage photograph with sepia tones"
		}
		
		// Create server instance
		server, err := NewReplicateImageMCPServer()
		if err != nil {
			log.Fatalf("Failed to create server: %v", err)
		}
		
		ctx := context.Background()
		
		// Map model names
		modelName := "kontext-" + editModel
		if editModel != "pro" && editModel != "max" && editModel != "dev" {
			fmt.Printf("Invalid Kontext model: %s\n", editModel)
			fmt.Println("Valid models: pro, max, dev")
			os.Exit(1)
		}
		
		fmt.Printf("Testing FLUX Kontext %s\n", strings.ToUpper(editModel))
		fmt.Printf("Input image: %s\n", inputImage)
		fmt.Printf("Prompt: %s\n", editPrompt)
		fmt.Println("---")
		
		startTime := time.Now()
		
		// Prepare parameters
		params := map[string]interface{}{
			"file_path": inputImage,
			"prompt":    editPrompt,
			"model":     modelName,
		}
		
		// Add output filename if specified
		if outputFile != "" {
			params["filename"] = outputFile
		}
		
		// Add dev-specific parameters for testing
		if editModel == "dev" {
			params["go_fast"] = true
			params["guidance"] = 2.5
			params["num_inference_steps"] = 30.0
		}
		
		// Call the editImage function
		result, err := server.editImage(ctx, params)
		
		elapsed := time.Since(startTime)
		
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Printf("✅ Success! Time: %v\n", elapsed)
		fmt.Printf("Result:\n%s\n", result)
		
		// Check if we need to continue operation
		if strings.Contains(result, "prediction_id:") && strings.Contains(result, "PROCESSING") {
			// Extract prediction ID from result
			lines := strings.Split(result, "\n")
			for _, line := range lines {
				if strings.Contains(line, "prediction_id:") {
					parts := strings.Split(line, ":")
					if len(parts) >= 2 {
						predictionID := strings.TrimSpace(parts[1])
						fmt.Printf("\nContinuing operation for prediction_id: %s\n", predictionID)
						
						// Wait for completion
						continueResult, err := server.continueOperation(ctx, map[string]interface{}{
							"prediction_id": predictionID,
							"wait_time":     30.0,
						})
						
						if err != nil {
							fmt.Printf("❌ Continue error: %v\n", err)
							os.Exit(1)
						}
						
						fmt.Printf("Final Result:\n%s\n", continueResult)
						break
					}
				}
			}
		}
		
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