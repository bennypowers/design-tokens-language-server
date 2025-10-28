package diagnostic

import (
	"encoding/json"
	"testing"

	"bennypowers.dev/dtls/internal/tokens"
	"bennypowers.dev/dtls/lsp/testutil"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestGetDiagnostics_DeprecatedToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()

	// Add a deprecated token
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:               "color.old",
		Value:              "#ff0000",
		Type:               "color",
		Deprecated:         true,
		DeprecationMessage: "Use color.primary instead",
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-old); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	diagnostics, err := GetDiagnostics(ctx, uri)
	require.NoError(t, err)
	require.Len(t, diagnostics, 1)

	assert.Equal(t, protocol.DiagnosticSeverityInformation, *diagnostics[0].Severity)
	assert.Contains(t, diagnostics[0].Message, "deprecated")
	assert.Contains(t, diagnostics[0].Message, "Use color.primary instead")
	assert.Equal(t, []protocol.DiagnosticTag{protocol.DiagnosticTagDeprecated}, diagnostics[0].Tags)
}

func TestGetDiagnostics_IncorrectFallback(t *testing.T) {
	ctx := testutil.NewMockServerContext()

	// Add a token
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#0000ff",
		Type:  "color",
	})

	uri := "file:///test.css"
	// Fallback is #ff0000 but token value is #0000ff
	cssContent := `.button { color: var(--color-primary, #ff0000); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	diagnostics, err := GetDiagnostics(ctx, uri)
	require.NoError(t, err)
	require.Len(t, diagnostics, 1)

	assert.Equal(t, protocol.DiagnosticSeverityError, *diagnostics[0].Severity)
	assert.Contains(t, diagnostics[0].Message, "fallback does not match")
	assert.Contains(t, diagnostics[0].Message, "#0000ff")
}

func TestGetDiagnostics_CorrectFallback(t *testing.T) {
	ctx := testutil.NewMockServerContext()

	// Add a token
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
		Type:  "color",
	})

	uri := "file:///test.css"
	// Fallback matches token value
	cssContent := `.button { color: var(--color-primary, #ff0000); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	diagnostics, err := GetDiagnostics(ctx, uri)
	require.NoError(t, err)
	assert.Empty(t, diagnostics, "Should not report diagnostic for correct fallback")
}

func TestGetDiagnostics_UnknownToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()

	uri := "file:///test.css"
	// Reference to unknown token (no diagnostic expected)
	cssContent := `.button { color: var(--unknown-token); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	diagnostics, err := GetDiagnostics(ctx, uri)
	require.NoError(t, err)
	assert.Empty(t, diagnostics, "Unknown tokens should not produce diagnostics")
}

func TestGetDiagnostics_NonCSSDocument(t *testing.T) {
	ctx := testutil.NewMockServerContext()

	uri := "file:///test.json"
	jsonContent := `{"test": "value"}`
	_ = ctx.DocumentManager().DidOpen(uri, "json", 1, jsonContent)

	diagnostics, err := GetDiagnostics(ctx, uri)
	require.NoError(t, err)
	// LSP protocol requires array, not nil - nil serializes to JSON null which crashes clients
	require.NotNil(t, diagnostics, "Should return empty array, not nil")
	assert.Empty(t, diagnostics, "Non-CSS documents should return empty diagnostics")
}

func TestGetDiagnostics_DocumentNotFound(t *testing.T) {
	ctx := testutil.NewMockServerContext()

	diagnostics, err := GetDiagnostics(ctx, "file:///nonexistent.css")
	require.NoError(t, err)
	// LSP protocol requires array, not nil - nil serializes to JSON null which crashes clients
	require.NotNil(t, diagnostics, "Should return empty array, not nil")
	assert.Empty(t, diagnostics)
}

func TestGetDiagnostics_InvalidCSS(t *testing.T) {
	ctx := testutil.NewMockServerContext()

	uri := "file:///test.css"
	// Totally invalid CSS that might fail to parse
	cssContent := `{ { { invalid }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	_, err := GetDiagnostics(ctx, uri)
	// Should not error, just return nil or empty
	require.NoError(t, err)
}

func TestGetDiagnostics_MultipleIssues(t *testing.T) {
	ctx := testutil.NewMockServerContext()

	// Add tokens
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:       "color.deprecated",
		Value:      "#ff0000",
		Type:       "color",
		Deprecated: true,
	})
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.wrong",
		Value: "#0000ff",
		Type:  "color",
	})

	uri := "file:///test.css"
	cssContent := `.button {
		color: var(--color-deprecated);
		background: var(--color-wrong, #ff0000);
	}`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	diagnostics, err := GetDiagnostics(ctx, uri)
	require.NoError(t, err)
	assert.Len(t, diagnostics, 2, "Should report both deprecated and incorrect fallback")
}

