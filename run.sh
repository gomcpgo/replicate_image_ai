#!/bin/bash

# Build and run script for Replicate Image AI MCP Server

# Source .env file if it exists
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

case "$1" in
    "build")
        echo "Building Replicate Image AI MCP server..."
        go build -o bin/replicate_image_ai cmd/main.go
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
            go run cmd/main.go -g "$model" -p "$prompt"
        else
            go run cmd/main.go -g "$model"
        fi
        ;;
    
    "list-models")
        go run cmd/main.go -list
        ;;
    
    "test-all")
        if [ -z "$REPLICATE_API_TOKEN" ]; then
            echo "Error: REPLICATE_API_TOKEN environment variable is required"
            exit 1
        fi
        go run cmd/main.go -test
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
        go run cmd/main.go -test-id "$2"
        ;;
    
    "run")
        echo "Running Replicate Image AI MCP server..."
        go run cmd/main.go
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
        echo "Usage: $0 {build|test|integration-test|generate|list-models|test-all|test-id|run|clean}"
        echo ""
        echo "Commands:"
        echo "  build                - Build the server binary"
        echo "  test                 - Run unit tests"
        echo "  integration-test     - Run integration tests"
        echo "  generate <model>     - Generate image with specific model"
        echo "  list-models          - List available models"
        echo "  test-all             - Test all models"
        echo "  test-id <model-id>   - Test a specific model ID directly"
        echo "  run                  - Run the MCP server"
        echo "  clean                - Remove build artifacts"
        echo ""
        echo "Examples:"
        echo "  $0 generate flux-schnell"
        echo "  $0 generate sdxl \"a beautiful landscape\""
        echo "  $0 test-id stability-ai/stable-diffusion"
        echo "  $0 test-all"
        echo "  $0 list-models"
        exit 1
        ;;
esac