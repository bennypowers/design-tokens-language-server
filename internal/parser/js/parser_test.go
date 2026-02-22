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
			name:     "typescript class",
			fixture:  "testdata/typescript-class.js",
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
			name:    "typescript class",
			fixture: "testdata/typescript-class.js",
			golden:  "testdata/golden/typescript-class.json",
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

			assert.Equal(t, len(expected.Variables), len(result.Variables), "variable count")
			assert.Equal(t, len(expected.VarCalls), len(result.VarCalls), "var call count")

			for i, v := range result.Variables {
				if i < len(expected.Variables) {
					assert.Equal(t, expected.Variables[i].Name, v.Name, "variable %d name", i)
					assert.Equal(t, expected.Variables[i].Range, v.Range, "variable %d range", i)
				}
			}

			for i, vc := range result.VarCalls {
				if i < len(expected.VarCalls) {
					assert.Equal(t, expected.VarCalls[i].TokenName, vc.TokenName, "var call %d token name", i)
					assert.Equal(t, expected.VarCalls[i].Range, vc.Range, "var call %d range", i)
				}
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
