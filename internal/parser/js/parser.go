package js

import (
	"fmt"
	"sync"

	"bennypowers.dev/dtls/internal/parser/css"
	htmlparser "bennypowers.dev/dtls/internal/parser/html"
	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
)

// Parser handles parsing JS/TS to extract CSS from tagged template literals
type Parser struct {
	parser        *sitter.Parser
	templateQuery *sitter.Query
}

var jsLang = sitter.NewLanguage(tree_sitter_javascript.Language())

// parserPool is a pool of reusable JS parsers
var parserPool = sync.Pool{
	New: func() any {
		parser := sitter.NewParser()
		_ = parser.SetLanguage(jsLang)

		templateQuery, qerr := sitter.NewQuery(jsLang, `
			(call_expression
				function: (identifier) @tag
				arguments: (template_string) @template)
		`)
		if qerr != nil {
			panic(fmt.Sprintf("failed to compile template query: %v", qerr))
		}

		return &Parser{
			parser:        parser,
			templateQuery: templateQuery,
		}
	},
}

// AcquireParser gets a parser from the pool
func AcquireParser() *Parser {
	p := parserPool.Get().(*Parser)
	p.parser.Reset()
	return p
}

// ReleaseParser returns a parser to the pool
func ReleaseParser(p *Parser) {
	if p != nil {
		parserPool.Put(p)
	}
}

// Close closes the parser and releases its resources
func (p *Parser) Close() {
	if p.parser != nil {
		p.parser.Close()
	}
	if p.templateQuery != nil {
		p.templateQuery.Close()
	}
}

// ClosePool closes all parsers in the pool
func ClosePool() {
	for range 100 {
		if p, ok := parserPool.Get().(*Parser); ok && p != nil {
			p.Close()
		}
	}
}

// ParseTemplates finds css/html tagged template literals and splits them at ${...} boundaries
func (p *Parser) ParseTemplates(source string) []TemplateRegion {
	sourceBytes := []byte(source)
	tree := p.parser.Parse(sourceBytes, nil)
	if tree == nil {
		return nil
	}
	defer tree.Close()

	root := tree.RootNode()
	var regions []TemplateRegion

	cursor := sitter.NewQueryCursor()
	defer cursor.Close()

	matches := cursor.Matches(p.templateQuery, root, sourceBytes)
	for match := matches.Next(); match != nil; match = matches.Next() {
		var tagName string
		var templateNode *sitter.Node

		for _, capture := range match.Captures {
			captureName := p.templateQuery.CaptureNames()[capture.Index]
			switch captureName {
			case "tag":
				tagName = string(sourceBytes[capture.Node.StartByte():capture.Node.EndByte()])
			case "template":
				templateNode = &capture.Node
			}
		}

		if tagName != "css" && tagName != "html" {
			continue
		}

		if templateNode == nil {
			continue
		}

		segments := extractSegments(templateNode, sourceBytes)
		if len(segments) > 0 {
			regions = append(regions, TemplateRegion{
				Segments: segments,
				Tag:      tagName,
			})
		}
	}

	return regions
}

// extractSegments splits a template_string node into literal text segments
// (string_fragment nodes), skipping ${...} substitutions
func extractSegments(templateNode *sitter.Node, sourceBytes []byte) []Segment {
	var segments []Segment

	for i := uint(0); i < templateNode.ChildCount(); i++ {
		child := templateNode.Child(i)
		if child.Kind() == "string_fragment" {
			content := string(sourceBytes[child.StartByte():child.EndByte()])
			segments = append(segments, Segment{
				Content:  content,
				StartLine: child.StartPosition().Row,
				StartCol:  child.StartPosition().Column,
			})
		}
	}

	return segments
}

// ParseCSS extracts and parses CSS from tagged template literals in JS/TS source
func (p *Parser) ParseCSS(source string) (*css.ParseResult, error) {
	templates := p.ParseTemplates(source)
	if len(templates) == 0 {
		return &css.ParseResult{
			Variables: []*css.Variable{},
			VarCalls:  []*css.VarCall{},
		}, nil
	}

	result := &css.ParseResult{
		Variables: []*css.Variable{},
		VarCalls:  []*css.VarCall{},
	}

	cssParser := css.AcquireParser()
	defer css.ReleaseParser(cssParser)

	for _, tmpl := range templates {
		switch tmpl.Tag {
		case "css":
			if err := parseCSSSegments(cssParser, tmpl.Segments, result); err != nil {
				continue
			}
		case "html":
			if err := parseHTMLSegments(tmpl.Segments, result); err != nil {
				continue
			}
		}
	}

	return result, nil
}

// parseCSSSegments parses each segment of a css tagged template as CSS
func parseCSSSegments(cssParser *css.Parser, segments []Segment, result *css.ParseResult) error {
	for _, seg := range segments {
		parsed, err := cssParser.Parse(seg.Content)
		if err != nil {
			continue
		}
		offsetSegmentResults(parsed, seg)
		result.Variables = append(result.Variables, parsed.Variables...)
		result.VarCalls = append(result.VarCalls, parsed.VarCalls...)
	}
	return nil
}

// parseHTMLSegments parses each segment of an html tagged template as HTML, then extracts CSS
func parseHTMLSegments(segments []Segment, result *css.ParseResult) error {
	htmlParser := htmlparser.AcquireParser()
	defer htmlparser.ReleaseParser(htmlParser)

	for _, seg := range segments {
		parsed, err := htmlParser.ParseCSS(seg.Content)
		if err != nil {
			continue
		}
		offsetSegmentResults(parsed, seg)
		result.Variables = append(result.Variables, parsed.Variables...)
		result.VarCalls = append(result.VarCalls, parsed.VarCalls...)
	}
	return nil
}

// offsetSegmentResults adjusts CSS parse results positions to account for the segment's
// position in the original JS/TS source
func offsetSegmentResults(parsed *css.ParseResult, seg Segment) {
	for _, v := range parsed.Variables {
		v.Range = offsetSegmentRange(v.Range, seg)
	}
	for _, vc := range parsed.VarCalls {
		vc.Range = offsetSegmentRange(vc.Range, seg)
	}
}

// offsetSegmentRange adjusts a CSS range to account for the segment's position
func offsetSegmentRange(r css.Range, seg Segment) css.Range {
	r.Start = offsetSegmentPosition(r.Start, seg)
	r.End = offsetSegmentPosition(r.End, seg)
	return r
}

// offsetSegmentPosition adjusts a CSS position to the segment's position in the JS/TS source.
// For the first line, both line and column are offset.
// For subsequent lines, only the line is offset.
func offsetSegmentPosition(pos css.Position, seg Segment) css.Position {
	if pos.Line == 0 {
		pos.Character += uint32(seg.StartCol) //nolint:gosec // G115: segment positions from tree-sitter are bounded by file size
	}
	pos.Line += uint32(seg.StartLine) //nolint:gosec // G115: segment positions from tree-sitter are bounded by file size
	return pos
}
