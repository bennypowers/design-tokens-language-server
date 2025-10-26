package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"bennypowers.dev/dtls/lsp/methods/textDocument"
	"bennypowers.dev/dtls/lsp/methods/textDocument/hover"
	"bennypowers.dev/dtls/lsp/methods/workspace"
	"bennypowers.dev/dtls/test/integration/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestFileWatching_TokenFileChange tests that changing a token file updates hover/diagnostics
func TestFileWatching_TokenFileChange(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create initial token file
	tokensPath := filepath.Join(tmpDir, "tokens.json")
	initialTokens := `{
  "color": {
    "primary": {
      "$value": "#ff0000",
      "$type": "color",
      "$description": "Initial primary color"
    }
  }
}`
	err := os.WriteFile(tokensPath, []byte(initialTokens), 0644)
	require.NoError(t, err)

	// Create CSS file
	cssPath := filepath.Join(tmpDir, "test.css")
	cssContent := `.button {
  color: var(--color-primary);
}`
	err = os.WriteFile(cssPath, []byte(cssContent), 0644)
	require.NoError(t, err)

	// Create server and load tokens
	server := testutil.NewTestServer(t)
	defer server.Close()
	err = server.LoadTokenFile(tokensPath, "")
	require.NoError(t, err)

	// Open CSS document
	cssURI := "file://" + cssPath
	err = textDocument.DidOpen(server, nil, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        cssURI,
			LanguageID: "css",
			Version:    1,
			Text:       cssContent,
		},
	})
	require.NoError(t, err)

	// Test initial hover
	hover1, err := hover.Hover(server, nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: cssURI},
			Position:     protocol.Position{Line: 1, Character: 15},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, hover1)

	content1, ok := hover1.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, content1.Value, "#ff0000", "Should show initial color value")
	assert.Contains(t, content1.Value, "Initial primary color", "Should show initial description")

	// Update token file
	updatedTokens := `{
  "color": {
    "primary": {
      "$value": "#00ff00",
      "$type": "color",
      "$description": "Updated primary color"
    }
  }
}`
	err = os.WriteFile(tokensPath, []byte(updatedTokens), 0644)
	require.NoError(t, err)

	// Simulate file change notification
	tokensURI := "file://" + tokensPath
	err = workspace.DidChangeWatchedFiles(server, nil, &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{
				URI:  tokensURI,
				Type: protocol.FileChangeTypeChanged,
			},
		},
	})
	require.NoError(t, err)

	// Test hover after update
	hover2, err := hover.Hover(server, nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: cssURI},
			Position:     protocol.Position{Line: 1, Character: 15},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, hover2)

	content2, ok := hover2.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, content2.Value, "#00ff00", "Should show updated color value")
	assert.Contains(t, content2.Value, "Updated primary color", "Should show updated description")
	assert.NotContains(t, content2.Value, "#ff0000", "Should not show old color value")
}

// TestFileWatching_TokenFileDeleted tests behavior when a token file is deleted
func TestFileWatching_TokenFileDeleted(t *testing.T) {
	tmpDir := t.TempDir()

	tokensPath := filepath.Join(tmpDir, "tokens.json")
	tokens := `{
  "color": {
    "primary": {
      "$value": "#ff0000",
      "$type": "color"
    }
  }
}`
	err := os.WriteFile(tokensPath, []byte(tokens), 0644)
	require.NoError(t, err)

	cssPath := filepath.Join(tmpDir, "test.css")
	cssContent := `.button {
  color: var(--color-primary);
}`
	err = os.WriteFile(cssPath, []byte(cssContent), 0644)
	require.NoError(t, err)

	server := testutil.NewTestServer(t)
	defer server.Close()
	err = server.LoadTokenFile(tokensPath, "")
	require.NoError(t, err)

	cssURI := "file://" + cssPath
	err = textDocument.DidOpen(server, nil, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        cssURI,
			LanguageID: "css",
			Version:    1,
			Text:       cssContent,
		},
	})
	require.NoError(t, err)

	// Verify token exists initially
	hover1, err := hover.Hover(server, nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: cssURI},
			Position:     protocol.Position{Line: 1, Character: 15},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, hover1)

	// Delete the token file
	err = os.Remove(tokensPath)
	require.NoError(t, err)

	// Simulate file deletion notification
	tokensURI := "file://" + tokensPath
	err = workspace.DidChangeWatchedFiles(server, nil, &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{
				URI:  tokensURI,
				Type: protocol.FileChangeTypeDeleted,
			},
		},
	})
	require.NoError(t, err)

	// After deletion, hover should show "Unknown token"
	hover2, err := hover.Hover(server, nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: cssURI},
			Position:     protocol.Position{Line: 1, Character: 15},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, hover2)

	content2, ok := hover2.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, content2.Value, "Unknown token", "Should show unknown token message")
}

