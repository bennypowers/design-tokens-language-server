package css

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_css "github.com/tree-sitter/tree-sitter-css/bindings/go"
)

// Parser handles parsing CSS with tree-sitter
type Parser struct {
	parser *sitter.Parser
}

// NewParser creates a new CSS parser
func NewParser() *Parser {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(tree_sitter_css.Language())
	parser.SetLanguage(lang)

	return &Parser{
		parser: parser,
	}
}

// Parse parses CSS code and extracts variable declarations and var() calls
func (p *Parser) Parse(source string) (*ParseResult, error) {
	tree := p.parser.Parse([]byte(source), nil)
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
	p.walkTree(root, []byte(source), result)

	return result, nil
}

// walkTree recursively walks the tree to find CSS variables and var() calls
func (p *Parser) walkTree(node *sitter.Node, source []byte, result *ParseResult) {
	if node == nil {
		return
	}

	nodeKind := node.Kind()

	// Check for CSS custom property declaration
	if nodeKind == "declaration" {
		p.handleDeclaration(node, source, result)
	}

	// Check for var() function call
	if nodeKind == "call_expression" {
		p.handleCallExpression(node, source, result)
	}

	// Recursively walk children
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		p.walkTree(child, source, result)
	}
}

// handleDeclaration processes a CSS declaration node
func (p *Parser) handleDeclaration(node *sitter.Node, source []byte, result *ParseResult) {
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

	propertyName := string(source[propertyNode.StartByte():propertyNode.EndByte()])

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
			nodeText := string(source[valueNode.StartByte():valueNode.EndByte()])
			parts = append(parts, strings.TrimSpace(nodeText))
		}
		value = strings.Join(parts, " ")
	}

	// Create variable
	variable := &Variable{
		Name:  propertyName,
		Value: value,
		Type:  VariableDeclaration,
		Range: Range{
			Start: Position{
				Line:      uint32(node.StartPosition().Row),
				Character: uint32(node.StartPosition().Column),
			},
			End: Position{
				Line:      uint32(node.EndPosition().Row),
				Character: uint32(node.EndPosition().Column),
			},
		},
	}

	result.Variables = append(result.Variables, variable)
}

// handleCallExpression processes a function call expression (looking for var())
func (p *Parser) handleCallExpression(node *sitter.Node, source []byte, result *ParseResult) {
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

	functionName := string(source[functionNameNode.StartByte():functionNameNode.EndByte()])

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
			text := string(source[child.StartByte():child.EndByte()])
			tokenName = strings.TrimSpace(text)
			argCount++
		} else if argCount == 1 {
			// Second argument is the fallback
			text := string(source[child.StartByte():child.EndByte()])
			fb := strings.TrimSpace(text)
			fallback = &fb
			argCount++
		}
	}

	if tokenName == "" {
		return
	}

	// Create var call
	varCall := &VarCall{
		TokenName: tokenName,
		Fallback:  fallback,
		Type:      VarReference,
		Range: Range{
			Start: Position{
				Line:      uint32(node.StartPosition().Row),
				Character: uint32(node.StartPosition().Column),
			},
			End: Position{
				Line:      uint32(node.EndPosition().Row),
				Character: uint32(node.EndPosition().Column),
			},
		},
	}

	result.VarCalls = append(result.VarCalls, varCall)
}
