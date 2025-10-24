package documents

// Document represents a text document being managed by the language server
type Document struct {
	uri        string
	languageID string
	version    int
	content    string
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

// SetContent updates the document's content and version
func (d *Document) SetContent(content string, version int) {
	d.content = content
	d.version = version
}
