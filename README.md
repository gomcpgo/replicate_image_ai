# Replicate Image AI MCP Server

An MCP (Model Context Protocol) server that provides AI-powered image generation, enhancement, and editing capabilities through the Replicate API.

## Features

### Currently Implemented
- **Image Generation**: Generate AI images from text prompts using various models (Flux, SDXL, Ideogram, etc.)
- **Image Editing**: Transform images using natural language instructions with FLUX Kontext (no masks needed)
- **Face Enhancement**: Restore and enhance faces in photos
- **Image Upscaling**: Increase resolution using AI super-resolution
- **Background Removal**: Remove or replace backgrounds
- **Photo Restoration**: Restore old or damaged photos
- **Continuation Pattern**: Handle long-running operations with a 30-second timeout and continuation mechanism
- **Local Storage**: All images are stored locally with metadata in YAML format
- **Image Management**: List and retrieve generated images with full metadata

### Coming Soon
- **Batch Processing**: Process multiple images sequentially

## Prerequisites

- Go 1.21 or higher
- Replicate API token (get one at https://replicate.com)

## Installation

1. Clone the repository:
```bash
git clone https://github.com/gomcpgo/replicate_image_ai
cd replicate_image_ai
```

2. Build the server:
```bash
./run.sh build
```

## Configuration

Set the following environment variables:

### Required
```bash
export REPLICATE_API_TOKEN="your-api-token"       # Your Replicate API token
export REPLICATE_IMAGES_ROOT_FOLDER="/path/to/images"  # Where to store generated images
```

### Optional
```bash
export MAX_IMAGE_SIZE_MB=5                # Maximum image size in MB (default: 5)
export MAX_BATCH_SIZE=10                  # Maximum batch size (default: 10)
export OPERATION_TIMEOUT_SECONDS=30       # Operation timeout in seconds (default: 30)
export DEBUG_MODE=false                   # Enable debug logging (default: false)
```

## Usage

### Running the Server

```bash
# Run directly
./run.sh run

# Or run the built binary
./bin/replicate_image_ai
```

### MCP Client Configuration

Add to your MCP client configuration:

```json
{
  "mcpServers": {
    "replicate-image-ai": {
      "command": "/path/to/replicate_image_ai/bin/replicate_image_ai",
      "env": {
        "REPLICATE_API_TOKEN": "your-api-token",
        "REPLICATE_IMAGES_ROOT_FOLDER": "/path/to/images"
      }
    }
  }
}
```

## Available Tools

### generate_image
Generate an AI image from a text prompt.

**Parameters:**
- `prompt` (required): Text description of the desired image
- `model`: Model to use (flux-schnell, flux-pro, flux-dev, imagen-4, seedream-3, sdxl, ideogram-turbo)
- `width`: Image width in pixels (default: 1024) - Note: imagen-4 uses aspect_ratio instead
- `height`: Image height in pixels (default: 1024) - Note: imagen-4 uses aspect_ratio instead
- `aspect_ratio`: Aspect ratio for imagen-4 only (1:1, 9:16, 16:9, 3:4, 4:3)
- `safety_filter_level`: Safety filter for imagen-4 only (block_low_and_above, block_medium_and_above, block_only_high)
- `output_format`: Output format for imagen-4 only (jpg, png)
- `filename`: Optional filename for the generated image
- `seed`: Seed for reproducible generation
- `guidance_scale`: How closely to follow the prompt (1-20, default: 7.5) - Not supported by imagen-4
- `negative_prompt`: What to avoid in the image - Not supported by imagen-4

**Example (Standard models):**
```json
{
  "prompt": "A beautiful sunset over mountains",
  "model": "flux-schnell",
  "width": 1024,
  "height": 1024
}
```

**Example (Imagen-4):**
```json
{
  "prompt": "A photorealistic portrait of a cat with striking green eyes",
  "model": "imagen-4",
  "aspect_ratio": "1:1",
  "safety_filter_level": "block_only_high",
  "output_format": "jpg"
}
```

### continue_operation
Continue waiting for an in-progress operation.

**Parameters:**
- `prediction_id` (required): The prediction ID from a previous operation
- `wait_time`: How many seconds to wait (max 30, default: 30)

### list_images
List all generated/processed images.

**Returns:** JSON array of image information including ID, operation, timestamp, file path, and metadata.

### get_image
Get details about a specific image.

**Parameters:**
- `id` (required): The image ID

**Returns:** Full image details including metadata and file path.

### edit_image
Edit images using natural language instructions with FLUX Kontext models. Transform entire images without masks.

**Parameters:**
- `file_path` (required): Path to the local image file to edit
- `prompt` (required): Text instruction describing the desired changes
- `model`: Model variant - "kontext-pro" (recommended), "kontext-max" (highest quality), "kontext-dev" (advanced)
- `aspect_ratio`: Output aspect ratio (default: "match_input_image")
- `prompt_upsampling`: Auto-enhance prompt for better results (Pro/Max only)
- `safety_tolerance`: Content filter level 0-2 (default: 2)
- `output_format`: "png", "jpg", or "webp" (default: "png")
- `go_fast`: Speed up generation (Dev model only)
- `guidance`: Guidance strength 0-10 (Dev model only, default: 2.5)
- `num_inference_steps`: Number of steps 1-50 (Dev model only, default: 30)
- `seed`: Seed for reproducible generation
- `filename`: Optional output filename

**Example Prompts:**
- "Make it a 90s cartoon"
- "Change the car to red"
- "Make it nighttime with rain"
- "Convert to oil painting style"
- "Add sunglasses to the person"
- "Make the text 3D and glowing"

## Storage Structure

Images are stored in the following structure:
```
REPLICATE_IMAGES_ROOT_FOLDER/
├── abc12345/                 # Unique 8-character ID
│   ├── metadata.yaml         # Operation metadata
│   └── image.jpg            # Generated image
├── def67890/
│   ├── metadata.yaml
│   └── sunset.png
```

## Model Information

### Generation Models
- **flux-schnell**: Fast generation, good quality (default)
- **flux-pro**: Best quality, slower
- **flux-dev**: Development version
- **imagen-4**: Google's photorealistic model with superior text rendering and fine details
- **seedream-3**: State-of-the-art quality
- **sdxl**: Stable Diffusion XL
- **ideogram-turbo**: Best for text in images

### FLUX Kontext Models (Text-based Image Editing)
- **kontext-pro**: Balanced speed and quality (recommended default)
- **kontext-max**: Highest quality, premium tier (higher cost)
- **kontext-dev**: Advanced controls with more parameters

## Development

### Project Structure
```
replicate_image_ai/
├── cmd/
│   └── main.go              # Main server entry point
├── pkg/
│   ├── config/             # Configuration management
│   ├── types/              # Type definitions
│   ├── client/             # Replicate API client
│   └── storage/            # Local storage management
├── go.mod
├── go.sum
├── run.sh                  # Build and run script
└── README.md
```

### Building
```bash
./run.sh build              # Build the server
./run.sh test              # Run unit tests
./run.sh integration-test  # Run integration tests
./run.sh clean             # Clean build artifacts
```

### Testing
```bash
# Run unit tests
go test ./pkg/...

# Run integration test (requires API token)
./run.sh integration-test
```

## Error Handling

The server implements a fail-fast approach:
- Operations timeout after 30 seconds
- Failed operations return clear error messages
- Partial results are returned for batch operations
- All errors are logged when DEBUG_MODE is enabled

## Cost Considerations

Replicate charges per prediction. Approximate costs:
- flux-schnell: ~$0.003 per image
- flux-pro: ~$0.055 per image
- sdxl: ~$0.020 per image

Monitor your usage at https://replicate.com/account/billing

## Contributing

1. Follow the existing code structure
2. Add tests for new features
3. Update documentation
4. Test with `./run.sh test` before submitting

## License

[Add your license here]

## Support

For issues or questions:
1. Check the troubleshooting section
2. Review error messages in debug mode
3. Consult the Replicate API documentation
4. Open an issue on GitHub