// TestFileWatching_MultipleTokenFiles tests watching multiple token files
func TestFileWatching_MultipleTokenFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create first token file
	tokens1Path := filepath.Join(tmpDir, "colors.json")
	tokens1 := `{
  "color": {
    "primary": {
      "$value": "#ff0000",
      "$type": "color"
    }
  }
}`
	err := os.WriteFile(tokens1Path, []byte(tokens1), 0644)
	require.NoError(t, err)

	// Create second token file
	tokens2Path := filepath.Join(tmpDir, "spacing.json")
	tokens2 := `{
  "spacing": {
    "small": {
      "$value": "8px",
      "$type": "dimension"
    }
  }
}`
	err = os.WriteFile(tokens2Path, []byte(tokens2), 0644)
	require.NoError(t, err)

	cssPath := filepath.Join(tmpDir, "test.css")
	cssContent := `.button {
  color: var(--color-primary);
  padding: var(--spacing-small);
}`
	err = os.WriteFile(cssPath, []byte(cssContent), 0644)
	require.NoError(t, err)

	server := testutil.NewTestServer(t)
	defer server.Close()
	err = server.LoadTokenFile(tokens1Path, "")
	require.NoError(t, err)
	err = server.LoadTokenFile(tokens2Path, "")
	require.NoError(t, err)

	cssURI := "file://" + cssPath
	err = textDocument.DidOpen(server, nil, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        cssURI,
			LanguageID: "css",
			Version:    1,
			Text:       cssContent,
		},
	})
	require.NoError(t, err)

	// Update only the spacing file
	tokens2Updated := `{
  "spacing": {
    "small": {
      "$value": "16px",
      "$type": "dimension"
    }
  }
}`
	err = os.WriteFile(tokens2Path, []byte(tokens2Updated), 0644)
	require.NoError(t, err)

	// Simulate file change notification
	tokens2URI := "file://" + tokens2Path
	err = workspace.DidChangeWatchedFiles(server, nil, &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{
				URI:  tokens2URI,
				Type: protocol.FileChangeTypeChanged,
			},
		},
	})
	require.NoError(t, err)

	// Test that spacing value updated
	hover, err := hover.Hover(server, nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: cssURI},
			Position:     protocol.Position{Line: 2, Character: 17},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, hover)

	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, content.Value, "16px", "Should show updated spacing value")
	assert.NotContains(t, content.Value, "8px", "Should not show old spacing value")
}

// TestFileWatching_NonTokenFileIgnored tests that non-token file changes are ignored
func TestFileWatching_NonTokenFileIgnored(t *testing.T) {
	tmpDir := t.TempDir()

	tokensPath := filepath.Join(tmpDir, "tokens.json")
	tokens := `{
  "color": {
    "primary": {
      "$value": "#ff0000",
      "$type": "color"
    }
  }
}`
	err := os.WriteFile(tokensPath, []byte(tokens), 0644)
	require.NoError(t, err)

	// Create a non-token file
	pkgPath := filepath.Join(tmpDir, "package.json")
	pkg := `{"name": "test"}`
	err = os.WriteFile(pkgPath, []byte(pkg), 0644)
	require.NoError(t, err)

	server := testutil.NewTestServer(t)
	defer server.Close()
	err = server.LoadTokenFile(tokensPath, "")
	require.NoError(t, err)

	// Count initial tokens
	initialCount := server.TokenCount()

	// Simulate change to non-token file
	pkgURI := "file://" + pkgPath
	err = workspace.DidChangeWatchedFiles(server, nil, &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{
				URI:  pkgURI,
				Type: protocol.FileChangeTypeChanged,
			},
		},
	})
	require.NoError(t, err)

	// Token count should remain the same
	assert.Equal(t, initialCount, server.TokenCount(), "Token count should not change for non-token file changes")
}

// TestFileWatching_YmlExtension tests that .yml files are recognized (in addition to .yaml)
func TestFileWatching_YmlExtension(t *testing.T) {
	tmpDir := t.TempDir()

	// Test all .yml variants
	testFiles := []string{
		"tokens.yml",
		"design-tokens.yml",
		"my-app.tokens.yml",
	}

	for _, filename := range testFiles {
		t.Run(filename, func(t *testing.T) {
			// Create token file with .yml extension
			tokensPath := filepath.Join(tmpDir, filename)
			tokens := `color:
  primary:
    $value: "#0000ff"
    $type: color
`
			err := os.WriteFile(tokensPath, []byte(tokens), 0644)
			require.NoError(t, err)

			// Create server and load the .yml file
			server := testutil.NewTestServer(t)
			defer server.Close()
			err = server.LoadTokenFile(tokensPath, "")
			require.NoError(t, err)

			// Verify token was loaded
			assert.Equal(t, 1, server.TokenCount(), "Should load token from .yml file")

			// Simulate file change to .yml file
			tokensURI := "file://" + tokensPath
			err = workspace.DidChangeWatchedFiles(server, nil, &protocol.DidChangeWatchedFilesParams{
				Changes: []protocol.FileEvent{
					{
						URI:  tokensURI,
						Type: protocol.FileChangeTypeChanged,
					},
				},
			})
			require.NoError(t, err, "Should handle .yml file changes")
		})
	}
}
