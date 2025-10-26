package workspace

import (
	"testing"

	"bennypowers.dev/dtls/lsp/testutil"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestDidChangeConfiguration_WithValidConfig(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	ctx.SetGLSPContext(glspCtx)

	// Prepare configuration with tokens files
	settings := map[string]any{
		"designTokensLanguageServer": map[string]any{
			"prefix": "--custom",
			"tokensFiles": []any{
				"tokens.json",
			},
		},
	}

	params := &protocol.DidChangeConfigurationParams{
		Settings: settings,
	}

	err := DidChangeConfiguration(ctx, glspCtx, params)
	require.NoError(t, err)

	// Verify config was updated
	config := ctx.GetConfig()
	assert.Equal(t, "--custom", config.Prefix)
}

func TestDidChangeConfiguration_WithNilSettings(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}

	params := &protocol.DidChangeConfigurationParams{
		Settings: nil,
	}

	err := DidChangeConfiguration(ctx, glspCtx, params)
	require.NoError(t, err)

	// Should use default config
	config := ctx.GetConfig()
	assert.Equal(t, types.DefaultConfig().Prefix, config.Prefix)
}

func TestDidChangeConfiguration_WithInvalidSettings(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}

	// Settings that's not a map
	params := &protocol.DidChangeConfigurationParams{
		Settings: "invalid",
	}

	err := DidChangeConfiguration(ctx, glspCtx, params)
	// Should not error (warns and uses defaults)
	require.NoError(t, err)
}

func TestDidChangeConfiguration_WithAlternateKey(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}

	// Using hyphenated key instead of camelCase
	settings := map[string]any{
		"design-tokens-language-server": map[string]any{
			"prefix": "--alt",
		},
	}

	params := &protocol.DidChangeConfigurationParams{
		Settings: settings,
	}

	err := DidChangeConfiguration(ctx, glspCtx, params)
	require.NoError(t, err)

	config := ctx.GetConfig()
	assert.Equal(t, "--alt", config.Prefix)
}

func TestDidChangeConfiguration_WithoutGLSPContext(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	// Don't set GLSP context

	settings := map[string]any{
		"designTokensLanguageServer": map[string]any{
			"prefix": "--test",
		},
	}

	params := &protocol.DidChangeConfigurationParams{
		Settings: settings,
	}

	// Should not panic when glspCtx is nil
	err := DidChangeConfiguration(ctx, nil, params)
	require.NoError(t, err)
}

func TestParseConfiguration_DefaultConfig(t *testing.T) {
	config, err := parseConfiguration(nil)
	require.NoError(t, err)
	assert.Equal(t, types.DefaultConfig(), config)
}

func TestParseConfiguration_ValidSettings(t *testing.T) {
	settings := map[string]any{
		"designTokensLanguageServer": map[string]any{
			"prefix": "--my-prefix",
			"tokensFiles": []any{
				"tokens/colors.json",
				"tokens/spacing.json",
			},
			"groupMarkers": []any{"value"},
		},
	}

	config, err := parseConfiguration(settings)
	require.NoError(t, err)
	assert.Equal(t, "--my-prefix", config.Prefix)
	assert.Len(t, config.TokensFiles, 2)
	assert.Len(t, config.GroupMarkers, 1)
	assert.Equal(t, "value", config.GroupMarkers[0])
}

func TestParseConfiguration_WithComplexTokensFiles(t *testing.T) {
	settings := map[string]any{
		"designTokensLanguageServer": map[string]any{
			"prefix": "--",
			"tokensFiles": []any{
				"simple.json",
				map[string]any{
					"path":   "complex.json",
					"prefix": "--override",
				},
			},
		},
	}

	config, err := parseConfiguration(settings)
	require.NoError(t, err)
	assert.Len(t, config.TokensFiles, 2)
}

func TestParseConfiguration_InvalidMap(t *testing.T) {
	// Settings that's not a map
	settings := "not a map"

	_, err := parseConfiguration(settings)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a map")
}

func TestParseConfiguration_MissingKey(t *testing.T) {
	// Map without our configuration key
	settings := map[string]any{
		"someOtherKey": map[string]any{
			"value": "test",
		},
	}

	config, err := parseConfiguration(settings)
	require.NoError(t, err)
	// Should return default config
	assert.Equal(t, types.DefaultConfig(), config)
}

func TestParseConfiguration_InvalidJSON(t *testing.T) {
	// Create a value that can't be marshaled to JSON
	// (functions can't be marshaled)
	settings := map[string]any{
		"designTokensLanguageServer": map[string]any{
			"invalidField": func() {}, // Functions can't be marshaled to JSON
		},
	}

	_, err := parseConfiguration(settings)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "marshal")
}
