package lsp

import (
	"fmt"
	"os"
	"strings"

	"github.com/bennypowers/design-tokens-language-server/internal/parser/css"
	"github.com/bennypowers/design-tokens-language-server/internal/tokens"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// handleCodeAction handles the textDocument/codeAction request
func (s *Server) handleCodeAction(context *glsp.Context, params *protocol.CodeActionParams) (any, error) {
	return s.CodeAction(params)
}

// CodeAction provides code actions (exposed for testing)
func (s *Server) CodeAction(params *protocol.CodeActionParams) ([]protocol.CodeAction, error) {
	uri := params.TextDocument.URI

	fmt.Fprintf(os.Stderr, "[DTLS] CodeAction requested: %s\n", uri)

	// Get document
	doc := s.documents.Get(uri)
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
		if !rangesIntersect(params.Range, protocol.Range{
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
		token := s.tokens.Get(varCall.TokenName)
		if token == nil {
			continue
		}

		// Create code actions for deprecated tokens
		if token.Deprecated {
			actions = append(actions, s.createDeprecatedTokenActions(uri, *varCall, token, params.Context.Diagnostics)...)
		}

		// Create code actions for incorrect fallback
		if varCall.Fallback != nil {
			fallbackValue := *varCall.Fallback
			tokenValue := token.Value

			if !isCSSValueSemanticallyEquivalent(fallbackValue, tokenValue) {
				actions = append(actions, s.createFixFallbackAction(uri, *varCall, token, params.Context.Diagnostics))
			}
		} else if token.Type == "color" || token.Type == "dimension" {
			// Suggest adding fallback for color and dimension tokens
			actions = append(actions, s.createAddFallbackAction(uri, *varCall, token))
		}
	}

	fmt.Fprintf(os.Stderr, "[DTLS] Returning %d code actions\n", len(actions))

	return actions, nil
}

// handleCodeActionResolve handles the codeAction/resolve request
func (s *Server) handleCodeActionResolve(context *glsp.Context, params *protocol.CodeAction) (*protocol.CodeAction, error) {
	return s.CodeActionResolve(params)
}

// CodeActionResolve resolves a code action (exposed for testing)
func (s *Server) CodeActionResolve(action *protocol.CodeAction) (*protocol.CodeAction, error) {
	fmt.Fprintf(os.Stderr, "[DTLS] CodeActionResolve requested: %s\n", action.Title)

	// For now, we compute the edit immediately in CodeAction
	// This is here for future optimization where we could defer computing edits
	return action, nil
}

// createFixFallbackAction creates a code action to fix an incorrect fallback value
func (s *Server) createFixFallbackAction(uri string, varCall css.VarCall, token *tokens.Token, diagnostics []protocol.Diagnostic) protocol.CodeAction {
	// Find the matching diagnostic
	var matchingDiag *protocol.Diagnostic
	for i := range diagnostics {
		if diagnostics[i].Range.Start.Line == varCall.Range.Start.Line &&
			diagnostics[i].Range.Start.Character == varCall.Range.Start.Character {
			matchingDiag = &diagnostics[i]
			break
		}
	}

	// Create the replacement text
	newText := fmt.Sprintf("var(%s, %s)", varCall.TokenName, token.Value)

	kind := protocol.CodeActionKindQuickFix
	action := protocol.CodeAction{
		Title: fmt.Sprintf("Fix fallback value to '%s'", token.Value),
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

	return action
}

// createAddFallbackAction creates a code action to add a fallback value
func (s *Server) createAddFallbackAction(uri string, varCall css.VarCall, token *tokens.Token) protocol.CodeAction {
	// Create the replacement text
	newText := fmt.Sprintf("var(%s, %s)", varCall.TokenName, token.Value)

	kind := protocol.CodeActionKindQuickFix
	return protocol.CodeAction{
		Title: fmt.Sprintf("Add fallback value '%s'", token.Value),
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
}

// createDeprecatedTokenActions creates code actions for deprecated tokens
func (s *Server) createDeprecatedTokenActions(uri string, varCall css.VarCall, token *tokens.Token, diagnostics []protocol.Diagnostic) []protocol.CodeAction {
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

		replacementToken := s.tokens.Get(cssVarName)
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
	kind := protocol.CodeActionKindQuickFix
	removeAction := protocol.CodeAction{
		Title: fmt.Sprintf("Replace with literal value '%s'", token.Value),
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
						NewText: token.Value,
					},
				},
			},
		},
	}

	if matchingDiag != nil {
		removeAction.Diagnostics = []protocol.Diagnostic{*matchingDiag}
	}

	actions = append(actions, removeAction)

	return actions
}

// rangesIntersect checks if two ranges intersect
// Ranges are treated as half-open intervals [start, end) where the end position is exclusive
func rangesIntersect(a, b protocol.Range) bool {
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
