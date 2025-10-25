package yaml_test

import (
	"os"
	"testing"

	"github.com/bennypowers/design-tokens-language-server/internal/parser/yaml"
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
	defer os.Remove(tmpfile.Name())

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
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte("invalid: yaml: content: ["))
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	// Attempt to parse
	parser := yaml.NewParser()
	_, err = parser.ParseFile(tmpfile.Name(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse file")
}
