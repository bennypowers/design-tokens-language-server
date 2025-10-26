package codeaction

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/bennypowers/design-tokens-language-server/internal/parser/css"
	"github.com/bennypowers/design-tokens-language-server/internal/tokens"
	"github.com/bennypowers/design-tokens-language-server/lsp/types"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// formatTokenValueForCSS formats a token value for safe insertion into CSS.
// Returns the formatted value and a boolean indicating if the value is safe to use.
// Some token types cannot be safely converted to CSS fallback values and should be skipped.
func FormatTokenValueForCSS(token *tokens.Token) (string, bool) {
	value := token.Value
	tokenType := strings.ToLower(token.Type)

	// Safe types that can use raw values (no quoting needed)
	safeTypes := map[string]bool{
		"color":      true,
		"dimension":  true,
		"number":     true,
		"duration":   true,
		"cubicbezier": true,
	}

	// Font weight can be numeric (safe) or string (needs validation)
	if tokenType == "fontweight" {
		// Check if it's numeric (safe to use raw)
		matched, _ := regexp.MatchString(`^\d+$`, value)
		if matched {
			return value, true
		}
		// Predefined keywords like "bold", "normal" are safe
		keywords := map[string]bool{
			"normal": true, "bold": true, "bolder": true, "lighter": true,
			"100": true, "200": true, "300": true, "400": true, "500": true,
			"600": true, "700": true, "800": true, "900": true,
		}
		if keywords[strings.ToLower(value)] {
			return value, true
		}
		return "", false // Unsafe font weight value
	}

	// Font family needs special handling (quoting for values with spaces/special chars)
	if tokenType == "fontfamily" {
		return FormatFontFamilyValue(value)
	}

	// Check if this is a safe type
	if safeTypes[tokenType] {
		return value, true
	}

	// If no type specified, inspect the value to determine if it's safe
	if tokenType == "" {
		// Check if it looks like a safe CSS value (color, dimension, number)
		// Colors: hex, rgb(), hsl(), named colors
		if strings.HasPrefix(value, "#") || strings.HasPrefix(value, "rgb") ||
		   strings.HasPrefix(value, "hsl") || isNamedColor(value) {
			return value, true
		}

		// Dimensions: number followed by unit (px, rem, em, %, etc.)
		matched, _ := regexp.MatchString(`^-?\d+(\.\d+)?(px|rem|em|%|vh|vw|pt|cm|mm|in|pc|ex|ch|vmin|vmax)$`, value)
		if matched {
			return value, true
		}

		// Pure numbers
		matched, _ = regexp.MatchString(`^-?\d+(\.\d+)?$`, value)
		if matched {
			return value, true
		}

		// If it contains spaces or special characters, it might need quoting
		// but we can't be sure, so skip it for safety
		if strings.ContainsAny(value, " \t\n\"'()[]{}") {
			return "", false
		}

		// Simple identifiers without spaces are probably safe
		matched, _ = regexp.MatchString(`^[a-zA-Z][a-zA-Z0-9-]*$`, value)
		if matched {
			return value, true
		}

		return "", false // Unknown format, unsafe
	}

	// Composite types (stroke, border, transition, shadow, gradient, typography)
	// cannot be safely inserted as simple CSS values
	return "", false
}

// formatFontFamilyValue formats a font family value for CSS.
// Returns the formatted value and whether it's safe to use.
func FormatFontFamilyValue(value string) (string, bool) {
	value = strings.TrimSpace(value)

	// If it's already quoted, use as-is (assume it's properly formatted)
	if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
	   (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
		return value, true
	}

	// Generic font families don't need quotes
	genericFamilies := map[string]bool{
		"serif": true, "sans-serif": true, "monospace": true,
		"cursive": true, "fantasy": true, "system-ui": true,
	}
	if genericFamilies[strings.ToLower(value)] {
		return value, true
	}

	// If it contains spaces, commas, or special characters, it needs quoting
	needsQuoting := strings.ContainsAny(value, " \t\n,\"'")

	if needsQuoting {
		// Escape any internal quotes
		escaped := strings.ReplaceAll(value, "\"", "\\\"")
		return fmt.Sprintf("\"%s\"", escaped), true
	}

	// Single-word font names without special chars don't need quotes
	return value, true
}

