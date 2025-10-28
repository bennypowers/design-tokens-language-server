package workspace

import (
	"encoding/json"
	"fmt"
	"os"

	"bennypowers.dev/dtls/lsp/types"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// DidChangeConfiguration handles the workspace/didChangeConfiguration notification
func DidChangeConfiguration(req *types.RequestContext, params *protocol.DidChangeConfigurationParams) error {
	fmt.Fprintf(os.Stderr, "[DTLS] Configuration changed\n")

	// Parse the settings
	config, err := parseConfiguration(params.Settings)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[DTLS] Warning: failed to parse configuration: %v\n", err)
		return nil // Don't fail, just use defaults
	}

	// Update server configuration
	req.Server.SetConfig(config)

	fmt.Fprintf(os.Stderr, "[DTLS] New configuration: %+v\n", config)

	// Reload tokens with new configuration
	if err := req.Server.LoadTokensFromConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "[DTLS] Warning: failed to reload tokens: %v\n", err)
	}

	// Republish diagnostics for all open documents (only if using push model)
	// If client supports pull diagnostics (LSP 3.17), it will request them via textDocument/diagnostic
	if !req.Server.UsePullDiagnostics() {
		if req.GLSP != nil {
			for _, doc := range req.Server.AllDocuments() {
				if err := req.Server.PublishDiagnostics(req.GLSP, doc.URI()); err != nil {
					fmt.Fprintf(os.Stderr, "[DTLS] Warning: failed to publish diagnostics for %s: %v\n", doc.URI(), err)
				}
			}
		}
	}

	return nil
}

// parseConfiguration parses the configuration from the settings
func parseConfiguration(settings any) (types.ServerConfig, error) {
	// Default configuration
	config := types.DefaultConfig()

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
