package editing

import "time"

// EditParams contains parameters for image editing
type EditParams struct {
	ImagePath    string
	Prompt       string  // Edit instruction
	Model        string  // pro, max, dev
	Strength     float64 // Edit strength (0.0-1.0)
	GuidanceScale float64 // Guidance scale for edit
	NumOutputs   int     // Number of variations
	Seed         int     // Random seed
	Filename     string  // Optional output filename
}

// EditResult contains the result of an image edit operation
type EditResult struct {
	ID           string
	Operation    string // "edit_image"
	InputPath    string
	OutputPath   string
	OutputURL    string
	Model        string
	ModelName    string
	EditPrompt   string
	Parameters   map[string]interface{}
	Metrics      EditMetrics
	PredictionID string
}

// EditMetrics contains performance metrics for editing
type EditMetrics struct {
	ProcessingTime float64 // in seconds
	InputSize      int64   // in bytes
	OutputSize     int64   // in bytes
}

// EditError represents an error during editing
type EditError struct {
	Code    string
	Message string
	Details map[string]interface{}
}

func (e EditError) Error() string {
	return e.Message
}

// EditMetadata contains metadata about an edited image
type EditMetadata struct {
	Version    string                 `json:"version"`
	ID         string                 `json:"id"`
	Operation  string                 `json:"operation"`
	Timestamp  time.Time              `json:"timestamp"`
	Model      string                 `json:"model"`
	Parameters map[string]interface{} `json:"parameters"`
	Result     interface{}            `json:"result,omitempty"`
}