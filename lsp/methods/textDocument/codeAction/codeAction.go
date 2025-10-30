package codeaction

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"bennypowers.dev/dtls/internal/collections"
	"bennypowers.dev/dtls/internal/parser/css"
	"bennypowers.dev/dtls/internal/tokens"
	"bennypowers.dev/dtls/lsp/types"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Package-level Sets for static CSS type and value lookups
var (
	// safeCSSTypes are token types that can use raw values without quoting
	safeCSSTypes = collections.NewSet(
		"color", "dimension", "number", "duration", "cubicbezier",
	)

	// fontWeightKeywords are valid CSS font-weight keyword values
	fontWeightKeywords = collections.NewSet(
		"normal", "bold", "bolder", "lighter",
		"inherit", "initial", "unset",
	)

	// genericFontFamilies are CSS generic font family names that don't need quotes
	genericFontFamilies = collections.NewSet(
		"serif", "sans-serif", "monospace",
		"cursive", "fantasy", "system-ui",
	)

	// cssNamedColors are all valid CSS named color keywords
	cssNamedColors = collections.NewSet(
		"transparent", "black", "white", "red", "green",
		"blue", "yellow", "cyan", "magenta", "gray",
		"grey", "maroon", "purple", "fuchsia", "lime",
		"olive", "navy", "teal", "aqua", "orange",
		"aliceblue", "antiquewhite", "aquamarine", "azure",
		"beige", "bisque", "blanchedalmond", "blueviolet",
		"brown", "burlywood", "cadetblue", "chartreuse",
		"chocolate", "coral", "cornflowerblue", "cornsilk",
		"crimson", "darkblue", "darkcyan", "darkgoldenrod",
		"darkgray", "darkgrey", "darkgreen", "darkkhaki",
		"darkmagenta", "darkolivegreen", "darkorange", "darkorchid",
		"darkred", "darksalmon", "darkseagreen", "darkslateblue",
		"darkslategray", "darkslategrey", "darkturquoise", "darkviolet",
		"deeppink", "deepskyblue", "dimgray", "dimgrey",
		"dodgerblue", "firebrick", "floralwhite", "forestgreen",
		"gainsboro", "ghostwhite", "gold", "goldenrod",
		"greenyellow", "honeydew", "hotpink", "indianred",
		"indigo", "ivory", "khaki", "lavender",
		"lavenderblush", "lawngreen", "lemonchiffon", "lightblue",
		"lightcoral", "lightcyan", "lightgoldenrodyellow", "lightgray",
		"lightgrey", "lightgreen", "lightpink", "lightsalmon",
		"lightseagreen", "lightskyblue", "lightslategray", "lightslategrey",
		"lightsteelblue", "lightyellow", "limegreen", "linen",
		"mediumaquamarine", "mediumblue", "mediumorchid", "mediumpurple",
		"mediumseagreen", "mediumslateblue", "mediumspringgreen", "mediumturquoise",
		"mediumvioletred", "midnightblue", "mintcream", "mistyrose",
		"moccasin", "navajowhite", "oldlace", "olivedrab",
		"orangered", "orchid", "palegoldenrod", "palegreen",
		"paleturquoise", "palevioletred", "papayawhip", "peachpuff",
		"peru", "pink", "plum", "powderblue",
		"rosybrown", "royalblue", "saddlebrown", "salmon",
		"sandybrown", "seagreen", "seashell", "sienna",
		"silver", "skyblue", "slateblue", "slategray",
		"slategrey", "snow", "springgreen", "steelblue",
		"tan", "thistle", "tomato", "turquoise",
		"violet", "wheat", "whitesmoke", "yellowgreen",
	)
)

