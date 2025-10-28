package lsp

import (
	"encoding/json"
)

// DetectPullDiagnosticsSupport detects whether the client supports pull diagnostics
// by parsing the raw initialize request parameters for the textDocument.diagnostic capability.
//
// LSP 3.17 introduced pull diagnostics via the textDocument/diagnostic method. Clients that
// support this feature will include a "diagnostic" field in their textDocument capabilities.
// Since glsp v0.2.2 only supports LSP 3.16, we must parse the raw JSON to detect this field.
//
// Returns:
//   - true: Client explicitly declares diagnostic capability (LSP 3.17+)
//   - false: No diagnostic capability found, or error parsing (conservative default to push)
func DetectPullDiagnosticsSupport(rawParams json.RawMessage) bool {
	// Define a minimal struct to extract only what we need from the initialize params
	var initParams struct {
		Capabilities struct {
			TextDocument *struct {
				Diagnostic *json.RawMessage `json:"diagnostic"` // LSP 3.17 field
			} `json:"textDocument"`
		} `json:"capabilities"`
	}

	// Try to unmarshal the raw params
	if err := json.Unmarshal(rawParams, &initParams); err != nil {
		// Parse error: default to push diagnostics (safe fallback)
		return false
	}

	// Check if textDocument capabilities exist
	if initParams.Capabilities.TextDocument == nil {
		return false
	}

	// Check if diagnostic capability is present (even if null/empty, presence indicates support)
	// The field being present indicates the client knows about this LSP 3.17 feature
	if initParams.Capabilities.TextDocument.Diagnostic != nil {
		return true
	}

	// No diagnostic capability found: default to push diagnostics
	return false
}
