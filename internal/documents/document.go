package documents

import "fmt"

// Document represents a text document being managed by the language server
type Document struct {
	uri        string
	languageID string
	content    string
	version    int
}

// NewDocument creates a new document
func NewDocument(uri, languageID string, version int, content string) *Document {
	return &Document{
		uri:        uri,
		languageID: languageID,
		version:    version,
		content:    content,
	}
}

// URI returns the document's URI
func (d *Document) URI() string {
	return d.uri
}

// LanguageID returns the document's language identifier
func (d *Document) LanguageID() string {
	return d.languageID
}

// Version returns the document's version
func (d *Document) Version() int {
	return d.version
}

// Content returns the document's current content
func (d *Document) Content() string {
	return d.content
}

// SetContent updates the document's content and version.
// Returns an error if the provided version is older than the current document version,
// preventing stale updates from being applied.
func (d *Document) SetContent(content string, version int) error {
	if version < d.version {
		return fmt.Errorf("rejected stale update: document version is %d but update version is %d", d.version, version)
	}
	d.content = content
	d.version = version
	return nil
}
