package parser_test

import (
	"testing"

	"bennypowers.dev/dtls/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsCSSSupportedLanguage(t *testing.T) {
	supported := []string{
		"css",
		"html",
		"javascript",
		"javascriptreact",
		"typescript",
		"typescriptreact",
	}

	for _, lang := range supported {
		t.Run(lang, func(t *testing.T) {
			assert.True(t, parser.IsCSSSupportedLanguage(lang))
		})
	}

	unsupported := []string{
		"json",
		"yaml",
		"go",
		"python",
		"",
	}

	for _, lang := range unsupported {
		t.Run("unsupported_"+lang, func(t *testing.T) {
			assert.False(t, parser.IsCSSSupportedLanguage(lang))
		})
	}
}

func TestParseCSSFromDocumentCSS(t *testing.T) {
	content := `.button { color: var(--color-primary); }`

	result, err := parser.ParseCSSFromDocument(content, "css")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Len(t, result.VarCalls, 1)
	assert.Equal(t, "--color-primary", result.VarCalls[0].TokenName)
}

func TestParseCSSFromDocumentHTML(t *testing.T) {
	content := `<style>.button { color: var(--text-color); }</style>`

	result, err := parser.ParseCSSFromDocument(content, "html")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Len(t, result.VarCalls, 1)
	assert.Equal(t, "--text-color", result.VarCalls[0].TokenName)
}

func TestParseCSSFromDocumentJavaScript(t *testing.T) {
	content := "const s = css`\n  .button { color: var(--text-color); }\n`;"

	for _, lang := range []string{"javascript", "javascriptreact", "typescript", "typescriptreact"} {
		t.Run(lang, func(t *testing.T) {
			result, err := parser.ParseCSSFromDocument(content, lang)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Len(t, result.VarCalls, 1)
			assert.Equal(t, "--text-color", result.VarCalls[0].TokenName)
		})
	}
}

func TestParseCSSFromDocumentJSX(t *testing.T) {
	content := "import { css } from 'lit';\nconst s = css`\n  .card { color: var(--card-color); }\n`;\nexport function Card() { return (<div/>); }"

	result, err := parser.ParseCSSFromDocument(content, "javascriptreact")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Len(t, result.VarCalls, 1)
	assert.Equal(t, "--card-color", result.VarCalls[0].TokenName)
}

func TestParseCSSFromDocumentTSX(t *testing.T) {
	content := "import { css } from 'lit';\ninterface Props { x: string }\nconst s = css`\n  :host { color: var(--host-color); }\n`;\nexport function App(p: Props) { return (<div/>); }"

	result, err := parser.ParseCSSFromDocument(content, "typescriptreact")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Len(t, result.VarCalls, 1)
	assert.Equal(t, "--host-color", result.VarCalls[0].TokenName)
}

func TestParseCSSFromDocumentUnsupported(t *testing.T) {
	result, err := parser.ParseCSSFromDocument("{}", "json")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestCSSContentSpansCSS(t *testing.T) {
	content := `.button { color: var(--x); }`
	spans := parser.CSSContentSpans(content, "css")
	require.Len(t, spans, 1)
	assert.Equal(t, content, spans[0])
}

func TestCSSContentSpansHTML(t *testing.T) {
	content := `<style>.a { color: red; }</style><div style="color: blue"></div>`
	spans := parser.CSSContentSpans(content, "html")
	require.Len(t, spans, 2)
	assert.Contains(t, spans[0], ".a { color: red; }")
	assert.Contains(t, spans[1], "x{color: blue}")
}

func TestCSSContentSpansJS(t *testing.T) {
	content := "const s = css`\n  .a { color: red; }\n`;"
	spans := parser.CSSContentSpans(content, "javascript")
	require.Len(t, spans, 1)
	assert.Contains(t, spans[0], ".a { color: red; }")
}

func TestCSSContentSpansJSHTMLTemplate(t *testing.T) {
	content := "const t = html`\n  <style>.b { color: blue; }</style>\n  <div style=\"margin: 0\"></div>\n`;"
	spans := parser.CSSContentSpans(content, "javascript")
	require.GreaterOrEqual(t, len(spans), 1)
	// Should find the style tag CSS content
	found := false
	for _, s := range spans {
		if s == ".b { color: blue; }" {
			found = true
		}
	}
	assert.True(t, found, "should have extracted CSS span '.b { color: blue; }' from html template")
}

func TestCSSContentSpansUnsupported(t *testing.T) {
	spans := parser.CSSContentSpans("{}", "json")
	assert.Nil(t, spans)
}

func TestCSSContentSpansEmpty(t *testing.T) {
	spans := parser.CSSContentSpans("<p>no css</p>", "html")
	assert.Empty(t, spans)
}
