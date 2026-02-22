package js_test

import (
	"encoding/json"
	"flag"
	"os"
	"testing"

	"bennypowers.dev/dtls/internal/parser/css"
	"bennypowers.dev/dtls/internal/parser/js"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update golden files")

func TestParseTemplates(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		wantCSS  int
		wantHTML int
	}{
		{
			name:     "css tagged template",
			fixture:  "testdata/css-template.js",
			wantCSS:  1,
			wantHTML: 0,
		},
		{
			name:     "html tagged template",
			fixture:  "testdata/html-template.js",
			wantCSS:  0,
			wantHTML: 1,
		},
		{
			name:     "template with expressions",
			fixture:  "testdata/template-with-expressions.js",
			wantCSS:  1,
			wantHTML: 0,
		},
		{
			name:     "no tagged templates",
			fixture:  "testdata/no-templates.js",
			wantCSS:  0,
			wantHTML: 0,
		},
		{
			name:     "lit-element class",
			fixture:  "testdata/lit-element-class.js",
			wantCSS:  1,
			wantHTML: 1,
		},
		{
			name:     "jsx component",
			fixture:  "testdata/jsx-component.jsx",
			wantCSS:  1,
			wantHTML: 0,
		},
		{
			name:     "tsx component",
			fixture:  "testdata/tsx-component.tsx",
			wantCSS:  1,
			wantHTML: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := os.ReadFile(tt.fixture)
			require.NoError(t, err)

			parser := js.AcquireParser()
			defer js.ReleaseParser(parser)

			templates := parser.ParseTemplates(string(source))

			cssCount := 0
			htmlCount := 0
			for _, tmpl := range templates {
				switch tmpl.Tag {
				case "css":
					cssCount++
				case "html":
					htmlCount++
				}
			}

			assert.Equal(t, tt.wantCSS, cssCount, "css template count")
			assert.Equal(t, tt.wantHTML, htmlCount, "html template count")
		})
	}
}

func TestParseTemplatesExpressionSplitting(t *testing.T) {
	source, err := os.ReadFile("testdata/template-with-expressions.js")
	require.NoError(t, err)

	parser := js.AcquireParser()
	defer js.ReleaseParser(parser)

	templates := parser.ParseTemplates(string(source))
	require.Len(t, templates, 1)

	tmpl := templates[0]
	assert.Equal(t, "css", tmpl.Tag)
	// Should have 2 segments (before and after ${someOtherStyles})
	assert.Equal(t, 2, len(tmpl.Segments), "should split at expression boundary")
}

func TestParseCSS(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		golden  string
	}{
		{
			name:    "css template",
			fixture: "testdata/css-template.js",
			golden:  "testdata/golden/css-template.json",
		},
		{
			name:    "html template",
			fixture: "testdata/html-template.js",
			golden:  "testdata/golden/html-template.json",
		},
		{
			name:    "template with expressions",
			fixture: "testdata/template-with-expressions.js",
			golden:  "testdata/golden/template-with-expressions.json",
		},
		{
			name:    "lit-element class",
			fixture: "testdata/lit-element-class.js",
			golden:  "testdata/golden/lit-element-class.json",
		},
		{
			name:    "jsx component",
			fixture: "testdata/jsx-component.jsx",
			golden:  "testdata/golden/jsx-component.json",
		},
		{
			name:    "tsx component",
			fixture: "testdata/tsx-component.tsx",
			golden:  "testdata/golden/tsx-component.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := os.ReadFile(tt.fixture)
			require.NoError(t, err)

			parser := js.AcquireParser()
			defer js.ReleaseParser(parser)

			result, err := parser.ParseCSS(string(source))
			require.NoError(t, err)
			require.NotNil(t, result)

			if *update {
				data, marshalErr := json.MarshalIndent(result, "", "  ")
				require.NoError(t, marshalErr)
				writeErr := os.WriteFile(tt.golden, append(data, '\n'), 0o644)
				require.NoError(t, writeErr)
				return
			}

			golden, err := os.ReadFile(tt.golden)
			require.NoError(t, err)

			var expected css.ParseResult
			err = json.Unmarshal(golden, &expected)
			require.NoError(t, err)

			require.Equal(t, len(expected.Variables), len(result.Variables), "variable count")
			require.Equal(t, len(expected.VarCalls), len(result.VarCalls), "var call count")

			for i, v := range result.Variables {
				assert.Equal(t, *expected.Variables[i], *v, "variable %d", i)
			}

			for i, vc := range result.VarCalls {
				assert.Equal(t, *expected.VarCalls[i], *vc, "var call %d", i)
			}
		})
	}
}

func TestParseCSSNoTemplates(t *testing.T) {
	source, err := os.ReadFile("testdata/no-templates.js")
	require.NoError(t, err)

	parser := js.AcquireParser()
	defer js.ReleaseParser(parser)

	result, err := parser.ParseCSS(string(source))
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Variables)
	assert.Empty(t, result.VarCalls)
}

