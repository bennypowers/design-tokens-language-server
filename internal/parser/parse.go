package parser

import (
	"bennypowers.dev/dtls/internal/parser/css"
	"bennypowers.dev/dtls/internal/parser/html"
	"bennypowers.dev/dtls/internal/parser/js"
)

// cssLanguages maps language IDs to the parser category they use.
// "css" → direct CSS, "html" → HTML parser, "js" → JS parser.
var cssLanguages = map[string]string{
	"css":             "css",
	"html":            "html",
	"javascript":      "js",
	"javascriptreact": "js",
	"typescript":      "js",
	"typescriptreact": "js",
}

// IsCSSSupportedLanguage returns true if the language supports CSS extraction
func IsCSSSupportedLanguage(languageID string) bool {
	_, ok := cssLanguages[languageID]
	return ok
}

// ParseCSSFromDocument extracts CSS parse results from any supported document type.
// Dispatches to the appropriate parser based on language ID.
func ParseCSSFromDocument(content, languageID string) (*css.ParseResult, error) {
	switch cssLanguages[languageID] {
	case "css":
		p := css.AcquireParser()
		defer css.ReleaseParser(p)
		return p.Parse(content)

	case "html":
		p := html.AcquireParser()
		defer html.ReleaseParser(p)
		return p.ParseCSS(content)

	case "js":
		p := js.AcquireParser()
		defer js.ReleaseParser(p)
		return p.ParseCSS(content)

	default:
		return nil, nil
	}
}

// CSSContentSpans returns the CSS text fragments from a document.
// For CSS files, this is the entire content. For HTML/JS files, these are the
// extracted CSS regions (style tags, style attributes, css tagged templates).
// Used by completion to scope brace counting to CSS content only.
func CSSContentSpans(content, languageID string) []string {
	switch cssLanguages[languageID] {
	case "css":
		return []string{content}

	case "html":
		p := html.AcquireParser()
		defer html.ReleaseParser(p)
		regions := p.ParseCSSRegions(content)
		spans := make([]string, 0, len(regions))
		for _, r := range regions {
			spans = append(spans, cssRegionSpan(r))
		}
		return spans

	case "js":
		p := js.AcquireParser()
		defer js.ReleaseParser(p)
		templates := p.ParseTemplates(content)
		var spans []string
		for _, tmpl := range templates {
			switch tmpl.Tag {
			case "css":
				for _, seg := range tmpl.Segments {
					spans = append(spans, seg.Content)
				}
			case "html":
				spans = append(spans, htmlCSSSpans(tmpl.Segments)...)
			}
		}
		return spans

	default:
		return nil
	}
}

// cssRegionSpan converts a CSS region to its text span.
// Style tags return raw content; style attributes are wrapped in "x{...}"
// to form valid CSS for brace counting.
func cssRegionSpan(r html.CSSRegion) string {
	if r.Type == html.StyleTag {
		return r.Content
	}
	return "x{" + r.Content + "}"
}

// htmlCSSSpans extracts CSS text spans from html tagged template segments.
func htmlCSSSpans(segments []js.Segment) []string {
	hp := html.AcquireParser()
	defer html.ReleaseParser(hp)

	var spans []string
	for _, seg := range segments {
		regions := hp.ParseCSSRegions(seg.Content)
		for _, r := range regions {
			spans = append(spans, cssRegionSpan(r))
		}
	}
	return spans
}
