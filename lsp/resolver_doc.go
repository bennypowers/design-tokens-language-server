package lsp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"bennypowers.dev/asimonim/load"
	"bennypowers.dev/asimonim/specifier"
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

	for i, entry := range resolutionOrder {
		entryPaths, err := extractSourcesFromEntry(entry, doc.Sets)
		if err != nil {
			return nil, fmt.Errorf("invalid resolutionOrder entry %d: %w", i, err)
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
		if rawName, ok := strings.CutPrefix(ref.Ref, "#/sets/"); ok {
			setName := unescapeJSONPointer(rawName)
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

	return nil, fmt.Errorf("unrecognized resolution order entry: %s", string(entry))
}

// extractFileRefsFromSources extracts file paths from source $ref entries,
// filtering out JSON pointer references (which start with #).
func extractFileRefsFromSources(sources []sourceRef) []string {
	var paths []string
	for _, src := range sources {
		if src.Ref == "" || strings.HasPrefix(src.Ref, "#") {
			continue
		}
		// Strip any fragment identifier (e.g., "palette.json#/brand" → "palette.json")
		path, _, _ := strings.Cut(src.Ref, "#")
		if path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}

// unescapeJSONPointer decodes a JSON Pointer token per RFC 6901:
// percent-decoding first, then replacing ~1 with / and ~0 with ~.
func unescapeJSONPointer(s string) string {
	if unescaped, err := url.PathUnescape(s); err == nil {
		s = unescaped
	}
	s = strings.ReplaceAll(s, "~1", "/")
	s = strings.ReplaceAll(s, "~0", "~")
	return s
}

// resolveRefPath resolves a $ref path relative to the resolver document's directory.
// URI-scheme refs (npm:, jsr:, http://, etc.) are returned unchanged.
func resolveRefPath(refPath, resolverDir string) string {
	if strings.Contains(refPath, "://") || strings.HasPrefix(refPath, "npm:") || strings.HasPrefix(refPath, "jsr:") {
		return refPath
	}
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

	cfg := s.GetConfig()
	state := s.GetState()

	// Create fetcher once if network fallback is enabled
	var fetcher load.Fetcher
	if cfg.NetworkFallback {
		fetcher = load.NewHTTPFetcher(load.DefaultMaxSize)
	}

	var errs []error
	for _, srcPath := range sourcePaths {
		normalizedPath, err := normalizePath(srcPath, state.RootPath)
		if err != nil {
			// Try CDN fallback for package specifiers
			if fetcher != nil && specifier.IsPackageSpecifier(srcPath) {
				count, cdnErr := s.loadFromCDN(fetcher, srcPath, opts, cfg)
				if cdnErr != nil && count == 0 {
					errs = append(errs, fmt.Errorf("failed to resolve resolver source %s: %w (CDN fallback also failed: %v)", srcPath, err, cdnErr))
				} else if cdnErr != nil {
					log.Warn("CDN fallback for resolver source %s loaded %d tokens but had errors: %v", srcPath, count, cdnErr)
				}
				continue
			}
			errs = append(errs, fmt.Errorf("failed to resolve resolver source %s: %w", srcPath, err))
			continue
		}

		if err := s.loadTokenFileAndLog(normalizedPath, opts); err != nil {
			errs = append(errs, fmt.Errorf("failed to load resolver source %s: %w", normalizedPath, err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
