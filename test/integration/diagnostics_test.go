package integration_test

import (
	"testing"

	"bennypowers.dev/dtls/lsp/methods/textDocument/diagnostic"
	"bennypowers.dev/dtls/test/integration/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestDiagnosticsIncorrectFallback tests diagnostics for incorrect fallback values
func TestDiagnosticsIncorrectFallback(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "incorrect-fallback.css")

	// Get diagnostics
	diagnostics, err := diagnostic.GetDiagnostics(server, "file:///test.css")
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
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "correct-fallback.css")

	// Get diagnostics
	diagnostics, err := diagnostic.GetDiagnostics(server, "file:///test.css")
	require.NoError(t, err)

	// Should have no diagnostics
	assert.Len(t, diagnostics, 0)
}

// TestDiagnosticsSemanticEquivalence tests that semantically equivalent CSS values are accepted
func TestDiagnosticsSemanticEquivalence(t *testing.T) {
	server := testutil.NewTestServer(t)

	// Load tokens with uppercase value
	tokens := []byte(`{
		"color": {
			"primary": {
				"$value": "#0000FF",
				"$type": "color"
			}
		}
	}`)
	err := server.LoadTokensFromJSON(tokens, "")
	require.NoError(t, err)

	// Load CSS with semantically equivalent fallback (different case)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "semantic-equivalence.css")

	// Get diagnostics
	diagnostics, err := diagnostic.GetDiagnostics(server, "file:///test.css")
	require.NoError(t, err)

	// Should have no diagnostics (values are semantically equivalent)
	assert.Len(t, diagnostics, 0)
}

// TestDiagnosticsDeprecatedToken tests diagnostics for deprecated tokens
func TestDiagnosticsDeprecatedToken(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "deprecated-token.css")

	// Get diagnostics
	diagnostics, err := diagnostic.GetDiagnostics(server, "file:///test.css")
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
	server := testutil.NewTestServer(t)
	// Don't load any tokens
	testutil.OpenCSSFixture(t, server, "file:///test.css", "unknown-token.css")

	// Get diagnostics
	diagnostics, err := diagnostic.GetDiagnostics(server, "file:///test.css")
	require.NoError(t, err)

	// Should have no diagnostics (unknown tokens are not errors, just missing hover info)
	assert.Len(t, diagnostics, 0)
}

// TestDiagnosticsMultipleIssues tests multiple diagnostics in one document
func TestDiagnosticsMultipleIssues(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "multiple-issues.css")

	// Get diagnostics
	diagnostics, err := diagnostic.GetDiagnostics(server, "file:///test.css")
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
