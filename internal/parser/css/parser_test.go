package css_test

import (
	"testing"

	"github.com/bennypowers/design-tokens-language-server/internal/parser/css"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseSimpleCSSVariable tests parsing a simple CSS custom property declaration
func TestParseSimpleCSSVariable(t *testing.T) {
	cssCode := `:root {
  --color-primary: #0000ff;
}`

	parser := css.NewParser()
	result, err := parser.Parse(cssCode)
	require.NoError(t, err, "Parsing should not error")
	require.NotNil(t, result, "Parse result should not be nil")

	// Should find one variable declaration
	variables := result.Variables
	require.Len(t, variables, 1, "Should find one variable")

	variable := variables[0]
	assert.Equal(t, "--color-primary", variable.Name)
	assert.Equal(t, "#0000ff", variable.Value)
	assert.Equal(t, css.VariableDeclaration, variable.Type)

	// Check position information
	assert.Equal(t, uint32(1), variable.Range.Start.Line, "Variable should be on line 1 (0-indexed)")
	assert.Greater(t, variable.Range.Start.Character, uint32(0), "Variable should have character position")
}

// TestParseMultipleCSSVariables tests parsing multiple CSS custom properties
func TestParseMultipleCSSVariables(t *testing.T) {
	cssCode := `:root {
  --color-primary: #0000ff;
  --color-secondary: #ff0000;
  --spacing-small: 8px;
}`

	parser := css.NewParser()
	result, err := parser.Parse(cssCode)
	require.NoError(t, err)

	variables := result.Variables
	require.Len(t, variables, 3, "Should find three variables")

	// Check each variable
	expectedVars := map[string]string{
		"--color-primary":   "#0000ff",
		"--color-secondary": "#ff0000",
		"--spacing-small":   "8px",
	}

	for _, v := range variables {
		expectedValue, ok := expectedVars[v.Name]
		require.True(t, ok, "Variable %s should be in expected list", v.Name)
		assert.Equal(t, expectedValue, v.Value, "Variable %s should have correct value", v.Name)
	}
}

// TestParseVarFunctionCall tests parsing var() function calls
func TestParseVarFunctionCall(t *testing.T) {
	cssCode := `.button {
  color: var(--color-primary);
}`

	parser := css.NewParser()
	result, err := parser.Parse(cssCode)
	require.NoError(t, err)

	varCalls := result.VarCalls
	require.Len(t, varCalls, 1, "Should find one var() call")

	varCall := varCalls[0]
	assert.Equal(t, "--color-primary", varCall.TokenName)
	assert.Nil(t, varCall.Fallback, "Should have no fallback")
	assert.Equal(t, css.VarReference, varCall.Type)
}

// TestParseVarFunctionWithFallback tests parsing var() with fallback values
func TestParseVarFunctionWithFallback(t *testing.T) {
	cssCode := `.button {
  color: var(--color-primary, #000);
}`

	parser := css.NewParser()
	result, err := parser.Parse(cssCode)
	require.NoError(t, err)

	varCalls := result.VarCalls
	require.Len(t, varCalls, 1, "Should find one var() call")

	varCall := varCalls[0]
	assert.Equal(t, "--color-primary", varCall.TokenName)
	require.NotNil(t, varCall.Fallback, "Should have fallback")
	assert.Equal(t, "#000", *varCall.Fallback)
}

// TestParseNestedVarCalls tests parsing nested var() calls (fallback contains var())
func TestParseNestedVarCalls(t *testing.T) {
	cssCode := `.button {
  color: var(--color-primary, var(--color-base));
}`

	parser := css.NewParser()
	result, err := parser.Parse(cssCode)
	require.NoError(t, err)

	// Should find both var() calls
	varCalls := result.VarCalls
	assert.GreaterOrEqual(t, len(varCalls), 2, "Should find at least two var() calls")

	// First call should be --color-primary
	found := false
	for _, call := range varCalls {
		if call.TokenName == "--color-primary" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should find var(--color-primary)")

	// Second call should be --color-base
	found = false
	for _, call := range varCalls {
		if call.TokenName == "--color-base" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should find var(--color-base)")
}

// TestParseMixedContent tests parsing CSS with both declarations and var() calls
func TestParseMixedContent(t *testing.T) {
	cssCode := `:root {
  --color-primary: #0000ff;
}

.button {
  color: var(--color-primary, #000);
  background: var(--color-secondary);
}`

	parser := css.NewParser()
	result, err := parser.Parse(cssCode)
	require.NoError(t, err)

	// Should find variable declarations
	assert.Len(t, result.Variables, 1, "Should find one variable declaration")
	assert.Equal(t, "--color-primary", result.Variables[0].Name)

	// Should find var() calls
	assert.Len(t, result.VarCalls, 2, "Should find two var() calls")
}

// TestParseInvalidCSS tests that invalid CSS is handled gracefully
func TestParseInvalidCSS(t *testing.T) {
	cssCode := `this is not valid css {{{`

	parser := css.NewParser()
	result, err := parser.Parse(cssCode)

	// Parser should not crash, but may return an error or empty result
	// Tree-sitter is error-tolerant, so it might still parse partially
	if err != nil {
		t.Logf("Parser returned error (acceptable): %v", err)
	}
	if result != nil {
		t.Logf("Parser returned result with %d variables and %d var calls",
			len(result.Variables), len(result.VarCalls))
	}
}

// TestParseEmptyCSS tests parsing empty CSS
func TestParseEmptyCSS(t *testing.T) {
	cssCode := ``

	parser := css.NewParser()
	result, err := parser.Parse(cssCode)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Variables, "Should find no variables")
	assert.Empty(t, result.VarCalls, "Should find no var() calls")
}
