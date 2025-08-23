package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gomcpgo/mcp/pkg/handler"
	"github.com/gomcpgo/mcp/pkg/protocol"
	"github.com/gomcpgo/mcp/pkg/server"
	"github.com/gomcpgo/replicate_image_ai/pkg/config"
	replhandler "github.com/gomcpgo/replicate_image_ai/pkg/handler"
)

const version = "2.0.0"

func main() {
	// Parse command line flags
	var (
		generateModel string
		listModels    bool
		versionFlag   bool
		testEnhance   string
		inputImage    string
		enhanceModel  string
		outputFile    string
		editModel     string
		editPrompt    string
		imagen4Flag   bool
		aspectRatio   string
		safetyFilter  string
		gen4Flag      bool
		refImages     string
		refTags       string
		resolution    string
		prompt        string
	)

	flag.StringVar(&generateModel, "g", "", "Generate an image using specified model")
	flag.BoolVar(&listModels, "list", false, "List all available models")
	flag.BoolVar(&versionFlag, "version", false, "Show version information")
	flag.StringVar(&prompt, "p", "", "Custom prompt for generation")
	
	// Enhancement flags
	flag.StringVar(&testEnhance, "enhance", "", "Test enhancement tool: remove-bg, upscale, face, restore")
	flag.StringVar(&inputImage, "input", "", "Input image path for enhancement tests")
	flag.StringVar(&enhanceModel, "model", "", "Model to use for enhancement")
	flag.StringVar(&outputFile, "output", "", "Output filename for enhanced image")
	
	// Edit image flags
	flag.StringVar(&editModel, "edit", "", "Test image editing with FLUX Kontext: pro, max, or dev")
	flag.StringVar(&editPrompt, "eprompt", "", "Prompt for image editing")
	
	// Imagen-4 flags
	flag.BoolVar(&imagen4Flag, "imagen4", false, "Test Google Imagen-4 photorealistic generation")
	flag.StringVar(&aspectRatio, "aspect", "16:9", "Aspect ratio for Imagen-4")
	flag.StringVar(&safetyFilter, "safety", "block_only_high", "Safety filter for Imagen-4")
	
	// Gen-4 flags
	flag.BoolVar(&gen4Flag, "gen4", false, "Test RunwayML Gen-4 with visual context")
	flag.StringVar(&refImages, "ref-images", "", "Comma-separated reference image paths")
	flag.StringVar(&refTags, "ref-tags", "", "Comma-separated reference tags")
	flag.StringVar(&resolution, "resolution", "1080p", "Resolution for Gen-4")

	flag.Parse()

	if versionFlag {
		fmt.Printf("Replicate Image AI MCP Server v%s\n", version)
		return
	}

	// Terminal mode operations
	if listModels || generateModel != "" || testEnhance != "" || editModel != "" || imagen4Flag || gen4Flag {
		// Get API key from environment
		apiKey := os.Getenv("REPLICATE_API_TOKEN")
		if apiKey == "" {
			log.Fatal("REPLICATE_API_TOKEN environment variable is required")
		}
		
		// Get root folder from environment or use default
		rootFolder := os.Getenv("REPLICATE_IMAGES_ROOT_FOLDER")
		if rootFolder == "" {
			homeDir, _ := os.UserHomeDir()
			rootFolder = fmt.Sprintf("%s/Library/Application Support/Savant/replicate_image_ai", homeDir)
		}
		
		// Create handler for terminal operations
		h, err := replhandler.NewReplicateImageHandler(apiKey, rootFolder, true)
		if err != nil {
			log.Fatalf("Failed to create handler: %v", err)
		}
		
		ctx := context.Background()
		
		// Handle terminal mode operations
		if listModels {
			listAvailableModels()
			return
		}
		
		if generateModel != "" {
			runGeneration(ctx, h, generateModel, prompt)
			return
		}
		
		if testEnhance != "" {
			runEnhancement(ctx, h, testEnhance, inputImage, enhanceModel, outputFile)
			return
		}
		
		if editModel != "" {
			runEdit(ctx, h, editModel, inputImage, editPrompt, outputFile)
			return
		}
		
		if imagen4Flag {
			runImagen4(ctx, h, prompt, aspectRatio, safetyFilter)
			return
		}
		
		if gen4Flag {
			runGen4(ctx, h, prompt, refImages, refTags, aspectRatio, resolution)
			return
		}
		
		return
	}

	// MCP Server mode
	fmt.Println("Starting Replicate Image AI MCP Server...")
	
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	
	// Create handler
	h, err := replhandler.NewReplicateImageHandler(cfg.ReplicateAPIToken, cfg.ReplicateImagesRoot, cfg.DebugMode)
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}
	
	// Create handler registry
	registry := handler.NewHandlerRegistry()
	registry.RegisterToolHandler(h)
	
	// Create and start server
	srv := server.New(server.Options{
		Name:     "replicate-image-ai",
		Version:  version,
		Registry: registry,
	})
	
	log.Printf("Server started (version %s)", version)
	if err := srv.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func listAvailableModels() {
	fmt.Println("\n=== Available Models ===\n")
	fmt.Println("Generation Models:")
	fmt.Println("  flux-schnell    - Fast generation (default)")
	fmt.Println("  flux-dev        - Development version")
	fmt.Println("  flux-pro        - High quality (paid)")
	fmt.Println("  imagen-4        - Google's photorealistic model")
	fmt.Println("  gen4-image      - RunwayML Gen-4 with visual context")
	fmt.Println("  sdxl            - Stable Diffusion XL")
	fmt.Println("  sdxl-lightning  - Fast SDXL variant")
	fmt.Println("  seedream        - High quality generation")
	fmt.Println("  ideogram        - Text in images")
	fmt.Println("  recraft         - Raster images")
	fmt.Println("  recraft-svg     - SVG generation")
	fmt.Println()
	fmt.Println("Enhancement Models:")
	fmt.Println("  remove-bg       - Background removal")
	fmt.Println("  rembg           - Alternative background removal")
	fmt.Println("  realesrgan      - Image upscaling")
	fmt.Println("  gfpgan          - Face enhancement")
	fmt.Println("  codeformer      - Face restoration")
	fmt.Println("  bopbtl          - Old photo restoration")
	fmt.Println()
	fmt.Println("Edit Models (FLUX Kontext):")
	fmt.Println("  pro             - Balanced speed/quality")
	fmt.Println("  max             - Highest quality")
	fmt.Println("  dev             - Advanced controls")
}

