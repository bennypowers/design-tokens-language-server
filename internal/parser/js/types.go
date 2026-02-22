package js

// Segment represents a literal text segment from a template string,
// between ${...} expression boundaries
type Segment struct {
	Content  string
	StartLine uint
	StartCol  uint
}

// TemplateRegion represents a tagged template literal found in JS/TS source
type TemplateRegion struct {
	Segments []Segment
	Tag      string // "css" or "html"
}