// formatTokenValueForCSS formats a token value for safe insertion into CSS.
// Returns the formatted value and an error if formatting/validation fails.
// Returns ("", error) for unsupported token types, invalid values or parse failures (caller should warn).
// Returns (value, nil) for successfully formatted values.
func FormatTokenValueForCSS(token *tokens.Token) (string, error) {
	value := token.Value
	tokenType := strings.ToLower(token.Type)

	// Font weight can be numeric (1-1000) or keyword (needs validation)
	if tokenType == "fontweight" {
		if fontWeightKeywords.Has(strings.ToLower(value)) {
			return value, nil
		}

		// Check if it's numeric (must be 1-1000 inclusive, reject 0)
		matched, _ := regexp.MatchString(`^[0-9]+$`, value)
		if matched {
			// Parse to integer to validate range
			var numValue int
			n, err := fmt.Sscanf(value, "%d", &numValue)
			if err != nil || n != 1 {
				return "", fmt.Errorf("failed to parse font-weight value %q: %w", value, err)
			}
			// Valid range is 1-1000 inclusive (CSS Fonts Level 4)
			// Explicitly reject 0 and out-of-range values
			if numValue >= 1 && numValue <= 1000 {
				return value, nil
			}
			// Invalid numeric value (0 or out of range)
			return "", fmt.Errorf("font-weight value %q out of range (must be 1-1000)", value)
		}
		return "", fmt.Errorf("invalid font-weight value %q (must be keyword or number)", value)
	}

	// Font family needs special handling (quoting for values with spaces/special chars)
	if tokenType == "fontfamily" {
		return FormatFontFamilyValue(value)
	}

	// Check if this is a safe type
	if safeCSSTypes.Has(tokenType) {
		return value, nil
	}

	// If no type specified, inspect the value to determine if it's safe
	if tokenType == "" {
		// Check if it looks like a safe CSS value (color, dimension, number)
		// Colors: hex, rgb(), hsl(), named colors
		if strings.HasPrefix(value, "#") || strings.HasPrefix(value, "rgb") ||
			strings.HasPrefix(value, "hsl") || isNamedColor(value) {
			return value, nil
		}

		// Dimensions: number followed by unit (px, rem, em, %, etc.)
		matched, _ := regexp.MatchString(`^-?\d+(\.\d+)?(px|rem|em|%|vh|vw|pt|cm|mm|in|pc|ex|ch|vmin|vmax)$`, value)
		if matched {
			return value, nil
		}

		// Pure numbers
		matched, _ = regexp.MatchString(`^-?\d+(\.\d+)?$`, value)
		if matched {
			return value, nil
		}

		// If it contains spaces or special characters, it might need quoting
		// but we can't be sure, so skip it for safety
		if strings.ContainsAny(value, " \t\n\"'()[]{}") {
			return "", fmt.Errorf("value %q contains special characters and cannot be safely formatted", value)
		}

		// Simple identifiers without spaces are probably safe
		matched, _ = regexp.MatchString(`^[a-zA-Z][a-zA-Z0-9-]*$`, value)
		if matched {
			return value, nil
		}

		return "", fmt.Errorf("value %q has unknown format", value)
	}

	// Composite types (stroke, border, transition, shadow, gradient, typography)
	// cannot be safely inserted as simple CSS values
	return "", fmt.Errorf("token type %q cannot be used as CSS fallback value", tokenType)
}

// formatFontFamilyValue formats a font family value for CSS.
// Returns the formatted value or an error if formatting fails.
func FormatFontFamilyValue(value string) (string, error) {
	value = strings.TrimSpace(value)

	// If it's already quoted, use as-is (assume it's properly formatted)
	if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
		(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
		return value, nil
	}

	// Generic font families don't need quotes
	if genericFontFamilies.Has(strings.ToLower(value)) {
		return value, nil
	}

	// If it contains a comma, it's likely a font-family list (e.g., "Arial, sans-serif")
	// These are typically already properly formatted, so return as-is
	if strings.Contains(value, ",") {
		// Check if it looks like a valid font-family list with quoted names
		// If it contains quotes, assume it's already properly formatted
		if strings.Contains(value, "\"") || strings.Contains(value, "'") {
			return value, nil
		}
		// Otherwise, it's a simple comma-separated list, also safe to use
		return value, nil
	}

	// If it contains spaces or special characters (but no comma), it needs quoting
	needsQuoting := strings.ContainsAny(value, " \t\n\"'")

	if needsQuoting {
		// Use %q to automatically quote and escape the string
		return fmt.Sprintf("%q", value), nil
	}

	// Single-word font names without special chars don't need quotes
	return value, nil
}