func runGeneration(ctx context.Context, h *replhandler.ReplicateImageHandler, model, prompt string) {
	if prompt == "" {
		prompt = "A beautiful sunset over mountains with a lake in the foreground"
	}
	
	fmt.Printf("Generating image with %s...\n", model)
	fmt.Printf("Prompt: %s\n", prompt)
	
	req := &protocol.CallToolRequest{
		Name: "generate_image",
		Arguments: map[string]interface{}{
			"prompt": prompt,
			"model":  model,
		},
	}
	
	resp, err := h.CallTool(ctx, req)
	if err != nil {
		log.Fatalf("Generation failed: %v", err)
	}
	
	printResponse(resp)
}

func runEnhancement(ctx context.Context, h *replhandler.ReplicateImageHandler, tool, inputPath, model, outputFile string) {
	if inputPath == "" {
		log.Fatal("Input image path is required for enhancement")
	}
	
	var toolName string
	args := map[string]interface{}{
		"file_path": inputPath,
	}
	
	switch tool {
	case "remove-bg":
		toolName = "remove_background"
		if model != "" {
			args["model"] = model
		}
	case "upscale":
		toolName = "upscale_image"
		if model != "" {
			args["model"] = model
		}
		args["scale"] = 2
	case "face":
		toolName = "enhance_face"
		if model != "" {
			args["model"] = model
		}
	case "restore":
		toolName = "restore_photo"
		if model != "" {
			args["model"] = model
		}
	default:
		log.Fatalf("Unknown enhancement tool: %s", tool)
	}
	
	if outputFile != "" {
		args["filename"] = outputFile
	}
	
	fmt.Printf("Running %s on %s...\n", toolName, inputPath)
	
	req := &protocol.CallToolRequest{
		Name:      toolName,
		Arguments: args,
	}
	
	resp, err := h.CallTool(ctx, req)
	if err != nil {
		log.Fatalf("Enhancement failed: %v", err)
	}
	
	printResponse(resp)
}

