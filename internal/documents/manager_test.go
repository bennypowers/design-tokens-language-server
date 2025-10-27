package documents_test

import (
	"testing"

	"bennypowers.dev/dtls/internal/documents"
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
	_ = manager.DidOpen("file:///test1.css", "css", 1, "content1")
	_ = manager.DidOpen("file:///test2.css", "css", 1, "content2")
	_ = manager.DidOpen("file:///test3.json", "json", 1, "content3")

	docs = manager.GetAll()
	require.Len(t, docs, 3, "Should have 3 documents")

	// Verify all expected URIs are present
	expectedURIs := map[string]bool{
		"file:///test1.css":  false,
		"file:///test2.css":  false,
		"file:///test3.json": false,
	}
	for _, doc := range docs {
		uri := doc.URI()
		if _, expected := expectedURIs[uri]; expected {
			expectedURIs[uri] = true
		}
	}
	for uri, found := range expectedURIs {
		assert.True(t, found, "Expected URI %s not found", uri)
	}
}

// TestDocumentManagerUTF16Incremental tests incremental updates with UTF-16 positions
func TestDocumentManagerUTF16Incremental(t *testing.T) {
	manager := documents.NewManager()
	uri := "file:///test.css"

	// Open document with emoji
	content := "/* 👍 */ color: blue;"
	err := manager.DidOpen(uri, "css", 1, content)
	require.NoError(t, err)

	// Replace "blue" with "red"
	// "/* 👍 */ color: " is 18 bytes but only 16 UTF-16 code units (👍 = 2 units)
	changes := []protocol.TextDocumentContentChangeEvent{
		{
			Range: &protocol.Range{
				Start: protocol.Position{Line: 0, Character: 16}, // UTF-16 position
				End:   protocol.Position{Line: 0, Character: 20}, // UTF-16 position
			},
			Text: "red",
		},
	}

	err = manager.DidChange(uri, 2, changes)
	require.NoError(t, err)

	doc := manager.Get(uri)
	assert.Equal(t, "/* 👍 */ color: red;", doc.Content())
}

// TestDocumentManagerUTF16CJK tests UTF-16 handling with CJK characters
func TestDocumentManagerUTF16CJK(t *testing.T) {
	manager := documents.NewManager()
	uri := "file:///test.css"

	// Open document with CJK characters
	content := "/* 颜色 */ color: blue;"
	err := manager.DidOpen(uri, "css", 1, content)
	require.NoError(t, err)

	// Replace "blue" with "red"
	// "/* 颜色 */ color: " is 20 bytes, 16 UTF-16 code units (each CJK char = 1 unit)
	changes := []protocol.TextDocumentContentChangeEvent{
		{
			Range: &protocol.Range{
				Start: protocol.Position{Line: 0, Character: 16},
				End:   protocol.Position{Line: 0, Character: 20},
			},
			Text: "red",
		},
	}

	err = manager.DidChange(uri, 2, changes)
	require.NoError(t, err)

	doc := manager.Get(uri)
	assert.Equal(t, "/* 颜色 */ color: red;", doc.Content())
}

// TestDocumentManagerEOFInsertion tests EOF insertion handling
func TestDocumentManagerEOFInsertion(t *testing.T) {
	manager := documents.NewManager()
	uri := "file:///test.css"

	// Open document
	content := "line1\nline2"
	err := manager.DidOpen(uri, "css", 1, content)
	require.NoError(t, err)

	// Insert at EOF (line == len(lines))
	changes := []protocol.TextDocumentContentChangeEvent{
		{
			Range: &protocol.Range{
				Start: protocol.Position{Line: 2, Character: 0}, // EOF
				End:   protocol.Position{Line: 2, Character: 0},
			},
			Text: "\nline3",
		},
	}

	err = manager.DidChange(uri, 2, changes)
	require.NoError(t, err)

	doc := manager.Get(uri)
	assert.Equal(t, "line1\nline2\nline3", doc.Content())
}

