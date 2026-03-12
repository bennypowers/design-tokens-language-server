package lsp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bennypowers.dev/dtls/internal/log"
)

// resolverDocument represents the structure of a DTCG resolver document.
type resolverDocument struct {
	Version         string            `json:"version"`
	Sets            map[string]setDef `json:"sets"`
	ResolutionOrder json.RawMessage   `json:"resolutionOrder"`
}

// setDef represents a named set in a resolver document.
type setDef struct {
	Sources []sourceRef `json:"sources"`
}

// sourceRef represents a source reference in a resolver document.
type sourceRef struct {
	Ref string `json:"$ref"`
}

// isResolverDocument checks if JSON data represents a resolver document
// by looking for the "resolutionOrder" field at the root.
func isResolverDocument(data []byte) bool {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return false
	}
	_, hasResolutionOrder := doc["resolutionOrder"]
	return hasResolutionOrder
}

// extractResolverSourcePaths extracts source file paths from a resolver document.
// It resolves $ref entries in both inline sources and named sets.
// Returns paths relative to the resolver document's directory.
func extractResolverSourcePaths(data []byte, resolverDir string) ([]string, error) {
	var doc resolverDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse resolver document: %w", err)
	}

	var paths []string
	seen := make(map[string]bool)

	// Parse resolutionOrder - can contain inline sources or $ref to sets
	var resolutionOrder []json.RawMessage
	if err := json.Unmarshal(doc.ResolutionOrder, &resolutionOrder); err != nil {
		return nil, fmt.Errorf("failed to parse resolutionOrder: %w", err)
	}

	for _, entry := range resolutionOrder {
		entryPaths, err := extractSourcesFromEntry(entry, doc.Sets)
		if err != nil {
			log.Warn("Failed to extract sources from resolution order entry: %v", err)
			continue
		}
		for _, p := range entryPaths {
			absPath := resolveRefPath(p, resolverDir)
			if !seen[absPath] {
				seen[absPath] = true
				paths = append(paths, absPath)
			}
		}
	}

	return paths, nil
}

// extractSourcesFromEntry extracts source file paths from a resolution order entry.
// An entry can be:
//   - An inline set with "sources": [{"$ref": "./file.json"}, ...]
//   - A reference to a named set: {"$ref": "#/sets/base"}
func extractSourcesFromEntry(entry json.RawMessage, sets map[string]setDef) ([]string, error) {
	var ref sourceRef
	if err := json.Unmarshal(entry, &ref); err == nil && ref.Ref != "" {
		// Check if it's a JSON pointer reference to a named set
		if setName, ok := strings.CutPrefix(ref.Ref, "#/sets/"); ok {
			set, ok := sets[setName]
			if !ok {
				return nil, fmt.Errorf("referenced set %q not found", setName)
			}
			return extractFileRefsFromSources(set.Sources), nil
		}
	}

	// Try as an inline set with sources
	var inlineSet struct {
		Sources []sourceRef `json:"sources"`
	}
	if err := json.Unmarshal(entry, &inlineSet); err == nil && len(inlineSet.Sources) > 0 {
		return extractFileRefsFromSources(inlineSet.Sources), nil
	}

	return nil, nil
}

// extractFileRefsFromSources extracts file paths from source $ref entries,
// filtering out JSON pointer references (which start with #).
func extractFileRefsFromSources(sources []sourceRef) []string {
	var paths []string
	for _, src := range sources {
		if src.Ref != "" && !strings.HasPrefix(src.Ref, "#") {
			paths = append(paths, src.Ref)
		}
	}
	return paths
}

// resolveRefPath resolves a $ref path relative to the resolver document's directory.
func resolveRefPath(refPath, resolverDir string) string {
	if filepath.IsAbs(refPath) {
		return filepath.Clean(refPath)
	}
	return filepath.Clean(filepath.Join(resolverDir, refPath))
}

// loadResolverDocument reads a resolver document and loads its source token files.
func (s *Server) loadResolverDocument(resolverPath string, opts *TokenFileOptions) error {
	data, err := os.ReadFile(filepath.Clean(resolverPath))
	if err != nil {
		return fmt.Errorf("failed to read resolver document %s: %w", resolverPath, err)
	}

	resolverDir := filepath.Dir(resolverPath)
	sourcePaths, err := extractResolverSourcePaths(data, resolverDir)
	if err != nil {
		return fmt.Errorf("failed to extract sources from resolver %s: %w", resolverPath, err)
	}

	log.Info("Resolver %s has %d source files", resolverPath, len(sourcePaths))

	var errs []error
	for _, srcPath := range sourcePaths {
		if err := s.loadTokenFileAndLog(srcPath, opts); err != nil {
			errs = append(errs, fmt.Errorf("failed to load resolver source %s: %w", srcPath, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors loading resolver sources: %w", errs[0])
	}
	return nil
}
