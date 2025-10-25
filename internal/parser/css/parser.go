package css

import (
	"fmt"
	"strings"
	"sync"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_css "github.com/tree-sitter/tree-sitter-css/bindings/go"
)

// Parser handles parsing CSS with tree-sitter
type Parser struct {
	parser *sitter.Parser
}

// parserPool is a pool of reusable CSS parsers for performance
var parserPool = sync.Pool{
	New: func() interface{} {
		parser := sitter.NewParser()
		lang := sitter.NewLanguage(tree_sitter_css.Language())
		parser.SetLanguage(lang)
		return &Parser{parser: parser}
	},
}

// NewParser creates a new CSS parser
// Deprecated: Use AcquireParser/ReleaseParser for better performance
func NewParser() *Parser {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(tree_sitter_css.Language())
	parser.SetLanguage(lang)

	return &Parser{
		parser: parser,
	}
}

// AcquireParser gets a parser from the pool
func AcquireParser() *Parser {
	p := parserPool.Get().(*Parser)
	p.parser.Reset() // Reset state for reuse
	return p
}

// ReleaseParser returns a parser to the pool
func ReleaseParser(p *Parser) {
	if p != nil {
		parserPool.Put(p)
	}
}

// Close closes the parser and releases its resources
// This should be called when the parser is no longer needed
func (p *Parser) Close() {
	if p.parser != nil {
		p.parser.Close()
	}
}

// ClosePool closes all parsers in the pool
// This should be called on server shutdown
func ClosePool() {
	// Drain the pool by repeatedly getting and closing parsers
	// Note: This is a best-effort cleanup; sync.Pool doesn't provide
	// a way to iterate over all items
	for i := 0; i < 100; i++ {
		if p, ok := parserPool.Get().(*Parser); ok && p != nil {
			p.Close()
		}
	}
}

// Parse parses CSS code and extracts variable declarations and var() calls
// Positions are converted to UTF-16 code units for LSP compatibility
func (p *Parser) Parse(source string) (*ParseResult, error) {
	sourceBytes := []byte(source)
	tree := p.parser.Parse(sourceBytes, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse CSS")
	}
	defer tree.Close()

	root := tree.RootNode()
	result := &ParseResult{
		Variables: []*Variable{},
		VarCalls:  []*VarCall{},
	}

	// Walk the tree to find declarations and var() calls
	// Note: tree-sitter positions from Parse() are byte-based, we'll convert them
	p.walkTree(root, sourceBytes, source, result)

	return result, nil
}

// positionToUTF16 converts a tree-sitter Point (which uses byte offsets for Column)
// to LSP Position (which uses UTF-16 code units for Character)
func positionToUTF16(source string, point sitter.Point) Position {
	lines := strings.Split(source, "\n")
	if point.Row >= uint(len(lines)) {
		return Position{Line: uint32(point.Row), Character: uint32(point.Column)}
	}

	line := lines[point.Row]
	// point.Column is a byte offset within the line
	// Convert it to UTF-16 code units
	if point.Column > uint(len(line)) {
		point.Column = uint(len(line))
	}

	utf16Count := uint32(0)
	for _, r := range []rune(line[:point.Column]) {
		if r <= 0xFFFF {
			utf16Count++
		} else {
			utf16Count += 2 // Surrogate pair
		}
	}

	return Position{
		Line:      uint32(point.Row),
		Character: utf16Count,
	}
}

// walkTree recursively walks the tree to find CSS variables and var() calls
func (p *Parser) walkTree(node *sitter.Node, sourceBytes []byte, source string, result *ParseResult) {
	if node == nil {
		return
	}

	nodeKind := node.Kind()

	// Check for CSS custom property declaration
	if nodeKind == "declaration" {
		p.handleDeclaration(node, sourceBytes, source, result)
	}

	// Check for var() function call
	if nodeKind == "call_expression" {
		p.handleCallExpression(node, sourceBytes, source, result)
	}

	// Recursively walk children
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		p.walkTree(child, sourceBytes, source, result)
	}
}

// handleDeclaration processes a CSS declaration node
func (p *Parser) handleDeclaration(node *sitter.Node, sourceBytes []byte, source string, result *ParseResult) {
	// Find property name node
	var propertyNode *sitter.Node
	var valueNodes []*sitter.Node

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		kind := child.Kind()
		switch kind {
		case "property_name":
			propertyNode = child
		case "plain_value", "integer_value", "float_value", "color_value":
			valueNodes = append(valueNodes, child)
		}
	}

	if propertyNode == nil {
		return
	}

	propertyName := string(sourceBytes[propertyNode.StartByte():propertyNode.EndByte()])

	// Only process custom properties (starting with --)
	if !strings.HasPrefix(propertyName, "--") {
		return
	}

	// Extract value
	var value string
	if len(valueNodes) > 0 {
		// Concatenate all value nodes
		var parts []string
		for _, valueNode := range valueNodes {
			nodeText := string(sourceBytes[valueNode.StartByte():valueNode.EndByte()])
			parts = append(parts, strings.TrimSpace(nodeText))
		}
		value = strings.Join(parts, " ")
	}

	// Create variable with UTF-16 positions for LSP
	variable := &Variable{
		Name:  propertyName,
		Value: value,
		Type:  VariableDeclaration,
		Range: Range{
			Start: positionToUTF16(source, node.StartPosition()),
			End:   positionToUTF16(source, node.EndPosition()),
		},
	}

	result.Variables = append(result.Variables, variable)
}

// handleCallExpression processes a function call expression (looking for var())
func (p *Parser) handleCallExpression(node *sitter.Node, sourceBytes []byte, source string, result *ParseResult) {
	// Find function name
	var functionNameNode *sitter.Node
	var argumentsNode *sitter.Node

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		kind := child.Kind()
		switch kind {
		case "function_name":
			functionNameNode = child
		case "arguments":
			argumentsNode = child
		}
	}

	if functionNameNode == nil {
		return
	}

	functionName := string(sourceBytes[functionNameNode.StartByte():functionNameNode.EndByte()])

	// Only process var() calls
	if functionName != "var" {
		return
	}

	if argumentsNode == nil {
		return
	}

	// Extract token name and optional fallback
	var tokenName string
	var fallback *string

	argCount := 0
	for i := uint(0); i < argumentsNode.ChildCount(); i++ {
		child := argumentsNode.Child(i)
		kind := child.Kind()

		// Skip punctuation like '(' ')' ','
		if kind == "(" || kind == ")" || kind == "," {
			continue
		}

		// First argument is the token name
		if argCount == 0 {
			text := string(sourceBytes[child.StartByte():child.EndByte()])
			tokenName = strings.TrimSpace(text)
			argCount++
		} else if argCount == 1 {
			// Second argument is the fallback
			text := string(sourceBytes[child.StartByte():child.EndByte()])
			fb := strings.TrimSpace(text)
			fallback = &fb
			argCount++
		}
	}

	if tokenName == "" {
		return
	}

	// Create var call with UTF-16 positions for LSP
	varCall := &VarCall{
		TokenName: tokenName,
		Fallback:  fallback,
		Type:      VarReference,
		Range: Range{
			Start: positionToUTF16(source, node.StartPosition()),
			End:   positionToUTF16(source, node.EndPosition()),
		},
	}

	result.VarCalls = append(result.VarCalls, varCall)
}
