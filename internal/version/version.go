package version

import (
	"fmt"
	"runtime/debug"
	"strings"
)

var (
	// Version information, set at build time via ldflags
	Version   = "dev"     // Version string (e.g., "v0.3.0")
	GitCommit = "unknown" // Git commit hash
	GitTag    = "unknown" // Git tag
	BuildTime = "unknown" // Build timestamp
	GitDirty  = ""        // "dirty" if working directory has uncommitted changes
)

// GetVersion returns the version string for the application
func GetVersion() string {
	// If Version was set via ldflags, use it
	if Version != "dev" {
		return Version
	}

	// Fallback: try to get version from build info
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "(devel)" && info.Main.Version != "" {
			return info.Main.Version
		}
	}

	// Final fallback: construct from git information if available
	if GitTag != "unknown" && GitCommit != "unknown" {
		version := GitTag
		if GitCommit != "" {
			// Safe substring that handles short commits
			commitSuffix := GitCommit
			if len(GitCommit) > 7 {
				commitSuffix = GitCommit[:7]
			}
			if !strings.HasSuffix(GitTag, commitSuffix) {
				// Add commit hash if tag doesn't contain it
				version = fmt.Sprintf("%s-%s", GitTag, commitSuffix)
			}
		}
		if GitDirty == "dirty" {
			version += "-dirty"
		}
		return version
	}

	return "dev"
}

// GetFullVersion returns detailed version information
func GetFullVersion() string {
	version := GetVersion()
	if GitCommit != "unknown" {
		return fmt.Sprintf("%s (commit: %s)", version, GitCommit)
	}
	return version
}

// GetBuildInfo returns detailed build information
func GetBuildInfo() map[string]string {
	return map[string]string{
		"version":   GetVersion(),
		"gitCommit": GitCommit,
		"gitTag":    GitTag,
		"buildTime": BuildTime,
		"gitDirty":  GitDirty,
	}
}
