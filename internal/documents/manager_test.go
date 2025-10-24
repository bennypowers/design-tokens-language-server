package documents_test

import (
	"testing"

	"github.com/bennypowers/design-tokens-language-server/internal/documents"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestDocumentManagerOpenClose tests opening and closing documents
func TestDocumentManagerOpenClose(t *testing.T) {
	manager := documents.NewManager()

	uri := "file:///test.css"
	content := `:root {
  --color-primary: #0000ff;
}`

	// Initially, document should not exist
	doc := manager.Get(uri)
	assert.Nil(t, doc, "Document should not exist initially")

	// Open document
	err := manager.DidOpen(uri, "css", 1, content)
	require.NoError(t, err, "DidOpen should not error")

	// Document should now exist
	doc = manager.Get(uri)
	require.NotNil(t, doc, "Document should exist after open")
	assert.Equal(t, uri, doc.URI())
	assert.Equal(t, content, doc.Content())
	assert.Equal(t, "css", doc.LanguageID())
	assert.Equal(t, 1, doc.Version())

	// Close document
	err = manager.DidClose(uri)
	require.NoError(t, err, "DidClose should not error")

	// Document should be removed
	doc = manager.Get(uri)
	assert.Nil(t, doc, "Document should not exist after close")
}

// TestDocumentManagerFullUpdate tests full document content updates
func TestDocumentManagerFullUpdate(t *testing.T) {
	manager := documents.NewManager()

	uri := "file:///test.css"
	initialContent := `:root { --color: red; }`

	// Open document
	err := manager.DidOpen(uri, "css", 1, initialContent)
	require.NoError(t, err)

	doc := manager.Get(uri)
	assert.Equal(t, initialContent, doc.Content())
	assert.Equal(t, 1, doc.Version())

	// Update with full content
	newContent := `:root { --color: blue; }`
	changes := []protocol.TextDocumentContentChangeEvent{
		{
			Text: newContent,
			// No Range means full document update
		},
	}

	err = manager.DidChange(uri, 2, changes)
	require.NoError(t, err, "DidChange should not error")

	// Content should be updated
	doc = manager.Get(uri)
	assert.Equal(t, newContent, doc.Content())
	assert.Equal(t, 2, doc.Version())
}

// TestDocumentManagerIncrementalUpdate tests incremental document updates
func TestDocumentManagerIncrementalUpdate(t *testing.T) {
	manager := documents.NewManager()

	uri := "file:///test.css"
	initialContent := `:root {
  --color: red;
}`

	// Open document
	err := manager.DidOpen(uri, "css", 1, initialContent)
	require.NoError(t, err)

	// Incremental update: change "red" to "blue"
	// Line 1, character 11-14 is "red"
	startPos := protocol.Position{Line: 1, Character: 11}
	endPos := protocol.Position{Line: 1, Character: 14}
	changes := []protocol.TextDocumentContentChangeEvent{
		{
			Range: &protocol.Range{
				Start: startPos,
				End:   endPos,
			},
			Text: "blue",
		},
	}

	err = manager.DidChange(uri, 2, changes)
	require.NoError(t, err, "Incremental change should not error")

	// Content should be updated
	expectedContent := `:root {
  --color: blue;
}`
	doc := manager.Get(uri)
	assert.Equal(t, expectedContent, doc.Content())
	assert.Equal(t, 2, doc.Version())
}

// TestDocumentManagerMultipleIncrementalUpdates tests multiple incremental updates in sequence
func TestDocumentManagerMultipleIncrementalUpdates(t *testing.T) {
	manager := documents.NewManager()

	uri := "file:///test.css"
	initialContent := "hello world"

	err := manager.DidOpen(uri, "css", 1, initialContent)
	require.NoError(t, err)

	// Change 1: Replace "hello" with "goodbye"
	changes1 := []protocol.TextDocumentContentChangeEvent{
		{
			Range: &protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 5},
			},
			Text: "goodbye",
		},
	}
	err = manager.DidChange(uri, 2, changes1)
	require.NoError(t, err)

	doc := manager.Get(uri)
	assert.Equal(t, "goodbye world", doc.Content())

	// Change 2: Replace "world" with "universe"
	changes2 := []protocol.TextDocumentContentChangeEvent{
		{
			Range: &protocol.Range{
				Start: protocol.Position{Line: 0, Character: 8},
				End:   protocol.Position{Line: 0, Character: 13},
			},
			Text: "universe",
		},
	}
	err = manager.DidChange(uri, 3, changes2)
	require.NoError(t, err)

	doc = manager.Get(uri)
	assert.Equal(t, "goodbye universe", doc.Content())
}

