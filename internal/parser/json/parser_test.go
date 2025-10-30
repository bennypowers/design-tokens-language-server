package json_test

import (
	"os"
	"testing"

	"bennypowers.dev/dtls/internal/parser/json"
	"bennypowers.dev/dtls/internal/tokens"
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
	parsed, err := parser.Parse([]byte(jsonData), "")
	require.NoError(t, err)
	require.Len(t, parsed, 1)

	token := parsed[0]
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
	parsed, err := parser.Parse([]byte(jsonData), "")
	require.NoError(t, err)
	require.Len(t, parsed, 2)

	// Check token names are properly namespaced
	names := map[string]bool{}
	for _, token := range parsed {
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
	parsed, err := parser.Parse([]byte(jsonData), "")
	require.NoError(t, err)
	require.Len(t, parsed, 3)

	typeCount := map[string]int{}
	for _, token := range parsed {
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
	parsed, err := parser.Parse([]byte(jsonData), "my-prefix")
	require.NoError(t, err)
	require.Len(t, parsed, 1)

	token := parsed[0]
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
	parsed, err := parser.Parse([]byte(jsonData), "")
	require.NoError(t, err)
	require.Len(t, parsed, 1)

	token := parsed[0]
	assert.True(t, token.Deprecated)
	assert.Equal(t, "Use color.primary instead", token.DeprecationMessage)
}

// TestParseInvalidJSON tests error handling for invalid JSON
func TestParseInvalidJSON(t *testing.T) {
	jsonData := `{ "color": { "primary": { "$value": #invalid } } }`

	parser := json.NewParser()
	_, err := parser.Parse([]byte(jsonData), "")
	assert.Error(t, err)
}

// TestParseEmptyJSON tests parsing empty JSON
func TestParseEmptyJSON(t *testing.T) {
	jsonData := `{}`

	parser := json.NewParser()
	parsed, err := parser.Parse([]byte(jsonData), "")
	require.NoError(t, err)
	assert.Empty(t, parsed)
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
	parsed, err := parser.Parse([]byte(jsonData), "")
	require.NoError(t, err)
	require.Len(t, parsed, 1)

	token := parsed[0]
	assert.Equal(t, "color-primary", token.Name)
	assert.Equal(t, "#0000ff", token.Value)
}

// TestParseFile tests parsing a JSON file from disk
func TestParseFile(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "tokens-*.json")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpfile.Name()) }()

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
	_, err = tmpfile.WriteString(jsonData)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	// Parse the file
	parser := json.NewParser()
	parsed, err := parser.ParseFile(tmpfile.Name(), "test")
	require.NoError(t, err)
	require.Len(t, parsed, 2)

	// Tokens should be returned in alphabetical order (primary, then secondary)
	assert.Equal(t, "color-primary", parsed[0].Name)
	assert.Equal(t, "#0000ff", parsed[0].Value)
	assert.Equal(t, "color", parsed[0].Type)
	assert.Equal(t, "Primary color", parsed[0].Description)
	assert.Equal(t, "test", parsed[0].Prefix)

	// Verify second token
	assert.Equal(t, "color-secondary", parsed[1].Name)
	assert.Equal(t, "#ff0000", parsed[1].Value)
	assert.Equal(t, "color", parsed[1].Type)
	assert.Equal(t, "test", parsed[1].Prefix)
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
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	_, err = tmpfile.WriteString(`{ "color": { "primary": { "$value": #invalid } } }`)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	// Attempt to parse
	parser := json.NewParser()
	_, err = parser.ParseFile(tmpfile.Name(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse file")
}

// TestParseWithGroupMarkers tests parsing tokens where a node is both a token and a group
func TestParseWithGroupMarkers(t *testing.T) {
	t.Run("basic group marker - node with $value and children", func(t *testing.T) {
		jsonData := `{
  "color": {
    "$value": "#ff0000",
    "$type": "color",
    "primary": {
      "$value": "#0000ff",
      "$type": "color"
    }
  }
}`

		parser := json.NewParser()
		groupMarkers := []string{"color"}
		parsed, err := parser.ParseWithGroupMarkers([]byte(jsonData), "", groupMarkers)
		require.NoError(t, err)
		require.Len(t, parsed, 2, "Should extract both 'color' and 'color-primary'")

		// Check color token
		colorToken := findTokenByName(parsed, "color")
		require.NotNil(t, colorToken, "Should have 'color' token")
		assert.Equal(t, "#ff0000", colorToken.Value)
		assert.Equal(t, "color", colorToken.Type)

		// Check color-primary token
		primaryToken := findTokenByName(parsed, "color-primary")
		require.NotNil(t, primaryToken, "Should have 'color-primary' token")
		assert.Equal(t, "#0000ff", primaryToken.Value)
		assert.Equal(t, "color", primaryToken.Type)
	})

	t.Run("nested group marker", func(t *testing.T) {
		jsonData := `{
  "spacing": {
    "scale": {
      "$value": "4px",
      "$type": "dimension",
      "small": {
        "$value": "8px",
        "$type": "dimension"
      },
      "large": {
        "$value": "16px",
        "$type": "dimension"
      }
    }
  }
}`

		parser := json.NewParser()
		groupMarkers := []string{"scale"}
		parsed, err := parser.ParseWithGroupMarkers([]byte(jsonData), "", groupMarkers)
		require.NoError(t, err)
		require.Len(t, parsed, 3)

		// Check scale token
		scaleToken := findTokenByName(parsed, "spacing-scale")
		require.NotNil(t, scaleToken)
		assert.Equal(t, "4px", scaleToken.Value)

		// Check children
		smallToken := findTokenByName(parsed, "spacing-scale-small")
		require.NotNil(t, smallToken)
		assert.Equal(t, "8px", smallToken.Value)

		largeToken := findTokenByName(parsed, "spacing-scale-large")
		require.NotNil(t, largeToken)
		assert.Equal(t, "16px", largeToken.Value)
	})

	t.Run("multiple group markers", func(t *testing.T) {
		jsonData := `{
  "color": {
    "$value": "#000000",
    "primary": {
      "$value": "#0000ff"
    }
  },
  "size": {
    "$value": "16px",
    "small": {
      "$value": "12px"
    }
  }
}`

		parser := json.NewParser()
		groupMarkers := []string{"color", "size"}
		parsed, err := parser.ParseWithGroupMarkers([]byte(jsonData), "", groupMarkers)
		require.NoError(t, err)
		require.Len(t, parsed, 4)

		assert.NotNil(t, findTokenByName(parsed, "color"))
		assert.NotNil(t, findTokenByName(parsed, "color-primary"))
		assert.NotNil(t, findTokenByName(parsed, "size"))
		assert.NotNil(t, findTokenByName(parsed, "size-small"))
	})

	t.Run("without group markers - should fail on node with $value and children", func(t *testing.T) {
		jsonData := `{
  "color": {
    "$value": "#ff0000",
    "primary": {
      "$value": "#0000ff"
    }
  }
}`

		parser := json.NewParser()
		parsed, err := parser.Parse([]byte(jsonData), "")

		// Without groupMarkers, this structure should still parse but only extract the parent token
		// (ignoring the children since the parent has $value)
		require.NoError(t, err)
		require.Len(t, parsed, 1, "Should only extract parent token when not using groupMarkers")
		assert.Equal(t, "color", parsed[0].Name)
	})
}

func TestParseTracksPositions(t *testing.T) {
	jsonData := []byte(`{
  "color": {
    "primary": {
      "$value": "#ff0000",
      "$type": "color"
    },
    "secondary": {
      "$value": "#00ff00"
    }
  }
}`)

	parser := json.NewParser()
	parsed, err := parser.Parse(jsonData, "")
	require.NoError(t, err)
	require.Len(t, parsed, 2)

	// Find the primary token
	primaryToken := findTokenByName(parsed, "color-primary")
	require.NotNil(t, primaryToken, "should find primary token")
	// Line 2 is where "primary" key is
	assert.Equal(t, uint32(2), primaryToken.Line, "primary token should be on line 2")
	assert.Greater(t, primaryToken.Character, uint32(0), "primary token should have non-zero character position")

	// Find the secondary token
	secondaryToken := findTokenByName(parsed, "color-secondary")
	require.NotNil(t, secondaryToken, "should find secondary token")
	// Line 6 is where "secondary" key is
	assert.Equal(t, uint32(6), secondaryToken.Line, "secondary token should be on line 6")
	assert.Greater(t, secondaryToken.Character, uint32(0), "secondary token should have non-zero character position")
}

// TestParseJSONWithExtensions tests parsing tokens with $extensions
func TestParseJSONWithExtensions(t *testing.T) {
	t.Run("simple extensions", func(t *testing.T) {
		jsonData := `{
  "color": {
    "primary": {
      "$value": "#0000ff",
      "$type": "color",
      "$extensions": {
        "com.figma": {
          "nodeId": "123:456"
        },
        "custom": {
          "category": "brand"
        }
      }
    }
  }
}`
		parser := json.NewParser()
		parsed, err := parser.Parse([]byte(jsonData), "")
		require.NoError(t, err)
		require.Len(t, parsed, 1)

		token := parsed[0]
		require.NotNil(t, token.Extensions)
		assert.Contains(t, token.Extensions, "com.figma")
		assert.Contains(t, token.Extensions, "custom")

		figma := token.Extensions["com.figma"].(map[string]interface{})
		assert.Equal(t, "123:456", figma["nodeId"])

		custom := token.Extensions["custom"].(map[string]interface{})
		assert.Equal(t, "brand", custom["category"])
	})

	t.Run("nested extensions", func(t *testing.T) {
		jsonData := `{
  "color": {
    "primary": {
      "$value": "#0000ff",
      "$extensions": {
        "org.example": {
          "metadata": {
            "version": "1.0",
            "deprecated": false
          }
        }
      }
    }
  }
}`
		parser := json.NewParser()
		parsed, err := parser.Parse([]byte(jsonData), "")
		require.NoError(t, err)

		token := parsed[0]
		require.NotNil(t, token.Extensions)

		org := token.Extensions["org.example"].(map[string]interface{})
		metadata := org["metadata"].(map[string]interface{})
		assert.Equal(t, "1.0", metadata["version"])
		assert.Equal(t, false, metadata["deprecated"])
	})

	t.Run("extensions with arrays", func(t *testing.T) {
		jsonData := `{
  "color": {
    "primary": {
      "$value": "#0000ff",
      "$extensions": {
        "tags": ["brand", "primary", "blue"]
      }
    }
  }
}`
		parser := json.NewParser()
		parsed, err := parser.Parse([]byte(jsonData), "")
		require.NoError(t, err)

		token := parsed[0]
		require.NotNil(t, token.Extensions)

		tags := token.Extensions["tags"].([]interface{})
		require.Len(t, tags, 3)
		assert.Equal(t, "brand", tags[0])
		assert.Equal(t, "primary", tags[1])
		assert.Equal(t, "blue", tags[2])
	})

	t.Run("empty extensions", func(t *testing.T) {
		jsonData := `{
  "color": {
    "primary": {
      "$value": "#0000ff",
      "$extensions": {}
    }
  }
}`
		parser := json.NewParser()
		parsed, err := parser.Parse([]byte(jsonData), "")
		require.NoError(t, err)

		token := parsed[0]
		require.NotNil(t, token.Extensions)
		assert.Empty(t, token.Extensions)
	})

	t.Run("no extensions", func(t *testing.T) {
		jsonData := `{
  "color": {
    "primary": {
      "$value": "#0000ff",
      "$type": "color"
    }
  }
}`
		parser := json.NewParser()
		parsed, err := parser.Parse([]byte(jsonData), "")
		require.NoError(t, err)

		token := parsed[0]
		assert.Nil(t, token.Extensions)
	})

	t.Run("extensions with multiple data types", func(t *testing.T) {
		jsonData := `{
  "color": {
    "primary": {
      "$value": "#0000ff",
      "$extensions": {
        "stringValue": "test",
        "numberValue": 42,
        "boolValue": true,
        "nullValue": null
      }
    }
  }
}`
		parser := json.NewParser()
		parsed, err := parser.Parse([]byte(jsonData), "")
		require.NoError(t, err)

		token := parsed[0]
		require.NotNil(t, token.Extensions)
		assert.Equal(t, "test", token.Extensions["stringValue"])
		assert.Equal(t, 42, token.Extensions["numberValue"]) // yaml.v3 decodes as int when possible
		assert.Equal(t, true, token.Extensions["boolValue"])
		assert.Nil(t, token.Extensions["nullValue"])
	})

	t.Run("extensions with JSONC comments", func(t *testing.T) {
		jsonData := `{
  "color": {
    "primary": {
      "$value": "#0000ff",
      // Comment in extensions
      "$extensions": {
        "custom": "value"
      }
    }
  }
}`
		parser := json.NewParser()
		parsed, err := parser.Parse([]byte(jsonData), "")
		require.NoError(t, err)

		token := parsed[0]
		require.NotNil(t, token.Extensions)
		assert.Equal(t, "value", token.Extensions["custom"])
	})
}

// Helper function to find a token by name in a slice
func findTokenByName(parsed []*tokens.Token, name string) *tokens.Token {
	for _, token := range parsed {
		if token.Name == name {
			return token
		}
	}
	return nil
}