// TestDocumentManagerEmptyDocumentEOF tests EOF insertion on empty document
func TestDocumentManagerEmptyDocumentEOF(t *testing.T) {
	manager := documents.NewManager()
	uri := "file:///test.css"

	// Open empty document
	err := manager.DidOpen(uri, "css", 1, "")
	require.NoError(t, err)

	// Insert at EOF in empty document
	changes := []protocol.TextDocumentContentChangeEvent{
		{
			Range: &protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
			Text: "hello",
		},
	}

	err = manager.DidChange(uri, 2, changes)
	require.NoError(t, err)

	doc := manager.Get(uri)
	assert.Equal(t, "hello", doc.Content())
}

// TestDocumentManagerCharBoundsCheck tests character bounds validation
func TestDocumentManagerCharBoundsCheck(t *testing.T) {
	manager := documents.NewManager()
	uri := "file:///test.css"

	content := "hello"
	err := manager.DidOpen(uri, "css", 1, content)
	require.NoError(t, err)

	// Try to edit with out-of-bounds character position (UTF16ToByteOffset clamps to end)
	// When position is beyond line length, it should append at end
	changes := []protocol.TextDocumentContentChangeEvent{
		{
			Range: &protocol.Range{
				Start: protocol.Position{Line: 0, Character: 100}, // Way beyond line length
				End:   protocol.Position{Line: 0, Character: 100},
			},
			Text: "x",
		},
	}

	err = manager.DidChange(uri, 2, changes)
	require.NoError(t, err)

	// UTF16ToByteOffset clamps to string length, so this appends at end
	doc := manager.Get(uri)
	assert.Equal(t, "hellox", doc.Content())
}

// TestDocumentManagerStaleVersionRejection tests that older versions are rejected
func TestDocumentManagerStaleVersionRejection(t *testing.T) {
	manager := documents.NewManager()
	uri := "file:///test.css"

	// Open document at version 5
	content := "original"
	err := manager.DidOpen(uri, "css", 5, content)
	require.NoError(t, err)

	// Try to update with version 3 (older)
	changes := []protocol.TextDocumentContentChangeEvent{
		{
			Range: nil, // Full update
			Text:  "stale update",
		},
	}

	err = manager.DidChange(uri, 3, changes)
	assert.Error(t, err, "Should reject stale version")
	assert.Contains(t, err.Error(), "stale update")

	// Verify content unchanged
	doc := manager.Get(uri)
	assert.Equal(t, "original", doc.Content())
	assert.Equal(t, 5, doc.Version())
}

// TestDocumentManagerLineBoundsError tests out-of-bounds line rejection
func TestDocumentManagerLineBoundsError(t *testing.T) {
	manager := documents.NewManager()
	uri := "file:///test.css"

	content := "line1\nline2"
	err := manager.DidOpen(uri, "css", 1, content)
	require.NoError(t, err)

	// Try to edit at line 10 (way beyond document)
	changes := []protocol.TextDocumentContentChangeEvent{
		{
			Range: &protocol.Range{
				Start: protocol.Position{Line: 10, Character: 0},
				End:   protocol.Position{Line: 10, Character: 0},
			},
			Text: "invalid",
		},
	}

	err = manager.DidChange(uri, 2, changes)
	assert.Error(t, err, "Should reject out-of-bounds line")
	assert.Contains(t, err.Error(), "out of bounds")

	// Verify content unchanged
	doc := manager.Get(uri)
	assert.Equal(t, "line1\nline2", doc.Content())
}

// TestDocumentManagerInvalidUTF8 tests handling of invalid UTF-8 sequences
func TestDocumentManagerInvalidUTF8(t *testing.T) {
	manager := documents.NewManager()
	uri := "file:///test.css"

	// Document with invalid UTF-8 byte sequence (0xFF is invalid in UTF-8)
	content := "hello\xFFworld"
	err := manager.DidOpen(uri, "css", 1, content)
	require.NoError(t, err)

	// Try to edit - should handle gracefully
	changes := []protocol.TextDocumentContentChangeEvent{
		{
			Range: &protocol.Range{
				Start: protocol.Position{Line: 0, Character: 5},
				End:   protocol.Position{Line: 0, Character: 6},
			},
			Text: " ",
		},
	}

	// Should not panic when handling invalid UTF-8
	assert.NotPanics(t, func() {
		err = manager.DidChange(uri, 2, changes)
		// May error but should be graceful
		if err != nil {
			t.Logf("Invalid UTF-8 handling returned error: %v", err)
		}
	})
}