// isNamedColor checks if a value is a named CSS color
func isNamedColor(value string) bool {
	namedColors := map[string]bool{
		"transparent": true, "black": true, "white": true, "red": true, "green": true,
		"blue": true, "yellow": true, "cyan": true, "magenta": true, "gray": true,
		"grey": true, "maroon": true, "purple": true, "fuchsia": true, "lime": true,
		"olive": true, "navy": true, "teal": true, "aqua": true, "orange": true,
		"aliceblue": true, "antiquewhite": true, "aquamarine": true, "azure": true,
		"beige": true, "bisque": true, "blanchedalmond": true, "blueviolet": true,
		"brown": true, "burlywood": true, "cadetblue": true, "chartreuse": true,
		"chocolate": true, "coral": true, "cornflowerblue": true, "cornsilk": true,
		"crimson": true, "darkblue": true, "darkcyan": true, "darkgoldenrod": true,
		"darkgray": true, "darkgrey": true, "darkgreen": true, "darkkhaki": true,
		"darkmagenta": true, "darkolivegreen": true, "darkorange": true, "darkorchid": true,
		"darkred": true, "darksalmon": true, "darkseagreen": true, "darkslateblue": true,
		"darkslategray": true, "darkslategrey": true, "darkturquoise": true, "darkviolet": true,
		"deeppink": true, "deepskyblue": true, "dimgray": true, "dimgrey": true,
		"dodgerblue": true, "firebrick": true, "floralwhite": true, "forestgreen": true,
		"gainsboro": true, "ghostwhite": true, "gold": true, "goldenrod": true,
		"greenyellow": true, "honeydew": true, "hotpink": true, "indianred": true,
		"indigo": true, "ivory": true, "khaki": true, "lavender": true,
		"lavenderblush": true, "lawngreen": true, "lemonchiffon": true, "lightblue": true,
		"lightcoral": true, "lightcyan": true, "lightgoldenrodyellow": true, "lightgray": true,
		"lightgrey": true, "lightgreen": true, "lightpink": true, "lightsalmon": true,
		"lightseagreen": true, "lightskyblue": true, "lightslategray": true, "lightslategrey": true,
		"lightsteelblue": true, "lightyellow": true, "limegreen": true, "linen": true,
		"mediumaquamarine": true, "mediumblue": true, "mediumorchid": true, "mediumpurple": true,
		"mediumseagreen": true, "mediumslateblue": true, "mediumspringgreen": true, "mediumturquoise": true,
		"mediumvioletred": true, "midnightblue": true, "mintcream": true, "mistyrose": true,
		"moccasin": true, "navajowhite": true, "oldlace": true, "olivedrab": true,
		"orangered": true, "orchid": true, "palegoldenrod": true, "palegreen": true,
		"paleturquoise": true, "palevioletred": true, "papayawhip": true, "peachpuff": true,
		"peru": true, "pink": true, "plum": true, "powderblue": true,
		"rosybrown": true, "royalblue": true, "saddlebrown": true, "salmon": true,
		"sandybrown": true, "seagreen": true, "seashell": true, "sienna": true,
		"silver": true, "skyblue": true, "slateblue": true, "slategray": true,
		"slategrey": true, "snow": true, "springgreen": true, "steelblue": true,
		"tan": true, "thistle": true, "tomato": true, "turquoise": true,
		"violet": true, "wheat": true, "whitesmoke": true, "yellowgreen": true,
	}
	return namedColors[strings.ToLower(value)]
}

// handleCodeAction handles the textDocument/codeAction request

// CodeAction handles the textDocument/codeAction request
func CodeAction(ctx types.ServerContext, context *glsp.Context, params *protocol.CodeActionParams) (any, error) {
	uri := params.TextDocument.URI

	fmt.Fprintf(os.Stderr, "[DTLS] CodeAction requested: %s\n", uri)

	// Get document
	doc := ctx.Document(uri)
	if doc == nil {
		return nil, nil
	}

	// Only process CSS files
	if doc.LanguageID() != "css" {
		return nil, nil
	}

	// Parse CSS to find var() calls
	parser := css.AcquireParser()
	defer css.ReleaseParser(parser)
	result, err := parser.Parse(doc.Content())
	if err != nil {
		fmt.Fprintf(os.Stderr, "[DTLS] Failed to parse CSS: %v\n", err)
		return nil, nil
	}

	var actions []protocol.CodeAction

	// Check each var() call in the requested range
	for _, varCall := range result.VarCalls {
		// Check if var call intersects with the requested range
		if !RangesIntersect(params.Range, protocol.Range{
			Start: protocol.Position{
				Line:      varCall.Range.Start.Line,
				Character: varCall.Range.Start.Character,
			},
			End: protocol.Position{
				Line:      varCall.Range.End.Line,
				Character: varCall.Range.End.Character,
			},
		}) {
			continue
		}

		// Look up the token
		token := ctx.Token(varCall.TokenName)
		if token == nil {
			continue
		}

		// Create code actions for deprecated tokens
		if token.Deprecated {
			actions = append(actions, CreateDeprecatedTokenActions(ctx, uri, *varCall, token, params.Context.Diagnostics)...)
		}

		// Create code actions for incorrect fallback
		if varCall.Fallback != nil {
			fallbackValue := *varCall.Fallback
			tokenValue := token.Value

			if !isCSSValueSemanticallyEquivalent(fallbackValue, tokenValue) {
				if action := CreateFixFallbackAction(uri, *varCall, token, params.Context.Diagnostics); action != nil {
					actions = append(actions, *action)
				}
			}
		} else if token.Type == "color" || token.Type == "dimension" {
			// Suggest adding fallback for color and dimension tokens
			if action := CreateAddFallbackAction(uri, *varCall, token); action != nil {
				actions = append(actions, *action)
			}
		}
	}

	fmt.Fprintf(os.Stderr, "[DTLS] Returning %d code actions\n", len(actions))

	return actions, nil
}

