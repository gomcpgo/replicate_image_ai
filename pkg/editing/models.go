package editing

// Model IDs for image editing models on Replicate

// FLUX Kontext models for text-based image editing
const (
	ModelFluxKontextPro = "black-forest-labs/flux-kontext-pro"
	ModelFluxKontextMax = "black-forest-labs/flux-kontext-max"
	ModelFluxKontextDev = "black-forest-labs/flux-kontext-dev"
)

// ModelInfo contains information about an editing model
type ModelInfo struct {
	ID          string
	Name        string
	Description string
	Category    string
	Features    []string
}

// GetModelInfo returns information about an editing model
func GetModelInfo(modelID string) ModelInfo {
	models := map[string]ModelInfo{
		ModelFluxKontextPro: {
			ID:          ModelFluxKontextPro,
			Name:        "FLUX Kontext Pro",
			Description: "Professional text-based image editing with balanced speed and quality",
			Category:    "text-edit",
			Features:    []string{"balanced", "professional", "text-based", "fast"},
		},
		ModelFluxKontextMax: {
			ID:          ModelFluxKontextMax,
			Name:        "FLUX Kontext Max",
			Description: "Maximum quality text-based image editing, premium tier",
			Category:    "text-edit",
			Features:    []string{"highest-quality", "premium", "text-based", "detailed"},
		},
		ModelFluxKontextDev: {
			ID:          ModelFluxKontextDev,
			Name:        "FLUX Kontext Dev",
			Description: "Development version with advanced controls for text-based editing",
			Category:    "text-edit",
			Features:    []string{"advanced-controls", "experimental", "text-based", "flexible"},
		},
	}
	
	if info, ok := models[modelID]; ok {
		return info
	}
	
	// Return basic info for unknown models
	return ModelInfo{
		ID:       modelID,
		Name:     "Unknown Model",
		Category: "unknown",
	}
}

// GetModelFromAlias returns the model ID from common aliases
func GetModelFromAlias(alias string) string {
	switch alias {
	case "pro", "kontext-pro", "flux-kontext-pro":
		return ModelFluxKontextPro
	case "max", "kontext-max", "flux-kontext-max":
		return ModelFluxKontextMax
	case "dev", "kontext-dev", "flux-kontext-dev":
		return ModelFluxKontextDev
	default:
		return ModelFluxKontextPro // Default to Pro
	}
}