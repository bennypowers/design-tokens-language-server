package js

// Segment represents a literal text segment from a template string,
// between ${...} expression boundaries
type Segment struct {
	// Content is the literal text of this segment
	Content string
	// StartLine is the 0-indexed line in the JS/TS source where this segment begins
	StartLine uint
	// StartCol is the 0-indexed column in the JS/TS source where this segment begins
	StartCol uint
}

// TemplateRegion represents a tagged template literal found in JS/TS source
type TemplateRegion struct {
	// Segments contains the literal text parts of the template, split at ${...} boundaries
	Segments []Segment
	// Tag is the template tag function name ("css" or "html")
	Tag string
}