// TestDocumentManagerBatchChanges tests applying multiple changes in a single DidChange
// Note: LSP batch changes are applied sequentially, with each change affecting
// the result of the previous change
func TestDocumentManagerBatchChanges(t *testing.T) {
	manager := documents.NewManager()

	uri := "file:///test.css"
	initialContent := "line 1\nline 2\nline 3"

	err := manager.DidOpen(uri, "css", 1, initialContent)
	require.NoError(t, err)

	// Apply multiple changes at once
	// Change 1: Replace "1" with " ONE" at position 5-6 of line 0
	// Change 2: Replace "2" with " TWO" at position 5-6 of line 1 (in modified document)
	changes := []protocol.TextDocumentContentChangeEvent{
		{
			Range: &protocol.Range{
				Start: protocol.Position{Line: 0, Character: 5},
				End:   protocol.Position{Line: 0, Character: 6},
			},
			Text: " ONE",
		},
		{
			Range: &protocol.Range{
				Start: protocol.Position{Line: 1, Character: 5},
				End:   protocol.Position{Line: 1, Character: 6},
			},
			Text: " TWO",
		},
	}

	err = manager.DidChange(uri, 2, changes)
	require.NoError(t, err)

	// After change 1: "line  ONE\nline 2\nline 3"
	// After change 2: "line  ONE\nline  TWO\nline 3"
	expectedContent := "line  ONE\nline  TWO\nline 3"
	doc := manager.Get(uri)
	assert.Equal(t, expectedContent, doc.Content())
}

// TestDocumentManagerInsertText tests inserting text (empty range)
func TestDocumentManagerInsertText(t *testing.T) {
	manager := documents.NewManager()

	uri := "file:///test.css"
	initialContent := "hello world"

	err := manager.DidOpen(uri, "css", 1, initialContent)
	require.NoError(t, err)

	// Insert " beautiful" at position 5 (after "hello")
	changes := []protocol.TextDocumentContentChangeEvent{
		{
			Range: &protocol.Range{
				Start: protocol.Position{Line: 0, Character: 5},
				End:   protocol.Position{Line: 0, Character: 5},
			},
			Text: " beautiful",
		},
	}

	err = manager.DidChange(uri, 2, changes)
	require.NoError(t, err)

	expectedContent := "hello beautiful world"
	doc := manager.Get(uri)
	assert.Equal(t, expectedContent, doc.Content())
}

// TestDocumentManagerDeleteText tests deleting text
func TestDocumentManagerDeleteText(t *testing.T) {
	manager := documents.NewManager()

	uri := "file:///test.css"
	initialContent := "hello beautiful world"

	err := manager.DidOpen(uri, "css", 1, initialContent)
	require.NoError(t, err)

	// Delete " beautiful" (characters 5-15)
	changes := []protocol.TextDocumentContentChangeEvent{
		{
			Range: &protocol.Range{
				Start: protocol.Position{Line: 0, Character: 5},
				End:   protocol.Position{Line: 0, Character: 15},
			},
			Text: "",
		},
	}

	err = manager.DidChange(uri, 2, changes)
	require.NoError(t, err)

	expectedContent := "hello world"
	doc := manager.Get(uri)
	assert.Equal(t, expectedContent, doc.Content())
}

// TestDocumentManagerMultiLineChanges tests changes across multiple lines
func TestDocumentManagerMultiLineChanges(t *testing.T) {
	manager := documents.NewManager()

	uri := "file:///test.css"
	initialContent := `line 1
line 2
line 3`

	err := manager.DidOpen(uri, "css", 1, initialContent)
	require.NoError(t, err)

	// Replace from middle of line 1 to middle of line 2
	changes := []protocol.TextDocumentContentChangeEvent{
		{
			Range: &protocol.Range{
				Start: protocol.Position{Line: 0, Character: 5},
				End:   protocol.Position{Line: 1, Character: 5},
			},
			Text: " REPLACED ",
		},
	}

	err = manager.DidChange(uri, 2, changes)
	require.NoError(t, err)

	expectedContent := `line  REPLACED 2
line 3`
	doc := manager.Get(uri)
	assert.Equal(t, expectedContent, doc.Content())
}

// TestDocumentManagerErrorHandling tests error cases
func TestDocumentManagerErrorHandling(t *testing.T) {
	manager := documents.NewManager()

	// Test changing non-existent document
	err := manager.DidChange("file:///nonexistent.css", 2, []protocol.TextDocumentContentChangeEvent{})
	assert.Error(t, err, "Changing non-existent document should error")

	// Test closing non-existent document
	err = manager.DidClose("file:///nonexistent.css")
	assert.Error(t, err, "Closing non-existent document should error")
}

// TestDocumentManagerGetAll tests retrieving all documents
func TestDocumentManagerGetAll(t *testing.T) {
	manager := documents.NewManager()

	// Initially empty
	docs := manager.GetAll()
	assert.Empty(t, docs, "Should have no documents initially")

	// Open multiple documents
	manager.DidOpen("file:///test1.css", "css", 1, "content1")
	manager.DidOpen("file:///test2.css", "css", 1, "content2")
	manager.DidOpen("file:///test3.json", "json", 1, "content3")

	docs = manager.GetAll()
	assert.Len(t, docs, 3, "Should have 3 documents")

	// Verify URIs
	uris := make(map[string]bool)
	for _, doc := range docs {
		uris[doc.URI()] = true
	}
	assert.True(t, uris["file:///test1.css"])
	assert.True(t, uris["file:///test2.css"])
	assert.True(t, uris["file:///test3.json"])
}
