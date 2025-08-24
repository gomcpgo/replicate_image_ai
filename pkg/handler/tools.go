package handler

import (
	"context"
	"encoding/json"

	"github.com/gomcpgo/mcp/pkg/protocol"
)

// ListTools provides a list of all available tools
func (h *ReplicateImageHandler) ListTools(ctx context.Context) (*protocol.ListToolsResponse, error) {
	tools := []protocol.Tool{
		{
			Name:        "generate_image",
			Description: `[CREATE NEW] Generate completely new images from text prompts WITHOUT reference images. Use when: Starting fresh, creating first concepts, standalone images. NOT for: Creating variations, maintaining consistency with existing images. Examples: "a red cube", "sunset over mountains", "futuristic city". Models: FLUX (speed/quality), SDXL (control), Imagen-4 (photorealism), Ideogram (text), Recraft (design).`,
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"prompt": {
						"type": "string",
						"description": "Text description of the image to generate. Be descriptive for best results."
					},
					"model": {
						"type": "string",
						"description": "Model to use: flux-schnell (fast), flux-pro (professional), flux-dev (experimental), sdxl (detailed), sdxl-lightning (ultra-fast), ideogram (text rendering), recraft (design), seedream (artistic), imagen-4 (photorealistic), gen4-image (visual context)",
						"default": "flux-schnell"
					},
					"width": {
						"type": "integer",
						"description": "Image width in pixels (most models). Common sizes: 512, 768, 1024. Not used for Imagen-4 or Gen-4.",
						"default": 1024
					},
					"height": {
						"type": "integer",
						"description": "Image height in pixels (most models). Common sizes: 512, 768, 1024. Not used for Imagen-4 or Gen-4.",
						"default": 1024
					},
					"aspect_ratio": {
						"type": "string",
						"description": "Aspect ratio for Imagen-4 and Gen-4 models: 1:1, 16:9, 9:16, 4:3, 3:4",
						"enum": ["1:1", "16:9", "9:16", "4:3", "3:4"]
					},
					"resolution": {
						"type": "string",
						"description": "Resolution for Gen-4 model: 720p, 1080p",
						"enum": ["720p", "1080p"],
						"default": "1080p"
					},
					"guidance_scale": {
						"type": "number",
						"description": "How closely to follow the prompt (1-20). Higher values = more literal interpretation.",
						"default": 7.5
					},
					"negative_prompt": {
						"type": "string",
						"description": "What to avoid in the image (SDXL and similar models only)"
					},
					"seed": {
						"type": "integer",
						"description": "Random seed for reproducible results"
					},
					"num_outputs": {
						"type": "integer",
						"description": "Number of images to generate (1-4)",
						"default": 1,
						"minimum": 1,
						"maximum": 4
					},
					"safety_filter_level": {
						"type": "string",
						"description": "Safety filter level for Imagen-4: block_low_and_above, block_medium_and_above, block_only_high",
						"enum": ["block_low_and_above", "block_medium_and_above", "block_only_high"],
						"default": "block_only_high"
					},
					"output_format": {
						"type": "string",
						"description": "Output format for Imagen-4: jpg, png",
						"enum": ["jpg", "png"],
						"default": "jpg"
					},
					"filename": {
						"type": "string",
						"description": "Custom filename for the generated image"
					}
				},
				"required": ["prompt"]
			}`),
		},
		{
			Name:        "generate_with_visual_context",
			Description: `[CREATE VARIATIONS] Generate variations, iterations, or consistent series from existing images using RunwayML Gen-4. Use when: Need multiple views of same subject, maintaining character/object identity, consistent style across images. NOT for: Simple edits, changing attributes of existing image. Examples: Character in different poses/scenes, product from multiple angles, consistent art style series. Requires 1-3 reference images with @tags in prompt (e.g., "@person at beach", "@product on shelf"). IMPORTANT: This may return 'processing' status - if so, use continue_operation tool with the provided prediction_id, DO NOT call this tool again!`,
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"prompt": {
						"type": "string",
						"description": "Transformation prompt with @tag references. Describe the new scene/pose/context using @tags to maintain visual identity (e.g., '@character sitting', '@product on table')"
					},
					"reference_images": {
						"type": "array",
						"items": {
							"type": "string"
						},
						"description": "Source images (1-3) whose visual identity, appearance, or style you want to maintain in the new generation",
						"minItems": 1,
						"maxItems": 3
					},
					"reference_tags": {
						"type": "array",
						"items": {
							"type": "string"
						},
						"description": "Short labels for each image (3-15 chars), used with @ in prompt to specify where/how to use each reference (must match number of images)"
					},
					"aspect_ratio": {
						"type": "string",
						"description": "Output aspect ratio",
						"enum": ["1:1", "16:9", "9:16", "4:3", "3:4"],
						"default": "16:9"
					},
					"resolution": {
						"type": "string",
						"description": "Output resolution",
						"enum": ["720p", "1080p"],
						"default": "1080p"
					},
					"seed": {
						"type": "integer",
						"description": "Random seed for reproducible results"
					},
					"filename": {
						"type": "string",
						"description": "Custom filename for the generated image"
					}
				},
				"required": ["prompt", "reference_images", "reference_tags"]
			}`),
		},
		{
			Name:        "edit_image",
			Description: `[MODIFY EXISTING] Transform images while preserving original composition using FLUX Kontext. Use when: Changing style/colors/weather/time, adding/removing elements, style transfer. NOT for: Creating variations, multiple views, or new compositions. Examples: "Make it winter", "Change to cartoon style", "Add rain", "Make it nighttime". Three models: pro (balanced), max (quality), dev (experimental).`,
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"file_path": {
						"type": "string",
						"description": "Path to the original image you want to modify (composition will be preserved)"
					},
					"prompt": {
						"type": "string",
						"description": "Transformation instruction describing the change, not the final result (e.g., 'Make it vintage', 'Change season to autumn', 'Convert to oil painting')"
					},
					"model": {
						"type": "string",
						"description": "FLUX Kontext model variant: pro (balanced), max (highest quality), dev (experimental)",
						"enum": ["pro", "max", "dev"],
						"default": "pro"
					},
					"strength": {
						"type": "number",
						"description": "Edit strength (0.0-1.0). Higher values make more dramatic changes.",
						"minimum": 0,
						"maximum": 1,
						"default": 0.8
					},
					"guidance_scale": {
						"type": "number",
						"description": "How closely to follow the edit prompt (1-20)",
						"default": 7.5
					},
					"seed": {
						"type": "integer",
						"description": "Random seed for reproducible results"
					},
					"filename": {
						"type": "string",
						"description": "Custom filename for the edited image"
					}
				},
				"required": ["file_path", "prompt"]
			}`),
		},
		{
			Name:        "remove_background",
			Description: "[EXTRACT SUBJECT] Remove or replace image backgrounds using AI. Use when: Isolating subjects, creating transparent PNGs, changing backgrounds. Models: remove-bg (fast), rembg (robust), dis (detailed).",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"file_path": {
						"type": "string",
						"description": "Path to the image file"
					},
					"model": {
						"type": "string",
						"description": "Model to use: remove-bg (fast), rembg (robust), dis (detailed)",
						"enum": ["remove-bg", "rembg", "dis"],
						"default": "remove-bg"
					},
					"filename": {
						"type": "string",
						"description": "Custom filename for the output image"
					}
				},
				"required": ["file_path"]
			}`),
		},
		{
			Name:        "upscale_image",
			Description: "[ENHANCE RESOLUTION] Increase image resolution and quality using AI. Use when: Making images larger, improving details, preparing for print. Scales: 2x, 4x, 8x. Models: realesrgan (general), esrgan (detailed), swinir (flexible).",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"file_path": {
						"type": "string",
						"description": "Path to the image file to upscale"
					},
					"scale": {
						"type": "integer",
						"description": "Upscale factor (2, 4, or 8)",
						"enum": [2, 4, 8],
						"default": 4
					},
					"model": {
						"type": "string",
						"description": "Model to use: realesrgan (general), esrgan (detailed), swinir (flexible)",
						"enum": ["realesrgan", "esrgan", "swinir"],
						"default": "realesrgan"
					},
					"face_enhance": {
						"type": "boolean",
						"description": "Enhance faces during upscaling (RealESRGAN only)",
						"default": false
					},
					"filename": {
						"type": "string",
						"description": "Custom filename for the upscaled image"
					}
				},
				"required": ["file_path"]
			}`),
		},
		{
			Name:        "enhance_face",
			Description: "[IMPROVE FACES] Enhance facial details and quality using specialized AI. Use when: Fixing blurry faces, restoring portraits, improving selfies. Models: gfpgan (balanced), codeformer (versatile), restoreformer (natural).",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"file_path": {
						"type": "string",
						"description": "Path to the image file containing faces"
					},
					"model": {
						"type": "string",
						"description": "Model to use: gfpgan (balanced), codeformer (versatile), restoreformer (natural)",
						"enum": ["gfpgan", "codeformer", "restoreformer"],
						"default": "gfpgan"
					},
					"fidelity": {
						"type": "number",
						"description": "Balance between quality and faithfulness to original (0.0-1.0). Higher = more faithful.",
						"minimum": 0,
						"maximum": 1,
						"default": 0.5
					},
					"only_center": {
						"type": "boolean",
						"description": "Only enhance the center face in the image",
						"default": false
					},
					"background_enhance": {
						"type": "boolean",
						"description": "Also enhance the background (CodeFormer only)",
						"default": false
					},
					"filename": {
						"type": "string",
						"description": "Custom filename for the enhanced image"
					}
				},
				"required": ["file_path"]
			}`),
		},
		{
			Name:        "restore_photo",
			Description: "[RESTORE QUALITY] Repair old or damaged photos using AI. Use when: Fixing old photos, removing scratches, restoring faded images, colorizing B&W. Models: bopbtl (old photos), gfpgan (faces), codeformer (general).",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"file_path": {
						"type": "string",
						"description": "Path to the photo to restore"
					},
					"model": {
						"type": "string",
						"description": "Model to use: bopbtl (old photos), gfpgan (faces), codeformer (general)",
						"enum": ["bopbtl", "gfpgan", "codeformer"],
						"default": "bopbtl"
					},
					"face_enhance": {
						"type": "boolean",
						"description": "Enhance faces during restoration",
						"default": true
					},
					"scratch_removal": {
						"type": "boolean",
						"description": "Remove scratches and damage (BOPBTL only)",
						"default": true
					},
					"colorize": {
						"type": "boolean",
						"description": "Colorize black and white photos",
						"default": false
					},
					"filename": {
						"type": "string",
						"description": "Custom filename for the restored image"
					}
				},
				"required": ["file_path"]
			}`),
		},
		{
			Name:        "continue_operation",
			Description: "[REQUIRED FOR ASYNC] ALWAYS use this tool when you receive 'processing' status with a prediction_id. DO NOT call the original tool again - that creates a NEW operation! This tool checks the status of an EXISTING operation. The message will explicitly say 'Use continue_operation with prediction_id=XXX'. You MUST use this exact prediction_id, not generate a new one.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"prediction_id": {
						"type": "string",
						"description": "The prediction ID from the previous operation that's still processing"
					},
					"wait_time": {
						"type": "integer",
						"description": "How many seconds to wait for completion (max 30)",
						"default": 30,
						"minimum": 5,
						"maximum": 30
					}
				},
				"required": ["prediction_id"]
			}`),
		},
	}
	
	return &protocol.ListToolsResponse{
		Tools: tools,
	}, nil
}