package enhancement

// Model IDs for enhancement models on Replicate

// Background removal models
const (
	ModelRemoveBG     = "pollinations/remove-bg-model:78409a3e5845eb27dffe672de6e8c8c2c3f12fa7419b48e93e8ae8fba3bb4d27"
	ModelRembg        = "cjwbw/rembg:fb8af171cfa1616ddcf1242c093f9c46bcada5ad4cf6f2fbe8b81b330ec5c003"
	ModelDISBGRemoval = "pollinations/dis-background-removal:a29bbfaa10cf0c99b2c8f10e5fa3c9cd4a29c47798f45f86c93ff7c96fb907fa"
)

// Upscaling models
const (
	ModelRealESRGAN = "nightmareai/real-esrgan:f121d640bd286e1fdc67f9799164c1d5be36ff74576ee11c803ae5b665dd46aa"
	ModelESRGAN     = "mv-lab/esrgan:7c2e97f640b7e199d5bb86d17dc4d1d6e317c0c45e1f6ac1c827e87b3c5b7c96"
	ModelSwinIR     = "jingyunliang/swinir:660d922d33153019e8c263a3bba265de882e7f4f70396546b6c9c8f9d47a021a"
)

// Face enhancement models
const (
	ModelGFPGAN         = "tencentarc/gfpgan:9283608cc6b7be6b65a8e44983db012355fde4132009bf99d976b2f0896856a3"
	ModelCodeFormer     = "sczhou/codeformer:7de2ea26c616d5bf2245ad0d5e24f0ff9a6204578a5c876db53142edd9d2cd56"
	ModelRestoreFormer  = "jingyunliang/restoreformer:65b8e87b48cbdc7e5e91703c8e18b5d2e4f20dcbc49f3c45cdba5e4c481e973c"
)

// Photo restoration models
const (
	ModelBOPBTL = "pollinations/bopbtl:52dd5a901af15c1c5c8c9f9b43e205a31bbc0e6a13802c53e09bf5be5cad40c9"
)

// ModelInfo contains information about an enhancement model
type ModelInfo struct {
	ID          string
	Name        string
	Description string
	Category    string
	Features    []string
}

// GetModelInfo returns information about an enhancement model
func GetModelInfo(modelID string) ModelInfo {
	models := map[string]ModelInfo{
		// Background removal models
		ModelRemoveBG: {
			ID:          ModelRemoveBG,
			Name:        "Remove BG",
			Description: "Fast and accurate background removal",
			Category:    "background-removal",
			Features:    []string{"fast", "accurate", "preserves-edges"},
		},
		ModelRembg: {
			ID:          ModelRembg,
			Name:        "Rembg",
			Description: "Robust background removal with U2-Net",
			Category:    "background-removal",
			Features:    []string{"robust", "u2-net", "high-quality"},
		},
		ModelDISBGRemoval: {
			ID:          ModelDISBGRemoval,
			Name:        "DIS Background Removal",
			Description: "Advanced background removal with DIS model",
			Category:    "background-removal",
			Features:    []string{"advanced", "dis-model", "detailed"},
		},
		
		// Upscaling models
		ModelRealESRGAN: {
			ID:          ModelRealESRGAN,
			Name:        "Real-ESRGAN",
			Description: "High-quality image upscaling with face enhancement",
			Category:    "upscaling",
			Features:    []string{"high-quality", "face-enhancement", "4x-upscale"},
		},
		ModelESRGAN: {
			ID:          ModelESRGAN,
			Name:        "ESRGAN",
			Description: "Enhanced Super-Resolution GAN for image upscaling",
			Category:    "upscaling",
			Features:    []string{"super-resolution", "gan", "detailed"},
		},
		ModelSwinIR: {
			ID:          ModelSwinIR,
			Name:        "SwinIR",
			Description: "Transformer-based image restoration and upscaling",
			Category:    "upscaling",
			Features:    []string{"transformer", "restoration", "flexible-scale"},
		},
		
		// Face enhancement models
		ModelGFPGAN: {
			ID:          ModelGFPGAN,
			Name:        "GFPGAN",
			Description: "Face restoration with generative facial prior",
			Category:    "face-enhancement",
			Features:    []string{"face-restoration", "generative", "high-fidelity"},
		},
		ModelCodeFormer: {
			ID:          ModelCodeFormer,
			Name:        "CodeFormer",
			Description: "Robust face restoration via discrete code modeling",
			Category:    "face-enhancement",
			Features:    []string{"robust", "code-modeling", "versatile"},
		},
		ModelRestoreFormer: {
			ID:          ModelRestoreFormer,
			Name:        "RestoreFormer",
			Description: "High-quality blind face restoration",
			Category:    "face-enhancement",
			Features:    []string{"blind-restoration", "high-quality", "natural"},
		},
		
		// Photo restoration models
		ModelBOPBTL: {
			ID:          ModelBOPBTL,
			Name:        "BOPBTL",
			Description: "Bringing old photos back to life",
			Category:    "photo-restoration",
			Features:    []string{"old-photos", "restoration", "colorization"},
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
func GetModelFromAlias(operation, alias string) string {
	switch operation {
	case "remove_background":
		switch alias {
		case "remove-bg", "removebg":
			return ModelRemoveBG
		case "rembg":
			return ModelRembg
		case "dis":
			return ModelDISBGRemoval
		default:
			return ModelRemoveBG // Default
		}
		
	case "upscale":
		switch alias {
		case "realesrgan", "real-esrgan":
			return ModelRealESRGAN
		case "esrgan":
			return ModelESRGAN
		case "swinir":
			return ModelSwinIR
		default:
			return ModelRealESRGAN // Default
		}
		
	case "enhance_face":
		switch alias {
		case "gfpgan":
			return ModelGFPGAN
		case "codeformer":
			return ModelCodeFormer
		case "restoreformer":
			return ModelRestoreFormer
		default:
			return ModelGFPGAN // Default
		}
		
	case "restore_photo":
		switch alias {
		case "bopbtl":
			return ModelBOPBTL
		case "gfpgan":
			return ModelGFPGAN
		case "codeformer":
			return ModelCodeFormer
		default:
			return ModelBOPBTL // Default
		}
		
	default:
		return ""
	}
}