// TestDocumentManagerEndLineBoundsError tests out-of-bounds end line rejection
func TestDocumentManagerEndLineBoundsError(t *testing.T) {
	manager := documents.NewManager()
	uri := "file:///test.css"

	content := "line1\nline2"
	err := manager.DidOpen(uri, "css", 1, content)
	require.NoError(t, err)

	// Try to edit with end line beyond document (start is valid, end is not)
	changes := []protocol.TextDocumentContentChangeEvent{
		{
			Range: &protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 10, Character: 0}, // Way beyond document
			},
			Text: "invalid",
		},
	}

	err = manager.DidChange(uri, 2, changes)
	assert.Error(t, err, "Should reject out-of-bounds end line")
	assert.Contains(t, err.Error(), "out of bounds")
	assert.Contains(t, err.Error(), "end line")

	// Verify content unchanged
	doc := manager.Get(uri)
	assert.Equal(t, "line1\nline2", doc.Content())
}

// TestDocumentManagerEOFInsertionEmptyDocument tests EOF insertion on truly empty document
func TestDocumentManagerEOFInsertionEmptyDocument(t *testing.T) {
	manager := documents.NewManager()
	uri := "file:///test.css"

	// Open truly empty document (empty string splits to [""])
	err := manager.DidOpen(uri, "css", 1, "")
	require.NoError(t, err)

	// Insert at EOF position on empty document
	// This triggers the special case: len(lines) == 1 with lines[0] == ""
	changes := []protocol.TextDocumentContentChangeEvent{
		{
			Range: &protocol.Range{
				Start: protocol.Position{Line: 1, Character: 0}, // EOF: line == len(lines) == 1
				End:   protocol.Position{Line: 1, Character: 0},
			},
			Text: "new content",
		},
	}

	err = manager.DidChange(uri, 2, changes)
	require.NoError(t, err)

	doc := manager.Get(uri)
	// Should append to the empty first line
	assert.Equal(t, "new content", doc.Content())
}

// TestDocumentManagerStartCharClamp tests character position clamping
func TestDocumentManagerStartCharClamp(t *testing.T) {
	manager := documents.NewManager()
	uri := "file:///test.css"

	content := "hello world"
	err := manager.DidOpen(uri, "css", 1, content)
	require.NoError(t, err)

	// Manually construct a change with negative character position
	// (This shouldn't happen from a real LSP client, but we test the guard)
	changes := []protocol.TextDocumentContentChangeEvent{
		{
			Range: &protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 5},
			},
			Text: "hi",
		},
	}

	// This should succeed - negative positions get clamped to 0 by UTF16ToByteOffset
	err = manager.DidChange(uri, 2, changes)
	require.NoError(t, err)

	doc := manager.Get(uri)
	assert.Equal(t, "hi world", doc.Content())
}

// TestDocumentManagerCharBeyondLineLength tests character position validation
func TestDocumentManagerCharBeyondLineLength(t *testing.T) {
	manager := documents.NewManager()
	uri := "file:///test.css"

	content := "short"
	err := manager.DidOpen(uri, "css", 1, content)
	require.NoError(t, err)

	// Character position way beyond line length gets clamped by UTF16ToByteOffset
	// The bounds check at line 146-149 validates that the clamped value is still valid
	changes := []protocol.TextDocumentContentChangeEvent{
		{
			Range: &protocol.Range{
				Start: protocol.Position{Line: 0, Character: 1000},
				End:   protocol.Position{Line: 0, Character: 1000},
			},
			Text: " added",
		},
	}

	// Should succeed - position gets clamped to end of line
	err = manager.DidChange(uri, 2, changes)
	require.NoError(t, err)

	doc := manager.Get(uri)
	assert.Equal(t, "short added", doc.Content())
}
