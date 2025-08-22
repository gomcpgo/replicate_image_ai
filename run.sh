#!/bin/bash

# Build and run script for Replicate Image AI MCP Server

# Source .env file if it exists
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

case "$1" in
    "build")
        echo "Building Replicate Image AI MCP server..."
        go build -o bin/replicate_image_ai cmd/main.go cmd/enhancements.go
        echo "Build complete: bin/replicate_image_ai"
        ;;
    
    "test")
        echo "Running unit tests..."
        go test ./pkg/...
        ;;
    
    "integration-test")
        if [ -z "$REPLICATE_API_TOKEN" ]; then
            echo "Error: REPLICATE_API_TOKEN environment variable is required"
            exit 1
        fi
        echo "Running integration tests..."
        go test -v ./test -run TestAllModels -timeout 10m
        ;;
    
    "generate")
        # Generate image with specific model
        if [ -z "$2" ]; then
            echo "Usage: ./run.sh generate <model> [custom prompt]"
            echo "Models: flux-schnell, flux-dev, sdxl, seedream, ideogram, recraft, etc."
            exit 1
        fi
        if [ -z "$REPLICATE_API_TOKEN" ]; then
            echo "Error: REPLICATE_API_TOKEN environment variable is required"
            exit 1
        fi
        model="$2"
        prompt="${3:-}"
        if [ -n "$prompt" ]; then
            go run cmd/main.go cmd/enhancements.go -g "$model" -p "$prompt"
        else
            go run cmd/main.go cmd/enhancements.go -g "$model"
        fi
        ;;
    
    "list-models")
        go run cmd/main.go cmd/enhancements.go -list
        ;;
    
    "test-all")
        if [ -z "$REPLICATE_API_TOKEN" ]; then
            echo "Error: REPLICATE_API_TOKEN environment variable is required"
            exit 1
        fi
        go run cmd/main.go cmd/enhancements.go -test
        ;;
    
    "test-id")
        # Test a specific model ID directly
        if [ -z "$2" ]; then
            echo "Usage: ./run.sh test-id <model-id>"
            echo "Example: ./run.sh test-id stability-ai/stable-diffusion"
            exit 1
        fi
        if [ -z "$REPLICATE_API_TOKEN" ]; then
            echo "Error: REPLICATE_API_TOKEN environment variable is required"
            exit 1
        fi
        go run cmd/main.go cmd/enhancements.go -test-id "$2"
        ;;
    
    "kontext")
        # Test FLUX Kontext text-based image editing
        if [ -z "$2" ] || [ -z "$3" ]; then
            echo "Usage: ./run.sh kontext <model> <image_path> [prompt] [output_file]"
            echo ""
            echo "Models:"
            echo "  pro - Balanced speed/quality (recommended)"
            echo "  max - Highest quality, premium tier"
            echo "  dev - Advanced controls"
            echo ""
            echo "Examples:"
            echo "  ./run.sh kontext pro photo.jpg \"Make it a 90s cartoon\""
            echo "  ./run.sh kontext max car.jpg \"Change the car to red\""
            echo "  ./run.sh kontext dev landscape.jpg \"Add rain and fog\" rainy.png"
            exit 1
        fi
        if [ -z "$REPLICATE_API_TOKEN" ]; then
            echo "Error: REPLICATE_API_TOKEN environment variable is required"
            exit 1
        fi
        model="$2"
        image="$3"
        prompt="${4:-Make it a vintage photograph with sepia tones}"
        output="${5:-}"
        
        cmd="go run cmd/main.go cmd/enhancements.go -kontext $model -input \"$image\" -kprompt \"$prompt\""
        if [ -n "$output" ]; then
            cmd="$cmd -output \"$output\""
        fi
        eval $cmd
        ;;
    
    "enhance")
        # Test enhancement functions
        if [ -z "$2" ] || [ -z "$3" ]; then
            echo "Usage: ./run.sh enhance <tool> <image_path> [model/mask] [output_file] [prompt]"
            echo ""
            echo "Tools:"
            echo "  remove-bg - Remove background from image"
            echo "  upscale   - Upscale image to higher resolution"
            echo "  face      - Enhance faces in image"
            echo "  restore   - Restore old/damaged photos"
            echo "  edit      - Edit parts of image with AI inpainting"
            echo ""
            echo "Examples:"
            echo "  ./run.sh enhance remove-bg photo.jpg"
            echo "  ./run.sh enhance upscale photo.jpg realesrgan"
            echo "  ./run.sh enhance face portrait.jpg gfpgan"
            echo "  ./run.sh enhance restore old_photo.jpg"
            echo "  ./run.sh enhance edit image.jpg"
            echo "  ./run.sh enhance edit image.jpg mask.png"
            echo "  ./run.sh enhance edit image.jpg \"\" output.png \"Add a sunset\""
            exit 1
        fi
        if [ -z "$REPLICATE_API_TOKEN" ]; then
            echo "Error: REPLICATE_API_TOKEN environment variable is required"
            exit 1
        fi
        tool="$2"
        image="$3"
        model="${4:-}"
        output="${5:-}"
        prompt="${6:-}"
        
        cmd="go run cmd/main.go cmd/enhancements.go -enhance $tool -input $image"
        if [ -n "$model" ]; then
            cmd="$cmd -model $model"
        fi
        if [ -n "$output" ]; then
            cmd="$cmd -output $output"
        fi
        # For edit command, use -p flag for custom prompt
        if [ "$tool" = "edit" ] && [ -n "$prompt" ]; then
            cmd="$cmd -p \"$prompt\""
        fi
        eval $cmd
        ;;
    
    "run")
        echo "Running Replicate Image AI MCP server..."
        go run cmd/main.go cmd/enhancements.go
        ;;
    
    "clean")
        echo "Cleaning build artifacts..."
        rm -rf bin/
        rm -rf test_output/
        ;;
    
    *)
        echo "Replicate Image AI MCP Server Build Script"
        echo "=========================================="
        echo ""
        echo "Usage: $0 {build|test|integration-test|generate|kontext|enhance|list-models|test-all|test-id|run|clean}"
        echo ""
        echo "Commands:"
        echo "  build                       - Build the server binary"
        echo "  test                        - Run unit tests"
        echo "  integration-test            - Run integration tests"
        echo "  generate <model>            - Generate image with specific model"
        echo "  kontext <model> <image>     - Edit image with FLUX Kontext (text-based editing)"
        echo "  enhance <tool> <image>      - Enhance image with AI tools"
        echo "  list-models                 - List available models"
        echo "  test-all                    - Test all models"
        echo "  test-id <model-id>          - Test a specific model ID directly"
        echo "  run                         - Run the MCP server"
        echo "  clean                       - Remove build artifacts"
        echo ""
        echo "Examples:"
        echo "  $0 generate flux-schnell"
        echo "  $0 generate sdxl \"a beautiful landscape\""
        echo "  $0 kontext pro car.jpg \"Change the car to red\""
        echo "  $0 kontext max photo.jpg \"Make it a 90s cartoon\""
        echo "  $0 enhance remove-bg photo.jpg"
        echo "  $0 enhance upscale image.png realesrgan"
        echo "  $0 enhance face portrait.jpg gfpgan"
        echo "  $0 test-id stability-ai/stable-diffusion"
        echo "  $0 test-all"
        echo "  $0 list-models"
        exit 1
        ;;
esac