// isNamedColor checks if a value is a named CSS color
func isNamedColor(value string) bool {
	return cssNamedColors.Has(strings.ToLower(value))
}

// handleCodeAction handles the textDocument/codeAction request

// CodeAction handles the textDocument/codeAction request
func CodeAction(req *types.RequestContext, params *protocol.CodeActionParams) (any, error) {
	uri := params.TextDocument.URI

	fmt.Fprintf(os.Stderr, "[DTLS] CodeAction requested: %s\n", uri)

	// Get document
	doc := req.Server.Document(uri)
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
		return nil, fmt.Errorf("failed to parse CSS: %w", err)
	}

	var actions []protocol.CodeAction

	// Collect var calls in range for toggle actions
	var varCallsInRange []css.VarCall

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

		varCallsInRange = append(varCallsInRange, *varCall)

		// Look up the token
		token := req.Server.Token(varCall.TokenName)
		if token == nil {
			continue
		}

		// Create code actions for deprecated tokens
		if token.Deprecated {
			actions = append(actions, CreateDeprecatedTokenActions(req, uri, *varCall, token, params.Context.Diagnostics)...)
		}

		// Create code actions for incorrect fallback
		if varCall.Fallback != nil {
			fallbackValue := *varCall.Fallback
			tokenValue := token.Value

			if !isCSSValueSemanticallyEquivalent(fallbackValue, tokenValue) {
				if action := CreateFixFallbackAction(req, uri, *varCall, token, params.Context.Diagnostics); action != nil {
					actions = append(actions, *action)
				}
			}
		} else if token.Type == "color" || token.Type == "dimension" {
			// Suggest adding fallback for color and dimension tokens
			if action := CreateAddFallbackAction(req, uri, *varCall, token); action != nil {
				actions = append(actions, *action)
			}
		}
	}

	// Add toggle fallback actions
	// Check if range is collapsed (cursor position) or expanded (selection)
	isCollapsed := params.Range.Start == params.Range.End

	if isCollapsed && len(varCallsInRange) > 0 {
		// Collapsed cursor - create toggleFallback action for the var() at cursor
		if action := CreateToggleFallbackAction(req, uri, varCallsInRange[0]); action != nil {
			actions = append(actions, *action)
		}
	} else if !isCollapsed && len(varCallsInRange) > 0 {
		// Expanded selection - create toggleRangeFallbacks action for all var() calls in range
		if action := CreateToggleRangeFallbacksAction(req, uri, varCallsInRange); action != nil {
			actions = append(actions, *action)
		}
	}

	// Add fixAll action if there are multiple incorrect-fallback diagnostics
	if len(params.Context.Diagnostics) >= 2 {
		hasMultipleIncorrectFallbacks := 0
		for i := range params.Context.Diagnostics {
			diag := &params.Context.Diagnostics[i]
			if diag.Code != nil && diag.Code.Value == "incorrect-fallback" {
				hasMultipleIncorrectFallbacks++
			}
		}
		if hasMultipleIncorrectFallbacks >= 2 {
			if action := CreateFixAllFallbacksAction(uri, result.VarCalls); action != nil {
				actions = append(actions, *action)
			}
		}
	}

	fmt.Fprintf(os.Stderr, "[DTLS] Returning %d code actions\n", len(actions))

	return actions, nil
}

// handleCodeActionResolve handles the codeAction/resolve request

// CodeActionResolve handles the codeAction/resolve request
func CodeActionResolve(req *types.RequestContext, action *protocol.CodeAction) (*protocol.CodeAction, error) {
	fmt.Fprintf(os.Stderr, "[DTLS] CodeActionResolve requested: %s\n", action.Title)

	// Handle fixAllFallbacks which uses lazy resolution
	if action.Title == "Fix all token fallback values" {
		return resolveFixAllFallbacks(req, action)
	}

	// For other actions (fixFallback, toggle, add, deprecated),
	// we compute the edit immediately in CodeAction, so just return as-is
	return action, nil
}

