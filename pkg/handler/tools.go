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
			Description: `Generate images from text prompts using various AI models including FLUX, Stable Diffusion XL, Google Imagen-4, and more. Each model has unique strengths - FLUX for speed and quality, SDXL for fine control, Imagen-4 for photorealism, Ideogram for text rendering, and Recraft for design work.`,
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
			Description: `Generate images using RunwayML Gen-4 with visual reference images for maintaining consistent visual elements across generated images. This tool excels at preserving character identity, object appearance, and style consistency. Use @tags in your prompt to reference specific images (e.g., "@person in a coffee shop" where "person" is the tag for a reference image of a specific person).`,
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"prompt": {
						"type": "string",
						"description": "Generation prompt with @tag references. Use @tag to reference specific images (e.g., '@character in @location with @object')"
					},
					"reference_images": {
						"type": "array",
						"items": {
							"type": "string"
						},
						"description": "Array of 1-3 local image file paths to use as visual references",
						"minItems": 1,
						"maxItems": 3
					},
					"reference_tags": {
						"type": "array",
						"items": {
							"type": "string"
						},
						"description": "Tags for each reference image (3-15 alphanumeric characters). These tags are used with @ in the prompt to reference specific images."
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
			Description: `Edit images using text instructions with FLUX Kontext models. Transform existing images through natural language commands like "Make it a winter scene", "Change the car to red", or "Convert to cartoon style". Three model variants available: pro (balanced speed/quality), max (highest quality), and dev (experimental features).`,
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"file_path": {
						"type": "string",
						"description": "Path to the image file to edit"
					},
					"prompt": {
						"type": "string",
						"description": "Text instruction describing how to edit the image (e.g., 'Make it a sunset scene', 'Change hair color to blonde', 'Add snow')"
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
			Description: "Remove or replace the background of an image using AI models. Produces a transparent PNG or can replace with a new background.",
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
			Description: "Upscale images to higher resolution using AI super-resolution models. Can enhance details and optionally improve faces.",
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
			Description: "Enhance and restore faces in images using specialized AI models. Improves facial details, removes artifacts, and can restore old or damaged portraits.",
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
			Description: "Restore old, damaged, or low-quality photos using AI. Can remove scratches, enhance details, fix fading, and optionally colorize black and white images.",
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
	}
	
	return &protocol.ListToolsResponse{
		Tools: tools,
	}, nil
}