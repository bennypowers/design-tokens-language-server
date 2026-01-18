package schema

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors for error type checking
var (
	// ErrSchemaDetectionFailed indicates schema version could not be determined
	ErrSchemaDetectionFailed = errors.New("schema version detection failed")

	// ErrInvalidSchema indicates the schema is not valid or recognized
	ErrInvalidSchema = errors.New("invalid schema")

	// ErrMixedSchemaFeatures indicates file contains features from multiple schema versions
	ErrMixedSchemaFeatures = errors.New("mixed schema features")

	// ErrConflictingRootTokens indicates both $root and groupMarkers are present
	ErrConflictingRootTokens = errors.New("conflicting root tokens")

	// ErrInvalidColorFormat indicates color value format doesn't match schema version
	ErrInvalidColorFormat = errors.New("invalid color format for schema")

	// ErrCircularReference indicates a circular reference was detected
	ErrCircularReference = errors.New("circular reference detected")
)

// SchemaDetectionError represents failure to detect schema version
type SchemaDetectionError struct {
	FilePath string
	Reason   string
}

func (e *SchemaDetectionError) Error() string {
	return fmt.Sprintf("failed to detect schema version for %s: %s\nSuggestion: Add explicit $schema field to the file", e.FilePath, e.Reason)
}

func (e *SchemaDetectionError) Unwrap() error {
	return ErrSchemaDetectionFailed
}

// NewSchemaDetectionError creates a new schema detection error
func NewSchemaDetectionError(filePath, reason string) error {
	return &SchemaDetectionError{
		FilePath: filePath,
		Reason:   reason,
	}
}

// InvalidSchemaError represents an invalid or unrecognized schema
type InvalidSchemaError struct {
	FilePath      string
	SchemaVersion string
	Reason        string
}

func (e *InvalidSchemaError) Error() string {
	return fmt.Sprintf("invalid schema %s in %s: %s", e.SchemaVersion, e.FilePath, e.Reason)
}

func (e *InvalidSchemaError) Unwrap() error {
	return ErrInvalidSchema
}

// NewInvalidSchemaError creates a new invalid schema error
func NewInvalidSchemaError(filePath, schemaVersion, reason string) error {
	return &InvalidSchemaError{
		FilePath:      filePath,
		SchemaVersion: schemaVersion,
		Reason:        reason,
	}
}

// MixedSchemaFeaturesError represents file containing features from multiple schemas
type MixedSchemaFeaturesError struct {
	FilePath            string
	DeclaredSchema      string
	ConflictingFeatures []string
}

func (e *MixedSchemaFeaturesError) Error() string {
	features := strings.Join(e.ConflictingFeatures, ", ")
	return fmt.Sprintf("file %s declares schema '%s' but contains features from other schema versions: %s\nSuggestion: Remove incompatible features or update $schema field",
		e.FilePath, e.DeclaredSchema, features)
}

func (e *MixedSchemaFeaturesError) Unwrap() error {
	return ErrMixedSchemaFeatures
}

// NewMixedSchemaFeaturesError creates a new mixed schema features error
func NewMixedSchemaFeaturesError(filePath, declaredSchema string, conflictingFeatures []string) error {
	return &MixedSchemaFeaturesError{
		FilePath:            filePath,
		DeclaredSchema:      declaredSchema,
		ConflictingFeatures: conflictingFeatures,
	}
}

// ConflictingRootTokensError represents both $root and groupMarkers present
type ConflictingRootTokensError struct {
	FilePath      string
	GroupPath     string
	RootTokenName string
	MarkerName    string
}

func (e *ConflictingRootTokensError) Error() string {
	return fmt.Sprintf("file %s has conflicting root tokens in group '%s': both '%s' and '%s' found\nSuggestion: Use only $root for 2025.10+ schemas, or only groupMarkers for draft schemas",
		e.FilePath, e.GroupPath, e.RootTokenName, e.MarkerName)
}

func (e *ConflictingRootTokensError) Unwrap() error {
	return ErrConflictingRootTokens
}

// NewConflictingRootTokensError creates a new conflicting root tokens error
func NewConflictingRootTokensError(filePath, groupPath, rootTokenName, markerName string) error {
	return &ConflictingRootTokensError{
		FilePath:      filePath,
		GroupPath:     groupPath,
		RootTokenName: rootTokenName,
		MarkerName:    markerName,
	}
}

// InvalidColorFormatError represents color value format mismatch
type InvalidColorFormatError struct {
	FilePath       string
	TokenPath      string
	SchemaVersion  string
	FoundFormat    string
	ExpectedFormat string
}

func (e *InvalidColorFormatError) Error() string {
	return fmt.Sprintf("invalid color format for token '%s' in %s: schema '%s' expects %s, but found %s\nSuggestion: Convert color value to match schema version, or update $schema field",
		e.TokenPath, e.FilePath, e.SchemaVersion, e.ExpectedFormat, e.FoundFormat)
}

func (e *InvalidColorFormatError) Unwrap() error {
	return ErrInvalidColorFormat
}

// NewInvalidColorFormatError creates a new invalid color format error
func NewInvalidColorFormatError(filePath, tokenPath, schemaVersion, foundFormat, expectedFormat string) error {
	return &InvalidColorFormatError{
		FilePath:       filePath,
		TokenPath:      tokenPath,
		SchemaVersion:  schemaVersion,
		FoundFormat:    foundFormat,
		ExpectedFormat: expectedFormat,
	}
}

// CircularReferenceError represents a circular reference
type CircularReferenceError struct {
	FilePath       string
	ReferenceChain []string
}

func (e *CircularReferenceError) Error() string {
	chain := strings.Join(e.ReferenceChain, " → ")
	return fmt.Sprintf("circular reference detected in %s: %s\nSuggestion: Break the circular dependency chain",
		e.FilePath, chain)
}

func (e *CircularReferenceError) Unwrap() error {
	return ErrCircularReference
}

// NewCircularReferenceError creates a new circular reference error
func NewCircularReferenceError(filePath string, chain []string) error {
	return &CircularReferenceError{
		FilePath:       filePath,
		ReferenceChain: chain,
	}
}
