package resolver

import (
	"fmt"
	"strings"

	"bennypowers.dev/dtls/internal/schema"
	"bennypowers.dev/dtls/internal/tokens"
)

// ResolveAliases resolves all alias references in the token list
// Updates ResolvedValue and IsResolved fields on each token
func ResolveAliases(tokenList []*tokens.Token, version schema.SchemaVersion) error {
	// Build dependency graph
	graph := BuildDependencyGraph(tokenList)

	// Check for circular dependencies
	if graph.HasCycle() {
		cycle := graph.FindCycle()
		return schema.NewCircularReferenceError("", cycle)
	}

	// Get topological sort order (dependencies first)
	sortedNames, err := graph.TopologicalSort()
	if err != nil {
		return err
	}

	// Build lookup map
	tokenByName := make(map[string]*tokens.Token)
	for _, tok := range tokenList {
		tokenByName[tok.Name] = tok
	}

	// Resolve tokens in dependency order
	for _, name := range sortedNames {
		tok := tokenByName[name]
		if tok == nil {
			continue
		}

		if err := resolveToken(tok, tokenByName, version); err != nil {
			return err
		}
	}

	return nil
}

// resolveToken resolves a single token's value
func resolveToken(tok *tokens.Token, tokenByName map[string]*tokens.Token, version schema.SchemaVersion) error {
	// If already resolved, nothing to do
	if tok.IsResolved {
		return nil
	}

	// Check if this is an alias
	isAlias := false

	// Check for curly brace reference
	if strings.Contains(tok.Value, "{") {
		isAlias = true
		resolved, err := resolveCurlyBraceReference(tok.Value, tokenByName)
		if err != nil {
			return err
		}
		tok.ResolvedValue = resolved
	} else if version != schema.Draft && strings.HasPrefix(tok.Value, "#/") {
		// Check for JSON Pointer reference ($ref)
		isAlias = true
		resolved, err := resolveJSONPointerReference(tok.Value, tokenByName)
		if err != nil {
			return err
		}
		tok.ResolvedValue = resolved
	}

	// If not an alias, resolve to its own value
	if !isAlias {
		if tok.RawValue != nil {
			tok.ResolvedValue = tok.RawValue
		} else {
			tok.ResolvedValue = tok.Value
		}
	}

	tok.IsResolved = true
	return nil
}

// resolveCurlyBraceReference resolves a curly brace reference like {color.base}
func resolveCurlyBraceReference(value string, tokenByName map[string]*tokens.Token) (interface{}, error) {
	// Extract the reference
	refs := extractCurlyBraceReferences(value)
	if len(refs) == 0 {
		// No reference, return the value as-is
		return value, nil
	}

	// For now, only support single whole-token references
	// TODO: Phase 4b will add support for embedded references and property-level refs
	if len(refs) > 1 || !strings.HasPrefix(value, "{") || !strings.HasSuffix(value, "}") {
		// Not a whole-token reference, return value as-is
		return value, nil
	}

	ref := refs[0]
	// Convert dot-separated path to hyphenated token name
	tokenName := strings.ReplaceAll(ref, ".", "-")

	// Look up the referenced token
	refToken := tokenByName[tokenName]
	if refToken == nil {
		return nil, fmt.Errorf("reference to non-existent token: %s", ref)
	}

	// Return the resolved value of the referenced token
	if !refToken.IsResolved {
		return nil, fmt.Errorf("referenced token not yet resolved: %s", ref)
	}

	return refToken.ResolvedValue, nil
}

// resolveJSONPointerReference resolves a JSON Pointer reference like #/color/base
func resolveJSONPointerReference(value string, tokenByName map[string]*tokens.Token) (interface{}, error) {
	// Extract token name from JSON Pointer path
	// e.g., "#/color/base" -> "color-base"
	path := strings.TrimPrefix(value, "#/")
	tokenName := strings.ReplaceAll(path, "/", "-")

	// Look up the referenced token
	refToken := tokenByName[tokenName]
	if refToken == nil {
		return nil, fmt.Errorf("$ref points to non-existent token: %s", value)
	}

	// Return the resolved value of the referenced token
	if !refToken.IsResolved {
		return nil, fmt.Errorf("referenced token not yet resolved: %s", value)
	}

	return refToken.ResolvedValue, nil
}
