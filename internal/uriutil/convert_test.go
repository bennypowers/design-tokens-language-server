package uriutil

import (
	"path/filepath"
	"runtime"
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
			input:    "file:///C:/project/file.txt",
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
