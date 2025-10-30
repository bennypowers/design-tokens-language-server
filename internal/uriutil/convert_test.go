package uriutil

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathToURI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		windows  bool // Only run on Windows
		posix    bool // Only run on POSIX
	}{
		// POSIX tests
		{
			name:     "POSIX absolute path",
			input:    "/home/user/project",
			expected: "file:///home/user/project",
			posix:    true,
		},
		{
			name:     "POSIX root path",
			input:    "/",
			expected: "file:///",
			posix:    true,
		},
		// Windows tests
		{
			name:     "Windows absolute path",
			input:    "C:\\project\\file.txt",
			expected: "file:///C:/project/file.txt",
			windows:  true,
		},
		{
			name:     "Windows forward slash path",
			input:    "C:/project/file.txt",
			expected: "file:///C:/project/file.txt",
			windows:  true,
		},
		{
			name:     "Windows UNC path",
			input:    "\\\\server\\share\\file.txt",
			expected: "file://server/share/file.txt",
			windows:  true,
		},
		{
			name:     "Path with spaces (POSIX)",
			input:    "/home/user/my project",
			expected: "file:///home/user/my%20project",
			posix:    true,
		},
		{
			name:     "Path with spaces (Windows)",
			input:    "C:\\Foo Bar\\file.txt",
			expected: "file:///C:/Foo%20Bar/file.txt",
			windows:  true,
		},
		{
			name:     "Path with unicode (POSIX)",
			input:    "/home/user/文件",
			expected: "file:///home/user/%E6%96%87%E4%BB%B6",
			posix:    true,
		},
		// Win32 extended-length prefix tests
		{
			name:     "Windows extended UNC path (backslash)",
			input:    `\\?\UNC\server\share\file.txt`,
			expected: "file://server/share/file.txt",
			windows:  true,
		},
		{
			name:     "Windows extended UNC path (forward slash)",
			input:    `//?/UNC/server/share/file.txt`,
			expected: "file://server/share/file.txt",
			windows:  true,
		},
		{
			name:     "Windows extended UNC path (mixed case)",
			input:    `\\?\unc\server\share\file.txt`,
			expected: "file://server/share/file.txt",
			windows:  true,
		},
		{
			name:     "Windows extended device drive path (backslash)",
			input:    `\\?\C:\project\file.txt`,
			expected: "file:///C:/project/file.txt",
			windows:  true,
		},
		{
			name:     "Windows extended device drive path (forward slash)",
			input:    `//?/C:/project/file.txt`,
			expected: "file:///C:/project/file.txt",
			windows:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip platform-specific tests
			if tt.windows && runtime.GOOS != "windows" {
				t.Skip("Windows-only test")
			}
			if tt.posix && runtime.GOOS == "windows" {
				t.Skip("POSIX-only test")
			}

			got := PathToURI(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestURIToPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		windows  bool // Only run on Windows
		posix    bool // Only run on POSIX
	}{
		// POSIX tests
		{
			name:     "POSIX file URI",
			input:    "file:///home/user/project",
			expected: "/home/user/project",
			posix:    true,
		},
		{
			name:     "POSIX root URI",
			input:    "file:///",
			expected: "/",
			posix:    true,
		},
		{
			name:     "POSIX URI with spaces (percent-encoded)",
			input:    "file:///home/user/my%20project",
			expected: "/home/user/my project", // Percent-decoded
			posix:    true,
		},
		// Windows tests
		{
			name:     "Windows file URI",
			input:    "file:///C:/project/file.txt",
			expected: "C:" + string(filepath.Separator) + "project" + string(filepath.Separator) + "file.txt",
			windows:  true,
		},
		{
			name:     "Windows file URI with backslashes in input",
			input:    `file:///C:\project\file.txt`,
			expected: "C:" + string(filepath.Separator) + "project" + string(filepath.Separator) + "file.txt",
			windows:  true,
		},
		{
			name:     "Windows UNC URI",
			input:    "file://server/share/file.txt",
			expected: "\\\\" + "server" + string(filepath.Separator) + "share" + string(filepath.Separator) + "file.txt",
			windows:  true,
		},
		{
			name:     "Windows URI with spaces (percent-encoded)",
			input:    "file:///C:/Foo%20Bar/file.txt",
			expected: "C:" + string(filepath.Separator) + "Foo Bar" + string(filepath.Separator) + "file.txt",
			windows:  true,
		},
		{
			name:     "URI with unicode (percent-encoded)",
			input:    "file:///home/user/%E6%96%87%E4%BB%B6",
			expected: "/home/user/文件",
			posix:    true,
		},
		// Cross-platform tests (should work on both)
		{
			name:     "file:// with two slashes",
			input:    "file://C:/project",
			expected: "C:" + string(filepath.Separator) + "project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip platform-specific tests
			if tt.windows && runtime.GOOS != "windows" {
				t.Skip("Windows-only test")
			}
			if tt.posix && runtime.GOOS == "windows" {
				t.Skip("POSIX-only test")
			}

			got := URIToPath(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		windows bool
		posix   bool
	}{
		{
			name:  "POSIX home directory",
			path:  "/home/user",
			posix: true,
		},
		{
			name:  "POSIX nested path",
			path:  "/home/user/projects/design-tokens",
			posix: true,
		},
		{
			name:    "Windows C drive",
			path:    "C:\\Users\\user\\project",
			windows: true,
		},
		{
			name:    "Windows D drive",
			path:    "D:\\workspace\\tokens",
			windows: true,
		},
		{
			name:  "POSIX path with spaces",
			path:  "/home/user/my project",
			posix: true,
		},
		{
			name:    "Windows path with spaces",
			path:    "C:\\Foo Bar\\baz",
			windows: true,
		},
		{
			name:  "POSIX path with unicode",
			path:  "/home/user/文件",
			posix: true,
		},
		{
			name:    "Windows UNC path",
			path:    "\\\\server\\share\\file.txt",
			windows: true,
		},
		{
			name:    "Windows extended UNC path",
			path:    `\\?\UNC\server\share\file.txt`,
			windows: true,
		},
		{
			name:    "Windows extended device drive path",
			path:    `\\?\C:\project\file.txt`,
			windows: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip platform-specific tests
			if tt.windows && runtime.GOOS != "windows" {
				t.Skip("Windows-only test")
			}
			if tt.posix && runtime.GOOS == "windows" {
				t.Skip("POSIX-only test")
			}

			// Convert path -> URI -> path
			uri := PathToURI(tt.path)
			roundTrip := URIToPath(uri)

			// Normalize both paths for comparison (resolve . and .., clean up separators)
			expectedClean := filepath.Clean(tt.path)
			gotClean := filepath.Clean(roundTrip)

			assert.Equal(t, expectedClean, gotClean, "Round trip should preserve path")
		})
	}
}

// TestSpecificCases tests the exact cases mentioned in the user's request
func TestSpecificCases(t *testing.T) {
	t.Run("file:///C:/proj -> C:\\proj (Windows)", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			t.Skip("Windows-only test")
		}
		got := URIToPath("file:///C:/proj")
		assert.Equal(t, "C:\\proj", got)
	})

	t.Run("C:\\proj -> file:///C:/proj (Windows)", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			t.Skip("Windows-only test")
		}
		got := PathToURI("C:\\proj")
		assert.Equal(t, "file:///C:/proj", got)
	})

	t.Run("/home/user -> file:///home/user (POSIX)", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("POSIX-only test")
		}
		got := PathToURI("/home/user")
		assert.Equal(t, "file:///home/user", got)
	})

	t.Run("file:///home/user -> /home/user (POSIX)", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("POSIX-only test")
		}
		got := URIToPath("file:///home/user")
		assert.Equal(t, "/home/user", got)
	})
}

