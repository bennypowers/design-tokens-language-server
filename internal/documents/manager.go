package documents

import (
	"fmt"
	"strings"
	"sync"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Manager manages text documents for the language server
type Manager struct {
	documents map[string]*Document
	mu        sync.RWMutex
}

// NewManager creates a new document manager
func NewManager() *Manager {
	return &Manager{
		documents: make(map[string]*Document),
	}
}

// Get retrieves a document by URI
func (m *Manager) Get(uri string) *Document {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.documents[uri]
}

// GetAll returns all managed documents
func (m *Manager) GetAll() []*Document {
	m.mu.RLock()
	defer m.mu.RUnlock()

	docs := make([]*Document, 0, len(m.documents))
	for _, doc := range m.documents {
		docs = append(docs, doc)
	}
	return docs
}

// DidOpen handles the textDocument/didOpen notification
func (m *Manager) DidOpen(uri, languageID string, version int, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	doc := NewDocument(uri, languageID, version, content)
	m.documents[uri] = doc
	return nil
}

// DidClose handles the textDocument/didClose notification
func (m *Manager) DidClose(uri string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.documents[uri]; !exists {
		return fmt.Errorf("document not found: %s", uri)
	}

	delete(m.documents, uri)
	return nil
}

// DidChange handles the textDocument/didChange notification
func (m *Manager) DidChange(uri string, version int, changes []protocol.TextDocumentContentChangeEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	doc, exists := m.documents[uri]
	if !exists {
		return fmt.Errorf("document not found: %s", uri)
	}

	// Apply changes
	newContent, err := m.applyChanges(doc.Content(), changes)
	if err != nil {
		return fmt.Errorf("failed to apply changes: %w", err)
	}

	if err := doc.SetContent(newContent, version); err != nil {
		return fmt.Errorf("failed to set document content: %w", err)
	}
	return nil
}

// applyChanges applies a list of content changes to the document
func (m *Manager) applyChanges(content string, changes []protocol.TextDocumentContentChangeEvent) (string, error) {
	result := content

	for _, change := range changes {
		// If no range is provided, this is a full document update
		if change.Range == nil {
			result = change.Text
			continue
		}

		// Incremental update
		newContent, err := applyIncrementalChange(result, *change.Range, change.Text)
		if err != nil {
			return "", err
		}
		result = newContent
	}

	return result, nil
}

// applyIncrementalChange applies a single incremental change to the content
func applyIncrementalChange(content string, changeRange protocol.Range, text string) (string, error) {
	lines := strings.Split(content, "\n")

	// Validate range
	if int(changeRange.Start.Line) >= len(lines) {
		return "", fmt.Errorf("start line %d out of bounds (total lines: %d)", changeRange.Start.Line, len(lines))
	}
	if int(changeRange.End.Line) >= len(lines) {
		return "", fmt.Errorf("end line %d out of bounds (total lines: %d)", changeRange.End.Line, len(lines))
	}

	startLine := int(changeRange.Start.Line)
	startChar := int(changeRange.Start.Character)
	endLine := int(changeRange.End.Line)
	endChar := int(changeRange.End.Character)

	// Build the new content
	var result strings.Builder

	// Lines before the change
	for i := 0; i < startLine; i++ {
		result.WriteString(lines[i])
		result.WriteString("\n")
	}

	// Start line prefix (before change)
	if startChar < len(lines[startLine]) {
		result.WriteString(lines[startLine][:startChar])
	} else {
		result.WriteString(lines[startLine])
	}

	// Insert new text
	result.WriteString(text)

	// End line suffix (after change)
	if endLine < len(lines) && endChar < len(lines[endLine]) {
		result.WriteString(lines[endLine][endChar:])
	}

	// Lines after the change
	for i := endLine + 1; i < len(lines); i++ {
		result.WriteString("\n")
		result.WriteString(lines[i])
	}

	return result.String(), nil
}
