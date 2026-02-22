package html

// RegionType identifies the kind of CSS region found in HTML
type RegionType int

const (
	// UnknownRegion is the zero value, indicating an uninitialized region type
	UnknownRegion RegionType = iota
	// StyleTag represents CSS inside a <style> element
	StyleTag
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