// handleCodeActionResolve handles the codeAction/resolve request

// CodeActionResolve handles the codeAction/resolve request
func CodeActionResolve(ctx types.ServerContext, context *glsp.Context, action *protocol.CodeAction) (*protocol.CodeAction, error) {
	fmt.Fprintf(os.Stderr, "[DTLS] CodeActionResolve requested: %s\n", action.Title)

	// For now, we compute the edit immediately in CodeAction
	// This is here for future optimization where we could defer computing edits
	return action, nil
}

// createFixFallbackAction creates a code action to fix an incorrect fallback value.
// Returns nil if the token value cannot be safely formatted for CSS.
func CreateFixFallbackAction(uri string, varCall css.VarCall, token *tokens.Token, diagnostics []protocol.Diagnostic) *protocol.CodeAction {
	// Format the token value for safe CSS insertion
	formattedValue, safe := FormatTokenValueForCSS(token)
	if !safe {
		// Skip this code action - value cannot be safely inserted
		return nil
	}

	// Find the matching diagnostic
	var matchingDiag *protocol.Diagnostic
	for i := range diagnostics {
		if diagnostics[i].Range.Start.Line == varCall.Range.Start.Line &&
			diagnostics[i].Range.Start.Character == varCall.Range.Start.Character {
			matchingDiag = &diagnostics[i]
			break
		}
	}

	// Create the replacement text with formatted value
	newText := fmt.Sprintf("var(%s, %s)", varCall.TokenName, formattedValue)

	kind := protocol.CodeActionKindQuickFix
	action := protocol.CodeAction{
		Title: fmt.Sprintf("Fix fallback value to '%s'", formattedValue),
		Kind:  &kind,
		Edit: &protocol.WorkspaceEdit{
			Changes: map[string][]protocol.TextEdit{
				uri: {
					{
						Range: protocol.Range{
							Start: protocol.Position{
								Line:      varCall.Range.Start.Line,
								Character: varCall.Range.Start.Character,
							},
							End: protocol.Position{
								Line:      varCall.Range.End.Line,
								Character: varCall.Range.End.Character,
							},
						},
						NewText: newText,
					},
				},
			},
		},
	}

	if matchingDiag != nil {
		action.Diagnostics = []protocol.Diagnostic{*matchingDiag}
		preferred := true
		action.IsPreferred = &preferred
	}

	return &action
}

// createAddFallbackAction creates a code action to add a fallback value.
// Returns nil if the token value cannot be safely formatted for CSS.
func CreateAddFallbackAction(uri string, varCall css.VarCall, token *tokens.Token) *protocol.CodeAction {
	// Format the token value for safe CSS insertion
	formattedValue, safe := FormatTokenValueForCSS(token)
	if !safe {
		// Skip this code action - value cannot be safely inserted
		return nil
	}

	// Create the replacement text with formatted value
	newText := fmt.Sprintf("var(%s, %s)", varCall.TokenName, formattedValue)

	kind := protocol.CodeActionKindQuickFix
	action := protocol.CodeAction{
		Title: fmt.Sprintf("Add fallback value '%s'", formattedValue),
		Kind:  &kind,
		Edit: &protocol.WorkspaceEdit{
			Changes: map[string][]protocol.TextEdit{
				uri: {
					{
						Range: protocol.Range{
							Start: protocol.Position{
								Line:      varCall.Range.Start.Line,
								Character: varCall.Range.Start.Character,
							},
							End: protocol.Position{
								Line:      varCall.Range.End.Line,
								Character: varCall.Range.End.Character,
							},
						},
						NewText: newText,
					},
				},
			},
		},
	}
	return &action
}

