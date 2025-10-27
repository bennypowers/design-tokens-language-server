package types

import (
	"github.com/tliron/glsp"
)

// RequestContext contains all request-scoped data for an LSP method call.
// It wraps both the server-wide context and the GLSP protocol context,
// and provides storage for request-scoped warnings.
type RequestContext struct {
	Server   ServerContext   // Server-wide context (documents, tokens, config)
	GLSP     *glsp.Context   // GLSP protocol context (Notify, Call methods)
	warnings []error         // Request-scoped warnings (collected during handler execution)
}

// NewRequestContext creates a new request context
func NewRequestContext(server ServerContext, glsp *glsp.Context) *RequestContext {
	return &RequestContext{
		Server: server,
		GLSP:   glsp,
	}
}

// AddWarning adds a non-fatal warning to this request.
// Warnings are logged by middleware after successful handler completion.
func (r *RequestContext) AddWarning(err error) {
	if err != nil {
		r.warnings = append(r.warnings, err)
	}
}

// Warnings returns all warnings collected during this request.
// Returns nil if no warnings were added.
func (r *RequestContext) Warnings() []error {
	return r.warnings
}

// HasWarnings returns true if any warnings were collected
func (r *RequestContext) HasWarnings() bool {
	return len(r.warnings) > 0
}
