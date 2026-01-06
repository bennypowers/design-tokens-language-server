package schema

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ValidateSchemaConsistency validates that file content matches the detected schema version
func ValidateSchemaConsistency(content []byte, detectedVersion SchemaVersion) error {
	return ValidateSchemaConsistencyWithPath("", content, detectedVersion)
}

// ValidateSchemaConsistencyWithPath validates schema consistency with file path context
func ValidateSchemaConsistencyWithPath(filePath string, content []byte, detectedVersion SchemaVersion) error {
	// Parse JSON to inspect structure
	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Collect features that conflict with declared schema
	var conflictingFeatures []string
	hasColorFormatIssue := false

	// Check for 2025.10-only features in draft schema
	if detectedVersion == Draft {
		if hasStructuredColorObjects(data) {
			conflictingFeatures = append(conflictingFeatures, "structured color objects (2025.10+ only)")
			hasColorFormatIssue = true
		}

		if hasFeature(data, "$ref") {
			conflictingFeatures = append(conflictingFeatures, "$ref (2025.10+ only)")
		}
		if hasFeature(data, "$extends") {
			conflictingFeatures = append(conflictingFeatures, "$extends (2025.10+ only)")
		}
		if hasFeature(data, "resolutionOrder") {
			conflictingFeatures = append(conflictingFeatures, "resolutionOrder (2025.10+ only)")
		}
	}

	// Check for draft-only patterns in 2025.10 schema
	if detectedVersion == V2025_10 {
		if hasStringColorValues(data) {
			return NewInvalidColorFormatError(filePath, "color tokens", detectedVersion.String(), "string value", "structured object with colorSpace")
		}
	}

	// Check for conflicting root token patterns
	if err := validateRootTokens(filePath, data, detectedVersion); err != nil {
		return err
	}

	// Return error if conflicting features found
	if len(conflictingFeatures) > 0 {
		// If ONLY color format issue (no other features), return specific error
		if len(conflictingFeatures) == 1 && hasColorFormatIssue {
			return NewInvalidColorFormatError(filePath, "color tokens", detectedVersion.String(), "structured object with colorSpace", "string value")
		}

		// Multiple incompatible features - return general error
		return NewMixedSchemaFeaturesError(filePath, detectedVersion.String(), conflictingFeatures)
	}

	return nil
}

// hasFeature checks if a feature (field name) exists anywhere in the structure
func hasFeature(data map[string]interface{}, featureName string) bool {
	if _, exists := data[featureName]; exists {
		return true
	}

	// Recursively check nested objects
	for _, value := range data {
		if obj, ok := value.(map[string]interface{}); ok {
			if hasFeature(obj, featureName) {
				return true
			}
		}
	}

	return false
}

// hasStructuredColorObjects checks if file contains 2025.10-style structured color values
func hasStructuredColorObjects(data map[string]interface{}) bool {
	return checkForStructuredColors(data)
}

func checkForStructuredColors(obj interface{}) bool {
	switch v := obj.(type) {
	case map[string]interface{}:
		// Check if this is a color token with structured value
		if colorType, ok := v["$type"].(string); ok && colorType == "color" {
			if value, ok := v["$value"].(map[string]interface{}); ok {
				// Check for colorSpace field (2025.10 indicator)
				if _, hasColorSpace := value["colorSpace"]; hasColorSpace {
					return true
				}
			}
		}

		// Recursively check nested objects
		for _, child := range v {
			if checkForStructuredColors(child) {
				return true
			}
		}
	}

	return false
}

// hasStringColorValues checks if file contains draft-style string color values
func hasStringColorValues(data map[string]interface{}) bool {
	return checkForStringColors(data)
}

func checkForStringColors(obj interface{}) bool {
	switch v := obj.(type) {
	case map[string]interface{}:
		// Check if this is a color token with string value
		if colorType, ok := v["$type"].(string); ok && colorType == "color" {
			if value, ok := v["$value"].(string); ok && value != "" {
				return true
			}
		}

		// Recursively check nested objects
		for _, child := range v {
			if checkForStringColors(child) {
				return true
			}
		}
	}

	return false
}

// validateRootTokens checks for conflicting root token patterns
func validateRootTokens(filePath string, data map[string]interface{}, schemaVersion SchemaVersion) error {
	// Common group marker names (draft-only)
	groupMarkers := []string{"_", "@", "DEFAULT"}

	return checkGroupsForRootConflicts(filePath, data, "", schemaVersion, groupMarkers)
}

func checkGroupsForRootConflicts(filePath string, obj map[string]interface{}, currentPath string, schemaVersion SchemaVersion, groupMarkers []string) error {
	for key, value := range obj {
		// Skip reserved fields
		if strings.HasPrefix(key, "$") && key != "$root" {
			continue
		}

		// Check if this is a group (has nested tokens)
		if child, ok := value.(map[string]interface{}); ok {
			// Check if this group might be a token group
			if hasTokenChildren(child) {
				// Check for both $root and group markers
				hasRoot := false
				var foundMarker string

				if _, exists := child["$root"]; exists {
					hasRoot = true
				}

				for _, marker := range groupMarkers {
					if _, exists := child[marker]; exists {
						foundMarker = marker
						break
					}
				}

				// Error if both present
				if hasRoot && foundMarker != "" {
					groupPath := currentPath
					if groupPath == "" {
						groupPath = key
					} else {
						groupPath = groupPath + "." + key
					}
					return NewConflictingRootTokensError(filePath, groupPath, "$root", foundMarker)
				}

				// Warn if 2025.10 uses group markers
				if schemaVersion == V2025_10 && foundMarker != "" && !hasRoot {
					// This is an error in 2025.10 - should use $root
					return NewInvalidSchemaError(filePath, schemaVersion.String(),
						fmt.Sprintf("group '%s' uses draft-style marker '%s' instead of $root", key, foundMarker))
				}
			}

			// Recursively check nested groups
			newPath := key
			if currentPath != "" {
				newPath = currentPath + "." + key
			}
			if err := checkGroupsForRootConflicts(filePath, child, newPath, schemaVersion, groupMarkers); err != nil {
				return err
			}
		}
	}

	return nil
}

// hasTokenChildren checks if an object has children that look like tokens
func hasTokenChildren(obj map[string]interface{}) bool {
	// Check if object has $value or $type (it's a token itself)
	if _, hasValue := obj["$value"]; hasValue {
		return false // This is a token, not a group
	}
	if _, hasType := obj["$type"]; hasType {
		return false // This is a token, not a group
	}

	// Check if children have $value or $type
	for key, value := range obj {
		if strings.HasPrefix(key, "$") {
			continue
		}

		if child, ok := value.(map[string]interface{}); ok {
			if _, hasValue := child["$value"]; hasValue {
				return true
			}
			if _, hasType := child["$type"]; hasType {
				return true
			}
		}
	}

	return false
}
