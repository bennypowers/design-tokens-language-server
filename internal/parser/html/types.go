package html

// RegionType identifies the kind of CSS region found in HTML
type RegionType int

const (
	// StyleTag represents CSS inside a <style> element
	StyleTag RegionType = iota
	// StyleAttribute represents CSS inside a style="..." attribute
	StyleAttribute
)

// CSSRegion represents a region of CSS content found in an HTML document
type CSSRegion struct {
	Content   string
	StartLine uint
	StartCol  uint
	Type      RegionType
}
