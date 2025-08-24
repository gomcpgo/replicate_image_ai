package handler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gomcpgo/mcp/pkg/protocol"
	"github.com/gomcpgo/replicate_image_ai/pkg/client"
	"github.com/gomcpgo/replicate_image_ai/pkg/editing"
	"github.com/gomcpgo/replicate_image_ai/pkg/enhancement"
	"github.com/gomcpgo/replicate_image_ai/pkg/generation"
	"github.com/gomcpgo/replicate_image_ai/pkg/storage"
)

// ReplicateImageHandler handles MCP requests for image operations
type ReplicateImageHandler struct {
	generator  *generation.Generator
	enhancer   *enhancement.Enhancer
	editor     *editing.Editor
	storage    *storage.Storage
	client     client.Client
	pendingOps *PendingOperationsManager
	debug      bool
}

// NewReplicateImageHandler creates a new handler instance
func NewReplicateImageHandler(apiKey string, rootFolder string, debug bool) (*ReplicateImageHandler, error) {
	// Initialize storage
	store := storage.NewStorage(rootFolder)
	
	// Initialize Replicate client
	replicateClient := client.NewReplicateClient(apiKey)
	
	// Initialize core components
	gen := generation.NewGenerator(replicateClient, store, debug)
	enh := enhancement.NewEnhancer(replicateClient, store, debug)
	edit := editing.NewEditor(replicateClient, store, debug)
	
	// Initialize pending operations manager
	pendingOps := NewPendingOperationsManager()
	
	return &ReplicateImageHandler{
		generator:  gen,
		enhancer:   enh,
		editor:     edit,
		storage:    store,
		client:     replicateClient,
		pendingOps: pendingOps,
		debug:      debug,
	}, nil
}

// CallTool handles execution of image tools
func (h *ReplicateImageHandler) CallTool(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResponse, error) {
	log.Printf("DEBUG: MCP CallTool received: %s", req.Name)
	deadline, hasDeadline := ctx.Deadline()
	if hasDeadline {
		log.Printf("DEBUG: Context has deadline: %v (in %v)", deadline, time.Until(deadline))
	} else {
		log.Printf("DEBUG: Context has no deadline")
	}
	
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
	
	// Async operation management
	case "continue_operation":
		return h.handleContinueOperation(ctx, req.Arguments)
		
	default:
		return nil, fmt.Errorf("unknown tool: %s", req.Name)
	}
}