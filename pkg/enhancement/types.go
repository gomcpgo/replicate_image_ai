package enhancement

import "time"

// RemoveBackgroundParams contains parameters for background removal
type RemoveBackgroundParams struct {
	ImagePath string
	Model     string // remove-bg, rembg, dis
	Alpha     bool   // Preserve alpha channel
	Filename  string // Optional output filename
}

// UpscaleParams contains parameters for image upscaling
type UpscaleParams struct {
	ImagePath   string
	Scale       int    // Upscale factor (2, 4, 8)
	Model       string // realesrgan, esrgan, swinir
	FaceEnhance bool   // Enhance faces during upscaling
	Filename    string // Optional output filename
}

// EnhanceFaceParams contains parameters for face enhancement
type EnhanceFaceParams struct {
	ImagePath      string
	Model          string  // gfpgan, codeformer, restoreformer
	Fidelity       float64 // 0.0-1.0, higher = more faithful to original
	OnlyCenter     bool    // Only enhance center face
	HasAligned     bool    // Whether faces are already aligned
	BackgroundEnhance bool  // Also enhance background
	Filename       string  // Optional output filename
}

// RestorePhotoParams contains parameters for photo restoration
type RestorePhotoParams struct {
	ImagePath      string
	Model          string  // bopbtl, gfpgan, codeformer
	Fidelity       float64 // For models that support it
	FaceEnhance    bool    // Enhance faces during restoration
	Colorize       bool    // Colorize black and white photos
	ScratchRemoval bool    // Remove scratches
	Filename       string  // Optional output filename
}

// EnhancementResult contains the result of an enhancement operation
type EnhancementResult struct {
	ID           string
	Operation    string // "remove_background", "upscale", "enhance_face", "restore_photo"
	InputPath    string
	OutputPath   string
	OutputURL    string
	Model        string
	ModelName    string
	Parameters   map[string]interface{}
	Metrics      EnhancementMetrics
	PredictionID string
}

// EnhancementMetrics contains performance metrics
type EnhancementMetrics struct {
	ProcessingTime float64 // in seconds
	InputSize      int64   // in bytes
	OutputSize     int64   // in bytes
	ScaleFactor    int     // For upscaling
}

// EnhancementError represents an error during enhancement
type EnhancementError struct {
	Code    string
	Message string
	Details map[string]interface{}
}

func (e EnhancementError) Error() string {
	return e.Message
}

// EnhancementMetadata contains metadata about an enhanced image
type EnhancementMetadata struct {
	Version    string                 `json:"version"`
	ID         string                 `json:"id"`
	Operation  string                 `json:"operation"`
	Timestamp  time.Time              `json:"timestamp"`
	Model      string                 `json:"model"`
	Parameters map[string]interface{} `json:"parameters"`
	Result     interface{}            `json:"result,omitempty"`
}