// resolveFixAllFallbacks resolves the fixAll action by computing edits for all incorrect fallbacks
func resolveFixAllFallbacks(req *types.RequestContext, action *protocol.CodeAction) (*protocol.CodeAction, error) {
	// Get the URI from the data field
	data, ok := action.Data.(map[string]any)
	if !ok {
		return action, nil
	}

	uriVal, ok := data["uri"]
	if !ok {
		return action, nil
	}
	uri := uriVal.(string)

	// Get document
	doc := req.Server.Document(uri)
	if doc == nil {
		return action, nil
	}

	// Parse CSS to find all var() calls
	parser := css.AcquireParser()
	defer css.ReleaseParser(parser)
	result, err := parser.Parse(doc.Content())
	if err != nil {
		return action, nil
	}

	var edits []protocol.TextEdit

	// Fix all var() calls with incorrect fallbacks
	for _, varCall := range result.VarCalls {
		token := req.Server.Token(varCall.TokenName)
		if token == nil {
			continue
		}

		// Only fix if there's a fallback that's incorrect
		if varCall.Fallback != nil {
			fallbackValue := *varCall.Fallback
			tokenValue := token.Value

			if !isCSSValueSemanticallyEquivalent(fallbackValue, tokenValue) {
				// Format the token value
				formattedValue, err := FormatTokenValueForCSS(token)
				if err != nil {
					req.AddWarning(fmt.Errorf("cannot format token %q: %w", token.Name, err))
					continue
				}

				// Create edit to fix this fallback
				newText := fmt.Sprintf("var(%s, %s)", varCall.TokenName, formattedValue)
				edits = append(edits, protocol.TextEdit{
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
				})
			}
		}
	}

	// Add edits to the action
	action.Edit = &protocol.WorkspaceEdit{
		Changes: map[string][]protocol.TextEdit{
			uri: edits,
		},
	}

	return action, nil
}

// createFixFallbackAction creates a code action to fix an incorrect fallback value.
// The edit is created immediately for simplicity (Go implementation),
// but can also be resolved lazily if Edit is nil (TypeScript compat).
func CreateFixFallbackAction(req *types.RequestContext, uri string, varCall css.VarCall, token *tokens.Token, diagnostics []protocol.Diagnostic) *protocol.CodeAction {
	// Try to find the matching diagnostic
	var matchingDiag *protocol.Diagnostic
	for i := range diagnostics {
		if diagnostics[i].Range.Start.Line == varCall.Range.Start.Line &&
			diagnostics[i].Range.Start.Character == varCall.Range.Start.Character {
			matchingDiag = &diagnostics[i]
			break
		}
	}

	// Format the token value for CSS
	formattedValue, err := FormatTokenValueForCSS(token)
	if err != nil {
		req.AddWarning(fmt.Errorf("cannot format token %q for fallback: %w", token.Name, err))
		return nil
	}

	// Create the replacement text
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

	// Include diagnostic if we found one
	if matchingDiag != nil {
		action.Diagnostics = []protocol.Diagnostic{*matchingDiag}
		preferred := true
		action.IsPreferred = &preferred
	}

	return &action
}

