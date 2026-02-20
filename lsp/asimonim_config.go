package lsp

import (
	"path/filepath"

	"bennypowers.dev/asimonim/config"
	"bennypowers.dev/asimonim/fs"
	"bennypowers.dev/dtls/internal/log"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/bmatcuk/doublestar/v4"
)

// ReadAsimonimConfig reads design tokens configuration from .config/design-tokens.{yaml,json}.
// Returns nil if no config file exists (not an error).
func ReadAsimonimConfig(rootPath string) (*types.ServerConfig, error) {
	if rootPath == "" {
		return nil, nil
	}

	filesystem := fs.NewOSFileSystem()
	cfg, err := config.Load(filesystem, rootPath)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, nil // No config file found
	}

	// Expand glob patterns in file paths
	expandedPaths, err := cfg.ExpandFiles(filesystem, rootPath)
	if err != nil {
		log.Warn("Failed to expand file globs (falling back to literal paths): %v", err)
		// Fall back to raw paths if expansion fails
		return AsimonimConfigToServerConfig(cfg), nil
	}

	return asimonimConfigToServerConfigWithPaths(cfg, expandedPaths), nil
}

// AsimonimConfigToServerConfig converts an asimonim Config to a dtls ServerConfig.
func AsimonimConfigToServerConfig(cfg *config.Config) *types.ServerConfig {
	if cfg == nil {
		return nil
	}

	serverConfig := &types.ServerConfig{
		Prefix:       cfg.Prefix,
		GroupMarkers: cfg.GroupMarkers,
		CDN:          cfg.CDN,
	}

	// Convert asimonim FileSpecs to dtls TokensFiles
	for _, spec := range cfg.Files {
		if spec.Prefix != "" || len(spec.GroupMarkers) > 0 {
			// Use object form for files with overrides
			serverConfig.TokensFiles = append(serverConfig.TokensFiles, types.TokenFileSpec{
				Path:         spec.Path,
				Prefix:       spec.Prefix,
				GroupMarkers: spec.GroupMarkers,
			})
		} else {
			// Use simple string form
			serverConfig.TokensFiles = append(serverConfig.TokensFiles, spec.Path)
		}
	}

	return serverConfig
}

// asimonimConfigToServerConfigWithPaths converts an asimonim Config to a dtls ServerConfig
// using pre-expanded file paths instead of the raw FileSpec paths.
// It preserves per-file overrides (Prefix, GroupMarkers) by matching expanded paths
// back to their source FileSpec patterns.
func asimonimConfigToServerConfigWithPaths(cfg *config.Config, expandedPaths []string) *types.ServerConfig {
	if cfg == nil {
		return nil
	}

	serverConfig := &types.ServerConfig{
		Prefix:       cfg.Prefix,
		GroupMarkers: cfg.GroupMarkers,
		CDN:          cfg.CDN,
	}

	// For each expanded path, find the matching FileSpec to preserve per-file overrides
	for _, path := range expandedPaths {
		spec := findMatchingFileSpec(cfg.Files, path)
		if spec != nil && (spec.Prefix != "" || len(spec.GroupMarkers) > 0) {
			// Use object form for files with overrides
			serverConfig.TokensFiles = append(serverConfig.TokensFiles, types.TokenFileSpec{
				Path:         path,
				Prefix:       spec.Prefix,
				GroupMarkers: spec.GroupMarkers,
			})
		} else {
			// Use simple string form
			serverConfig.TokensFiles = append(serverConfig.TokensFiles, path)
		}
	}

	return serverConfig
}

// findMatchingFileSpec finds the FileSpec whose pattern matches the given expanded path.
// Returns nil if no matching FileSpec is found.
func findMatchingFileSpec(files []config.FileSpec, expandedPath string) *config.FileSpec {
	for i := range files {
		spec := &files[i]
		// Check for exact match first
		if spec.Path == expandedPath {
			return spec
		}
		// Check if the pattern matches (for glob patterns)
		matched, err := doublestar.PathMatch(spec.Path, expandedPath)
		if err == nil && matched {
			return spec
		}
		// For relative patterns against absolute expanded paths, try matching
		// the relative suffix of the expanded path against the pattern.
		// E.g., pattern "tokens/*.yaml" should match "/tmp/project/tokens/color.yaml"
		if !filepath.IsAbs(spec.Path) && filepath.IsAbs(expandedPath) {
			// Try to match against progressively shorter suffixes of the expanded path
			parts := splitPath(expandedPath)
			for j := 1; j < len(parts); j++ {
				suffix := filepath.Join(parts[j:]...)
				matched, err = doublestar.PathMatch(spec.Path, suffix)
				if err == nil && matched {
					return spec
				}
			}
		}
	}
	return nil
}

// splitPath splits a path into its components.
func splitPath(path string) []string {
	var parts []string
	for path != "" && path != "/" && path != "." {
		dir, file := filepath.Split(path)
		if file != "" {
			parts = append([]string{file}, parts...)
		}
		// Remove trailing separator from dir
		path = filepath.Clean(dir)
		if path == "/" || path == "." {
			break
		}
	}
	return parts
}
