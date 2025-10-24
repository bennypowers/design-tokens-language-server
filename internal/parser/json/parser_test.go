package json_test

import (
	"testing"

	"github.com/bennypowers/design-tokens-language-server/internal/parser/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseSimpleTokens tests parsing a simple DTCG token file
func TestParseSimpleTokens(t *testing.T) {
	jsonData := `{
  "color": {
    "primary": {
      "$value": "#0000ff",
      "$type": "color",
      "$description": "Primary brand color"
    }
  }
}`

	parser := json.NewParser()
	tokens, err := parser.Parse([]byte(jsonData), "")
	require.NoError(t, err)
	require.Len(t, tokens, 1)

	token := tokens[0]
	assert.Equal(t, "color-primary", token.Name)
	assert.Equal(t, "#0000ff", token.Value)
	assert.Equal(t, "color", token.Type)
	assert.Equal(t, "Primary brand color", token.Description)
}

// TestParseNestedTokens tests parsing nested token groups
func TestParseNestedTokens(t *testing.T) {
	jsonData := `{
  "color": {
    "brand": {
      "primary": {
        "$value": "#0000ff",
        "$type": "color"
      },
      "secondary": {
        "$value": "#ff0000",
        "$type": "color"
      }
    }
  }
}`

	parser := json.NewParser()
	tokens, err := parser.Parse([]byte(jsonData), "")
	require.NoError(t, err)
	require.Len(t, tokens, 2)

	// Check token names are properly namespaced
	names := map[string]bool{}
	for _, token := range tokens {
		names[token.Name] = true
	}
	assert.True(t, names["color-brand-primary"])
	assert.True(t, names["color-brand-secondary"])
}

// TestParseMultipleTypes tests parsing tokens of different types
func TestParseMultipleTypes(t *testing.T) {
	jsonData := `{
  "color": {
    "primary": {
      "$value": "#0000ff",
      "$type": "color"
    }
  },
  "spacing": {
    "small": {
      "$value": "8px",
      "$type": "dimension"
    }
  },
  "font": {
    "size": {
      "base": {
        "$value": "16px",
        "$type": "dimension"
      }
    }
  }
}`

	parser := json.NewParser()
	tokens, err := parser.Parse([]byte(jsonData), "")
	require.NoError(t, err)
	require.Len(t, tokens, 3)

	typeCount := map[string]int{}
	for _, token := range tokens {
		typeCount[token.Type]++
	}
	assert.Equal(t, 1, typeCount["color"])
	assert.Equal(t, 2, typeCount["dimension"])
}

// TestParseWithPrefix tests parsing with a CSS variable prefix
func TestParseWithPrefix(t *testing.T) {
	jsonData := `{
  "color": {
    "primary": {
      "$value": "#0000ff",
      "$type": "color"
    }
  }
}`

	parser := json.NewParser()
	tokens, err := parser.Parse([]byte(jsonData), "my-prefix")
	require.NoError(t, err)
	require.Len(t, tokens, 1)

	token := tokens[0]
	assert.Equal(t, "color-primary", token.Name)
	assert.Equal(t, "my-prefix", token.Prefix)
	assert.Equal(t, "--my-prefix-color-primary", token.CSSVariableName())
}

// TestParseDeprecatedTokens tests parsing tokens with deprecated flag
func TestParseDeprecatedTokens(t *testing.T) {
	jsonData := `{
  "color": {
    "old-primary": {
      "$value": "#0000ff",
      "$type": "color",
      "deprecated": true,
      "deprecationMessage": "Use color.primary instead"
    }
  }
}`

	parser := json.NewParser()
	tokens, err := parser.Parse([]byte(jsonData), "")
	require.NoError(t, err)
	require.Len(t, tokens, 1)

	token := tokens[0]
	assert.True(t, token.Deprecated)
	assert.Equal(t, "Use color.primary instead", token.DeprecationMessage)
}

// TestParseInvalidJSON tests error handling for invalid JSON
func TestParseInvalidJSON(t *testing.T) {
	jsonData := `{ invalid json }`

	parser := json.NewParser()
	_, err := parser.Parse([]byte(jsonData), "")
	assert.Error(t, err)
}

// TestParseEmptyJSON tests parsing empty JSON
func TestParseEmptyJSON(t *testing.T) {
	jsonData := `{}`

	parser := json.NewParser()
	tokens, err := parser.Parse([]byte(jsonData), "")
	require.NoError(t, err)
	assert.Empty(t, tokens)
}

// TestParseJSONWithComments tests parsing JSONC (JSON with comments)
func TestParseJSONWithComments(t *testing.T) {
	jsonData := `{
  // This is a comment
  "color": {
    /* Multi-line
       comment */
    "primary": {
      "$value": "#0000ff",
      "$type": "color"
    }
  }
}`

	parser := json.NewParser()
	tokens, err := parser.Parse([]byte(jsonData), "")
	require.NoError(t, err)
	require.Len(t, tokens, 1)

	token := tokens[0]
	assert.Equal(t, "color-primary", token.Name)
	assert.Equal(t, "#0000ff", token.Value)
}
