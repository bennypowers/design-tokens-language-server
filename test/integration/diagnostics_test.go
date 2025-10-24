package integration_test

import (
	"testing"

	"github.com/bennypowers/design-tokens-language-server/internal/lsp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestDiagnosticsIncorrectFallback tests diagnostics for incorrect fallback values
func TestDiagnosticsIncorrectFallback(t *testing.T) {
	server, err := lsp.NewServer()
	require.NoError(t, err)

	// Load tokens
	tokenJSON := []byte(`{
		"color": {
			"primary": {
				"$value": "#0000ff",
				"$type": "color"
			}
		}
	}`)

	err = server.LoadTokensFromJSON(tokenJSON, "")
	require.NoError(t, err)

	// CSS with incorrect fallback
	cssContent := `.button {
  color: var(--color-primary, #ff0000);
}`

	err = server.DidOpen("file:///test.css", "css", 1, cssContent)
	require.NoError(t, err)

	// Get diagnostics
	diagnostics, err := server.GetDiagnostics("file:///test.css")
	require.NoError(t, err)
	require.NotNil(t, diagnostics)

	// Should have one diagnostic for incorrect fallback
	assert.Len(t, diagnostics, 1)
	require.NotNil(t, diagnostics[0].Severity)
	assert.Equal(t, protocol.DiagnosticSeverityError, *diagnostics[0].Severity)
	assert.Contains(t, diagnostics[0].Message, "fallback does not match")
	assert.Contains(t, diagnostics[0].Message, "#0000ff")
}

// TestDiagnosticsCorrectFallback tests that correct fallback values produce no diagnostics
func TestDiagnosticsCorrectFallback(t *testing.T) {
	server, err := lsp.NewServer()
	require.NoError(t, err)

	// Load tokens
	tokenJSON := []byte(`{
		"color": {
			"primary": {
				"$value": "#0000ff",
				"$type": "color"
			}
		}
	}`)

	err = server.LoadTokensFromJSON(tokenJSON, "")
	require.NoError(t, err)

	// CSS with correct fallback
	cssContent := `.button {
  color: var(--color-primary, #0000ff);
}`

	err = server.DidOpen("file:///test.css", "css", 1, cssContent)
	require.NoError(t, err)

	// Get diagnostics
	diagnostics, err := server.GetDiagnostics("file:///test.css")
	require.NoError(t, err)

	// Should have no diagnostics
	assert.Len(t, diagnostics, 0)
}

// TestDiagnosticsSemanticEquivalence tests that semantically equivalent CSS values are accepted
func TestDiagnosticsSemanticEquivalence(t *testing.T) {
	server, err := lsp.NewServer()
	require.NoError(t, err)

	// Load tokens
	tokenJSON := []byte(`{
		"color": {
			"primary": {
				"$value": "#0000FF",
				"$type": "color"
			}
		}
	}`)

	err = server.LoadTokensFromJSON(tokenJSON, "")
	require.NoError(t, err)

	// CSS with semantically equivalent fallback (different case, spaces)
	cssContent := `.button {
  color: var(--color-primary, #0000ff);
}`

	err = server.DidOpen("file:///test.css", "css", 1, cssContent)
	require.NoError(t, err)

	// Get diagnostics
	diagnostics, err := server.GetDiagnostics("file:///test.css")
	require.NoError(t, err)

	// Should have no diagnostics (values are semantically equivalent)
	assert.Len(t, diagnostics, 0)
}

// TestDiagnosticsDeprecatedToken tests diagnostics for deprecated tokens
func TestDiagnosticsDeprecatedToken(t *testing.T) {
	server, err := lsp.NewServer()
	require.NoError(t, err)

	// Load tokens with deprecated token
	tokenJSON := []byte(`{
		"color": {
			"old-primary": {
				"$value": "#ff0000",
				"$type": "color",
				"$deprecated": "Use color.primary instead"
			}
		}
	}`)

	err = server.LoadTokensFromJSON(tokenJSON, "")
	require.NoError(t, err)

	// CSS using deprecated token
	cssContent := `.button {
  color: var(--color-old-primary);
}`

	err = server.DidOpen("file:///test.css", "css", 1, cssContent)
	require.NoError(t, err)

	// Get diagnostics
	diagnostics, err := server.GetDiagnostics("file:///test.css")
	require.NoError(t, err)
	require.NotNil(t, diagnostics)

	// Should have one diagnostic for deprecated token
	assert.Len(t, diagnostics, 1)
	require.NotNil(t, diagnostics[0].Severity)
	assert.Equal(t, protocol.DiagnosticSeverityInformation, *diagnostics[0].Severity)
	assert.Contains(t, diagnostics[0].Message, "deprecated")
	assert.Contains(t, diagnostics[0].Message, "Use color.primary instead")
	assert.Contains(t, diagnostics[0].Tags, protocol.DiagnosticTagDeprecated)
}

// TestDiagnosticsUnknownToken tests that unknown tokens produce no diagnostics
// (they are handled by hover instead)
func TestDiagnosticsUnknownToken(t *testing.T) {
	server, err := lsp.NewServer()
	require.NoError(t, err)

	// Don't load any tokens

	// CSS with unknown token
	cssContent := `.button {
  color: var(--unknown-token);
}`

	err = server.DidOpen("file:///test.css", "css", 1, cssContent)
	require.NoError(t, err)

	// Get diagnostics
	diagnostics, err := server.GetDiagnostics("file:///test.css")
	require.NoError(t, err)

	// Should have no diagnostics (unknown tokens are not errors, just missing hover info)
	assert.Len(t, diagnostics, 0)
}

// TestDiagnosticsMultipleIssues tests multiple diagnostics in one document
func TestDiagnosticsMultipleIssues(t *testing.T) {
	server, err := lsp.NewServer()
	require.NoError(t, err)

	// Load tokens
	tokenJSON := []byte(`{
		"color": {
			"primary": {
				"$value": "#0000ff",
				"$type": "color"
			},
			"old-secondary": {
				"$value": "#00ff00",
				"$type": "color",
				"$deprecated": true
			}
		}
	}`)

	err = server.LoadTokensFromJSON(tokenJSON, "")
	require.NoError(t, err)

	// CSS with multiple issues
	cssContent := `.button {
  color: var(--color-primary, #ff0000);
  background: var(--color-old-secondary);
}`

	err = server.DidOpen("file:///test.css", "css", 1, cssContent)
	require.NoError(t, err)

	// Get diagnostics
	diagnostics, err := server.GetDiagnostics("file:///test.css")
	require.NoError(t, err)
	require.NotNil(t, diagnostics)

	// Should have two diagnostics
	assert.Len(t, diagnostics, 2)

	// One for incorrect fallback
	var incorrectFallback *protocol.Diagnostic
	var deprecated *protocol.Diagnostic

	for i := range diagnostics {
		if diagnostics[i].Severity != nil && *diagnostics[i].Severity == protocol.DiagnosticSeverityError {
			incorrectFallback = &diagnostics[i]
		} else if diagnostics[i].Severity != nil && *diagnostics[i].Severity == protocol.DiagnosticSeverityInformation {
			deprecated = &diagnostics[i]
		}
	}

	require.NotNil(t, incorrectFallback, "Should have error diagnostic for incorrect fallback")
	require.NotNil(t, deprecated, "Should have info diagnostic for deprecated token")

	assert.Contains(t, incorrectFallback.Message, "fallback does not match")
	assert.Contains(t, deprecated.Message, "deprecated")
}
