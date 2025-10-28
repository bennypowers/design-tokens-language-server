package diagnostic

import (
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// LSP 3.17 Pull Diagnostics types
//
// WORKAROUND: These types are defined here because glsp v0.2.2 only implements LSP 3.16.
// Pull diagnostics (textDocument/diagnostic) was introduced in LSP 3.17.
//
// When glsp is updated to support LSP 3.17, these type definitions can be removed
// and replaced with the library's native types from protocol_3_17 package.
//
// See: https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_diagnostic

// DocumentDiagnosticParams represents the parameters for textDocument/diagnostic request
type DocumentDiagnosticParams struct {
	// The text document
	TextDocument protocol.TextDocumentIdentifier `json:"textDocument"`

	// The additional identifier provided during registration
	Identifier string `json:"identifier,omitempty"`

	// The result id of a previous response if provided
	PreviousResultID string `json:"previousResultId,omitempty"`
}

// DocumentDiagnosticReportKind represents the kind of diagnostic report
type DocumentDiagnosticReportKind string

const (
	// DiagnosticFull represents a full document diagnostic report
	DiagnosticFull DocumentDiagnosticReportKind = "full"
	// DiagnosticUnchanged represents an unchanged diagnostic report
	DiagnosticUnchanged DocumentDiagnosticReportKind = "unchanged"
)

// RelatedFullDocumentDiagnosticReport represents a full diagnostic report
type RelatedFullDocumentDiagnosticReport struct {
	// The kind of diagnostic report
	Kind string `json:"kind"`

	// An optional result id
	ResultID string `json:"resultId,omitempty"`

	// The actual items
	Items []protocol.Diagnostic `json:"items"`

	// Related documents (not used in our implementation)
	RelatedDocuments map[string]any `json:"relatedDocuments,omitempty"`
}

// DiagnosticOptions represents server capabilities for pull diagnostics
type DiagnosticOptions struct {
	// Whether the server has inter-file dependencies
	InterFileDependencies bool `json:"interFileDependencies"`

	// Whether the server supports workspace diagnostics
	WorkspaceDiagnostics bool `json:"workspaceDiagnostics"`
}
