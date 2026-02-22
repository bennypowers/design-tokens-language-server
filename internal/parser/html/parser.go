package html

import (
	"fmt"
	"sync"

	"bennypowers.dev/dtls/internal/parser/css"
	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_html "github.com/tree-sitter/tree-sitter-html/bindings/go"
)

// Parser handles parsing HTML to extract CSS regions
type Parser struct {
	parser     *sitter.Parser
	styleQuery *sitter.Query
	attrQuery  *sitter.Query
}

var htmlLang = sitter.NewLanguage(tree_sitter_html.Language())

// parserPool is a pool of reusable HTML parsers
var parserPool = sync.Pool{
	New: func() any {
		parser := sitter.NewParser()
		if err := parser.SetLanguage(htmlLang); err != nil {
			panic(fmt.Sprintf("failed to set HTML language: %v", err))
		}

		styleQuery, qerr := sitter.NewQuery(htmlLang, `(style_element (raw_text) @css)`)
		if qerr != nil {
			panic(fmt.Sprintf("failed to compile style query: %v", qerr))
		}

		attrQuery, qerr := sitter.NewQuery(htmlLang, `
			(attribute
				(attribute_name) @attr_name
				(quoted_attribute_value (attribute_value) @attr_value)
				(#eq? @attr_name "style"))
		`)
		if qerr != nil {
			panic(fmt.Sprintf("failed to compile attribute query: %v", qerr))
		}

		return &Parser{
			parser:     parser,
			styleQuery: styleQuery,
			attrQuery:  attrQuery,
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
	if p.styleQuery != nil {
		p.styleQuery.Close()
	}
	if p.attrQuery != nil {
		p.attrQuery.Close()
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

// ParseCSSRegions extracts CSS regions from HTML source
func (p *Parser) ParseCSSRegions(source string) []CSSRegion {
	sourceBytes := []byte(source)
	tree := p.parser.Parse(sourceBytes, nil)
	if tree == nil {
		return nil
	}
	defer tree.Close()

	root := tree.RootNode()
	var regions []CSSRegion

	// Find <style> tag contents
	cursor := sitter.NewQueryCursor()
	defer cursor.Close()

	matches := cursor.Matches(p.styleQuery, root, sourceBytes)
	for match := matches.Next(); match != nil; match = matches.Next() {
		for _, capture := range match.Captures {
			node := capture.Node
			content := string(sourceBytes[node.StartByte():node.EndByte()])
			regions = append(regions, CSSRegion{
				Content:   content,
				StartLine: node.StartPosition().Row,
				StartCol:  node.StartPosition().Column,
				Type:      StyleTag,
			})
		}
	}

	// Find style="..." attribute values
	cursor2 := sitter.NewQueryCursor()
	defer cursor2.Close()

	attrMatches := cursor2.Matches(p.attrQuery, root, sourceBytes)
	for match := attrMatches.Next(); match != nil; match = attrMatches.Next() {
		for _, capture := range match.Captures {
			captureName := p.attrQuery.CaptureNames()[capture.Index]
			if captureName != "attr_value" {
				continue
			}
			node := capture.Node
			content := string(sourceBytes[node.StartByte():node.EndByte()])
			regions = append(regions, CSSRegion{
				Content:   content,
				StartLine: node.StartPosition().Row,
				StartCol:  node.StartPosition().Column,
				Type:      StyleAttribute,
			})
		}
	}

	return regions
}

// ParseCSS extracts CSS from HTML and parses it, mapping positions back to HTML coordinates
func (p *Parser) ParseCSS(source string) (*css.ParseResult, error) {
	regions := p.ParseCSSRegions(source)
	if len(regions) == 0 {
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

	for _, region := range regions {
		switch region.Type {
		case StyleTag:
			parsed, err := cssParser.Parse(region.Content)
			if err != nil {
				continue
			}
			offsetStyleTagResults(parsed, region)
			result.Variables = append(result.Variables, parsed.Variables...)
			result.VarCalls = append(result.VarCalls, parsed.VarCalls...)

		case StyleAttribute:
			parsed, err := parseStyleAttribute(cssParser, region)
			if err != nil {
				continue
			}
			result.Variables = append(result.Variables, parsed.Variables...)
			result.VarCalls = append(result.VarCalls, parsed.VarCalls...)
		}
	}

	return result, nil
}

// offsetStyleTagResults adjusts CSS parse results to account for the style tag's position in the HTML
func offsetStyleTagResults(parsed *css.ParseResult, region CSSRegion) {
	for _, v := range parsed.Variables {
		v.Range = offsetRange(v.Range, region)
	}
	for _, vc := range parsed.VarCalls {
		vc.Range = offsetRange(vc.Range, region)
	}
}

// offsetRange adjusts a CSS range to account for the region's position in the HTML document
func offsetRange(r css.Range, region CSSRegion) css.Range {
	r.Start = offsetPosition(r.Start, region)
	r.End = offsetPosition(r.End, region)
	return r
}

// offsetPosition adjusts a CSS position to account for the region's position in the HTML document.
// For the first line of CSS content, both line and column are offset.
// For subsequent lines, only the line is offset (columns are absolute within the CSS content).
func offsetPosition(pos css.Position, region CSSRegion) css.Position {
	if pos.Line == 0 {
		pos.Character += uint32(region.StartCol) //nolint:gosec // G115: region positions from tree-sitter are bounded by file size
	}
	pos.Line += uint32(region.StartLine) //nolint:gosec // G115: region positions from tree-sitter are bounded by file size
	return pos
}

// parseStyleAttribute parses CSS from a style attribute value
// Wraps the content in "x{...}" to make it a valid CSS rule, then adjusts positions
func parseStyleAttribute(cssParser *css.Parser, region CSSRegion) (*css.ParseResult, error) {
	// Wrap in a dummy rule to make valid CSS
	wrapped := "x{" + region.Content + "}"
	parsed, err := cssParser.Parse(wrapped)
	if err != nil {
		return nil, err
	}

	// Adjust positions: subtract the "x{" prefix (2 chars), add attribute position
	for _, v := range parsed.Variables {
		v.Range = adjustAttributeRange(v.Range, region)
	}
	for _, vc := range parsed.VarCalls {
		vc.Range = adjustAttributeRange(vc.Range, region)
	}

	return parsed, nil
}

// adjustAttributeRange adjusts positions from the wrapped CSS back to the HTML document
func adjustAttributeRange(r css.Range, region CSSRegion) css.Range {
	r.Start = adjustAttributePosition(r.Start, region)
	r.End = adjustAttributePosition(r.End, region)
	return r
}

// adjustAttributePosition adjusts a position from the wrapped CSS back to the HTML document.
// The wrapper "x{" adds 2 characters on line 0, so we subtract those and add the attribute's position.
func adjustAttributePosition(pos css.Position, region CSSRegion) css.Position {
	if pos.Line == 0 {
		// Subtract the "x{" wrapper prefix, add the attribute value's column
		col := uint32(region.StartCol) //nolint:gosec // G115: region positions from tree-sitter are bounded by file size
		if pos.Character >= 2 {
			col += pos.Character - 2
		}
		pos.Character = col
	}
	pos.Line += uint32(region.StartLine) //nolint:gosec // G115: region positions from tree-sitter are bounded by file size
	return pos
}
