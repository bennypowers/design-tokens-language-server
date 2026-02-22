package parser

import (
	"bennypowers.dev/dtls/internal/parser/css"
	"bennypowers.dev/dtls/internal/parser/html"
	"bennypowers.dev/dtls/internal/parser/js"
)

// IsCSSSupportedLanguage returns true if the language supports CSS extraction
func IsCSSSupportedLanguage(languageID string) bool {
	switch languageID {
	case "css", "html",
		"javascript", "javascriptreact",
		"typescript", "typescriptreact":
		return true
	default:
		return false
	}
}

// ParseCSSFromDocument extracts CSS parse results from any supported document type.
// Dispatches to the appropriate parser based on language ID.
func ParseCSSFromDocument(content, languageID string) (*css.ParseResult, error) {
	switch languageID {
	case "css":
		p := css.AcquireParser()
		defer css.ReleaseParser(p)
		return p.Parse(content)

	case "html":
		p := html.AcquireParser()
		defer html.ReleaseParser(p)
		return p.ParseCSS(content)

	case "javascript", "javascriptreact",
		"typescript", "typescriptreact":
		p := js.AcquireParser()
		defer js.ReleaseParser(p)
		return p.ParseCSS(content)

	default:
		return nil, nil
	}
}
