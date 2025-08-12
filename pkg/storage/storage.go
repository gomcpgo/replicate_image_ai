package storage

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
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
	// Determine filename
	if filename == "" {
		// Generate a simple filename based on ID
		ext := ".png" // default extension
		if strings.Contains(imageURL, ".jpg") || strings.Contains(imageURL, ".jpeg") {
			ext = ".jpg"
		} else if strings.Contains(imageURL, ".webp") {
			ext = ".webp"
		}
		filename = "image" + ext
	}

	imagePath := filepath.Join(s.rootPath, id, filename)

	// Download or decode the image
	var imageData []byte
	var err error

	if strings.HasPrefix(imageURL, "data:") {
		// Base64 encoded data
		parts := strings.SplitN(imageURL, ",", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid base64 data")
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

		imageData, err = io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read image data: %w", err)
		}
	}

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