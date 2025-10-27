package diagnostic

import (
	"fmt"
	"os"
	"strings"

	"bennypowers.dev/dtls/internal/parser/css"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// handleDocumentDiagnostic handles the textDocument/diagnostic request (pull diagnostics)
//
// This is an LSP 3.17 feature. Since glsp v0.2.2 only supports LSP 3.16, this handler
// is called via CustomHandler which intercepts the method before it reaches protocol.Handler.

// DocumentDiagnostic handles the textDocument/diagnostic request (pull diagnostics)
func DocumentDiagnostic(ctx types.ServerContext, context *glsp.Context, params *DocumentDiagnosticParams) (any, error) {
	uri := params.TextDocument.URI
	fmt.Fprintf(os.Stderr, "[DTLS] Pull diagnostics requested for: %s\n", uri)

	diagnostics, err := GetDiagnostics(ctx, uri)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[DTLS] Error getting diagnostics: %v\n", err)
		return nil, err
	}

	// Return a full document diagnostic report
	return RelatedFullDocumentDiagnosticReport{
		Kind:  string(DiagnosticFull),
		Items: diagnostics,
	}, nil
}

// GetDiagnostics returns diagnostics for a document
func GetDiagnostics(ctx types.ServerContext, uri string) ([]protocol.Diagnostic, error) {
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
		return nil, fmt.Errorf("failed to parse CSS: %w", err)
	}

	var diagnostics []protocol.Diagnostic

	// Check each var() call
	for _, varCall := range result.VarCalls {
		// Look up the token
		token := ctx.Token(varCall.TokenName)
		if token == nil {
			// Unknown tokens are not errors - they're handled by hover
			continue
		}

		// Check for deprecated token
		if token.Deprecated {
			message := fmt.Sprintf("%s is deprecated", varCall.TokenName)
			if token.DeprecationMessage != "" {
				message += ": " + token.DeprecationMessage
			}

			severity := protocol.DiagnosticSeverityInformation
			diagnostics = append(diagnostics, protocol.Diagnostic{
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
				Severity: &severity,
				Message:  message,
				Tags:     []protocol.DiagnosticTag{protocol.DiagnosticTagDeprecated},
			})
		}

		// Check for incorrect fallback
		if varCall.Fallback != nil {
			fallbackValue := *varCall.Fallback
			tokenValue := token.Value

			// Check semantic equivalence (case-insensitive, whitespace-normalized)
			if !isCSSValueSemanticallyEquivalent(fallbackValue, tokenValue) {
				severity := protocol.DiagnosticSeverityError
				diagnostics = append(diagnostics, protocol.Diagnostic{
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
					Severity: &severity,
					Message:  fmt.Sprintf("Token fallback does not match expected value: %s", tokenValue),
				})
			}
		}
	}

	return diagnostics, nil
}

// PublishDiagnostics publishes diagnostics for a document

// isCSSValueSemanticallyEquivalent checks if two CSS values are semantically equivalent
// Ignores whitespace and case differences
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
