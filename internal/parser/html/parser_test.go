package html_test

import (
	"encoding/json"
	"flag"
	"os"
	"testing"

	"bennypowers.dev/dtls/internal/parser/css"
	"bennypowers.dev/dtls/internal/parser/html"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update golden files")

func TestParseCSSRegions(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		wantTags int
		wantAttr int
	}{
		{
			name:     "style tag",
			fixture:  "testdata/style-tag.html",
			wantTags: 1,
			wantAttr: 0,
		},
		{
			name:     "style attributes",
			fixture:  "testdata/style-attribute.html",
			wantTags: 0,
			wantAttr: 2,
		},
		{
			name:     "multiple styles",
			fixture:  "testdata/multiple-styles.html",
			wantTags: 2,
			wantAttr: 2,
		},
		{
			name:     "no CSS",
			fixture:  "testdata/no-css.html",
			wantTags: 0,
			wantAttr: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := os.ReadFile(tt.fixture)
			require.NoError(t, err)

			parser := html.AcquireParser()
			defer html.ReleaseParser(parser)

			regions := parser.ParseCSSRegions(string(source))

			tags := 0
			attrs := 0
			for _, r := range regions {
				switch r.Type {
				case html.StyleTag:
					tags++
				case html.StyleAttribute:
					attrs++
				}
			}

			assert.Equal(t, tt.wantTags, tags, "style tag count")
			assert.Equal(t, tt.wantAttr, attrs, "style attribute count")
		})
	}
}

func TestParseCSS(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		golden  string
	}{
		{
			name:    "style tag",
			fixture: "testdata/style-tag.html",
			golden:  "testdata/golden/style-tag.json",
		},
		{
			name:    "style attribute",
			fixture: "testdata/style-attribute.html",
			golden:  "testdata/golden/style-attribute.json",
		},
		{
			name:    "multiple styles",
			fixture: "testdata/multiple-styles.html",
			golden:  "testdata/golden/multiple-styles.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := os.ReadFile(tt.fixture)
			require.NoError(t, err)

			parser := html.AcquireParser()
			defer html.ReleaseParser(parser)

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

func TestParseCSSNoCSS(t *testing.T) {
	source, err := os.ReadFile("testdata/no-css.html")
	require.NoError(t, err)

	parser := html.AcquireParser()
	defer html.ReleaseParser(parser)

	result, err := parser.ParseCSS(string(source))
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Variables)
	assert.Empty(t, result.VarCalls)
}

func TestParseCSSEmptyStyleTag(t *testing.T) {
	source := `<style></style>`

	parser := html.AcquireParser()
	defer html.ReleaseParser(parser)

	result, err := parser.ParseCSS(source)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Variables)
	assert.Empty(t, result.VarCalls)
}

func TestParseCSSEmptyStyleAttribute(t *testing.T) {
	source := `<div style=""></div>`

	parser := html.AcquireParser()
	defer html.ReleaseParser(parser)

	result, err := parser.ParseCSS(source)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Variables)
	assert.Empty(t, result.VarCalls)
}

func TestStyleTagPositionMapping(t *testing.T) {
	// Verify position mapping accuracy for style tags
	source := `<html>
<head>
<style>
.button {
  color: var(--color-primary);
}
</style>
</head>
</html>`

	parser := html.AcquireParser()
	defer html.ReleaseParser(parser)

	result, err := parser.ParseCSS(source)
	require.NoError(t, err)
	require.Len(t, result.VarCalls, 1)

	vc := result.VarCalls[0]
	assert.Equal(t, "--color-primary", vc.TokenName)
	// var(--color-primary) is on line 4 (0-indexed) in the HTML document
	// "  color: var(--color-primary);" — var starts at column 9
	assert.Equal(t, uint32(4), vc.Range.Start.Line, "var call should be on line 4")
	assert.Equal(t, uint32(9), vc.Range.Start.Character, "var call should start at char 9")
}

func TestStyleAttributePositionMapping(t *testing.T) {
	// Verify position mapping accuracy for style attributes
	source := `<div style="color: var(--text-color)">Hello</div>`

	parser := html.AcquireParser()
	defer html.ReleaseParser(parser)

	result, err := parser.ParseCSS(source)
	require.NoError(t, err)
	require.Len(t, result.VarCalls, 1)

	vc := result.VarCalls[0]
	assert.Equal(t, "--text-color", vc.TokenName)
	// style="color: var(--text-color)" — attribute value starts at col 12
	// "color: var(--text-color)" — var() starts at offset 7 within the attribute value
	// So in the HTML document, var() starts at col 12 + 7 = 19
	assert.Equal(t, uint32(0), vc.Range.Start.Line, "var call should be on line 0")
	assert.Equal(t, uint32(19), vc.Range.Start.Character, "var call should start at char 19")
}
