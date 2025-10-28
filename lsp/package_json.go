package lsp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"bennypowers.dev/dtls/lsp/types"
	"github.com/tidwall/jsonc"
)

// ReadPackageJsonConfig reads designTokensLanguageServer configuration from package.json
// Returns nil if package.json doesn't exist or doesn't have the configuration
func ReadPackageJsonConfig(rootPath string) (*types.ServerConfig, error) {
	if rootPath == "" {
		return nil, nil
	}

	packageJSONPath := filepath.Join(rootPath, "package.json")

	// Check if package.json exists
	if _, err := os.Stat(packageJSONPath); os.IsNotExist(err) {
		return nil, nil // Not an error, just no config
	}

	// Read package.json
	data, err := os.ReadFile(packageJSONPath)
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

	// Extract designTokensLanguageServer field
	dtlsConfig, ok := pkgJSON["designTokensLanguageServer"]
	if !ok {
		return nil, nil // No config, not an error
	}

	configMap, ok := dtlsConfig.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("designTokensLanguageServer must be an object")
	}

	// Build ServerConfig
	config := &types.ServerConfig{}

	// Parse prefix
	if prefix, ok := configMap["prefix"].(string); ok {
		config.Prefix = prefix
	}

	// Parse groupMarkers
	if gm, ok := configMap["groupMarkers"]; ok {
		switch v := gm.(type) {
		case []any:
			for _, item := range v {
				if str, ok := item.(string); ok {
					config.GroupMarkers = append(config.GroupMarkers, str)
				}
			}
		case []string:
			config.GroupMarkers = v
		}
	}

	// Parse tokensFiles
	if tf, ok := configMap["tokensFiles"]; ok {
		switch v := tf.(type) {
		case string:
			// Single string
			config.TokensFiles = []any{v}
		case []any:
			// Array of strings or objects
			config.TokensFiles = v
		case []string:
			// Array of strings
			for _, str := range v {
				config.TokensFiles = append(config.TokensFiles, str)
			}
		}
	}

	return config, nil
}
