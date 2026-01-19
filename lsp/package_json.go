package lsp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"bennypowers.dev/dtls/lsp/types"
	"github.com/tidwall/jsonc"
)

// readPackageJsonFile reads and parses package.json from the given root path.
// Returns the parsed JSON as a map, or nil if the file doesn't exist.
func readPackageJsonFile(rootPath string) (map[string]any, error) {
	packageJSONPath := filepath.Join(rootPath, "package.json")

	// Check if package.json exists
	if _, err := os.Stat(packageJSONPath); os.IsNotExist(err) {
		return nil, nil // Not an error, just no config
	}

	// Read package.json
	data, err := os.ReadFile(packageJSONPath) //nolint:gosec // G304: Reading workspace package.json - local trusted environment
	if err != nil {
		return nil, fmt.Errorf("failed to read package.json: %w", err)
	}

	// Parse as JSONC (allows comments)
	data = jsonc.ToJSON(data)

	// Parse JSON
	var pkgJSON map[string]any
	if err := json.Unmarshal(data, &pkgJSON); err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}

	return pkgJSON, nil
}

// extractConfigMap extracts the designTokensLanguageServer configuration map.
// Returns nil if the field doesn't exist (not an error).
func extractConfigMap(pkgJSON map[string]any) (map[string]any, error) {
	dtlsConfig, ok := pkgJSON["designTokensLanguageServer"]
	if !ok {
		return nil, nil // No config, not an error
	}

	configMap, ok := dtlsConfig.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("designTokensLanguageServer must be an object")
	}

	return configMap, nil
}

// parseGroupMarkersField parses the groupMarkers field from configuration.
// Handles both []string and []any types, returning nil if not present.
func parseGroupMarkersField(configMap map[string]any) []string {
	gm, ok := configMap["groupMarkers"]
	if !ok {
		return nil
	}

	switch v := gm.(type) {
	case []any:
		var groupMarkers []string
		for _, item := range v {
			if str, ok := item.(string); ok {
				groupMarkers = append(groupMarkers, str)
			}
		}
		return groupMarkers
	case []string:
		return v
	}

	return nil
}

// parseTokensFilesField parses the tokensFiles field from configuration.
// Handles string, []any, and []string types, returning nil if not present.
func parseTokensFilesField(configMap map[string]any) []any {
	tf, ok := configMap["tokensFiles"]
	if !ok {
		return nil
	}

	switch v := tf.(type) {
	case string:
		// Single string - wrap in array
		return []any{v}
	case []any:
		// Array of strings or objects - return as-is
		return v
	case []string:
		// Array of strings - convert to []any
		var tokensFiles []any
		for _, str := range v {
			tokensFiles = append(tokensFiles, str)
		}
		return tokensFiles
	}

	return nil
}

// buildServerConfig constructs a ServerConfig from the parsed configuration map.
// Extracts and parses all fields (prefix, groupMarkers, tokensFiles).
func buildServerConfig(configMap map[string]any) *types.ServerConfig {
	config := &types.ServerConfig{}

	// Parse prefix
	if prefix, ok := configMap["prefix"].(string); ok {
		config.Prefix = prefix
	}

	// Parse groupMarkers
	config.GroupMarkers = parseGroupMarkersField(configMap)

	// Parse tokensFiles
	config.TokensFiles = parseTokensFilesField(configMap)

	return config
}

// ReadPackageJsonConfig reads designTokensLanguageServer configuration from package.json.
// Falls back to .config/design-tokens.{yaml,json} if no package.json config is found.
// Returns nil if no configuration exists (not an error).
func ReadPackageJsonConfig(rootPath string) (*types.ServerConfig, error) {
	if rootPath == "" {
		return nil, nil
	}

	pkgJSON, err := readPackageJsonFile(rootPath)
	if err != nil {
		return nil, err
	}

	if pkgJSON != nil {
		configMap, err := extractConfigMap(pkgJSON)
		if err != nil {
			return nil, err
		}
		if configMap != nil {
			// Found package.json config
			config := buildServerConfig(configMap)
			return config, nil
		}
	}

	// Fallback to asimonim config (.config/design-tokens.{yaml,json})
	return ReadAsimonimConfig(rootPath)
}
