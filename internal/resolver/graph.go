package resolver

import (
	"fmt"
	"strings"

	"bennypowers.dev/dtls/internal/parser/common"
	"bennypowers.dev/dtls/internal/schema"
	"bennypowers.dev/dtls/internal/tokens"
)

// DependencyGraph represents a directed graph of token dependencies
type DependencyGraph struct {
	// adjacency list: token name -> list of tokens it depends on
	dependencies map[string][]string
	// reverse lookup: token name -> list of tokens that depend on it
	dependents map[string][]string
	// all token names in the graph
	nodes map[string]bool
}

// BuildDependencyGraph builds a dependency graph from a list of tokens
func BuildDependencyGraph(tokenList []*tokens.Token) *DependencyGraph {
	graph := &DependencyGraph{
		dependencies: make(map[string][]string),
		dependents:   make(map[string][]string),
		nodes:        make(map[string]bool),
	}

	// Build token name lookup
	tokenByName := make(map[string]*tokens.Token)
	for _, tok := range tokenList {
		graph.nodes[tok.Name] = true
		tokenByName[tok.Name] = tok
	}

	// Extract dependencies from each token
	for _, tok := range tokenList {
		deps := extractDependencies(tok)
		if len(deps) > 0 {
			graph.dependencies[tok.Name] = deps
			for _, dep := range deps {
				graph.dependents[dep] = append(graph.dependents[dep], tok.Name)
			}
		}
	}

	return graph
}

// extractDependencies extracts token names that this token depends on
func extractDependencies(tok *tokens.Token) []string {
	deps := []string{}

	// Check for curly brace references in Value
	if strings.Contains(tok.Value, "{") {
		// Extract {token.path} references
		refs := extractCurlyBraceReferences(tok.Value)
		for _, ref := range refs {
			// Convert dot-separated path to hyphenated token name
			// e.g., "color.base" -> "color-base"
			tokenName := strings.ReplaceAll(ref, ".", "-")
			deps = append(deps, tokenName)
		}
	}

	// Check for JSON Pointer references ($ref field)
	// For 2025.10, the Value field contains the JSON Pointer path
	if tok.SchemaVersion != schema.Draft && strings.HasPrefix(tok.Value, "#/") {
		// Extract token name from JSON Pointer
		// e.g., "#/color/base" -> "color-base"
		path := strings.TrimPrefix(tok.Value, "#/")
		tokenName := strings.ReplaceAll(path, "/", "-")
		deps = append(deps, tokenName)
	}

	return deps
}

// extractCurlyBraceReferences extracts token paths from curly brace references
// Uses the shared regex pattern from internal/parser/common for consistency
func extractCurlyBraceReferences(value string) []string {
	refs := []string{}
	matches := common.CurlyBraceReferenceRegexp.FindAllStringSubmatch(value, -1)
	for _, match := range matches {
		if len(match) > 1 {
			refs = append(refs, match[1]) // The captured group (path)
		}
	}
	return refs
}

// GetDependencies returns the list of tokens that the given token depends on
func (g *DependencyGraph) GetDependencies(tokenName string) []string {
	if deps, ok := g.dependencies[tokenName]; ok {
		return deps
	}
	return []string{}
}

// HasCycle returns true if the graph contains a circular dependency
func (g *DependencyGraph) HasCycle() bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for node := range g.nodes {
		if g.hasCycleDFS(node, visited, recStack) {
			return true
		}
	}

	return false
}

// hasCycleDFS performs depth-first search to detect cycles
func (g *DependencyGraph) hasCycleDFS(node string, visited, recStack map[string]bool) bool {
	if recStack[node] {
		return true
	}
	if visited[node] {
		return false
	}

	visited[node] = true
	recStack[node] = true

	for _, dep := range g.dependencies[node] {
		if g.hasCycleDFS(dep, visited, recStack) {
			return true
		}
	}

	recStack[node] = false
	return false
}

// FindCycle returns the cycle path if one exists, or nil if no cycle
func (g *DependencyGraph) FindCycle() []string {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := []string{}

	for node := range g.nodes {
		if cycle := g.findCycleDFS(node, visited, recStack, path); cycle != nil {
			return cycle
		}
	}

	return nil
}

// findCycleDFS finds a cycle and returns the path
func (g *DependencyGraph) findCycleDFS(node string, visited, recStack map[string]bool, path []string) []string {
	if recStack[node] {
		// Found a cycle - return the path from this node
		// Invariant: node must be in path because we add it immediately after setting recStack[node] = true
		cycleStart := -1
		for i, n := range path {
			if n == node {
				cycleStart = i
				break
			}
		}
		if cycleStart == -1 {
			// This should never happen - indicates a broken invariant in the DFS algorithm
			panic(fmt.Sprintf("cycle detection invariant violated: node %q in recStack but not in path %v", node, path))
		}
		return append(path[cycleStart:], node)
	}
	if visited[node] {
		return nil
	}

	visited[node] = true
	recStack[node] = true
	path = append(path, node)

	for _, dep := range g.dependencies[node] {
		if cycle := g.findCycleDFS(dep, visited, recStack, path); cycle != nil {
			return cycle
		}
	}

	recStack[node] = false
	return nil
}

// TopologicalSort returns tokens in dependency order (dependencies first)
// Returns error if graph contains a cycle
func (g *DependencyGraph) TopologicalSort() ([]string, error) {
	// Check for cycles first using dedicated cycle detection.
	// Note: This performs a separate DFS before the sorting DFS below, which is
	// intentionally redundant for clarity. The separation allows FindCycle() to
	// provide detailed error reporting with the full cycle path, while the sort
	// DFS below is optimized for building the result list. This trade-off favors
	// clear error messages and maintainability over performance optimization.
	if cycle := g.FindCycle(); cycle != nil {
		return nil, schema.NewCircularReferenceError("", cycle)
	}

	visited := make(map[string]bool)
	result := []string{}

	// Visit all nodes
	for node := range g.nodes {
		if !visited[node] {
			g.topologicalSortDFS(node, visited, &result)
		}
	}

	// Result is already in topological order (dependencies first)
	// because we add nodes to result after visiting their dependencies
	return result, nil
}

// topologicalSortDFS performs DFS for topological sort
func (g *DependencyGraph) topologicalSortDFS(node string, visited map[string]bool, stack *[]string) {
	visited[node] = true

	// Visit all dependencies first
	for _, dep := range g.dependencies[node] {
		if !visited[dep] {
			g.topologicalSortDFS(dep, visited, stack)
		}
	}

	// Push node to stack after visiting all dependencies
	*stack = append(*stack, node)
}