// createDeprecatedTokenActions creates code actions for deprecated tokens
func CreateDeprecatedTokenActions(ctx types.ServerContext, uri string, varCall css.VarCall, token *tokens.Token, diagnostics []protocol.Diagnostic) []protocol.CodeAction {
	var actions []protocol.CodeAction

	// Find the matching diagnostic
	var matchingDiag *protocol.Diagnostic
	for i := range diagnostics {
		if diagnostics[i].Range.Start.Line == varCall.Range.Start.Line &&
			diagnostics[i].Range.Start.Character == varCall.Range.Start.Character {
			matchingDiag = &diagnostics[i]
			break
		}
	}

	// Try to extract recommended replacement from deprecation message
	// Common patterns: "Use X instead", "Use X.Y instead", "Replaced by X"
	var recommendedToken string
	if token.DeprecationMessage != "" {
		msg := token.DeprecationMessage

		// Pattern: "Use X instead" or "Use X.Y instead"
		if idx := strings.Index(msg, "Use "); idx != -1 {
			rest := msg[idx+4:]
			if endIdx := strings.Index(rest, " instead"); endIdx != -1 {
				recommendedToken = strings.TrimSpace(rest[:endIdx])
			}
		}

		// Pattern: "Replaced by X"
		if recommendedToken == "" {
			if idx := strings.Index(msg, "Replaced by "); idx != -1 {
				rest := msg[idx+12:]
				// Take until space or end of string
				if spaceIdx := strings.Index(rest, " "); spaceIdx != -1 {
					recommendedToken = rest[:spaceIdx]
				} else {
					recommendedToken = rest
				}
			}
		}
	}

	// If we found a recommended token, try to look it up
	if recommendedToken != "" {
		// Convert dot notation to CSS variable name
		cssVarName := "--" + strings.ReplaceAll(recommendedToken, ".", "-")

		replacementToken := ctx.Token(cssVarName)
		if replacementToken != nil {
			// Create replacement action
			newText := fmt.Sprintf("var(%s)", cssVarName)
			if varCall.Fallback != nil {
				newText = fmt.Sprintf("var(%s, %s)", cssVarName, replacementToken.Value)
			}

			kind := protocol.CodeActionKindQuickFix
			action := protocol.CodeAction{
				Title: fmt.Sprintf("Replace with '%s'", cssVarName),
				Kind:  &kind,
				Edit: &protocol.WorkspaceEdit{
					Changes: map[string][]protocol.TextEdit{
						uri: {
							{
								Range: protocol.Range{
									Start: protocol.Position{
										Line:      varCall.Range.Start.Line,
										Character: varCall.Range.Start.Character,
									},
									End: protocol.Position{
										Line:      varCall.Range.End.Line,
										Character: varCall.Range.End.Character,
									},
								},
								NewText: newText,
							},
						},
					},
				},
			}

			if matchingDiag != nil {
				action.Diagnostics = []protocol.Diagnostic{*matchingDiag}
				preferred := true
				action.IsPreferred = &preferred
			}

			actions = append(actions, action)
		}
	}

	// Add a generic "Remove deprecated token" action (shows the value inline)
	// Only offer this if the value can be safely formatted for CSS
	formattedValue, safe := FormatTokenValueForCSS(token)
	if safe {
		kind := protocol.CodeActionKindQuickFix
		removeAction := protocol.CodeAction{
			Title: fmt.Sprintf("Replace with literal value '%s'", formattedValue),
			Kind:  &kind,
			Edit: &protocol.WorkspaceEdit{
				Changes: map[string][]protocol.TextEdit{
					uri: {
						{
							Range: protocol.Range{
								Start: protocol.Position{
									Line:      varCall.Range.Start.Line,
									Character: varCall.Range.Start.Character,
								},
								End: protocol.Position{
									Line:      varCall.Range.End.Line,
									Character: varCall.Range.End.Character,
								},
							},
							NewText: formattedValue,
						},
					},
				},
			},
		}

		if matchingDiag != nil {
			removeAction.Diagnostics = []protocol.Diagnostic{*matchingDiag}
		}

		actions = append(actions, removeAction)
	}

	return actions
}

// rangesIntersect checks if two ranges intersect
// Ranges are treated as half-open intervals [start, end) where the end position is exclusive
func RangesIntersect(a, b protocol.Range) bool {
	// Check if a ends before or at the start of b (no intersection)
	if a.End.Line < b.Start.Line {
		return false
	}
	if a.End.Line == b.Start.Line && a.End.Character <= b.Start.Character {
		return false
	}

	// Check if b ends before or at the start of a (no intersection)
	if b.End.Line < a.Start.Line {
		return false
	}
	if b.End.Line == a.Start.Line && b.End.Character <= a.Start.Character {
		return false
	}

	return true
}
func isCSSValueSemanticallyEquivalent(a, b string) bool {
	// Normalize: remove all whitespace and convert to lowercase
	normalize := func(s string) string {
		s = strings.ReplaceAll(s, " ", "")
		s = strings.ReplaceAll(s, "\t", "")
		s = strings.ReplaceAll(s, "\n", "")
		s = strings.ToLower(s)
		return s
	}

	return normalize(a) == normalize(b)
}
