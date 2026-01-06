package schema

import (
	"encoding/json"
)

// DetectionConfig provides configuration for schema version detection
type DetectionConfig struct {
	// DefaultVersion is used when no other detection method succeeds
	DefaultVersion SchemaVersion
}

// DetectVersion detects the schema version from file content
// Priority order:
// 1. $schema field in file root
// 2. Config default version
// 3. Duck typing (detect reserved fields/structured formats)
// 4. Default to draft (backward compatibility)
func DetectVersion(content []byte, config *DetectionConfig) (SchemaVersion, error) {
	// Parse JSON
	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return Unknown, NewSchemaDetectionError("", "invalid JSON: "+err.Error())
	}

	// 1. Check for explicit $schema field
	if schemaURL, ok := data["$schema"].(string); ok {
		version, err := FromURL(schemaURL)
		if err == nil {
			return version, nil
		}
		// Invalid schema URL - continue with other methods
	}

	// 2. Check config default
	if config != nil && config.DefaultVersion != Unknown {
		return config.DefaultVersion, nil
	}

	// 3. Duck typing - check for unambiguous 2025.10 features
	if version := duckTypeSchema(data); version != Unknown {
		return version, nil
	}

	// 4. Default to draft for backward compatibility
	return Draft, nil
}

// DetectVersionWithValidation detects schema version and validates consistency
func DetectVersionWithValidation(filePath string, content []byte, config *DetectionConfig) (SchemaVersion, error) {
	version, err := DetectVersion(content, config)
	if err != nil {
		return Unknown, err
	}

	// Validate schema consistency
	if err := ValidateSchemaConsistencyWithPath(filePath, content, version); err != nil {
		return version, err // Return version but also error
	}

	return version, nil
}

// duckTypeSchema attempts to detect schema version from content patterns
func duckTypeSchema(data map[string]interface{}) SchemaVersion {
	// Check for 2025.10-only reserved fields
	if hasFeature(data, "$ref") {
		return V2025_10
	}
	if hasFeature(data, "$extends") {
		return V2025_10
	}
	if hasFeature(data, "resolutionOrder") {
		return V2025_10
	}

	// Check for structured color objects (2025.10)
	if hasStructuredColorObjects(data) {
		return V2025_10
	}

	// No definitive 2025.10 indicators found
	// Return Unknown to signal ambiguity (caller will default to draft)
	return Unknown
}