// TestUriFallback tests the fallback URI parsing function
func TestUriFallback(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		posix    bool
		windows  bool
	}{
		{
			name:     "file:/// with three slashes",
			input:    "file:///home/user",
			expected: "/home/user",
			posix:    true,
		},
		{
			name:     "file:// with two slashes",
			input:    "file://home/user",
			expected: "home/user",
			posix:    true,
		},
		{
			name:     "Windows drive with file:///",
			input:    "file:///C:/test",
			expected: "C:" + string(filepath.Separator) + "test",
		},
		{
			name:     "Windows drive with file://",
			input:    "file://C:/test",
			expected: "C:" + string(filepath.Separator) + "test",
		},
		{
			name:     "Plain path without file:// prefix",
			input:    "/home/user",
			expected: "/home/user",
			posix:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.windows && runtime.GOOS != "windows" {
				t.Skip("Windows-only test")
			}
			if tt.posix && runtime.GOOS == "windows" {
				t.Skip("POSIX-only test")
			}

			got := uriFallback(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// TestURIToPath_InvalidURIs tests URIToPath with invalid URIs that trigger fallback
func TestURIToPath_InvalidURIs(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		posix   bool
		windows bool
	}{
		{
			name:  "Invalid URI scheme",
			input: "http://example.com/path",
			posix: true,
		},
		{
			name:  "Malformed URI",
			input: "file://:invalid",
		},
		{
			name:  "URI with query params",
			input: "file:///path?query=value",
			posix: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.windows && runtime.GOOS != "windows" {
				t.Skip("Windows-only test")
			}
			if tt.posix && runtime.GOOS == "windows" {
				t.Skip("POSIX-only test")
			}

			// Should not panic, should return some path (even if not meaningful)
			result := URIToPath(tt.input)
			assert.NotEmpty(t, result)
		})
	}
}

// TestPathToURI_RelativePaths tests PathToURI with relative paths
func TestPathToURI_RelativePaths(t *testing.T) {
	// Test that relative paths are converted to absolute
	relPath := "test.txt"
	uri := PathToURI(relPath)

	// Should start with file:///
	assert.True(t, strings.HasPrefix(uri, "file:///") || strings.HasPrefix(uri, "file://"),
		"URI should start with file://")

	// Should not contain the relative path as-is
	// (it should be expanded to absolute)
	assert.NotEqual(t, "file:///test.txt", uri)
}

// TestPathToURI_EmptyPath tests PathToURI with empty path
func TestPathToURI_EmptyPath(t *testing.T) {
	uri := PathToURI("")
	// Should not panic and should return a valid URI
	assert.True(t, strings.HasPrefix(uri, "file://"))
}

// TestURIToPath_EmptyURI tests URIToPath with empty URI
func TestURIToPath_EmptyURI(t *testing.T) {
	path := URIToPath("")
	// Should not panic
	assert.Equal(t, "", path)
}

// TestPathToURI_SpecialCharacters tests various special characters
func TestPathToURI_SpecialCharacters(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX-only test")
	}

	tests := []struct {
		name     string
		input    string
		contains string // What the encoded URI should contain
	}{
		{
			name:     "Path with hash",
			input:    "/home/user/#tag",
			contains: "%23",
		},
		{
			name:     "Path with question mark",
			input:    "/home/user/?query",
			contains: "%3F",
		},
		{
			name:     "Path with percent",
			input:    "/home/user/100%",
			contains: "%25",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri := PathToURI(tt.input)
			assert.Contains(t, uri, tt.contains, "URI should contain percent-encoded character")
		})
	}
}