func runEdit(ctx context.Context, h *replhandler.ReplicateImageHandler, model, inputPath, editPrompt, outputFile string) {
	if inputPath == "" {
		log.Fatal("Input image path is required for editing")
	}
	
	if editPrompt == "" {
		editPrompt = "Make it a vintage photograph with sepia tones"
	}
	
	args := map[string]interface{}{
		"file_path": inputPath,
		"prompt":    editPrompt,
		"model":     model,
	}
	
	if outputFile != "" {
		args["filename"] = outputFile
	}
	
	fmt.Printf("Editing image with FLUX Kontext %s...\n", model)
	fmt.Printf("Edit prompt: %s\n", editPrompt)
	
	req := &protocol.CallToolRequest{
		Name:      "edit_image",
		Arguments: args,
	}
	
	resp, err := h.CallTool(ctx, req)
	if err != nil {
		log.Fatalf("Edit failed: %v", err)
	}
	
	printResponse(resp)
}

func runImagen4(ctx context.Context, h *replhandler.ReplicateImageHandler, prompt, aspectRatio, safetyFilter string) {
	if prompt == "" {
		prompt = "A photorealistic portrait of a young woman with vibrant red hair"
	}
	
	fmt.Println("Testing Google Imagen-4...")
	fmt.Printf("Prompt: %s\n", prompt)
	fmt.Printf("Aspect Ratio: %s\n", aspectRatio)
	
	req := &protocol.CallToolRequest{
		Name: "generate_image",
		Arguments: map[string]interface{}{
			"prompt":               prompt,
			"model":                "imagen-4",
			"aspect_ratio":         aspectRatio,
			"safety_filter_level":  safetyFilter,
		},
	}
	
	resp, err := h.CallTool(ctx, req)
	if err != nil {
		log.Fatalf("Imagen-4 generation failed: %v", err)
	}
	
	printResponse(resp)
}

func runGen4(ctx context.Context, h *replhandler.ReplicateImageHandler, prompt, refImages, refTags, aspectRatio, resolution string) {
	if refImages == "" || refTags == "" {
		log.Fatal("Reference images and tags are required for Gen-4")
	}
	
	images := strings.Split(refImages, ",")
	tags := strings.Split(refTags, ",")
	
	if len(images) != len(tags) {
		log.Fatal("Number of reference images must match number of tags")
	}
	
	if prompt == "" {
		// Generate default prompt using first tag
		prompt = fmt.Sprintf("@%s in a modern office setting", tags[0])
	}
	
	fmt.Println("Testing RunwayML Gen-4...")
	fmt.Printf("Prompt: %s\n", prompt)
	fmt.Printf("Reference Images: %s\n", refImages)
	fmt.Printf("Reference Tags: %s\n", refTags)
	
	req := &protocol.CallToolRequest{
		Name: "generate_with_visual_context",
		Arguments: map[string]interface{}{
			"prompt":           prompt,
			"reference_images": images,
			"reference_tags":   tags,
			"aspect_ratio":     aspectRatio,
			"resolution":       resolution,
		},
	}
	
	resp, err := h.CallTool(ctx, req)
	if err != nil {
		log.Fatalf("Gen-4 generation failed: %v", err)
	}
	
	printResponse(resp)
}

func printResponse(resp *protocol.CallToolResponse) {
	for _, content := range resp.Content {
		if content.Type == "text" {
			fmt.Println(content.Text)
		}
	}
}