func TestDocumentDiagnostic(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a deprecated token
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:       "spacing.old",
		Value:      "8px",
		Type:       "dimension",
		Deprecated: true,
	})

	uri := "file:///test.css"
	cssContent := `.button { padding: var(--spacing-old); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	params := &DocumentDiagnosticParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	}

	result, err := DocumentDiagnostic(req, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check that result is a RelatedFullDocumentDiagnosticReport
	report, ok := result.(RelatedFullDocumentDiagnosticReport)
	require.True(t, ok, "Result should be RelatedFullDocumentDiagnosticReport")
	assert.Equal(t, string(DiagnosticFull), report.Kind)
	assert.Len(t, report.Items, 1)
}

func TestIsCSSValueSemanticallyEquivalent(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{
			name:     "Exact match",
			a:        "#ff0000",
			b:        "#ff0000",
			expected: true,
		},
		{
			name:     "Case difference",
			a:        "#FF0000",
			b:        "#ff0000",
			expected: true,
		},
		{
			name:     "Whitespace difference",
			a:        "rgb(255, 0, 0)",
			b:        "rgb(255,0,0)",
			expected: true,
		},
		{
			name:     "Mixed whitespace and case",
			a:        "RGB( 255, 0, 0 )",
			b:        "rgb(255,0,0)",
			expected: true,
		},
		{
			name:     "Tab and newline",
			a:        "rgba(\n255,\t0,\t0,\t1\n)",
			b:        "rgba(255,0,0,1)",
			expected: true,
		},
		{
			name:     "Different values",
			a:        "#ff0000",
			b:        "#0000ff",
			expected: false,
		},
		{
			name:     "Empty strings",
			a:        "",
			b:        "",
			expected: true,
		},
		{
			name:     "One empty",
			a:        "#ff0000",
			b:        "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCSSValueSemanticallyEquivalent(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDiagnostics_FallbackSemanticEquivalence(t *testing.T) {
	ctx := testutil.NewMockServerContext()

	// Add a token with uppercase hex value
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#FF0000",
		Type:  "color",
	})

	uri := "file:///test.css"
	// Fallback uses lowercase (should be treated as equivalent)
	cssContent := `.button { color: var(--color-primary, #ff0000); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	diagnostics, err := GetDiagnostics(ctx, uri)
	require.NoError(t, err)
	assert.Empty(t, diagnostics, "Case-insensitive match should not produce diagnostic")
}

func TestGetDiagnostics_DeprecatedWithoutMessage(t *testing.T) {
	ctx := testutil.NewMockServerContext()

	// Deprecated token without custom message
	// Note: Token name uses DTCG dot notation, CSS variable uses hyphens
	// The conversion happens in CSSVariableName(): "color.legacy" → "--color-legacy"
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:       "color.legacy",
		Value:      "#ff0000",
		Type:       "color",
		Deprecated: true,
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-legacy); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	diagnostics, err := GetDiagnostics(ctx, uri)
	require.NoError(t, err)
	require.Len(t, diagnostics, 1)

	// Should just say "is deprecated" without additional message
	assert.Contains(t, diagnostics[0].Message, "--color-legacy is deprecated")
	// Make sure there's no colon followed by extra message
	assert.Equal(t, "--color-legacy is deprecated", diagnostics[0].Message)
}

func TestGetDiagnostics_EmptyArrayJSON(t *testing.T) {
	ctx := testutil.NewMockServerContext()

	uri := "file:///empty.css"
	cssContent := `.button { color: blue; }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	diagnostics, err := GetDiagnostics(ctx, uri)
	require.NoError(t, err)
	require.NotNil(t, diagnostics, "Must return non-nil slice")
	require.Empty(t, diagnostics)

	// Verify JSON serialization produces [] not null
	jsonBytes, err := json.Marshal(diagnostics)
	require.NoError(t, err)
	assert.Equal(t, "[]", string(jsonBytes), "Empty diagnostics must serialize to JSON [] not null")
}