func TestParseTemplatesGenericCSSTag(t *testing.T) {
	source, err := os.ReadFile("testdata/generic-css-tag.ts")
	require.NoError(t, err)

	parser := js.AcquireParser()
	defer js.ReleaseParser(parser)

	templates := parser.ParseTemplates(string(source))

	cssCount := 0
	for _, tmpl := range templates {
		if tmpl.Tag == "css" {
			cssCount++
		}
	}
	assert.Equal(t, 1, cssCount, "should find css tagged template with generic type parameter")

	// Also verify we can extract var calls
	result, err := parser.ParseCSS(string(source))
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.GreaterOrEqual(t, len(result.VarCalls), 2, "should find var calls from generic css template")

	tokenNames := make(map[string]bool)
	for _, vc := range result.VarCalls {
		tokenNames[vc.TokenName] = true
	}
	assert.True(t, tokenNames["--host-color"], "should find --host-color")
	assert.True(t, tokenNames["--content-padding"], "should find --content-padding")
}

func TestClosePool(t *testing.T) {
	// Exercise ClosePool — should not panic
	p := js.AcquireParser()
	js.ReleaseParser(p)
	js.ClosePool()
	// Pool is drained; acquiring again should still work
	p2 := js.AcquireParser()
	defer js.ReleaseParser(p2)
	templates := p2.ParseTemplates("const s = css`a{}`")
	assert.Len(t, templates, 1)
}

func TestParseTemplatesIgnoresNonCSSHTMLTags(t *testing.T) {
	// Tagged templates with names other than css/html should be ignored
	source := "const s = foo`some template`;\nconst t = bar`another one`;"

	parser := js.AcquireParser()
	defer js.ReleaseParser(parser)

	templates := parser.ParseTemplates(source)
	assert.Empty(t, templates, "should ignore non-css/html tagged templates")
}

func TestParseCSSWithHTMLTemplateStyleAttribute(t *testing.T) {
	// html template with a style attribute should extract var calls
	source := "const t = html`<div style=\"color: var(--x)\"></div>`;"

	parser := js.AcquireParser()
	defer js.ReleaseParser(parser)

	result, err := parser.ParseCSS(source)
	require.NoError(t, err)
	require.Len(t, result.VarCalls, 1)
	assert.Equal(t, "--x", result.VarCalls[0].TokenName)
}

func TestMultilineCSSTemplatePositionMapping(t *testing.T) {
	// Verify that CSS on lines after the first line of a template
	// gets only line offset (not column offset)
	source, err := os.ReadFile("testdata/css-template.js")
	require.NoError(t, err)

	parser := js.AcquireParser()
	defer js.ReleaseParser(parser)

	result, err := parser.ParseCSS(string(source))
	require.NoError(t, err)

	// The template has var calls on multiple lines
	require.GreaterOrEqual(t, len(result.VarCalls), 2)

	// Second var call is var(--bg-color, #fff) on line 8 (0-indexed)
	vc := result.VarCalls[1]
	assert.Equal(t, "--bg-color", vc.TokenName)
	assert.Equal(t, uint32(8), vc.Range.Start.Line, "second var call line")
	assert.Greater(t, vc.Range.Start.Character, uint32(0), "should have nonzero column")
}

func TestHTMLTemplateWithStyleAttribute(t *testing.T) {
	// Verify that html template with style attributes also works
	source, err := os.ReadFile("testdata/html-template.js")
	require.NoError(t, err)

	parser := js.AcquireParser()
	defer js.ReleaseParser(parser)

	result, err := parser.ParseCSS(string(source))
	require.NoError(t, err)

	// Should find var calls from both <style> tag and style attribute
	require.GreaterOrEqual(t, len(result.VarCalls), 2)

	// Check that we got both --text-color and --card-bg from the style tag
	tokenNames := make(map[string]bool)
	for _, vc := range result.VarCalls {
		tokenNames[vc.TokenName] = true
	}
	assert.True(t, tokenNames["--text-color"], "should find --text-color")
	assert.True(t, tokenNames["--card-bg"], "should find --card-bg")
}

func TestCSSTemplatePositionMapping(t *testing.T) {
	// Use the css-template.js fixture to verify position mapping
	source, err := os.ReadFile("testdata/css-template.js")
	require.NoError(t, err)

	parser := js.AcquireParser()
	defer js.ReleaseParser(parser)

	result, err := parser.ParseCSS(string(source))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result.VarCalls), 1)

	// First var call is var(--color-primary) on line 7 (0-indexed)
	// "    color: var(--color-primary);" — var starts at col 11
	vc := result.VarCalls[0]
	assert.Equal(t, "--color-primary", vc.TokenName)
	assert.Equal(t, uint32(7), vc.Range.Start.Line, "var call line")
	assert.Equal(t, uint32(11), vc.Range.Start.Character, "var call character")
}
