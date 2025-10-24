package yaml_test

import (
	"testing"

	"github.com/bennypowers/design-tokens-language-server/internal/parser/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseSimpleYAMLTokens tests parsing a simple DTCG YAML token file
func TestParseSimpleYAMLTokens(t *testing.T) {
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
	yamlData := `invalid: yaml: data: ::::`

	parser := yaml.NewParser()
	_, err := parser.Parse([]byte(yamlData), "")
	assert.Error(t, err)
}

// TestParseEmptyYAML tests parsing empty YAML
func TestParseEmptyYAML(t *testing.T) {
	yamlData := ``

	parser := yaml.NewParser()
	tokens, err := parser.Parse([]byte(yamlData), "")
	require.NoError(t, err)
	assert.Empty(t, tokens)
}
