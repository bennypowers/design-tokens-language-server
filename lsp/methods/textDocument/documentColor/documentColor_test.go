package documentcolor

import (
	"github.com/bennypowers/design-tokens-language-server/lsp/types"
	"testing"

	"github.com/bennypowers/design-tokens-language-server/internal/documents"
	"github.com/bennypowers/design-tokens-language-server/internal/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// mockServerContext implements types.ServerContext for testing
type mockServerContext struct {
	docs   *documents.Manager
	tokens *tokens.Manager
}

func (m *mockServerContext) Document(uri string) *documents.Document {
	return m.docs.Get(uri)
}

func (m *mockServerContext) DocumentManager() *documents.Manager {
	return m.docs
}

func (m *mockServerContext) AllDocuments() []*documents.Document {
	return m.docs.GetAll()
}

func (m *mockServerContext) Token(name string) *tokens.Token {
	return m.tokens.Get(name)
}

func (m *mockServerContext) TokenManager() *tokens.Manager {
	return m.tokens
}

func (m *mockServerContext) TokenCount() int {
	return m.tokens.Count()
}

func (m *mockServerContext) RootURI() string {
	return "file:///workspace"
}

func (m *mockServerContext) RootPath() string {
	return "/workspace"
}

func (m *mockServerContext) SetRootURI(uri string) {}

func (m *mockServerContext) SetRootPath(path string) {}

func (m *mockServerContext) LoadTokensFromConfig() error {
	return nil
}

func (m *mockServerContext) RegisterFileWatchers(ctx *glsp.Context) error {
	return nil
}

func (m *mockServerContext) GLSPContext() *glsp.Context {
	return nil
}

func (m *mockServerContext) SetGLSPContext(ctx *glsp.Context) {}



func (m *mockServerContext) GetConfig() types.ServerConfig {
	return types.DefaultConfig()
}

func (m *mockServerContext) SetConfig(config types.ServerConfig) {}

func (m *mockServerContext) IsTokenFile(path string) bool {
	return false
}

func (m *mockServerContext) PublishDiagnostics(context *glsp.Context, uri string) error {
	return nil
}

func newMockServerContext() *mockServerContext {
	return &mockServerContext{
		docs:   documents.NewManager(),
		tokens: tokens.NewManager(),
	}
}

func TestDocumentColor_ColorTokenInVar(t *testing.T) {
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	// Add a color token
	ctx.tokens.Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
		Type:  "color",
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	ctx.docs.DidOpen(uri, "css", 1, cssContent)

	result, err := DocumentColor(ctx, glspCtx, &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result, 1)

	// Check color value
	assert.Equal(t, protocol.Decimal(1.0), result[0].Color.Red)
	assert.Equal(t, protocol.Decimal(0.0), result[0].Color.Green)
	assert.Equal(t, protocol.Decimal(0.0), result[0].Color.Blue)
	assert.Equal(t, protocol.Decimal(1.0), result[0].Color.Alpha)
}

func TestDocumentColor_ColorTokenInDeclaration(t *testing.T) {
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	// Add a color token
	ctx.tokens.Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#00ff00",
		Type:  "color",
	})

	uri := "file:///test.css"
	cssContent := `:root { --color-primary: #00ff00; }`
	ctx.docs.DidOpen(uri, "css", 1, cssContent)

	result, err := DocumentColor(ctx, glspCtx, &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.GreaterOrEqual(t, len(result), 1)

	// Check that we found a green color
	foundGreen := false
	for _, colorInfo := range result {
		if colorInfo.Color.Green == 1.0 && colorInfo.Color.Red == 0.0 && colorInfo.Color.Blue == 0.0 {
			foundGreen = true
			break
		}
	}
	assert.True(t, foundGreen, "Should find green color in declarations")
}

func TestDocumentColor_NonColorToken(t *testing.T) {
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	// Add a non-color token
	ctx.tokens.Add(&tokens.Token{
		Name:  "spacing.small",
		Value: "8px",
		Type:  "dimension",
	})

	uri := "file:///test.css"
	cssContent := `.button { padding: var(--spacing-small); }`
	ctx.docs.DidOpen(uri, "css", 1, cssContent)

	result, err := DocumentColor(ctx, glspCtx, &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})

	require.NoError(t, err)
	assert.Empty(t, result) // Should not include non-color tokens
}

