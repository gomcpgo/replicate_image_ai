package types

import (
	"time"
)

// Model IDs for Replicate models - ALL VERIFIED TO EXIST
const (
	// ============== GENERATION MODELS ==============
	// All of these are CONFIRMED to exist on Replicate
	
	ModelFluxSchnell   = "black-forest-labs/flux-schnell"        // Fast generation (default)
	ModelFluxDev       = "black-forest-labs/flux-dev"            // Development version  
	ModelFluxPro       = "black-forest-labs/flux-pro"            // High quality (paid)
	ModelSDXL          = "stability-ai/sdxl:7762fd07cf82c948538e41f63f77d685e02b063e37e496e96eefd46c929f9bdc"                     // Stable Diffusion XL
	ModelSDXLLightning = "bytedance/sdxl-lightning-4step:6f7a773af6fc3e8de9d5a3c00be77c17308914bf67772726aff83496ba1e3bbe"        // Fast SDXL variant
	ModelSeedream3     = "bytedance/seedream-3"                  // High quality
	ModelIdeogramTurbo = "ideogram-ai/ideogram-v3-turbo"         // Text in images
	ModelRecraft       = "recraft-ai/recraft-v3"                 // Raster images
	ModelRecraftSVG    = "recraft-ai/recraft-v3-svg"             // SVG generation
	
	// ============== ENHANCEMENT MODELS ==============
	// All with version hashes for reliability
	
	ModelGFPGAN        = "tencentarc/gfpgan:297a243ce8643961d52f745f9b6c8c1bd96850a51c92be5f43628a0d3e08321a"                     // Face restoration
	ModelCodeFormer    = "sczhou/codeformer:cc4956dd26fa5a7185d5660cc9100fab1b8070a1d1654a8bb5eb6d443b020bb2"                     // Face enhancement  
	ModelRealESRGAN    = "nightmareai/real-esrgan:f121d640bd286e1fdc67f9799164c1d5be36ff74576ee11c803ae5b665dd46aa"               // Image upscaling
	ModelClarityUpscaler = "philz1337x/clarity-upscaler:dfad41707589d68ecdccd1dfa600d55a208f9310748e44bfe35b4a6291453d5e"         // Advanced upscaling
	
	// ============== BACKGROUND REMOVAL ==============
	// All with version hashes for reliability
	
	ModelRemoveBG      = "lucataco/remove-bg:95fcc2a26d3899cd6c2691c900465aaeff466285a65c14638cc5f36f34befaf1"                    // Fast removal
	ModelRembg         = "cjwbw/rembg:fb8af171cfa1616ddcf1242c093f9c46bcada5ad4cf6f2fbe8b81b330ec5c003"                           // Alternative BG removal
	ModelDISBGRemoval  = "lucataco/dis-background-removal:5b67a0da2b417b71066754fa8550d668a12edccfe3f32de062f7e20ac0b5d55e"       // High accuracy removal
	
	// ============== IMAGE EDITING ==============
	
	ModelInpainting    = "stability-ai/stable-diffusion-inpainting:95b7223104132402a9ae91cc677285bc5eb997834bd2349fa486f53910fd68b3"  // SD inpainting
	
	// ============== FLUX KONTEXT TEXT-BASED EDITING ==============
	// Natural language image editing without masks
	
	ModelFluxKontextPro = "black-forest-labs/flux-kontext-pro"  // Balanced speed/quality (recommended default)
	ModelFluxKontextMax = "black-forest-labs/flux-kontext-max"  // Highest quality, premium tier
	ModelFluxKontextDev = "black-forest-labs/flux-kontext-dev"  // Advanced controls, more parameters
	
	// ============== PHOTO RESTORATION ==============
	
	ModelOldPhotoRestore = "microsoft/bringing-old-photos-back-to-life:c75db81db6cbd809d93cc3b7e7a088a351a3349c9fa02b6d393e35e0d51ba799"  // Old photo restoration
)

// Prediction statuses from Replicate
const (
	StatusStarting   = "starting"
	StatusProcessing = "processing"
	StatusSucceeded  = "succeeded"
	StatusFailed     = "failed"
	StatusCanceled   = "canceled"
)

// ImageMetadata represents the metadata stored for each operation
type ImageMetadata struct {
	Version     string                 `yaml:"version"`
	ID          string                 `yaml:"id"`
	Operation   string                 `yaml:"operation"`
	Timestamp   time.Time              `yaml:"timestamp"`
	Model       string                 `yaml:"model"`
	Parameters  map[string]interface{} `yaml:"parameters"`
	Result      *OperationResult       `yaml:"result,omitempty"`
	Error       *string                `yaml:"error,omitempty"`
}

