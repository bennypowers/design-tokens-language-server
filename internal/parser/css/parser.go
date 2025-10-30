package css

import (
	"fmt"
	"strings"
	"sync"

	"bennypowers.dev/dtls/lsp/helpers"
	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_css "github.com/tree-sitter/tree-sitter-css/bindings/go"
)

// Parser handles parsing CSS with tree-sitter
type Parser struct {
	parser *sitter.Parser
}

// parserPool is a pool of reusable CSS parsers for performance
var parserPool = sync.Pool{
	New: func() any {
		parser := sitter.NewParser()
		lang := sitter.NewLanguage(tree_sitter_css.Language())
		_ = parser.SetLanguage(lang) // Error ignored - parser initialization is critical and will panic if it fails
		return &Parser{parser: parser}
	},
}

// NewParser creates a new CSS parser
// Deprecated: Use AcquireParser/ReleaseParser for better performance
func NewParser() *Parser {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(tree_sitter_css.Language())
	_ = parser.SetLanguage(lang) // Error ignored - parser initialization is critical and will panic if it fails

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
	for range 100 {
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
	if err := p.walkTree(root, sourceBytes, source, result); err != nil {
		return nil, fmt.Errorf("failed to walk parse tree: %w", err)
	}

	return result, nil
}

// walkTree recursively walks the tree to find CSS variables and var() calls
func (p *Parser) walkTree(node *sitter.Node, sourceBytes []byte, source string, result *ParseResult) error {
	if node == nil {
		return nil
	}

	nodeKind := node.Kind()

	// Check for CSS custom property declaration
	if nodeKind == "declaration" {
		if err := p.handleDeclaration(node, sourceBytes, source, result); err != nil {
			return err
		}
	}

	// Check for var() function call
	if nodeKind == "call_expression" {
		if err := p.handleCallExpression(node, sourceBytes, source, result); err != nil {
			return err
		}
	}

	// Recursively walk children
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if err := p.walkTree(child, sourceBytes, source, result); err != nil {
			return err
		}
	}

	return nil
}

// handleDeclaration processes a CSS declaration node
func (p *Parser) handleDeclaration(node *sitter.Node, sourceBytes []byte, source string, result *ParseResult) error {
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
		return nil
	}

	propertyName := string(sourceBytes[propertyNode.StartByte():propertyNode.EndByte()])

	// Only process custom properties (starting with --)
	if !strings.HasPrefix(propertyName, "--") {
		return nil
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

	// Convert positions with overflow checking
	posRange, err := createPositionRange(source, propertyNode)
	if err != nil {
		return err
	}

	// Create variable with UTF-16 positions for LSP
	// Range covers only the property name (LHS), not the entire declaration
	// This ensures hover only triggers on the property name, not on the value
	variable := &Variable{
		Name:  propertyName,
		Value: value,
		Type:  VariableDeclaration,
		Range: posRange,
	}

	result.Variables = append(result.Variables, variable)
	return nil
}

// extractVarArguments extracts token name and optional fallback from var() arguments
func extractVarArguments(argumentsNode *sitter.Node, sourceBytes []byte) (tokenName string, fallback *string) {
	argCount := 0
	for i := uint(0); i < argumentsNode.ChildCount(); i++ {
		child := argumentsNode.Child(i)
		kind := child.Kind()

		// Skip punctuation like '(' ')' ','
		if kind == "(" || kind == ")" || kind == "," {
			continue
		}

		// First argument is the token name
		switch argCount {
		case 0:
			text := string(sourceBytes[child.StartByte():child.EndByte()])
			tokenName = strings.TrimSpace(text)
			argCount++
		case 1:
			// Second argument is the fallback
			text := string(sourceBytes[child.StartByte():child.EndByte()])
			fb := strings.TrimSpace(text)
			fallback = &fb
			argCount++
		}
	}
	return tokenName, fallback
}

// createPositionRange converts tree-sitter node positions to LSP Range with overflow checking
func createPositionRange(source string, node *sitter.Node) (Range, error) {
	startProto, err := helpers.PositionToUTF16(source, node.StartPosition())
	if err != nil {
		return Range{}, fmt.Errorf("failed to convert start position: %w", err)
	}
	endProto, err := helpers.PositionToUTF16(source, node.EndPosition())
	if err != nil {
		return Range{}, fmt.Errorf("failed to convert end position: %w", err)
	}

	return Range{
		Start: Position{Line: startProto.Line, Character: startProto.Character},
		End:   Position{Line: endProto.Line, Character: endProto.Character},
	}, nil
}

// handleCallExpression processes a function call expression (looking for var())
func (p *Parser) handleCallExpression(node *sitter.Node, sourceBytes []byte, source string, result *ParseResult) error {
	// Find function name and arguments nodes
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

	// Validate function name is "var"
	if functionNameNode == nil {
		return nil
	}
	functionName := string(sourceBytes[functionNameNode.StartByte():functionNameNode.EndByte()])
	if functionName != "var" || argumentsNode == nil {
		return nil
	}

	// Extract arguments
	tokenName, fallback := extractVarArguments(argumentsNode, sourceBytes)
	if tokenName == "" {
		return nil
	}

	// Convert positions with overflow checking
	posRange, err := createPositionRange(source, node)
	if err != nil {
		return err
	}

	// Create var call with UTF-16 positions for LSP
	varCall := &VarCall{
		TokenName: tokenName,
		Fallback:  fallback,
		Type:      VarReference,
		Range:     posRange,
	}

	result.VarCalls = append(result.VarCalls, varCall)
	return nil
}
