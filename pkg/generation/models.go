package generation

// Model IDs for image generation models on Replicate
const (
	// FLUX models - High quality, fast generation
	ModelFluxSchnell = "black-forest-labs/flux-schnell"
	ModelFluxPro     = "black-forest-labs/flux-1.1-pro"
	ModelFluxDev     = "black-forest-labs/flux-dev"
	
	// Google Imagen
	ModelImagen4 = "google/imagen-4"
	
	// RunwayML Gen-4
	ModelGen4Image = "runwayml/gen4-image"
	
	// Stable Diffusion models
	ModelSDXL          = "stability-ai/sdxl:39ed52f2a78e934b3ba6e2a89f5b1c712de7dfea535525255b1aa35c5565e08b"
	ModelSDXLLightning = "bytedance/sdxl-lightning-4step:5599ed30703defd1d160a25a63321b4dec97101d98b4674bcc56e41f62f35637"
	
	// Ideogram model
	ModelIdeogramTurbo = "ideogram-ai/ideogram-turbo"
	
	// Recraft models
	ModelRecraft    = "recraft-ai/recraft-v3"
	ModelRecraftSVG = "recraft-ai/recraft-v3-svg"
	
	// Seedream model
	ModelSeedream3 = "viktorfa/seedream-3:847dc86c09e3e95f20ae908ad3e991b10e0e29e24d0ddce8f5e31b42bc16b49c"
)

// ModelInfo contains information about a model
type ModelInfo struct {
	ID          string
	Name        string
	Description string
	Category    string
	Features    []string
}

// GetModelInfo returns information about a model
func GetModelInfo(modelID string) ModelInfo {
	models := map[string]ModelInfo{
		ModelFluxSchnell: {
			ID:          ModelFluxSchnell,
			Name:        "FLUX Schnell",
			Description: "Fast, high-quality image generation",
			Category:    "flux",
			Features:    []string{"fast", "high-quality", "versatile"},
		},
		ModelFluxPro: {
			ID:          ModelFluxPro,
			Name:        "FLUX Pro",
			Description: "Professional-grade image generation with advanced controls",
			Category:    "flux",
			Features:    []string{"professional", "advanced-controls", "high-resolution"},
		},
		ModelFluxDev: {
			ID:          ModelFluxDev,
			Name:        "FLUX Dev",
			Description: "Development version with experimental features",
			Category:    "flux",
			Features:    []string{"experimental", "cutting-edge"},
		},
		ModelImagen4: {
			ID:          ModelImagen4,
			Name:        "Google Imagen-4",
			Description: "Photorealistic image generation with aspect ratio control",
			Category:    "photorealistic",
			Features:    []string{"photorealistic", "aspect-ratio", "safety-filter"},
		},
		ModelGen4Image: {
			ID:          ModelGen4Image,
			Name:        "RunwayML Gen-4",
			Description: "Advanced generation with visual context and reference images",
			Category:    "advanced",
			Features:    []string{"reference-images", "visual-context", "style-transfer"},
		},
		ModelSDXL: {
			ID:          ModelSDXL,
			Name:        "Stable Diffusion XL",
			Description: "High-resolution image generation with fine control",
			Category:    "stable-diffusion",
			Features:    []string{"high-resolution", "fine-control", "negative-prompt"},
		},
		ModelSDXLLightning: {
			ID:          ModelSDXLLightning,
			Name:        "SDXL Lightning",
			Description: "Ultra-fast 4-step SDXL generation",
			Category:    "stable-diffusion",
			Features:    []string{"ultra-fast", "4-step", "efficient"},
		},
		ModelIdeogramTurbo: {
			ID:          ModelIdeogramTurbo,
			Name:        "Ideogram Turbo",
			Description: "Fast generation with excellent text rendering",
			Category:    "specialized",
			Features:    []string{"text-rendering", "fast", "creative"},
		},
		ModelRecraft: {
			ID:          ModelRecraft,
			Name:        "Recraft V3",
			Description: "Design-focused generation for professional graphics",
			Category:    "design",
			Features:    []string{"design", "professional", "graphics"},
		},
		ModelRecraftSVG: {
			ID:          ModelRecraftSVG,
			Name:        "Recraft V3 SVG",
			Description: "Vector graphics generation in SVG format",
			Category:    "design",
			Features:    []string{"vector", "svg", "scalable"},
		},
		ModelSeedream3: {
			ID:          ModelSeedream3,
			Name:        "Seedream 3",
			Description: "Artistic and creative image generation",
			Category:    "artistic",
			Features:    []string{"artistic", "creative", "stylized"},
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
	case "flux-schnell", "flux", "schnell":
		return ModelFluxSchnell
	case "flux-pro", "pro":
		return ModelFluxPro
	case "flux-dev", "dev":
		return ModelFluxDev
	case "imagen-4", "imagen":
		return ModelImagen4
	case "gen4-image", "gen4", "runway":
		return ModelGen4Image
	case "sdxl":
		return ModelSDXL
	case "sdxl-lightning", "lightning":
		return ModelSDXLLightning
	case "ideogram", "ideogram-turbo":
		return ModelIdeogramTurbo
	case "recraft":
		return ModelRecraft
	case "recraft-svg":
		return ModelRecraftSVG
	case "seedream", "seedream-3":
		return ModelSeedream3
	default:
		return ModelFluxSchnell // Default fallback
	}
}