// OperationResult contains the result of an operation
type OperationResult struct {
	Filename        string  `yaml:"filename"`
	GenerationTime  float64 `yaml:"generation_time"`
	CostEstimate    float64 `yaml:"cost_estimate,omitempty"`
	PredictionID    string  `yaml:"prediction_id"`
	Width           int     `yaml:"width,omitempty"`
	Height          int     `yaml:"height,omitempty"`
}

// ReplicatePredictionRequest represents a request to create a prediction
type ReplicatePredictionRequest struct {
	Version string                 `json:"version"`
	Input   map[string]interface{} `json:"input"`
	Webhook string                 `json:"webhook,omitempty"`
}

// ReplicatePredictionResponse represents the response from Replicate
type ReplicatePredictionResponse struct {
	ID          string                 `json:"id"`
	Version     string                 `json:"version"`
	Status      string                 `json:"status"`
	Input       map[string]interface{} `json:"input"`
	Output      interface{}            `json:"output"`
	Error       interface{}            `json:"error"`
	Logs        string                 `json:"logs"`
	CreatedAt   string                 `json:"created_at"`
	StartedAt   *string                `json:"started_at"`
	CompletedAt *string                `json:"completed_at"`
	URLs        struct {
		Get    string `json:"get"`
		Cancel string `json:"cancel"`
	} `json:"urls"`
}

// GenerateImageParams represents parameters for image generation
type GenerateImageParams struct {
	Prompt          string  `json:"prompt"`
	Model           string  `json:"model,omitempty"`
	CustomModelID   string  `json:"custom_model_id,omitempty"`
	Width           int     `json:"width,omitempty"`
	Height          int     `json:"height,omitempty"`
	AspectRatio     string  `json:"aspect_ratio,omitempty"`
	Seed            *int    `json:"seed,omitempty"`
	GuidanceScale   float64 `json:"guidance_scale,omitempty"`
	NegativePrompt  string  `json:"negative_prompt,omitempty"`
	NumInferenceSteps int   `json:"num_inference_steps,omitempty"`
	OutputFormat    string  `json:"output_format,omitempty"`
	Filename        string  `json:"filename,omitempty"`
	QualityPreset   string  `json:"quality_preset,omitempty"`
}

// EnhanceFaceParams represents parameters for face enhancement
type EnhanceFaceParams struct {
	FilePath          string  `json:"file_path"`
	EnhancementModel  string  `json:"enhancement_model,omitempty"`
	Fidelity         float64 `json:"fidelity,omitempty"`
	Upscale          int     `json:"upscale,omitempty"`
	BackgroundEnhance bool    `json:"background_enhance,omitempty"`
	Filename         string  `json:"filename,omitempty"`
}

// UpscaleImageParams represents parameters for image upscaling
type UpscaleImageParams struct {
	FilePath     string  `json:"file_path"`
	Scale        int     `json:"scale,omitempty"`
	Model        string  `json:"model,omitempty"`
	FaceEnhance  bool    `json:"face_enhance,omitempty"`
	Denoise      float64 `json:"denoise,omitempty"`
	Filename     string  `json:"filename,omitempty"`
}

// RemoveBackgroundParams represents parameters for background removal
type RemoveBackgroundParams struct {
	FilePath         string  `json:"file_path"`
	OutputFormat     string  `json:"output_format,omitempty"`
	BackgroundColor  string  `json:"background_color,omitempty"`
	Model            string  `json:"model,omitempty"`
	EdgeSmoothing    float64 `json:"edge_smoothing,omitempty"`
	ReturnMask       bool    `json:"return_mask,omitempty"`
	Filename         string  `json:"filename,omitempty"`
}

// EditImageParams represents parameters for image editing
type EditImageParams struct {
	FilePath         string  `json:"file_path"`
	MaskPath         string  `json:"mask_path,omitempty"`
	SelectionPrompt  string  `json:"selection_prompt,omitempty"`
	EditPrompt       string  `json:"edit_prompt"`
	EditMode         string  `json:"edit_mode,omitempty"`
	Strength         float64 `json:"strength,omitempty"`
	GuidanceScale    float64 `json:"guidance_scale,omitempty"`
	Filename         string  `json:"filename,omitempty"`
}

// ContinueOperationParams represents parameters for continuing an operation
type ContinueOperationParams struct {
	PredictionID string `json:"prediction_id"`
	WaitTime     int    `json:"wait_time,omitempty"`
}

// ListImagesResponse represents the response from list_images
type ListImagesResponse struct {
	Images []ImageInfo `json:"images"`
	Total  int         `json:"total"`
}

// ImageInfo represents information about a stored image
type ImageInfo struct {
	ID        string                 `json:"id"`
	Operation string                 `json:"operation"`
	Timestamp time.Time              `json:"timestamp"`
	FilePath  string                 `json:"file_path"`
	Model     string                 `json:"model,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// GetImageResponse represents the response from get_image
type GetImageResponse struct {
	ID       string         `json:"id"`
	FilePath string         `json:"file_path"`
	Metadata *ImageMetadata `json:"metadata"`
}