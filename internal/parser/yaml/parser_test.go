package yaml_test

import (
	"os"
	"testing"

	"bennypowers.dev/dtls/internal/parser/yaml"
	"bennypowers.dev/dtls/internal/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseSimpleYAMLTokens tests parsing a simple DTCG YAML token file
func TestParseSimpleYAMLTokens(t *testing.T) {
	t.Parallel()
	yamlData := `color:
  primary:
    $value: "#0000ff"
    $type: color
    $description: Primary brand color
`

	parser := yaml.NewParser()
	tokens, err := parser.Parse([]byte(yamlData), "")
	require.NoError(t, err)
	require.Len(t, tokens, 1)

	token := tokens[0]
	assert.Equal(t, "color-primary", token.Name)
	assert.Equal(t, "#0000ff", token.Value)
	assert.Equal(t, "color", token.Type)
	assert.Equal(t, "Primary brand color", token.Description)
}

// TestParseNestedYAMLTokens tests parsing nested YAML token groups
func TestParseNestedYAMLTokens(t *testing.T) {
	t.Parallel()
	yamlData := `color:
  brand:
    primary:
      $value: "#0000ff"
      $type: color
    secondary:
      $value: "#ff0000"
      $type: color
`

	parser := yaml.NewParser()
	tokens, err := parser.Parse([]byte(yamlData), "")
	require.NoError(t, err)
	require.Len(t, tokens, 2)

	names := map[string]bool{}
	for _, token := range tokens {
		names[token.Name] = true
	}
	assert.True(t, names["color-brand-primary"])
	assert.True(t, names["color-brand-secondary"])
}

// TestParseYAMLWithPrefix tests parsing with a CSS variable prefix
func TestParseYAMLWithPrefix(t *testing.T) {
	t.Parallel()
	yamlData := `color:
  primary:
    $value: "#0000ff"
    $type: color
`

	parser := yaml.NewParser()
	tokens, err := parser.Parse([]byte(yamlData), "my-prefix")
	require.NoError(t, err)
	require.Len(t, tokens, 1)

	token := tokens[0]
	assert.Equal(t, "my-prefix", token.Prefix)
	assert.Equal(t, "--my-prefix-color-primary", token.CSSVariableName())
}

// TestParseInvalidYAML tests error handling for invalid YAML
func TestParseInvalidYAML(t *testing.T) {
	t.Parallel()
	yamlData := `invalid: yaml: data: ::::`

	parser := yaml.NewParser()
	_, err := parser.Parse([]byte(yamlData), "")
	assert.Error(t, err)
}

// TestParseEmptyYAML tests parsing empty YAML
func TestParseEmptyYAML(t *testing.T) {
	t.Parallel()
	yamlData := ``

	parser := yaml.NewParser()
	tokens, err := parser.Parse([]byte(yamlData), "")
	require.NoError(t, err)
	assert.Empty(t, tokens)
}

// TestParseFile tests parsing a YAML file from disk
func TestParseFile(t *testing.T) {
	t.Parallel()
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "tokens-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	// Write test data
	yamlData := `color:
  primary:
    $value: "#0000ff"
    $type: color
    $description: Primary color
  secondary:
    $value: "#ff0000"
    $type: color
`
	_, err = tmpfile.Write([]byte(yamlData))
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	// Parse the file
	parser := yaml.NewParser()
	tokens, err := parser.ParseFile(tmpfile.Name(), "test")
	require.NoError(t, err)
	require.Len(t, tokens, 2)

	// Verify first token
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
	t.Parallel()
	parser := yaml.NewParser()
	_, err := parser.ParseFile("/nonexistent/file.yaml", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
}

// TestParseFileInvalidYAML tests error handling for invalid YAML
func TestParseFileInvalidYAML(t *testing.T) {
	t.Parallel()
	// Create a temporary file with invalid YAML
	tmpfile, err := os.CreateTemp("", "invalid-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	_, err = tmpfile.Write([]byte("invalid: yaml: content: ["))
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	// Attempt to parse
	parser := yaml.NewParser()
	_, err = parser.ParseFile(tmpfile.Name(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse file")
}

// TestParseWithGroupMarkers tests parsing tokens where a node is both a token and a group
func TestParseWithGroupMarkers(t *testing.T) {
	t.Run("basic group marker - node with $value and children", func(t *testing.T) {
		yamlData := `
color:
  $value: '#ff0000'
  $type: color
  primary:
    $value: '#0000ff'
    $type: color
`

		parser := yaml.NewParser()
		groupMarkers := []string{"color"}
		tokens, err := parser.ParseWithGroupMarkers([]byte(yamlData), "", groupMarkers)
		require.NoError(t, err)
		require.Len(t, tokens, 2, "Should extract both 'color' and 'color-primary'")

		// Check color token
		colorToken := findTokenByName(tokens, "color")
		require.NotNil(t, colorToken, "Should have 'color' token")
		assert.Equal(t, "#ff0000", colorToken.Value)
		assert.Equal(t, "color", colorToken.Type)

		// Check color-primary token
		primaryToken := findTokenByName(tokens, "color-primary")
		require.NotNil(t, primaryToken, "Should have 'color-primary' token")
		assert.Equal(t, "#0000ff", primaryToken.Value)
		assert.Equal(t, "color", primaryToken.Type)
	})

	t.Run("nested group marker", func(t *testing.T) {
		yamlData := `
spacing:
  scale:
    $value: 4px
    $type: dimension
    small:
      $value: 8px
      $type: dimension
    large:
      $value: 16px
      $type: dimension
`

		parser := yaml.NewParser()
		groupMarkers := []string{"scale"}
		tokens, err := parser.ParseWithGroupMarkers([]byte(yamlData), "", groupMarkers)
		require.NoError(t, err)
		require.Len(t, tokens, 3)

		// Check scale token
		scaleToken := findTokenByName(tokens, "spacing-scale")
		require.NotNil(t, scaleToken)
		assert.Equal(t, "4px", scaleToken.Value)

		// Check children
		smallToken := findTokenByName(tokens, "spacing-scale-small")
		require.NotNil(t, smallToken)
		assert.Equal(t, "8px", smallToken.Value)

		largeToken := findTokenByName(tokens, "spacing-scale-large")
		require.NotNil(t, largeToken)
		assert.Equal(t, "16px", largeToken.Value)
	})

	t.Run("multiple group markers", func(t *testing.T) {
		yamlData := `
color:
  $value: '#000000'
  primary:
    $value: '#0000ff'
size:
  $value: 16px
  small:
    $value: 12px
`

		parser := yaml.NewParser()
		groupMarkers := []string{"color", "size"}
		tokens, err := parser.ParseWithGroupMarkers([]byte(yamlData), "", groupMarkers)
		require.NoError(t, err)
		require.Len(t, tokens, 4)

		assert.NotNil(t, findTokenByName(tokens, "color"))
		assert.NotNil(t, findTokenByName(tokens, "color-primary"))
		assert.NotNil(t, findTokenByName(tokens, "size"))
		assert.NotNil(t, findTokenByName(tokens, "size-small"))
	})
}

// TestParseYAMLWithExtensions tests parsing tokens with $extensions
func TestParseYAMLWithExtensions(t *testing.T) {
	t.Run("simple extensions", func(t *testing.T) {
		yamlData := `
color:
  primary:
    $value: "#0000ff"
    $type: color
    $extensions:
      com.figma:
        nodeId: "123:456"
      custom:
        category: brand
`
		parser := yaml.NewParser()
		tokens, err := parser.Parse([]byte(yamlData), "")
		require.NoError(t, err)
		require.Len(t, tokens, 1)

		token := tokens[0]
		require.NotNil(t, token.Extensions)
		assert.Contains(t, token.Extensions, "com.figma")
		assert.Contains(t, token.Extensions, "custom")

		figma := token.Extensions["com.figma"].(map[string]interface{})
		assert.Equal(t, "123:456", figma["nodeId"])

		custom := token.Extensions["custom"].(map[string]interface{})
		assert.Equal(t, "brand", custom["category"])
	})

	t.Run("nested extensions", func(t *testing.T) {
		yamlData := `
color:
  primary:
    $value: "#0000ff"
    $extensions:
      org.example:
        metadata:
          version: "1.0"
          deprecated: false
`
		parser := yaml.NewParser()
		tokens, err := parser.Parse([]byte(yamlData), "")
		require.NoError(t, err)

		token := tokens[0]
		require.NotNil(t, token.Extensions)

		org := token.Extensions["org.example"].(map[string]interface{})
		metadata := org["metadata"].(map[string]interface{})
		assert.Equal(t, "1.0", metadata["version"])
		assert.Equal(t, false, metadata["deprecated"])
	})

	t.Run("extensions with arrays", func(t *testing.T) {
		yamlData := `
color:
  primary:
    $value: "#0000ff"
    $extensions:
      tags:
        - brand
        - primary
        - blue
`
		parser := yaml.NewParser()
		tokens, err := parser.Parse([]byte(yamlData), "")
		require.NoError(t, err)

		token := tokens[0]
		require.NotNil(t, token.Extensions)

		tags := token.Extensions["tags"].([]interface{})
		require.Len(t, tags, 3)
		assert.Equal(t, "brand", tags[0])
		assert.Equal(t, "primary", tags[1])
		assert.Equal(t, "blue", tags[2])
	})

	t.Run("empty extensions", func(t *testing.T) {
		yamlData := `
color:
  primary:
    $value: "#0000ff"
    $extensions: {}
`
		parser := yaml.NewParser()
		tokens, err := parser.Parse([]byte(yamlData), "")
		require.NoError(t, err)

		token := tokens[0]
		require.NotNil(t, token.Extensions)
		assert.Empty(t, token.Extensions)
	})

	t.Run("no extensions", func(t *testing.T) {
		yamlData := `
color:
  primary:
    $value: "#0000ff"
    $type: color
`
		parser := yaml.NewParser()
		tokens, err := parser.Parse([]byte(yamlData), "")
		require.NoError(t, err)

		token := tokens[0]
		assert.Nil(t, token.Extensions)
	})

	t.Run("extensions with multiple data types", func(t *testing.T) {
		yamlData := `
color:
  primary:
    $value: "#0000ff"
    $extensions:
      stringValue: test
      numberValue: 42
      boolValue: true
      nullValue: null
`
		parser := yaml.NewParser()
		tokens, err := parser.Parse([]byte(yamlData), "")
		require.NoError(t, err)

		token := tokens[0]
		require.NotNil(t, token.Extensions)
		assert.Equal(t, "test", token.Extensions["stringValue"])
		assert.Equal(t, 42, token.Extensions["numberValue"])
		assert.Equal(t, true, token.Extensions["boolValue"])
		assert.Nil(t, token.Extensions["nullValue"])
	})
}

// Helper function to find a token by name in a slice
func findTokenByName(tokens []*tokens.Token, name string) *tokens.Token {
	for _, token := range tokens {
		if token.Name == name {
			return token
		}
	}
	return nil
}
