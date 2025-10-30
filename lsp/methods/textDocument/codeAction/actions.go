package codeaction

import (
	"fmt"
	"strings"

	cssparser "bennypowers.dev/dtls/internal/parser/css"
	"bennypowers.dev/dtls/internal/tokens"
	"bennypowers.dev/dtls/lsp/helpers/css"
	"bennypowers.dev/dtls/lsp/types"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// createReplacementAction creates a code action to replace a deprecated token with a recommended token.
// Returns nil if the replacement token cannot be formatted for CSS.
func createReplacementAction(req *types.RequestContext, uri string, varCall cssparser.VarCall, cssVarName string, replacementToken *tokens.Token, matchingDiag *protocol.Diagnostic) *protocol.CodeAction {
	// Build replacement text
	newText := fmt.Sprintf("var(%s)", cssVarName)
	if varCall.Fallback != nil {
		// Format the replacement token value for CSS
		formattedFallback, err := css.FormatTokenValueForCSS(replacementToken)
		if err != nil {
			req.AddWarning(fmt.Errorf("cannot format replacement token %q: %w", replacementToken.Name, err))
			return nil
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

	return &action
}

// createLiteralValueAction creates a code action to replace a var() call with a literal value.
// Returns nil if the token value cannot be formatted for CSS.
func createLiteralValueAction(uri string, varCall cssparser.VarCall, token *tokens.Token, matchingDiag *protocol.Diagnostic) *protocol.CodeAction {
	formattedValue, err := css.FormatTokenValueForCSS(token)
	if err != nil {
		return nil
	}

	kind := protocol.CodeActionKindQuickFix
	action := protocol.CodeAction{
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
		action.Diagnostics = []protocol.Diagnostic{*matchingDiag}
	}

	return &action
}

// createFixFallbackAction creates a code action to fix an incorrect fallback value.
// The edit is created immediately for simplicity (Go implementation),
// but can also be resolved lazily if Edit is nil (TypeScript compat).
func createFixFallbackAction(req *types.RequestContext, uri string, varCall cssparser.VarCall, token *tokens.Token, diagnostics []protocol.Diagnostic) *protocol.CodeAction {
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
	formattedValue, err := css.FormatTokenValueForCSS(token)
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
func createAddFallbackAction(req *types.RequestContext, uri string, varCall cssparser.VarCall, token *tokens.Token) *protocol.CodeAction {
	// Format the token value for safe CSS insertion
	formattedValue, err := css.FormatTokenValueForCSS(token)
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
func createDeprecatedTokenActions(req *types.RequestContext, uri string, varCall cssparser.VarCall, token *tokens.Token, diagnostics []protocol.Diagnostic) []protocol.CodeAction {
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
	recommendedToken := extractRecommendedToken(token.DeprecationMessage)

	// If we found a recommended token, try to create a replacement action
	if recommendedToken != "" {
		cssVarName := "--" + strings.ReplaceAll(recommendedToken, ".", "-")
		replacementToken := req.Server.Token(cssVarName)
		if replacementToken != nil {
			if action := createReplacementAction(req, uri, varCall, cssVarName, replacementToken, matchingDiag); action != nil {
				actions = append(actions, *action)
			}
		}
	}

	// Always try to add a literal value action as an alternative
	if action := createLiteralValueAction(uri, varCall, token, matchingDiag); action != nil {
		actions = append(actions, *action)
	}

	return actions
}

// createToggleFallbackAction creates a code action to toggle the fallback value for a single var() call.
// If the var() has a fallback, it removes it. If it doesn't, it adds one.
func createToggleFallbackAction(req *types.RequestContext, uri string, varCall cssparser.VarCall) *protocol.CodeAction {
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
		formattedValue, err := css.FormatTokenValueForCSS(token)
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

// createToggleRangeFallbacksAction creates a code action to toggle fallback values for multiple var() calls in a range.
func createToggleRangeFallbacksAction(req *types.RequestContext, uri string, varCalls []cssparser.VarCall) *protocol.CodeAction {
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
			formattedValue, err := css.FormatTokenValueForCSS(token)
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

// createFixAllFallbacksAction creates a source fixAll action to fix all incorrect fallback values.
// The actual edits are computed in the resolve step.
func createFixAllFallbacksAction(uri string, varCalls []*cssparser.VarCall) *protocol.CodeAction {
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

// createToggleActions creates toggle fallback actions based on the selection type.
// Returns actions for toggling fallbacks (single var or range of vars).
func createToggleActions(req *types.RequestContext, uri string, varCallsInRange []cssparser.VarCall, requestedRange protocol.Range) []protocol.CodeAction {
	var actions []protocol.CodeAction

	if len(varCallsInRange) == 0 {
		return actions
	}

	// Check if range is collapsed (cursor position) or expanded (selection)
	isCollapsed := requestedRange.Start == requestedRange.End

	if isCollapsed {
		// Collapsed cursor - create toggleFallback action for the var() at cursor
		if action := createToggleFallbackAction(req, uri, varCallsInRange[0]); action != nil {
			actions = append(actions, *action)
		}
	} else {
		// Expanded selection - create toggleRangeFallbacks action for all var() calls in range
		if action := createToggleRangeFallbacksAction(req, uri, varCallsInRange); action != nil {
			actions = append(actions, *action)
		}
	}

	return actions
}

// createFixAllActionIfNeeded creates a fix-all action if there are multiple incorrect-fallback diagnostics.
// Returns the action or nil if not needed.
func createFixAllActionIfNeeded(uri string, varCalls []*cssparser.VarCall, diagnostics []protocol.Diagnostic) *protocol.CodeAction {
	if len(diagnostics) < 2 {
		return nil
	}

	// Count incorrect-fallback diagnostics
	incorrectFallbackCount := 0
	for i := range diagnostics {
		diag := &diagnostics[i]
		if diag.Code != nil && diag.Code.Value == "incorrect-fallback" {
			incorrectFallbackCount++
		}
	}

	if incorrectFallbackCount < 2 {
		return nil
	}

	return createFixAllFallbacksAction(uri, varCalls)
}
