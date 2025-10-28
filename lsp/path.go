package lsp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// normalizePath resolves a token file path based on its prefix and workspace root.
// Supports:
//   - Absolute paths: returned as-is
//   - Relative paths (./foo or foo): resolved relative to workspaceRoot
//   - Home directory paths (~/foo): resolved using $HOME
//   - npm: protocol (npm:@scope/package/file): resolved via node_modules
func normalizePath(path, workspaceRoot string) (string, error) {
	// Absolute path - return as-is
	if filepath.IsAbs(path) {
		return path, nil
	}

	// Home directory expansion
	if strings.HasPrefix(path, "~/") {
		home := os.Getenv("HOME")
		if home == "" {
			return "", fmt.Errorf("HOME environment variable not set")
		}
		return filepath.Join(home, path[2:]), nil
	}

	// npm: protocol - resolve via node_modules
	if strings.HasPrefix(path, "npm:") {
		return resolveNpmPath(path[4:], workspaceRoot)
	}

	// Relative path - resolve relative to workspace root
	// Remove leading "./" if present
	cleanPath := strings.TrimPrefix(path, "./")
	return filepath.Join(workspaceRoot, cleanPath), nil
}

// resolveNpmPath resolves an npm: protocol path using Node.js module resolution.
// This includes support for package.json "exports" field and legacy resolution.
// Examples:
//   - npm:@scope/package/file.json -> resolved via exports or direct path
//   - npm:package/file.json -> resolved via exports or direct path
//   - npm:@scope/package -> resolves to package's main entry point
func resolveNpmPath(npmPath, workspaceRoot string) (string, error) {
	if npmPath == "" || strings.HasPrefix(npmPath, "/") {
		return "", fmt.Errorf("invalid npm package path: %q", npmPath)
	}

	// Split the npm path into package name and subpath
	// For scoped packages: @scope/package/file -> ["@scope/package", "file"]
	// For unscoped packages: package/file -> ["package", "file"]
	var packageName, subpath string

	if strings.HasPrefix(npmPath, "@") {
		// Scoped package: @scope/package/optional/file/path
		parts := strings.SplitN(npmPath, "/", 3)
		if len(parts) < 2 {
			return "", fmt.Errorf("invalid npm package path: %q (scoped packages require @scope/package format)", npmPath)
		}
		packageName = parts[0] + "/" + parts[1] // @scope/package
		if len(parts) > 2 {
			subpath = parts[2] // optional/file/path
		}
	} else {
		// Unscoped package: package/optional/file/path
		parts := strings.SplitN(npmPath, "/", 2)
		packageName = parts[0]
		if len(parts) > 1 {
			subpath = parts[1]
		}
	}

	// Find the package directory in node_modules
	packageDir := filepath.Join(workspaceRoot, "node_modules", packageName)
	if _, err := os.Stat(packageDir); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("npm package not found: %s (expected at %s)", packageName, packageDir)
		}
		return "", fmt.Errorf("error accessing npm package %s: %w", packageName, err)
	}

	// If no subpath, resolve to package's main entry point
	if subpath == "" {
		resolved, err := resolvePackageEntry(packageDir, ".")
		if err != nil {
			// Fallback to package directory if no entry point found
			return packageDir, nil
		}
		return resolved, nil
	}

	// Resolve the subpath using package exports or direct file access
	resolved, err := resolvePackageSubpath(packageDir, subpath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve npm:%s: %w", npmPath, err)
	}

	return resolved, nil
}

// PackageJSON represents relevant fields from package.json
type PackageJSON struct {
	Main    string `json:"main,omitempty"`
	Exports any    `json:"exports,omitempty"` // Can be string, map[string]any, or map[string]string
}

// resolvePackageEntry resolves a package entry point (e.g., "." for the main entry)
func resolvePackageEntry(packageDir, entry string) (string, error) {
	pkgJSON, err := readPackageJSON(packageDir)
	if err != nil {
		return "", err
	}

	// Try to resolve via exports field first (modern Node.js)
	if pkgJSON.Exports != nil {
		resolved, err := resolveExports(packageDir, pkgJSON.Exports, entry)
		if err == nil {
			return resolved, nil
		}
		// Fall through to legacy resolution if exports doesn't match
	}

	// Legacy resolution: use "main" field or default to index.js
	if pkgJSON.Main != "" {
		mainPath := filepath.Join(packageDir, pkgJSON.Main)
		if _, err := os.Stat(mainPath); err == nil {
			return mainPath, nil
		}
	}

	// Default to index.js
	indexPath := filepath.Join(packageDir, "index.js")
	if _, err := os.Stat(indexPath); err == nil {
		return indexPath, nil
	}

	return "", fmt.Errorf("no entry point found in package")
}

