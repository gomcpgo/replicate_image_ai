package storage

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gomcpgo/replicate_image_ai/pkg/types"
	"gopkg.in/yaml.v3"
)

// Storage handles local file storage for images
type Storage struct {
	rootPath string
}

// NewStorage creates a new storage instance
func NewStorage(rootPath string) *Storage {
	return &Storage{
		rootPath: rootPath,
	}
}

// detectImageFormat detects the image format from content and metadata
func detectImageFormat(data []byte, contentType string, url string) string {
	// 1. Try Content-Type header first (most reliable for HTTP responses)
	contentType = strings.ToLower(contentType)
	switch {
	case strings.Contains(contentType, "image/jpeg") || strings.Contains(contentType, "image/jpg"):
		return ".jpg"
	case strings.Contains(contentType, "image/png"):
		return ".png"
	case strings.Contains(contentType, "image/webp"):
		return ".webp"
	case strings.Contains(contentType, "image/gif"):
		return ".gif"
	case strings.Contains(contentType, "image/bmp"):
		return ".bmp"
	}
	
	// 2. Check magic bytes (file signatures) - most reliable for actual content
	if len(data) >= 12 {
		// WebP: RIFF....WEBP
		if bytes.HasPrefix(data, []byte("RIFF")) && len(data) >= 12 {
			if string(data[8:12]) == "WEBP" {
				return ".webp"
			}
		}
		
		// PNG: 89 50 4E 47 0D 0A 1A 0A
		if bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
			return ".png"
		}
		
		// JPEG: FF D8 FF
		if bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}) {
			return ".jpg"
		}
		
		// GIF: GIF87a or GIF89a
		if bytes.HasPrefix(data, []byte("GIF8")) {
			return ".gif"
		}
		
		// BMP: BM
		if bytes.HasPrefix(data, []byte{0x42, 0x4D}) {
			return ".bmp"
		}
	}
	
	// 3. Try to parse from URL as fallback
	urlLower := strings.ToLower(url)
	if strings.Contains(urlLower, ".webp") {
		return ".webp"
	}
	if strings.Contains(urlLower, ".png") {
		return ".png"
	}
	if strings.Contains(urlLower, ".jpg") || strings.Contains(urlLower, ".jpeg") {
		return ".jpg"
	}
	if strings.Contains(urlLower, ".gif") {
		return ".gif"
	}
	if strings.Contains(urlLower, ".bmp") {
		return ".bmp"
	}
	
	// 4. Default to WebP for Replicate (most common output format)
	return ".webp"
}

// GenerateID generates a unique 8-character alphanumeric ID
func (s *Storage) GenerateID() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	const idLength = 8
	maxRetries := 100

	for i := 0; i < maxRetries; i++ {
		b := make([]byte, idLength)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}

		id := make([]byte, idLength)
		for j := 0; j < idLength; j++ {
			id[j] = charset[b[j]%byte(len(charset))]
		}

		idStr := string(id)
		
		// Check if this ID already exists
		idPath := filepath.Join(s.rootPath, idStr)
		if _, err := os.Stat(idPath); os.IsNotExist(err) {
			// ID is unique, create the directory
			if err := os.MkdirAll(idPath, 0755); err != nil {
				return "", fmt.Errorf("failed to create directory: %w", err)
			}
			return idStr, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique ID after %d attempts", maxRetries)
}

