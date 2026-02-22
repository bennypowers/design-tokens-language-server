package js

import (
	"fmt"
	"sync"

	"bennypowers.dev/dtls/internal/log"
	"bennypowers.dev/dtls/internal/parser/css"
	htmlparser "bennypowers.dev/dtls/internal/parser/html"
	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
)

// Parser handles parsing JS/TS to extract CSS from tagged template literals
type Parser struct {
	parser        *sitter.Parser
	templateQuery *sitter.Query
	genericQuery  *sitter.Query // matches css<Type>`...` (generic form parsed by JS grammar as binary_expression)
}

var jsLang = sitter.NewLanguage(tree_sitter_javascript.Language())

// parserPool is a pool of reusable JS parsers
var parserPool = sync.Pool{
	New: func() any {
		parser := sitter.NewParser()
		if err := parser.SetLanguage(jsLang); err != nil {
			panic(fmt.Sprintf("failed to set JS language: %v", err))
		}

		templateQuery, qerr := sitter.NewQuery(jsLang, `
			(call_expression
				function: (identifier) @tag
				arguments: (template_string) @template)
		`)
		if qerr != nil {
			panic(fmt.Sprintf("failed to compile template query: %v", qerr))
		}

		// Generic form: css<Type>`...` is valid TypeScript (since TS 2.9) but both
		// tree-sitter-javascript and tree-sitter-typescript misparse it as binary
		// expressions instead of a call_expression with type_arguments.
		// See: https://github.com/tree-sitter/tree-sitter-typescript/issues/341
		genericQuery, qerr := sitter.NewQuery(jsLang, `
			(binary_expression
				left: (binary_expression
					left: (identifier) @tag)
				right: (template_string) @template)
		`)
		if qerr != nil {
			panic(fmt.Sprintf("failed to compile generic query: %v", qerr))
		}

		return &Parser{
			parser:        parser,
			templateQuery: templateQuery,
			genericQuery:  genericQuery,
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
	if p.genericQuery != nil {
		p.genericQuery.Close()
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

// ParseTemplates finds css/html tagged template literals and splits them at ${...} boundaries.
// Handles both standard form (css`...`) and generic form (css<Type>`...`).
func (p *Parser) ParseTemplates(source string) []TemplateRegion {
	sourceBytes := []byte(source)
	tree := p.parser.Parse(sourceBytes, nil)
	if tree == nil {
		return nil
	}
	defer tree.Close()

	root := tree.RootNode()
	var regions []TemplateRegion

	// Run both queries: standard tagged templates and generic form
	for _, query := range []*sitter.Query{p.templateQuery, p.genericQuery} {
		regions = p.runTemplateQuery(query, root, sourceBytes, regions)
	}

	return regions
}

// runTemplateQuery executes a single tree-sitter query against the parsed tree,
// extracting matching css/html tagged template regions and appending them to regions.
func (p *Parser) runTemplateQuery(query *sitter.Query, root *sitter.Node, sourceBytes []byte, regions []TemplateRegion) []TemplateRegion {
	cursor := sitter.NewQueryCursor()
	defer cursor.Close()

	matches := cursor.Matches(query, root, sourceBytes)
	for match := matches.Next(); match != nil; match = matches.Next() {
		var tagName string
		var templateNode sitter.Node
		foundTemplate := false

		for _, capture := range match.Captures {
			captureName := query.CaptureNames()[capture.Index]
			switch captureName {
			case "tag":
				tagName = string(sourceBytes[capture.Node.StartByte():capture.Node.EndByte()])
			case "template":
				templateNode = capture.Node
				foundTemplate = true
			}
		}

		if tagName != "css" && tagName != "html" {
			continue
		}

		if !foundTemplate {
			continue
		}

		segments := extractSegments(&templateNode, sourceBytes)
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
			parseCSSSegments(cssParser, tmpl.Segments, result)
		case "html":
			parseHTMLSegments(tmpl.Segments, result)
		}
	}

	return result, nil
}

// parseCSSSegments parses each segment of a css tagged template as CSS
func parseCSSSegments(cssParser *css.Parser, segments []Segment, result *css.ParseResult) {
	for _, seg := range segments {
		parsed, err := cssParser.Parse(seg.Content)
		if err != nil {
			log.Debug("Failed to parse CSS segment at %d:%d: %v", seg.StartLine, seg.StartCol, err)
			continue
		}
		offsetSegmentResults(parsed, seg)
		result.Variables = append(result.Variables, parsed.Variables...)
		result.VarCalls = append(result.VarCalls, parsed.VarCalls...)
	}
}

// parseHTMLSegments parses each segment of an html tagged template as HTML, then extracts CSS
func parseHTMLSegments(segments []Segment, result *css.ParseResult) {
	htmlParser := htmlparser.AcquireParser()
	defer htmlparser.ReleaseParser(htmlParser)

	for _, seg := range segments {
		parsed, err := htmlParser.ParseCSS(seg.Content)
		if err != nil {
			log.Debug("Failed to parse HTML segment at %d:%d: %v", seg.StartLine, seg.StartCol, err)
			continue
		}
		offsetSegmentResults(parsed, seg)
		result.Variables = append(result.Variables, parsed.Variables...)
		result.VarCalls = append(result.VarCalls, parsed.VarCalls...)
	}
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
