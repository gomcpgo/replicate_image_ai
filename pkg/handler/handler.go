package handler

import (
	"context"
	"fmt"

	"github.com/gomcpgo/mcp/pkg/protocol"
	"github.com/gomcpgo/replicate_image_ai/pkg/client"
	"github.com/gomcpgo/replicate_image_ai/pkg/editing"
	"github.com/gomcpgo/replicate_image_ai/pkg/enhancement"
	"github.com/gomcpgo/replicate_image_ai/pkg/generation"
	"github.com/gomcpgo/replicate_image_ai/pkg/storage"
)

// ReplicateImageHandler handles MCP requests for image operations
type ReplicateImageHandler struct {
	generator *generation.Generator
	enhancer  *enhancement.Enhancer
	editor    *editing.Editor
	storage   *storage.Storage
	debug     bool
}

// NewReplicateImageHandler creates a new handler instance
func NewReplicateImageHandler(apiKey string, rootFolder string, debug bool) (*ReplicateImageHandler, error) {
	// Initialize storage
	store, err := storage.NewStorage(rootFolder)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}
	
	// Initialize Replicate client
	replicateClient := client.NewReplicateClient(apiKey)
	
	// Initialize core components
	gen := generation.NewGenerator(replicateClient, store, debug)
	enh := enhancement.NewEnhancer(replicateClient, store, debug)
	edit := editing.NewEditor(replicateClient, store, debug)
	
	return &ReplicateImageHandler{
		generator: gen,
		enhancer:  enh,
		editor:    edit,
		storage:   store,
		debug:     debug,
	}, nil
}

// CallTool handles execution of image tools
func (h *ReplicateImageHandler) CallTool(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResponse, error) {
	switch req.Name {
	// Generation tools
	case "generate_image":
		return h.handleGenerateImage(ctx, req.Arguments)
	case "generate_with_visual_context":
		return h.handleGenerateWithVisualContext(ctx, req.Arguments)
		
	// Enhancement tools
	case "remove_background":
		return h.handleRemoveBackground(ctx, req.Arguments)
	case "upscale_image":
		return h.handleUpscaleImage(ctx, req.Arguments)
	case "enhance_face":
		return h.handleEnhanceFace(ctx, req.Arguments)
	case "restore_photo":
		return h.handleRestorePhoto(ctx, req.Arguments)
		
	// Editing tools
	case "edit_image":
		return h.handleEditImage(ctx, req.Arguments)
		
	default:
		return nil, fmt.Errorf("unknown tool: %s", req.Name)
	}
}