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
	// Content is the raw CSS text extracted from the region
	Content string
	// StartLine is the 0-indexed line in the HTML document where the CSS content begins
	StartLine uint
	// StartCol is the 0-indexed column in the HTML document where the CSS content begins
	StartCol uint
	// Type identifies whether this region comes from a <style> tag or a style attribute
	Type RegionType
}
