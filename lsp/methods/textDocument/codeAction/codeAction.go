package codeaction

import (
	"bennypowers.dev/dtls/internal/log"
	"fmt"
	"strings"

	"bennypowers.dev/dtls/internal/documents"
	cssparser "bennypowers.dev/dtls/internal/parser/css"
	"bennypowers.dev/dtls/lsp/helpers"
	"bennypowers.dev/dtls/lsp/helpers/css"
	"bennypowers.dev/dtls/lsp/types"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// validateCSSDocument validates that the document exists and is a CSS file.
// Returns the document and true if valid, or nil and false otherwise.
func validateCSSDocument(req *types.RequestContext, uri string) (*documents.Document, bool) {
	doc := req.Server.Document(uri)
	if doc == nil {
		return nil, false
	}
	if doc.LanguageID() != "css" {
		return nil, false
	}
	return doc, true
}

// parseVarCalls parses CSS content and extracts all var() calls.
// Returns the list of var calls and any parsing error.
func parseVarCalls(doc *documents.Document) ([]*cssparser.VarCall, error) {
	parser := cssparser.AcquireParser()
	defer cssparser.ReleaseParser(parser)
	result, err := parser.Parse(doc.Content())
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSS: %w", err)
	}
	return result.VarCalls, nil
}

// processVarCalls processes all var() calls in the requested range and generates code actions.
// Returns the list of code actions and the var calls that were in range.
func processVarCalls(req *types.RequestContext, uri string, varCalls []*cssparser.VarCall, params *protocol.CodeActionParams) ([]protocol.CodeAction, []cssparser.VarCall) {
	var actions []protocol.CodeAction
	var varCallsInRange []cssparser.VarCall

	// Check each var() call in the requested range
	for _, varCall := range varCalls {
		// Check if var call intersects with the requested range
		if !helpers.RangesIntersect(params.Range, protocol.Range{
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
			actions = append(actions, createDeprecatedTokenActions(req, uri, *varCall, token, params.Context.Diagnostics)...)
		}

		// Create code actions for incorrect fallback
		if varCall.Fallback != nil {
			fallbackValue := *varCall.Fallback
			tokenValue := token.Value

			if !css.IsCSSValueSemanticallyEquivalent(fallbackValue, tokenValue) {
				if action := createFixFallbackAction(req, uri, *varCall, token, params.Context.Diagnostics); action != nil {
					actions = append(actions, *action)
				}
			}
		} else if token.Type == "color" || token.Type == "dimension" {
			// Suggest adding fallback for color and dimension tokens
			if action := createAddFallbackAction(req, uri, *varCall, token); action != nil {
				actions = append(actions, *action)
			}
		}
	}

	return actions, varCallsInRange
}

// extractRecommendedToken extracts the recommended replacement token from a deprecation message.
// Returns the token name if found, empty string otherwise.
func extractRecommendedToken(deprecationMessage string) string {
	if deprecationMessage == "" {
		return ""
	}

	// Pattern: "Use X instead" or "Use X.Y instead"
	if idx := strings.Index(deprecationMessage, "Use "); idx != -1 {
		rest := deprecationMessage[idx+4:]
		if endIdx := strings.Index(rest, " instead"); endIdx != -1 {
			return strings.TrimSpace(rest[:endIdx])
		}
	}

	// Pattern: "Replaced by X"
	if idx := strings.Index(deprecationMessage, "Replaced by "); idx != -1 {
		rest := deprecationMessage[idx+12:]
		// Take until space or end of string
		if spaceIdx := strings.Index(rest, " "); spaceIdx != -1 {
			return rest[:spaceIdx]
		}
		return rest
	}

	return ""
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
	parser := cssparser.AcquireParser()
	defer cssparser.ReleaseParser(parser)
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

			if !css.IsCSSValueSemanticallyEquivalent(fallbackValue, tokenValue) {
				// Format the token value
				formattedValue, err := css.FormatTokenValueForCSS(token)
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

// CodeAction handles the textDocument/codeAction request
func CodeAction(req *types.RequestContext, params *protocol.CodeActionParams) (any, error) {
	uri := params.TextDocument.URI
	log.Info("CodeAction requested: %s", uri)

	// Check if client supports CodeAction literals
	// Legacy clients only support Command, which we don't implement
	if !req.Server.SupportsCodeActionLiterals() {
		log.Info("Client does not support CodeAction literals, returning nil")
		return nil, nil
	}

	// Validate document
	doc, ok := validateCSSDocument(req, uri)
	if !ok {
		return nil, nil
	}

	// Parse CSS to find var() calls
	varCalls, err := parseVarCalls(doc)
	if err != nil {
		return nil, err
	}

	// Process var calls and collect actions
	actions, varCallsInRange := processVarCalls(req, uri, varCalls, params)

	// Add toggle actions
	actions = append(actions, createToggleActions(req, uri, varCallsInRange, params.Range)...)

	// Add fix-all action if needed
	if fixAllAction := createFixAllActionIfNeeded(uri, varCalls, params.Context.Diagnostics); fixAllAction != nil {
		actions = append(actions, *fixAllAction)
	}

	log.Info("Returning %d code actions", len(actions))
	return actions, nil
}

// CodeActionResolve handles the codeAction/resolve request
func CodeActionResolve(req *types.RequestContext, action *protocol.CodeAction) (*protocol.CodeAction, error) {
	log.Info("CodeActionResolve requested: %s", action.Title)

	// Handle fixAllFallbacks which uses lazy resolution
	if action.Title == "Fix all token fallback values" {
		return resolveFixAllFallbacks(req, action)
	}

	// For other actions (fixFallback, toggle, add, deprecated),
	// we compute the edit immediately in CodeAction, so just return as-is
	return action, nil
}
