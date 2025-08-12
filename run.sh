#!/bin/bash

# Build and run script for Replicate Image AI MCP Server

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
        echo "Running integration test..."
        go run cmd/main.go -test
        ;;
    
    "run")
        echo "Running Replicate Image AI MCP server..."
        go run cmd/main.go
        ;;
    
    "clean")
        echo "Cleaning build artifacts..."
        rm -rf bin/
        ;;
    
    *)
        echo "Replicate Image AI MCP Server Build Script"
        echo "=========================================="
        echo ""
        echo "Usage: $0 {build|test|integration-test|run|clean}"
        echo ""
        echo "Commands:"
        echo "  build           - Build the server binary"
        echo "  test            - Run unit tests"
        echo "  integration-test - Run integration tests"
        echo "  run             - Run the server"
        echo "  clean           - Remove build artifacts"
        exit 1
        ;;
esac