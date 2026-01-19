package lsp

import (
	"bennypowers.dev/asimonim/config"
	"bennypowers.dev/asimonim/fs"
	"bennypowers.dev/dtls/internal/log"
	"bennypowers.dev/dtls/lsp/types"
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
		log.Warn("Failed to expand file globs: %v", err)
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
func asimonimConfigToServerConfigWithPaths(cfg *config.Config, expandedPaths []string) *types.ServerConfig {
	if cfg == nil {
		return nil
	}

	serverConfig := &types.ServerConfig{
		Prefix:       cfg.Prefix,
		GroupMarkers: cfg.GroupMarkers,
	}

	// Use the expanded paths directly
	for _, path := range expandedPaths {
		serverConfig.TokensFiles = append(serverConfig.TokensFiles, path)
	}

	return serverConfig
}