// TestIsWindowsDriveLetter tests the drive letter detection function
func TestIsWindowsDriveLetter(t *testing.T) {
	tests := []struct {
		name     string
		segment  string
		index    int
		expected bool
	}{
		{
			name:     "Valid drive letter at index 1",
			segment:  "C:",
			index:    1,
			expected: true,
		},
		{
			name:     "Lowercase drive letter",
			segment:  "d:",
			index:    1,
			expected: true,
		},
		{
			name:     "Drive letter at wrong index",
			segment:  "C:",
			index:    0,
			expected: false,
		},
		{
			name:     "Invalid - too long",
			segment:  "C::",
			index:    1,
			expected: false,
		},
		{
			name:     "Invalid - not a letter",
			segment:  "1:",
			index:    1,
			expected: false,
		},
		{
			name:     "Invalid - no colon",
			segment:  "CD",
			index:    1,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isWindowsDriveLetter(tt.segment, tt.index)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// TestHandleExtendedLengthPrefix tests Windows extended-length prefix handling
func TestHandleExtendedLengthPrefix(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		expectedPath      string
		expectedContinue  bool
		expectedExtended  bool
	}{
		{
			name:              "Not an extended path",
			input:             `C:\normal\path`,
			expectedPath:      `C:\normal\path`,
			expectedContinue:  true,
			expectedExtended:  false,
		},
		{
			name:              "Extended UNC path with backslash",
			input:             `\\?\UNC\server\share\file`,
			expectedPath:      `\\server\share\file`,
			expectedContinue:  true,
			expectedExtended:  true,
		},
		{
			name:              "Extended UNC path with forward slash",
			input:             `//?/UNC/server/share/file`,
			expectedPath:      `\\server/share/file`,
			expectedContinue:  true,
			expectedExtended:  true,
		},
		{
			name:              "Extended UNC path lowercase",
			input:             `\\?\unc\server\share`,
			expectedPath:      `\\server\share`,
			expectedContinue:  true,
			expectedExtended:  true,
		},
		{
			name:              "Extended device drive path",
			input:             `\\?\C:\project\file`,
			expectedPath:      "",
			expectedContinue:  false,
			expectedExtended:  false,
		},
		{
			name:              "Extended device drive path forward slash",
			input:             `//?/D:/workspace`,
			expectedPath:      "",
			expectedContinue:  false,
			expectedExtended:  false,
		},
		{
			name:              "Unrecognized extended path",
			input:             `\\?\UNKNOWN\path`,
			expectedPath:      "",
			expectedContinue:  false,
			expectedExtended:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, shouldContinue, wasExtended := handleExtendedLengthPrefix(tt.input)
			assert.Equal(t, tt.expectedPath, path)
			assert.Equal(t, tt.expectedContinue, shouldContinue)
			assert.Equal(t, tt.expectedExtended, wasExtended)
		})
	}
}

// TestHandleWindowsUNCPath tests UNC path handling
func TestHandleWindowsUNCPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test - UNC path handling requires Windows")
	}

	tests := []struct {
		name        string
		input       string
		expectedURI string
		expectedOK  bool
	}{
		{
			name:        "Standard UNC path with backslash",
			input:       `\\server\share\file.txt`,
			expectedURI: "file://server/share/file.txt",
			expectedOK:  true,
		},
		{
			name:        "UNC path with forward slash",
			input:       `//server/share/file.txt`,
			expectedURI: "file://server/share/file.txt",
			expectedOK:  true,
		},
		{
			name:        "Not a UNC path",
			input:       `C:\project\file`,
			expectedURI: "",
			expectedOK:  false,
		},
		{
			name:        "UNC path with spaces",
			input:       `\\server\my share\my file.txt`,
			expectedURI: "file://server/my%20share/my%20file.txt",
			expectedOK:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri, ok := handleWindowsUNCPath(tt.input)
			assert.Equal(t, tt.expectedOK, ok)
			if ok {
				assert.Equal(t, tt.expectedURI, uri)
			}
		})
	}
}

// TestNormalizeAndEncodeSegments tests segment normalization and encoding
func TestNormalizeAndEncodeSegments(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "Empty segments",
			input:    []string{"", "home", "", "user"},
			expected: []string{"", "home", "", "user"},
		},
		{
			name:     "Drive letter at index 1",
			input:    []string{"", "C:", "project"},
			expected: []string{"", "C:", "project"},
		},
		{
			name:     "Lowercase drive letter",
			input:    []string{"", "d:", "workspace"},
			expected: []string{"", "D:", "workspace"},
		},
		{
			name:     "Segments with spaces",
			input:    []string{"", "home", "my folder", "file"},
			expected: []string{"", "home", "my%20folder", "file"},
		},
		{
			name:     "Segments with special chars",
			input:    []string{"", "path", "file#1", "test?"},
			expected: []string{"", "path", "file%231", "test%3F"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeAndEncodeSegments(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}
