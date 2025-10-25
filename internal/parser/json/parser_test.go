package json_test

import (
	"os"
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
      "$deprecated": "Use color.primary instead"
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

// TestParseFile tests parsing a JSON file from disk
func TestParseFile(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "tokens-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	// Write test data
	jsonData := `{
  "color": {
    "primary": {
      "$value": "#0000ff",
      "$type": "color",
      "$description": "Primary color"
    },
    "secondary": {
      "$value": "#ff0000",
      "$type": "color"
    }
  }
}`
	_, err = tmpfile.Write([]byte(jsonData))
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	// Parse the file
	parser := json.NewParser()
	tokens, err := parser.ParseFile(tmpfile.Name(), "test")
	require.NoError(t, err)
	require.Len(t, tokens, 2)

	// Tokens should be returned in alphabetical order (primary, then secondary)
	assert.Equal(t, "color-primary", tokens[0].Name)
	assert.Equal(t, "#0000ff", tokens[0].Value)
	assert.Equal(t, "color", tokens[0].Type)
	assert.Equal(t, "Primary color", tokens[0].Description)
	assert.Equal(t, "test", tokens[0].Prefix)

	// Verify second token
	assert.Equal(t, "color-secondary", tokens[1].Name)
	assert.Equal(t, "#ff0000", tokens[1].Value)
	assert.Equal(t, "color", tokens[1].Type)
	assert.Equal(t, "test", tokens[1].Prefix)
}

// TestParseFileNotFound tests error handling when file doesn't exist
func TestParseFileNotFound(t *testing.T) {
	parser := json.NewParser()
	_, err := parser.ParseFile("/nonexistent/file.json", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
}

// TestParseFileInvalidJSON tests error handling for invalid JSON
func TestParseFileInvalidJSON(t *testing.T) {
	// Create a temporary file with invalid JSON
	tmpfile, err := os.CreateTemp("", "invalid-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte("{ invalid json }"))
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	// Attempt to parse
	parser := json.NewParser()
	_, err = parser.ParseFile(tmpfile.Name(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse file")
}
