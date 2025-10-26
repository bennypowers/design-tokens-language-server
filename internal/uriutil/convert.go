package uriutil

import (
	"path/filepath"
	"strings"
)

// PathToURI converts a file system path to a file:// URI.
// Handles both Windows and POSIX paths correctly:
//   - C:\proj -> file:///C:/proj
//   - /home/user -> file:///home/user
//
// The function:
//   - Converts to absolute path using filepath.Abs
//   - Normalizes path separators to forward slashes
//   - Ensures Windows paths have three slashes: file:///C:/
//   - Ensures POSIX paths have three slashes: file:///home/
func PathToURI(path string) string {
	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		// If Abs fails, use the original path
		absPath = path
	}

	// Convert to forward slashes for URI
	absPath = filepath.ToSlash(absPath)

	// Ensure path starts with / for URI
	// Windows: C:/proj -> /C:/proj
	// POSIX: /home/user already has /
	if !strings.HasPrefix(absPath, "/") {
		absPath = "/" + absPath
	}

	// Return file:// URI with three slashes total
	// file:// + /C:/proj = file:///C:/proj
	// file:// + /home/user = file:///home/user
	return "file://" + absPath
}

// URIToPath converts a file:// URI to a file system path.
// Handles both Windows and POSIX URIs correctly:
//   - file:///C:/proj -> C:\proj (on Windows) or C:/proj (on POSIX)
//   - file:///home/user -> /home/user
//
// The function:
//   - Strips file:// or file:/// prefix
//   - Handles leading /C:/ -> C:/ for Windows drive letters
//   - Converts forward slashes to OS-specific separators
func URIToPath(uri string) string {
	// Remove file:// or file:/// prefix (be lenient)
	path := uri
	if strings.HasPrefix(path, "file:///") {
		path = path[7:] // Remove "file://" keeping one slash
	} else if strings.HasPrefix(path, "file://") {
		path = path[7:] // Remove "file://"
	}

	// On Windows URIs, path might be /C:/proj
	// We need to detect and fix this: /C:/proj -> C:/proj
	if len(path) >= 3 && path[0] == '/' && path[2] == ':' {
		// Remove leading slash from /C:/path
		path = path[1:]
	}

	// Convert forward slashes to OS-specific separators
	// On Windows: C:/proj -> C:\proj
	// On POSIX: /home/user stays /home/user
	path = filepath.FromSlash(path)

	return path
}
