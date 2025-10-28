package css_test

import (
	"testing"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_css "github.com/tree-sitter/tree-sitter-css/bindings/go"
)

// TestTreeSitterBasic tests basic tree-sitter functionality
func TestTreeSitterBasic(t *testing.T) {
	cssCode := `:root {
  --color-primary: #0000ff;
}`

	// Check if language loads
	langPtr := tree_sitter_css.Language()
	t.Logf("Language pointer: %v", langPtr)

	lang := sitter.NewLanguage(langPtr)
	if lang == nil {
		t.Fatal("Language is nil")
	}
	t.Logf("Language created successfully")

	parser := sitter.NewParser()
	if parser == nil {
		t.Fatal("Parser is nil")
	}
	t.Logf("Parser created successfully")

	err := parser.SetLanguage(lang)
	if err != nil {
		t.Fatalf("Failed to set language: %v", err)
	}
	t.Logf("Language set successfully")

	tree := parser.Parse([]byte(cssCode), nil)
	if tree == nil {
		t.Fatal("Failed to parse CSS - tree is nil")
	}
	defer tree.Close()

	root := tree.RootNode()
	t.Logf("Root node kind: %s", root.Kind())
	t.Logf("Child count: %d", root.ChildCount())

	// Print tree structure
	printTree(t, root, []byte(cssCode), 0)
}

func printTree(t *testing.T, node *sitter.Node, source []byte, depth int) {
	if node == nil {
		return
	}

	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}

	nodeText := string(source[node.StartByte():node.EndByte()])
	if len(nodeText) > 50 {
		nodeText = nodeText[:50] + "..."
	}

	t.Logf("%s%s: %q", indent, node.Kind(), nodeText)

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		printTree(t, child, source, depth+1)
	}
}