// SaveImage saves an image from a URL or base64 data
func (s *Storage) SaveImage(id string, imageURL string, filename string) (string, error) {
	// Download or decode the image first to detect format
	var imageData []byte
	var contentType string
	var err error

	if strings.HasPrefix(imageURL, "data:") {
		// Base64 encoded data
		parts := strings.SplitN(imageURL, ",", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid base64 data")
		}
		
		// Extract MIME type from data URL if present
		if len(parts[0]) > 5 {
			// Format: data:image/png;base64
			typeInfo := parts[0][5:] // Remove "data:"
			if idx := strings.Index(typeInfo, ";"); idx != -1 {
				contentType = typeInfo[:idx]
			}
		}
		
		imageData, err = base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return "", fmt.Errorf("failed to decode base64: %w", err)
		}
	} else {
		// URL - download the image
		resp, err := http.Get(imageURL)
		if err != nil {
			return "", fmt.Errorf("failed to download image: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("failed to download image: status %d", resp.StatusCode)
		}

		// Get Content-Type header
		contentType = resp.Header.Get("Content-Type")

		imageData, err = io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read image data: %w", err)
		}
	}

	// Detect the actual image format
	detectedExt := detectImageFormat(imageData, contentType, imageURL)
	log.Printf("[Storage] Detected image format: %s (Content-Type: %s, URL: %s)", detectedExt, contentType, imageURL)
	
	// Determine final filename
	if filename == "" {
		// Generate a simple filename with detected extension
		filename = "image" + detectedExt
	} else {
		// Check if filename already has an extension
		existingExt := filepath.Ext(filename)
		if existingExt == "" {
			// Add the detected extension
			filename = filename + detectedExt
			log.Printf("[Storage] Added extension to filename: %s", filename)
		} else {
			// Filename already has an extension
			// Log if it differs from detected format
			if existingExt != detectedExt {
				log.Printf("[Storage] Warning: Provided extension %s differs from detected %s", existingExt, detectedExt)
			}
		}
	}

	imagePath := filepath.Join(s.rootPath, id, filename)

	// Save the image
	if err := os.WriteFile(imagePath, imageData, 0644); err != nil {
		return "", fmt.Errorf("failed to save image: %w", err)
	}

	return imagePath, nil
}

// SaveMetadata saves metadata for an operation
func (s *Storage) SaveMetadata(id string, metadata *types.ImageMetadata) error {
	metadataPath := filepath.Join(s.rootPath, id, "metadata.yaml")
	
	// Ensure version is set
	if metadata.Version == "" {
		metadata.Version = "1.0"
	}
	
	// Ensure timestamp is set
	if metadata.Timestamp.IsZero() {
		metadata.Timestamp = time.Now()
	}

	data, err := yaml.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	return nil
}

// LoadMetadata loads metadata for an operation
func (s *Storage) LoadMetadata(id string) (*types.ImageMetadata, error) {
	metadataPath := filepath.Join(s.rootPath, id, "metadata.yaml")

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata types.ImageMetadata
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &metadata, nil
}

// ListImages lists all stored images
func (s *Storage) ListImages() ([]types.ImageInfo, error) {
	entries, err := os.ReadDir(s.rootPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.ImageInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var images []types.ImageInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		id := entry.Name()
		metadata, err := s.LoadMetadata(id)
		if err != nil {
			// Skip entries without valid metadata
			continue
		}

		// Find the image file
		imagePath := ""
		if metadata.Result != nil && metadata.Result.Filename != "" {
			imagePath = filepath.Join(s.rootPath, id, metadata.Result.Filename)
		} else {
			// Look for any image file
			files, _ := os.ReadDir(filepath.Join(s.rootPath, id))
			for _, file := range files {
				name := file.Name()
				if strings.HasSuffix(name, ".png") || strings.HasSuffix(name, ".jpg") || 
				   strings.HasSuffix(name, ".jpeg") || strings.HasSuffix(name, ".webp") {
					imagePath = filepath.Join(s.rootPath, id, name)
					break
				}
			}
		}

		images = append(images, types.ImageInfo{
			ID:        id,
			Operation: metadata.Operation,
			Timestamp: metadata.Timestamp,
			FilePath:  imagePath,
			Model:     metadata.Model,
			Metadata:  metadata.Parameters,
		})
	}

	return images, nil
}

// GetImagePath returns the full path to an image
func (s *Storage) GetImagePath(id string, filename string) string {
	return filepath.Join(s.rootPath, id, filename)
}

// ImageToBase64 converts an image file to base64 data URL
func ImageToBase64(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Detect MIME type
	mimeType := "image/png" // default
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".webp":
		mimeType = "image/webp"
	case ".gif":
		mimeType = "image/gif"
	case ".bmp":
		mimeType = "image/bmp"
	}

	// Check file size (5MB limit)
	if len(data) > 5*1024*1024 {
		return "", fmt.Errorf("image file too large (max 5MB)")
	}

	// Create data URL
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(data))
	return dataURL, nil
}