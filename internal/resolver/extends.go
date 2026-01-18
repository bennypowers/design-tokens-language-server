package resolver

import (
	"strings"

	"bennypowers.dev/dtls/internal/schema"
	"bennypowers.dev/dtls/internal/tokens"
)

// ResolveGroupExtensions resolves $extends relationships in token groups
// Only applies to 2025.10 schema. Performs deep merge with parent groups.
// Returns the modified token list with inherited tokens added.
func ResolveGroupExtensions(tokenList []*tokens.Token) ([]*tokens.Token, error) {
	if len(tokenList) == 0 {
		return tokenList, nil
	}

	// Only process 2025.10 tokens
	has2025Tokens := false
	for _, tok := range tokenList {
		if tok.SchemaVersion == schema.V2025_10 {
			has2025Tokens = true
			break
		}
	}

	if !has2025Tokens {
		return tokenList, nil
	}

	// Find $extends relationships
	extendsMap := make(map[string]string) // childGroupPath -> parentGroupPath
	for _, tok := range tokenList {
		// Look for tokens with path ending in "$extends"
		if len(tok.Path) > 0 && tok.Path[len(tok.Path)-1] == "$extends" {
			// Get the group path (everything except "$extends")
			groupPath := strings.Join(tok.Path[:len(tok.Path)-1], "/")
			// tok.Value is the JSON Pointer like "#/baseColors"
			parentPath := strings.TrimPrefix(tok.Value, "#/")
			extendsMap[groupPath] = parentPath
		}
	}

	if len(extendsMap) == 0 {
		return tokenList, nil // No extensions to process
	}

	// Build dependency graph
	graph := &DependencyGraph{
		dependencies: make(map[string][]string),
		dependents:   make(map[string][]string),
		nodes:        make(map[string]bool),
	}

	for child, parent := range extendsMap {
		graph.nodes[child] = true
		graph.nodes[parent] = true
		graph.dependencies[child] = []string{parent}
		graph.dependents[parent] = append(graph.dependents[parent], child)
	}

	// Check for cycles
	if graph.HasCycle() {
		cycle := graph.FindCycle()
		return tokenList, schema.NewCircularReferenceError("", cycle)
	}

	// Get topological order (parents first)
	order, err := graph.TopologicalSort()
	if err != nil {
		return tokenList, err
	}

	// Pre-index tokens by group path for performance
	// This avoids O(n) scans for each getTokensForGroup call
	groupIndex := make(map[string][]*tokens.Token)
	for _, tok := range tokenList {
		// Skip $extends tokens
		if len(tok.Path) > 0 && tok.Path[len(tok.Path)-1] == "$extends" {
			continue
		}

		// Index token under all its parent group paths
		// e.g., "color/brand/primary" is indexed under "color", "color/brand", "color/brand/primary"
		for i := 1; i <= len(tok.Path); i++ {
			groupPath := strings.Join(tok.Path[:i], "/")
			groupIndex[groupPath] = append(groupIndex[groupPath], tok)
		}
	}

	// Helper function to get tokens matching a path prefix
	getTokensForGroup := func(groupPath string) []*tokens.Token {
		return groupIndex[groupPath]
	}

	// Process extensions in topological order
	for _, groupPath := range order {
		parentPath, hasExtends := extendsMap[groupPath]
		if !hasExtends {
			continue
		}

		// Get parent and child tokens using the helper function
		parentTokens := getTokensForGroup(parentPath)
		childTokens := getTokensForGroup(groupPath)

		// Parse group paths
		parentPathParts := strings.Split(parentPath, "/")
		childPathParts := strings.Split(groupPath, "/")

		// Build set of child token names (to detect overrides)
		childNames := make(map[string]bool)
		for _, tok := range childTokens {
			// Get relative name within the child group
			if len(tok.Path) > len(childPathParts) {
				relativeName := strings.Join(tok.Path[len(childPathParts):], "-")
				childNames[relativeName] = true
			}
		}

		// Clone parent tokens into child group (if not overridden)
		for _, parentTok := range parentTokens {
			// Get relative path within parent group
			if len(parentTok.Path) <= len(parentPathParts) {
				continue
			}
			relativePath := parentTok.Path[len(parentPathParts):]
			relativeName := strings.Join(relativePath, "-")

			// Skip if child already has this token (override)
			if childNames[relativeName] {
				continue
			}

			// Clone token with child group path
			newPath := make([]string, len(childPathParts)+len(relativePath))
			copy(newPath, childPathParts)
			copy(newPath[len(childPathParts):], relativePath)
			newName := strings.Join(newPath, "-")

			// Deep copy extensions to avoid shared references
			var extensions map[string]interface{}
			if parentTok.Extensions != nil {
				extensions = make(map[string]interface{}, len(parentTok.Extensions))
				for k, v := range parentTok.Extensions {
					extensions[k] = v // Note: nested structures still shared
				}
			}

			clonedToken := &tokens.Token{
				Name:               newName,
				Value:              parentTok.Value,
				Type:               parentTok.Type,
				Description:        parentTok.Description,
				Path:               newPath,
				Line:               parentTok.Line,
				Character:          parentTok.Character,
				FilePath:           parentTok.FilePath,
				DefinitionURI:      parentTok.DefinitionURI,
				Prefix:             parentTok.Prefix,
				SchemaVersion:      parentTok.SchemaVersion,
				RawValue:           parentTok.RawValue,
				ResolvedValue:      parentTok.ResolvedValue,
				IsResolved:         parentTok.IsResolved,
				Deprecated:         parentTok.Deprecated,
				DeprecationMessage: parentTok.DeprecationMessage,
				Extensions:         extensions,
			}

			// Add cloned token to the token list
			tokenList = append(tokenList, clonedToken)

			// Update the index so future extends can see this inherited token
			for i := 1; i <= len(newPath); i++ {
				indexPath := strings.Join(newPath[:i], "/")
				groupIndex[indexPath] = append(groupIndex[indexPath], clonedToken)
			}
		}
	}

	return tokenList, nil
}
