package css

// VariableType represents the type of CSS variable construct
type VariableType int

const (
	// VariableDeclaration represents a CSS custom property declaration (--var-name: value)
	VariableDeclaration VariableType = iota
	// VarReference represents a var() function call
	VarReference
)

// Position represents a position in a text document
type Position struct {
	Line      uint32
	Character uint32
}

// Range represents a range in a text document
type Range struct {
	Start Position
	End   Position
}

// Variable represents a CSS custom property declaration
type Variable struct {
	Name  string
	Value string
	Type  VariableType
	Range Range
}

// VarCall represents a var() function call
type VarCall struct {
	TokenName string
	Fallback  *string // Optional fallback value
	Type      VariableType
	Range     Range
}

// ParseResult contains the results of parsing CSS
type ParseResult struct {
	Variables []*Variable
	VarCalls  []*VarCall
}
