package schema

import (
	"fmt"
	"sync"

	"gopkg.in/yaml.v3"
)

// Registry manages SchemaHandlers for different schema versions
type Registry struct {
	handlers map[SchemaVersion]SchemaHandler
	mu       sync.RWMutex
}

var (
	// DefaultRegistry is the global schema handler registry
	DefaultRegistry = NewRegistry()
)

// NewRegistry creates a new schema handler registry
func NewRegistry() *Registry {
	r := &Registry{
		handlers: make(map[SchemaVersion]SchemaHandler),
	}

	// Register built-in handlers
	r.Register(&DraftSchemaHandler{})
	r.Register(&V2025_10SchemaHandler{})

	return r
}

// Register adds a schema handler to the registry
func (r *Registry) Register(handler SchemaHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[handler.Version()] = handler
}

// Get retrieves a handler for the specified schema version
// Returns error if no handler is registered for the version
func (r *Registry) Get(version SchemaVersion) (SchemaHandler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler, ok := r.handlers[version]
	if !ok {
		return nil, fmt.Errorf("no handler registered for schema version: %s", version)
	}
	return handler, nil
}

// Versions returns all registered schema versions
func (r *Registry) Versions() []SchemaVersion {
	r.mu.RLock()
	defer r.mu.RUnlock()

	versions := make([]SchemaVersion, 0, len(r.handlers))
	for version := range r.handlers {
		versions = append(versions, version)
	}
	return versions
}

// DraftSchemaHandler implements SchemaHandler for the editor's draft schema
type DraftSchemaHandler struct{}

func (h *DraftSchemaHandler) Version() SchemaVersion {
	return Draft
}

func (h *DraftSchemaHandler) ValidateTokenNode(node *yaml.Node) error {
	// Draft schema validation - tokens can have $type, $value, $description, $extensions
	// No structured color values, no $ref, no $extends
	return nil
}

func (h *DraftSchemaHandler) FormatColorForCSS(colorValue interface{}) string {
	// Draft schema: colors are strings (hex, rgb(), hsl(), named)
	if str, ok := colorValue.(string); ok {
		return str
	}
	return ""
}

func (h *DraftSchemaHandler) SupportsFeature(feature string) bool {
	switch feature {
	case "curly-brace-references":
		return true
	case "json-pointer", "extends", "root", "resolution-order":
		return false
	default:
		return false
	}
}

// V2025_10SchemaHandler implements SchemaHandler for the 2025.10 stable schema
type V2025_10SchemaHandler struct{}

func (h *V2025_10SchemaHandler) Version() SchemaVersion {
	return V2025_10
}

func (h *V2025_10SchemaHandler) ValidateTokenNode(node *yaml.Node) error {
	// 2025.10 schema validation - supports structured colors, $ref, $extends, $root
	return nil
}

func (h *V2025_10SchemaHandler) FormatColorForCSS(colorValue interface{}) string {
	// 2025.10 schema: colors can be structured objects or strings
	// Try to extract hex field from structured color
	if colorMap, ok := colorValue.(map[string]interface{}); ok {
		if hex, ok := colorMap["hex"].(string); ok {
			return hex
		}

		// TODO: Convert from colorSpace/components to CSS
		// For now, return empty string to indicate conversion needed
		return ""
	}

	// Fallback to string representation
	if str, ok := colorValue.(string); ok {
		return str
	}

	return ""
}

func (h *V2025_10SchemaHandler) SupportsFeature(feature string) bool {
	switch feature {
	case "curly-brace-references", "json-pointer", "extends", "root":
		return true
	case "resolution-order":
		// Post-MVP feature
		return false
	default:
		return false
	}
}
