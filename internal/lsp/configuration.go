package lsp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TokenFileSpec represents a token file specification
type TokenFileSpec struct {
	// Path to the token file (required)
	// Can be absolute, relative, or npm: protocol
	Path string `json:"path"`

	// Prefix for CSS variables from this file (optional)
	Prefix string `json:"prefix,omitempty"`

	// GroupMarkers are token names that can also be groups (optional)
	GroupMarkers []string `json:"groupMarkers,omitempty"`
}

// ServerConfig represents the server configuration
type ServerConfig struct {
	// TokensFiles specifies token files to load
	// Can be:
	//  - strings (paths): ["./tokens.json", "npm:@foo/tokens"]
	//  - objects: [{"path": "./tokens.json", "prefix": "ds"}]
	// If empty, falls back to searching for common patterns
	TokensFiles []any `json:"tokensFiles"`

	// Prefix is the global CSS variable prefix (can be overridden per-file)
	// Example: "ds" will generate "--ds-color-primary"
	Prefix string `json:"prefix"`

	// GroupMarkers are token names which will be treated as group names as well
	// Default: ["_", "@", "DEFAULT"]
	GroupMarkers []string `json:"groupMarkers"`
}

// DefaultConfig returns the default server configuration
func DefaultConfig() ServerConfig {
	return ServerConfig{
		TokensFiles: []any{}, // Empty = auto-discover
		Prefix:      "",
		GroupMarkers: []string{
			"_",
			"@",
			"DEFAULT",
		},
	}
}

// handleDidChangeConfiguration handles the workspace/didChangeConfiguration notification
func (s *Server) handleDidChangeConfiguration(context *glsp.Context, params *protocol.DidChangeConfigurationParams) error {
	fmt.Fprintf(os.Stderr, "[DTLS] Configuration changed\n")

	// Parse the settings
	config, err := s.parseConfiguration(params.Settings)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[DTLS] Warning: failed to parse configuration: %v\n", err)
		return nil // Don't fail, just use defaults
	}

	// Update server configuration
	s.config = config

	fmt.Fprintf(os.Stderr, "[DTLS] New configuration: %+v\n", config)

	// Reload tokens with new configuration
	if err := s.loadTokensFromConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "[DTLS] Warning: failed to reload tokens: %v\n", err)
	}

	// Republish diagnostics for all open documents
	if s.context != nil {
		for _, doc := range s.documents.GetAll() {
			if err := s.PublishDiagnostics(s.context, doc.URI()); err != nil {
				fmt.Fprintf(os.Stderr, "[DTLS] Warning: failed to publish diagnostics for %s: %v\n", doc.URI(), err)
			}
		}
	}

	return nil
}

// parseConfiguration parses the configuration from the settings
func (s *Server) parseConfiguration(settings any) (ServerConfig, error) {
	// Default configuration
	config := DefaultConfig()

	if settings == nil {
		return config, nil
	}

	// Settings come as a nested object: { "designTokensLanguageServer": { ... } }
	settingsMap, ok := settings.(map[string]any)
	if !ok {
		return config, fmt.Errorf("settings is not a map")
	}

	// Look for our configuration under "designTokensLanguageServer" key
	var ourSettings any
	if val, exists := settingsMap["designTokensLanguageServer"]; exists {
		ourSettings = val
	} else if val, exists := settingsMap["design-tokens-language-server"]; exists {
		ourSettings = val
	} else {
		// No configuration provided, use defaults
		return config, nil
	}

	// Convert to JSON and back to parse into struct
	jsonBytes, err := json.Marshal(ourSettings)
	if err != nil {
		return config, fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, &config); err != nil {
		return config, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	return config, nil
}

// GetConfig returns the current server configuration
func (s *Server) GetConfig() ServerConfig {
	if s.config.TokensFiles == nil {
		return DefaultConfig()
	}
	return s.config
}

// loadTokensFromConfig loads tokens based on current configuration
func (s *Server) loadTokensFromConfig() error {
	// Clear existing tokens
	s.tokens.Clear()

	// If tokensFiles is specified, load those files
	if len(s.config.TokensFiles) > 0 {
		return s.loadExplicitTokenFiles()
	}

	// Otherwise, auto-discover token files
	if s.rootPath != "" {
		return s.loadTokenFilesAutoDiscover()
	}

	return nil
}

// loadExplicitTokenFiles loads tokens from explicitly configured files
func (s *Server) loadExplicitTokenFiles() error {
	for _, item := range s.config.TokensFiles {
		var path, prefix string
		var groupMarkers []string

		// Parse the item - can be string or object
		switch v := item.(type) {
		case string:
			path = v
			prefix = s.config.Prefix
			groupMarkers = s.config.GroupMarkers
		case map[string]any:
			// Convert to TokenFileSpec
			pathVal, ok := v["path"]
			if !ok {
				continue
			}
			path, _ = pathVal.(string)
			if prefixVal, ok := v["prefix"]; ok {
				prefix, _ = prefixVal.(string)
			} else {
				prefix = s.config.Prefix
			}
			if gmVal, ok := v["groupMarkers"]; ok {
				if gmSlice, ok := gmVal.([]any); ok {
					for _, gm := range gmSlice {
						if gmStr, ok := gm.(string); ok {
							groupMarkers = append(groupMarkers, gmStr)
						}
					}
				}
			}
			if len(groupMarkers) == 0 {
				groupMarkers = s.config.GroupMarkers
			}
		default:
			continue
		}

		// Resolve path relative to workspace
		if s.rootPath != "" && !filepath.IsAbs(path) {
			path = filepath.Join(s.rootPath, path)
		}

		// TODO: Handle npm: protocol

		// Load the file
		if err := s.LoadTokenFile(path, prefix); err != nil {
			fmt.Fprintf(os.Stderr, "[DTLS] Warning: failed to load %s: %v\n", path, err)
			continue
		}

		fmt.Fprintf(os.Stderr, "[DTLS] Loaded %s (prefix: %s)\n", path, prefix)
	}

	return nil
}

// loadTokenFilesAutoDiscover auto-discovers and loads token files
func (s *Server) loadTokenFilesAutoDiscover() error {
	tokenConfig := TokenFileConfig{
		RootDir: s.rootPath,
		Patterns: []string{
			"**/tokens.json",
			"**/*.tokens.json",
			"**/design-tokens.json",
		},
		Prefix: s.config.Prefix,
	}

	return s.LoadTokenFiles(tokenConfig)
}