func TestDocumentColor_NonCSSDocument(t *testing.T) {
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	uri := "file:///test.json"
	jsonContent := `{"color": {"$value": "#ff0000"}}`
	ctx.docs.DidOpen(uri, "json", 1, jsonContent)

	result, err := DocumentColor(ctx, glspCtx, &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDocumentColor_DocumentNotFound(t *testing.T) {
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	result, err := DocumentColor(ctx, glspCtx, &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nonexistent.css"},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestColorPresentation_AllFormats(t *testing.T) {
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	color := protocol.Color{
		Red:   1.0,
		Green: 0.0,
		Blue:  0.0,
		Alpha: 1.0,
	}

	result, err := ColorPresentation(ctx, glspCtx, &protocol.ColorPresentationParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
		Color:        color,
	})

	require.NoError(t, err)
	require.Len(t, result, 4)

	// Check format labels
	labels := make([]string, len(result))
	for i, p := range result {
		labels[i] = p.Label
	}

	assert.Contains(t, labels, "#ff0000")               // Hex
	assert.Contains(t, labels, "rgb(255, 0, 0)")        // RGB
	assert.Contains(t, labels, "rgba(255, 0, 0, 1.00)") // RGBA
	// HSL format should also be present
	foundHSL := false
	for _, label := range labels {
		if len(label) > 3 && label[:3] == "hsl" {
			foundHSL = true
			break
		}
	}
	assert.True(t, foundHSL, "Should include HSL format")
}

func TestColorPresentation_WithAlpha(t *testing.T) {
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	color := protocol.Color{
		Red:   1.0,
		Green: 0.0,
		Blue:  0.0,
		Alpha: 0.5,
	}

	result, err := ColorPresentation(ctx, glspCtx, &protocol.ColorPresentationParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
		Color:        color,
	})

	require.NoError(t, err)
	require.Len(t, result, 4)

	// Hex with alpha should be 8 digits
	foundHexAlpha := false
	for _, p := range result {
		if len(p.Label) == 9 && p.Label[0] == '#' {
			foundHexAlpha = true
			assert.Equal(t, "#ff00007f", p.Label) // 0x7F = 127 = uint8(0.5 * 255)
			break
		}
	}
	assert.True(t, foundHexAlpha, "Should include hex with alpha")
}

func TestRgbToHSL(t *testing.T) {
	tests := []struct {
		name     string
		r, g, b  float64
		h, s, l  float64
	}{
		{
			name: "red",
			r:    1.0,
			g:    0.0,
			b:    0.0,
			h:    0.0,
			s:    1.0,
			l:    0.5,
		},
		{
			name: "green",
			r:    0.0,
			g:    1.0,
			b:    0.0,
			h:    120.0,
			s:    1.0,
			l:    0.5,
		},
		{
			name: "blue",
			r:    0.0,
			g:    0.0,
			b:    1.0,
			h:    240.0,
			s:    1.0,
			l:    0.5,
		},
		{
			name: "black",
			r:    0.0,
			g:    0.0,
			b:    0.0,
			h:    0.0,
			s:    0.0,
			l:    0.0,
		},
		{
			name: "white",
			r:    1.0,
			g:    1.0,
			b:    1.0,
			h:    0.0,
			s:    0.0,
			l:    1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, s, l := rgbToHSL(tt.r, tt.g, tt.b)
			assert.InDelta(t, tt.h, h, 0.1, "Hue mismatch")
			assert.InDelta(t, tt.s, s, 0.01, "Saturation mismatch")
			assert.InDelta(t, tt.l, l, 0.01, "Lightness mismatch")
		})
	}
}

// TestParseColor tests the parseColor helper function
func TestParseColor(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    *protocol.Color
		expectError bool
	}{
		{
			name:  "6-digit hex color",
			input: "#ff0000",
			expected: &protocol.Color{
				Red:   1.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "6-digit hex color uppercase",
			input: "#FF0000",
			expected: &protocol.Color{
				Red:   1.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "6-digit hex color with whitespace",
			input: "  #00ff00  ",
			expected: &protocol.Color{
				Red:   0.0,
				Green: 1.0,
				Blue:  0.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "3-digit hex color",
			input: "#f00",
			expected: &protocol.Color{
				Red:   1.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "3-digit hex color - blue",
			input: "#00f",
			expected: &protocol.Color{
				Red:   0.0,
				Green: 0.0,
				Blue:  1.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "8-digit hex color with alpha",
			input: "#ff000080",
			expected: &protocol.Color{
				Red:   1.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: protocol.Decimal(128.0 / 255.0), // ~0.502
			},
			expectError: false,
		},
		{
			name:  "8-digit hex color with full alpha",
			input: "#0000ffff",
			expected: &protocol.Color{
				Red:   0.0,
				Green: 0.0,
				Blue:  1.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "8-digit hex color with zero alpha",
			input: "#ff000000",
			expected: &protocol.Color{
				Red:   1.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: 0.0,
			},
			expectError: false,
		},
		{
			name:        "invalid hex - wrong length",
			input:       "#ff00",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "invalid hex - non-hex characters",
			input:       "#gggggg",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "rgb() format not supported yet",
			input:       "rgb(255, 0, 0)",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "named color not supported",
			input:       "red",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "just hash",
			input:       "#",
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseColor(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)

				// Compare with small tolerance for floating point
				const tolerance = 0.001
				assert.InDelta(t, tt.expected.Red, result.Red, tolerance, "Red channel mismatch")
				assert.InDelta(t, tt.expected.Green, result.Green, tolerance, "Green channel mismatch")
				assert.InDelta(t, tt.expected.Blue, result.Blue, tolerance, "Blue channel mismatch")
				assert.InDelta(t, tt.expected.Alpha, result.Alpha, tolerance, "Alpha channel mismatch")
			}
		})
	}
}
