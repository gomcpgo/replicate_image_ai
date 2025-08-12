package types

import (
	"time"
)

// Model IDs for Replicate models
const (
	// Generation models
	ModelFluxSchnell = "black-forest-labs/flux-schnell:5599ed30703defd1d160a25a63321b4dec97101d98b4674bcc56e41f62f35637"
	ModelFluxPro     = "black-forest-labs/flux-pro"
	ModelFluxDev     = "black-forest-labs/flux-dev"
	ModelSeedream3   = "bytedance/seedream-3"
	ModelSDXL        = "stability-ai/sdxl:39ed52f2a78e934b3ba6e2a89f5b1c712de7dfea535525255b1aa35c5565e08b"
	ModelIdeogramTurbo = "ideogram-ai/ideogram-v3-turbo"
	ModelRecraft     = "recraft-ai/recraft-v3-svg"
	
	// Enhancement models
	ModelGFPGAN      = "tencentarc/gfpgan:0fbacf7afc6c144e5be9767cff80f25aff23e52b0708f17e20f9879b2f21516c"
	ModelCodeFormer  = "sczhou/codeformer:7de2ea26c616d5bf2245ad0d5e24f0ff9a6204578a5c876db53142edd9d2cd56"
	ModelRealESRGAN  = "nightmareai/real-esrgan:f121d640bd286e1fdc67f9799164c1d5be36ff74576ee11c803ae5b665dd46aa"
	
	// Editing models
	ModelRemoveBG    = "lucataco/remove-bg:95fcc2a26d3899cd6c2691c900465aaeff466285a65c14638cc5f36f34befaf1"
	ModelInpainting  = "stability-ai/stable-diffusion-inpainting"
	ModelOldPhoto    = "microsoft/bringing-old-photos-back-to-life"
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