// createAddFallbackAction creates a code action to add a fallback value.
// Returns nil if the token value cannot be safely formatted for CSS.
func CreateAddFallbackAction(req *types.RequestContext, uri string, varCall css.VarCall, token *tokens.Token) *protocol.CodeAction {
	// Format the token value for safe CSS insertion
	formattedValue, err := FormatTokenValueForCSS(token)
	if err != nil {
		req.AddWarning(fmt.Errorf("cannot format token %q for fallback: %w", token.Name, err))
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
func CreateDeprecatedTokenActions(req *types.RequestContext, uri string, varCall css.VarCall, token *tokens.Token, diagnostics []protocol.Diagnostic) []protocol.CodeAction {
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

		replacementToken := req.Server.Token(cssVarName)
		if replacementToken != nil {
			// Create replacement action
			newText := fmt.Sprintf("var(%s)", cssVarName)
			if varCall.Fallback != nil {
				// Format the replacement token value for CSS (handles quoting for font-family, etc.)
				formattedFallback, err := FormatTokenValueForCSS(replacementToken)
				if err != nil {
					req.AddWarning(fmt.Errorf("cannot format replacement token %q: %w", replacementToken.Name, err))
					goto createLiteralAction
				}
				newText = fmt.Sprintf("var(%s, %s)", cssVarName, formattedFallback)
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

createLiteralAction:
	// Add a generic "Remove deprecated token" action (shows the value inline)
	// Only offer this if the value can be safely formatted for CSS
	formattedValue, err := FormatTokenValueForCSS(token)
	if err == nil {
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

// CreateToggleFallbackAction creates a code action to toggle the fallback value for a single var() call.
// If the var() has a fallback, it removes it. If it doesn't, it adds one.
func CreateToggleFallbackAction(req *types.RequestContext, uri string, varCall css.VarCall) *protocol.CodeAction {
	token := req.Server.Token(varCall.TokenName)
	if token == nil {
		return nil
	}

	var newText string
	if varCall.Fallback != nil {
		// Has fallback - remove it
		newText = fmt.Sprintf("var(%s)", varCall.TokenName)
	} else {
		// No fallback - add it
		formattedValue, err := FormatTokenValueForCSS(token)
		if err != nil {
			req.AddWarning(fmt.Errorf("cannot format token %q for fallback: %w", token.Name, err))
			return nil
		}
		newText = fmt.Sprintf("var(%s, %s)", varCall.TokenName, formattedValue)
	}

	kind := protocol.CodeActionKindRefactorRewrite
	action := protocol.CodeAction{
		Title: "Toggle design token fallback value",
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

// CreateToggleRangeFallbacksAction creates a code action to toggle fallback values for multiple var() calls in a range.
func CreateToggleRangeFallbacksAction(req *types.RequestContext, uri string, varCalls []css.VarCall) *protocol.CodeAction {
	var edits []protocol.TextEdit

	for _, varCall := range varCalls {
		token := req.Server.Token(varCall.TokenName)
		if token == nil {
			continue
		}

		var newText string
		if varCall.Fallback != nil {
			// Has fallback - remove it
			newText = fmt.Sprintf("var(%s)", varCall.TokenName)
		} else {
			// No fallback - add it
			formattedValue, err := FormatTokenValueForCSS(token)
			if err != nil {
				req.AddWarning(fmt.Errorf("cannot format token %q for fallback: %w", token.Name, err))
				continue
			}
			newText = fmt.Sprintf("var(%s, %s)", varCall.TokenName, formattedValue)
		}

		edits = append(edits, protocol.TextEdit{
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
		})
	}

	if len(edits) == 0 {
		return nil
	}

	kind := protocol.CodeActionKindRefactorRewrite
	action := protocol.CodeAction{
		Title: "Toggle design token fallback values (in range)",
		Kind:  &kind,
		Edit: &protocol.WorkspaceEdit{
			Changes: map[string][]protocol.TextEdit{
				uri: edits,
			},
		},
	}
	return &action
}

// CreateFixAllFallbacksAction creates a source fixAll action to fix all incorrect fallback values.
// The actual edits are computed in the resolve step.
func CreateFixAllFallbacksAction(uri string, varCalls []*css.VarCall) *protocol.CodeAction {
	kind := protocol.CodeActionKind("source.fixAll")
	action := protocol.CodeAction{
		Title: "Fix all token fallback values",
		Kind:  &kind,
		// Data field is used to pass var calls to resolve step
		Data: map[string]any{
			"uri":      uri,
			"varCalls": varCalls,
		},
	}
	return &action
}
