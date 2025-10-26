package uriutil

import (
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
)

// PathToURI converts a file system path to a file:// URI.
// Handles both Windows and POSIX paths correctly:
//   - C:\proj -> file:///C:/proj
//   - /home/user -> file:///home/user
//   - \\server\share -> file://server/share (UNC)
//   - C:\Foo Bar -> file:///C:/Foo%20Bar (percent-encoded)
//
// The function:
//   - Converts to absolute path using filepath.Abs
//   - Percent-encodes path segments (spaces, unicode, reserved chars)
//   - Correctly handles Windows UNC paths (\\server\share)
//   - Ensures Windows paths have three slashes: file:///C:/
//   - Ensures POSIX paths have three slashes: file:///home/
func PathToURI(path string) string {
	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		// If Abs fails, use the original path
		absPath = path
	}

	// Detect Windows UNC path (\\server\share or //server/share)
	if runtime.GOOS == "windows" && strings.HasPrefix(absPath, `\\`) {
		// UNC path: \\server\share\path -> file://server/share/path
		// Strip the leading \\
		uncPath := strings.TrimPrefix(absPath, `\\`)
		// Convert to forward slashes
		uncPath = filepath.ToSlash(uncPath)
		// Split into segments and percent-encode each
		segments := strings.Split(uncPath, "/")
		for i, seg := range segments {
			segments[i] = url.PathEscape(seg)
		}
		// Reconstruct: file://server/share/path (no extra slashes)
		return "file://" + strings.Join(segments, "/")
	}

	// Convert to forward slashes for URI
	absPath = filepath.ToSlash(absPath)

	// Ensure path starts with / for URI
	// Windows: C:/proj -> /C:/proj
	// POSIX: /home/user already has /
	if !strings.HasPrefix(absPath, "/") {
		absPath = "/" + absPath
	}

	// Split into segments and percent-encode each (skip the leading empty segment from /)
	segments := strings.Split(absPath, "/")
	for i, seg := range segments {
		if seg != "" { // Don't encode empty segments
			segments[i] = url.PathEscape(seg)
		}
	}
	encodedPath := strings.Join(segments, "/")

	// Return file:// URI with three slashes total
	// file:// + /C:/proj = file:///C:/proj
	// file:// + /home/user = file:///home/user
	return "file://" + encodedPath
}

// URIToPath converts a file:// URI to a file system path.
// Handles both Windows and POSIX URIs correctly:
//   - file:///C:/proj -> C:\proj (on Windows) or C:/proj (on POSIX)
//   - file:///home/user -> /home/user
//   - file://server/share -> \\server\share (UNC on Windows)
//   - file:///C:/Foo%20Bar -> C:\Foo Bar (percent-decoded)
//
// The function:
//   - Parses and validates the URI
//   - Percent-decodes path segments
//   - Handles Windows drive letters and UNC paths
//   - Converts forward slashes to OS-specific separators
func URIToPath(uri string) string {
	// Parse the URI to validate and extract components
	parsed, err := url.Parse(uri)
	if err != nil {
		// If parsing fails, fall back to simple string manipulation
		return uriFallback(uri)
	}

	// Verify it's a file:// URI
	if parsed.Scheme != "file" {
		return uriFallback(uri)
	}

	// Extract the path component
	path := parsed.Path

	// Handle UNC paths (file://server/share/path)
	if parsed.Host != "" {
		// UNC path: file://server/share/path -> \\server\share\path (on Windows)
		if runtime.GOOS == "windows" {
			// Decode the host and path
			host, _ := url.PathUnescape(parsed.Host)
			pathDecoded, _ := url.PathUnescape(path)
			// Reconstruct as \\server\share\path
			uncPath := `\\` + host + strings.ReplaceAll(pathDecoded, "/", `\`)
			return uncPath
		}
		// On POSIX, UNC paths are not supported, return as-is
		// This shouldn't normally happen
		return parsed.Host + path
	}

	// Percent-decode the path
	decodedPath, err := url.PathUnescape(path)
	if err != nil {
		// If decoding fails, use the original path
		decodedPath = path
	}

	// On Windows URIs, path might be /C:/proj
	// We need to detect and fix this: /C:/proj -> C:/proj
	if len(decodedPath) >= 3 && decodedPath[0] == '/' && decodedPath[2] == ':' {
		// Remove leading slash from /C:/path
		decodedPath = decodedPath[1:]
	}

	// Convert forward slashes to OS-specific separators
	// On Windows: C:/proj -> C:\proj
	// On POSIX: /home/user stays /home/user
	decodedPath = filepath.FromSlash(decodedPath)

	return decodedPath
}

// uriFallback provides a simple fallback for invalid URIs
func uriFallback(uri string) string {
	// Remove file:// or file:/// prefix (be lenient)
	path := uri
	if strings.HasPrefix(path, "file:///") {
		path = path[7:] // Remove "file://" keeping one slash
	} else if strings.HasPrefix(path, "file://") {
		path = path[7:] // Remove "file://"
	}

	// On Windows URIs, path might be /C:/proj
	if len(path) >= 3 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}

	// Convert forward slashes to OS-specific separators
	path = filepath.FromSlash(path)

	return path
}