// resolvePackageSubpath resolves a subpath within a package
func resolvePackageSubpath(packageDir, subpath string) (string, error) {
	pkgJSON, err := readPackageJSON(packageDir)
	if err != nil {
		// If no package.json, try direct file access
		directPath := filepath.Join(packageDir, subpath)
		if _, err := os.Stat(directPath); err == nil {
			return directPath, nil
		}
		return "", fmt.Errorf("file not found: %s", subpath)
	}

	// Try to resolve via exports field first
	if pkgJSON.Exports != nil {
		resolved, err := resolveExports(packageDir, pkgJSON.Exports, "./"+subpath)
		if err == nil {
			return resolved, nil
		}
		// Fall through to direct file access if exports doesn't match
	}

	// Direct file access (legacy behavior)
	directPath := filepath.Join(packageDir, subpath)
	if _, err := os.Stat(directPath); err == nil {
		return directPath, nil
	}

	return "", fmt.Errorf("file not found: %s", subpath)
}

// resolveExports resolves a path using package.json exports field
// Supports both simple string exports and complex export maps
func resolveExports(packageDir string, exports any, requestedPath string) (string, error) {
	switch exp := exports.(type) {
	case string:
		// Simple export: "exports": "./dist/index.js"
		// This applies only to "." entry point
		if requestedPath == "." || requestedPath == "./" {
			resolvedPath := filepath.Join(packageDir, exp)
			if _, err := os.Stat(resolvedPath); err == nil {
				return resolvedPath, nil
			}
		}
		return "", fmt.Errorf("exports does not match %s", requestedPath)

	case map[string]any:
		// Complex export map: "exports": { ".": "./dist/index.js", "./tokens": "./dist/tokens.json" }
		// Try exact match first
		if target, ok := exp[requestedPath]; ok {
			return resolveExportTarget(packageDir, target)
		}

		// Try with "./" prefix if not already present
		if !strings.HasPrefix(requestedPath, "./") {
			prefixedPath := "./" + requestedPath
			if target, ok := exp[prefixedPath]; ok {
				return resolveExportTarget(packageDir, target)
			}
		}

		// Try pattern matching (e.g., "./*": "./dist/*.js")
		for pattern, target := range exp {
			if matched, subst := matchExportPattern(pattern, requestedPath); matched {
				return resolveExportTarget(packageDir, expandPattern(target, subst))
			}
		}

		return "", fmt.Errorf("no export found for %s", requestedPath)

	default:
		return "", fmt.Errorf("unsupported exports type: %T", exports)
	}
}

// resolveExportTarget resolves an export target (can be string or nested object)
func resolveExportTarget(packageDir string, target any) (string, error) {
	switch t := target.(type) {
	case string:
		resolvedPath := filepath.Join(packageDir, t)
		if _, err := os.Stat(resolvedPath); err == nil {
			return resolvedPath, nil
		}
		return "", fmt.Errorf("export target not found: %s", t)

	case map[string]any:
		// Conditional exports: { "import": "./dist/index.mjs", "require": "./dist/index.js" }
		// For design tokens, we typically want the "default" or "require" export
		if defaultTarget, ok := t["default"]; ok {
			return resolveExportTarget(packageDir, defaultTarget)
		}
		if requireTarget, ok := t["require"]; ok {
			return resolveExportTarget(packageDir, requireTarget)
		}
		if importTarget, ok := t["import"]; ok {
			return resolveExportTarget(packageDir, importTarget)
		}
		return "", fmt.Errorf("no suitable conditional export found")

	default:
		return "", fmt.Errorf("unsupported export target type: %T", target)
	}
}

// matchExportPattern checks if a requested path matches an export pattern
// Returns (matched, substitution) where substitution is the captured wildcard
func matchExportPattern(pattern, requestedPath string) (bool, string) {
	// Simple wildcard matching for patterns like "./*"
	if !strings.Contains(pattern, "*") {
		return false, ""
	}

	// Split pattern on "*"
	parts := strings.Split(pattern, "*")
	if len(parts) != 2 {
		return false, "" // Only support single wildcard for now
	}

	prefix, suffix := parts[0], parts[1]

	// Check if requestedPath matches the pattern
	if !strings.HasPrefix(requestedPath, prefix) || !strings.HasSuffix(requestedPath, suffix) {
		return false, ""
	}

	// Extract the wildcarded part
	start := len(prefix)
	end := len(requestedPath) - len(suffix)
	if start > end {
		return false, ""
	}

	substitution := requestedPath[start:end]
	return true, substitution
}

// expandPattern expands a pattern with a substitution (e.g., "./dist/*.js" with "tokens" -> "./dist/tokens.js")
func expandPattern(pattern any, substitution string) any {
	if str, ok := pattern.(string); ok {
		return strings.Replace(str, "*", substitution, 1)
	}
	return pattern
}

// readPackageJSON reads and parses package.json from a directory
func readPackageJSON(packageDir string) (*PackageJSON, error) {
	pkgPath := filepath.Join(packageDir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil, err
	}

	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}

	return &pkg, nil
}
