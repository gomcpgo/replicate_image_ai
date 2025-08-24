#!/bin/bash

# Build and run script for Replicate Image AI MCP Server

# Source .env file if it exists
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

case "$1" in
    "build")
        echo "Building Replicate Image AI MCP server..."
        go build -o bin/replicate_image_ai ./cmd
        echo "Build complete: bin/replicate_image_ai"
        ;;
    
    "test")
        if [ "$2" = "async" ]; then
            echo "Running async tests with short timeouts..."
            export REPLICATE_INITIAL_WAIT=1
            export REPLICATE_CONTINUE_WAIT=2
            go test -v ./pkg/generation -run TestGenerateImage
        elif [ "$2" = "all" ]; then
            echo "Running all unit tests..."
            go test -v ./pkg/...
        elif [ "$2" = "quick" ]; then
            echo "Running quick unit tests (no network)..."
            go test -short ./pkg/...
        else
            echo "Running unit tests..."
            go test ./pkg/...
        fi
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
            go run ./cmd -g "$model" -p "$prompt"
        else
            go run ./cmd -g "$model"
        fi
        ;;
    
    "list-models")
        go run ./cmd -list
        ;;
    
    "test-all")
        if [ -z "$REPLICATE_API_TOKEN" ]; then
            echo "Error: REPLICATE_API_TOKEN environment variable is required"
            exit 1
        fi
        go run ./cmd -test
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
        go run ./cmd -test-id "$2"
        ;;
    
    "gen4")
        # Test RunwayML Gen-4 with reference images
        if [ -z "$REPLICATE_API_TOKEN" ]; then
            echo "Error: REPLICATE_API_TOKEN environment variable is required"
            exit 1
        fi
        if [ -z "$2" ] || [ -z "$3" ]; then
            echo "Usage: ./run.sh gen4 <ref_images> <ref_tags> [prompt] [aspect_ratio] [resolution]"
            echo ""
            echo "Examples:"
            echo "  ./run.sh gen4 'person.jpg' 'person' '@person in a coffee shop'"
            echo "  ./run.sh gen4 'person.jpg,product.jpg' 'person,product' '@person holding @product'"
            echo "  ./run.sh gen4 'img1.jpg,img2.jpg,img3.jpg' 'woman,robot,room' '@woman and @robot in @room'"
            exit 1
        fi
        
        ref_images="$2"
        ref_tags="$3"
        prompt="${4:-@${ref_tags%%,*} in a modern office setting with natural lighting}"
        aspect="${5:-16:9}"
        resolution="${6:-1080p}"
        
        echo "Testing RunwayML Gen-4..."
        echo "Reference Images: $ref_images"
        echo "Reference Tags: $ref_tags"
        echo "Prompt: $prompt"
        echo "Aspect Ratio: $aspect"
        echo "Resolution: $resolution"
        
        go run ./cmd -gen4 -ref-images "$ref_images" -ref-tags "$ref_tags" -aspect "$aspect" -resolution "$resolution" -p "$prompt"
        ;;
    
    "imagen4")
        # Test Google Imagen-4 photorealistic generation
        if [ -z "$REPLICATE_API_TOKEN" ]; then
            echo "Error: REPLICATE_API_TOKEN environment variable is required"
            exit 1
        fi
        prompt="${2:-A photorealistic portrait of a young woman with vibrant red hair, looking directly at the camera with a warm smile, soft golden hour lighting streaming through a window, shallow depth of field with blurred background, shot on professional camera}"
        aspect="${3:-16:9}"
        safety="${4:-block_only_high}"
        
        echo "Testing Google Imagen-4..."
        echo "Prompt: $prompt"
        echo "Aspect Ratio: $aspect"
        echo "Safety Filter: $safety"
        
        go run ./cmd -imagen4 -aspect "$aspect" -safety "$safety" -p "$prompt"
        ;;
    
    "edit")
        # Test image editing with FLUX Kontext
        if [ -z "$2" ] || [ -z "$3" ]; then
            echo "Usage: ./run.sh edit <model> <image_path> [prompt] [output_file]"
            echo ""
            echo "Models:"
            echo "  pro - Balanced speed/quality (recommended)"
            echo "  max - Highest quality, premium tier"
            echo "  dev - Advanced controls"
            echo ""
            echo "Examples:"
            echo "  ./run.sh edit pro photo.jpg \"Make it a 90s cartoon\""
            echo "  ./run.sh edit max car.jpg \"Change the car to red\""
            echo "  ./run.sh edit dev landscape.jpg \"Add rain and fog\" rainy.png"
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
        
        cmd="go run ./cmd -edit $model -input \"$image\" -eprompt \"$prompt\""
        if [ -n "$output" ]; then
            cmd="$cmd -output \"$output\""
        fi
        eval $cmd
        ;;
    
    "enhance")
        # Test enhancement functions
        if [ -z "$2" ] || [ -z "$3" ]; then
            echo "Usage: ./run.sh enhance <tool> <image_path> [model] [output_file]"
            echo ""
            echo "Tools:"
            echo "  remove-bg - Remove background from image"
            echo "  upscale   - Upscale image to higher resolution"
            echo "  face      - Enhance faces in image"
            echo "  restore   - Restore old/damaged photos"
            echo ""
            echo "Examples:"
            echo "  ./run.sh enhance remove-bg photo.jpg"
            echo "  ./run.sh enhance upscale photo.jpg realesrgan"
            echo "  ./run.sh enhance face portrait.jpg gfpgan"
            echo "  ./run.sh enhance restore old_photo.jpg"
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
        
        cmd="go run ./cmd -enhance $tool -input $image"
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
        go run ./cmd
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
        echo "Usage: $0 {build|test|integration-test|generate|gen4|imagen4|edit|enhance|list-models|test-all|test-id|run|clean}"
        echo ""
        echo "Commands:"
        echo "  build                       - Build the server binary"
        echo "  test [async|all|quick]      - Run unit tests (async: test async flow, all: verbose, quick: skip slow tests)"
        echo "  integration-test            - Run integration tests"
        echo "  generate <model>            - Generate image with specific model"
        echo "  gen4 <images> <tags>        - Generate with RunwayML Gen-4 using reference images"
        echo "  imagen4 [prompt] [aspect]   - Generate with Google Imagen-4 photorealistic model"
        echo "  edit <model> <image>        - Edit image with FLUX Kontext (text-based editing)"
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
        echo "  $0 gen4 'person.jpg,car.jpg' 'person,car' '@person standing next to @car'"
        echo "  $0 imagen4"
        echo "  $0 imagen4 \"A photorealistic cat\" 16:9"
        echo "  $0 edit pro car.jpg \"Change the car to red\""
        echo "  $0 edit max photo.jpg \"Make it a 90s cartoon\""
        echo "  $0 enhance remove-bg photo.jpg"
        echo "  $0 enhance upscale image.png realesrgan"
        echo "  $0 enhance face portrait.jpg gfpgan"
        echo "  $0 test-id stability-ai/stable-diffusion"
        echo "  $0 test-all"
        echo "  $0 list-models"
        exit 1
        ;;
esac