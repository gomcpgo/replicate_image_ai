package generation

import "time"

// GenerateParams contains parameters for image generation
type GenerateParams struct {
	Prompt         string
	Model          string
	Width          int
	Height         int
	AspectRatio    string  // For Imagen4 and Gen4
	Resolution     string  // For Gen4
	Seed           int
	GuidanceScale  float64
	NegativePrompt string
	NumOutputs     int
	SafetyFilter   string  // For Imagen4
	OutputFormat   string  // For Imagen4
	Filename       string  // Optional filename hint
}

// Gen4Params contains parameters specific to Gen-4 with visual context
type Gen4Params struct {
	Prompt          string
	ReferenceImages []string // Local file paths
	ReferenceTags   []string // Tags for reference images
	AspectRatio     string
	Resolution      string
	Seed            int
	Filename        string // Optional filename hint
}

// ImageResult contains the result of an image generation
type ImageResult struct {
	ID           string
	FilePath     string
	URL          string
	Model        string
	ModelName    string
	Prompt       string
	Parameters   map[string]interface{}
	Metrics      GenerationMetrics
	PredictionID string
	Status       string // "completed" or "processing"
	StorageID    string // For async operations
}

// GenerationMetrics contains performance metrics
type GenerationMetrics struct {
	GenerationTime float64 // in seconds
	FileSize       int64   // in bytes
	Width          int
	Height         int
}

// GenerationStatus represents the status of a generation request
type GenerationStatus string

const (
	StatusStarting   GenerationStatus = "starting"
	StatusProcessing GenerationStatus = "processing"
	StatusSucceeded  GenerationStatus = "succeeded"
	StatusFailed     GenerationStatus = "failed"
	StatusCanceled   GenerationStatus = "canceled"
)

// GenerationError represents an error during generation
type GenerationError struct {
	Code    string
	Message string
	Details map[string]interface{}
}

func (e GenerationError) Error() string {
	return e.Message
}

// ImageMetadata contains metadata about a generated image
type ImageMetadata struct {
	Version    string                 `json:"version"`
	ID         string                 `json:"id"`
	Operation  string                 `json:"operation"`
	Timestamp  time.Time              `json:"timestamp"`
	Model      string                 `json:"model"`
	Parameters map[string]interface{} `json:"parameters"`
	Result     interface{}            `json:"result,omitempty"`
}