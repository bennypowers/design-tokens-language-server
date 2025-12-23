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
			parentPath = strings.ReplaceAll(parentPath, "/", "/")
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

	// Group tokens by their group path
	tokensByGroup := make(map[string][]*tokens.Token)
	for _, tok := range tokenList {
		if len(tok.Path) == 0 {
			continue
		}
		// Skip $extends tokens themselves
		if tok.Path[len(tok.Path)-1] == "$extends" {
			continue
		}
		// Get the top-level group
		groupPath := tok.Path[0]
		tokensByGroup[groupPath] = append(tokensByGroup[groupPath], tok)
	}

	// Process extensions in topological order
	for _, groupPath := range order {
		parentPath, hasExtends := extendsMap[groupPath]
		if !hasExtends {
			continue
		}

		// Get parent and child tokens
		parentTokens := tokensByGroup[parentPath]
		childTokens := tokensByGroup[groupPath]

		// Build set of child token names (to detect overrides)
		childNames := make(map[string]bool)
		for _, tok := range childTokens {
			// Get relative name within the group
			if len(tok.Path) > 1 {
				relativeName := strings.Join(tok.Path[1:], "-")
				childNames[relativeName] = true
			}
		}

		// Clone parent tokens into child group (if not overridden)
		for _, parentTok := range parentTokens {
			// Get relative path within parent group
			if len(parentTok.Path) < 2 {
				continue
			}
			relativePath := parentTok.Path[1:]
			relativeName := strings.Join(relativePath, "-")

			// Skip if child already has this token (override)
			if childNames[relativeName] {
				continue
			}

			// Clone token with child group path
			newPath := append([]string{groupPath}, relativePath...)
			newName := strings.Join(newPath, "-")

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
			}

			// Add to token list and child group
			tokenList = append(tokenList, clonedToken)
			tokensByGroup[groupPath] = append(tokensByGroup[groupPath], clonedToken)
		}
	}

	return tokenList, nil
}
