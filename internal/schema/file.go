package schema

import (
	"time"
)

// TokenFile represents a token file with cached schema version
type TokenFile struct {
	// Path is the file path
	Path string

	// Content is the raw file content
	Content []byte

	// SchemaVersion is the detected schema version (cached)
	SchemaVersion SchemaVersion

	// LoadedAt is when the file was loaded
	LoadedAt time.Time
}

// NewTokenFile creates a new TokenFile with schema detection and validation
func NewTokenFile(path string, content []byte, config *DetectionConfig) (*TokenFile, error) {
	// Detect schema version with validation
	version, err := DetectVersionWithValidation(path, content, config)
	if err != nil {
		// If validation fails, don't create the file
		return nil, err
	}

	return &TokenFile{
		Path:          path,
		Content:       content,
		SchemaVersion: version,
		LoadedAt:      time.Now(),
	}, nil
}
