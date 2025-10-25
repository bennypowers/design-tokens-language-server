package lsp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bennypowers/design-tokens-language-server/internal/lsp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadTokenFile_ExtensionHandling tests various file extension scenarios
func TestLoadTokenFile_ExtensionHandling(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		content     string
		shouldError bool
		errorMsg    string
	}{
		{
			name:     "standard .json",
			filename: "tokens.json",
			content:  `{"color": {"primary": {"$value": "#ff0000", "$type": "color"}}}`,
		},
		{
			name:     "uppercase .JSON",
			filename: "tokens.JSON",
			content:  `{"color": {"primary": {"$value": "#ff0000", "$type": "color"}}}`,
		},
		{
			name:     "mixed case .Json",
			filename: "tokens.Json",
			content:  `{"color": {"primary": {"$value": "#ff0000", "$type": "color"}}}`,
		},
		{
			name:     "multiple extensions .tokens.json",
			filename: "design.tokens.json",
			content:  `{"color": {"primary": {"$value": "#ff0000", "$type": "color"}}}`,
		},
		{
			name:     "standard .yaml",
			filename: "tokens.yaml",
			content:  "color:\n  primary:\n    $value: \"#ff0000\"\n    $type: color\n",
		},
		{
			name:     "uppercase .YAML",
			filename: "tokens.YAML",
			content:  "color:\n  primary:\n    $value: \"#ff0000\"\n    $type: color\n",
		},
		{
			name:     "standard .yml",
			filename: "tokens.yml",
			content:  "color:\n  primary:\n    $value: \"#ff0000\"\n    $type: color\n",
		},
		{
			name:     "uppercase .YML",
			filename: "tokens.YML",
			content:  "color:\n  primary:\n    $value: \"#ff0000\"\n    $type: color\n",
		},
		{
			name:        "unsupported .txt",
			filename:    "tokens.txt",
			content:     "some text",
			shouldError: true,
			errorMsg:    "unsupported file type",
		},
		{
			name:        "unsupported .toml",
			filename:    "tokens.toml",
			content:     "[color]\nprimary = \"#ff0000\"\n",
			shouldError: true,
			errorMsg:    "unsupported file type",
		},
		{
			name:        "no extension",
			filename:    "tokens",
			content:     `{"color": {"primary": {"$value": "#ff0000"}}}`,
			shouldError: true,
			errorMsg:    "unsupported file type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, tt.filename)

			// Write test file
			err := os.WriteFile(filePath, []byte(tt.content), 0644)
			require.NoError(t, err)

			// Create server and try to load
			server, err := lsp.NewServer()
			require.NoError(t, err)
			err = server.LoadTokenFile(filePath, "")

			if tt.shouldError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err, "Failed to load %s", tt.filename)
			}
		})
	}
}
