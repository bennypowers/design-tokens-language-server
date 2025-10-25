package lsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMatchGlobPattern tests the glob pattern matching using doublestar
func TestMatchGlobPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		// Basic patterns
		{
			name:    "exact match",
			pattern: "tokens.json",
			path:    "tokens.json",
			want:    true,
		},
		{
			name:    "exact no match",
			pattern: "tokens.json",
			path:    "other.json",
			want:    false,
		},

		// Single star patterns
		{
			name:    "star matches filename",
			pattern: "*.json",
			path:    "tokens.json",
			want:    true,
		},
		{
			name:    "star no match different ext",
			pattern: "*.json",
			path:    "tokens.yaml",
			want:    false,
		},
		{
			name:    "star matches prefix",
			pattern: "tokens.*",
			path:    "tokens.json",
			want:    true,
		},

		// Double star patterns - recursive directory matching
		{
			name:    "double star matches any depth - single level",
			pattern: "**/tokens.json",
			path:    "tokens.json",
			want:    true,
		},
		{
			name:    "double star matches any depth - nested",
			pattern: "**/tokens.json",
			path:    "src/config/tokens.json",
			want:    true,
		},
		{
			name:    "double star matches any depth - deep nested",
			pattern: "**/tokens.json",
			path:    "a/b/c/d/e/tokens.json",
			want:    true,
		},
		{
			name:    "double star no match wrong filename",
			pattern: "**/tokens.json",
			path:    "src/config/colors.json",
			want:    false,
		},
		{
			name:    "double star no match - filename not at end",
			pattern: "**/tokens.json",
			path:    "tokens.json.backup",
			want:    false,
		},

		// Complex double star patterns
		{
			name:    "double star with wildcard extension",
			pattern: "**/*.tokens.json",
			path:    "src/design.tokens.json",
			want:    true,
		},
		{
			name:    "double star with wildcard extension nested",
			pattern: "**/*.tokens.json",
			path:    "src/config/design.tokens.json",
			want:    true,
		},
		{
			name:    "double star with wildcard no match",
			pattern: "**/*.tokens.json",
			path:    "src/config/tokens.json",
			want:    false,
		},

		// Prefix patterns
		{
			name:    "prefix double star",
			pattern: "src/**/tokens.json",
			path:    "src/tokens.json",
			want:    true,
		},
		{
			name:    "prefix double star nested",
			pattern: "src/**/tokens.json",
			path:    "src/config/tokens.json",
			want:    true,
		},
		{
			name:    "prefix double star no match wrong prefix",
			pattern: "src/**/tokens.json",
			path:    "lib/tokens.json",
			want:    false,
		},

		// Edge cases that previously caused false positives
		{
			name:    "substring false positive prevention - similar name",
			pattern: "**/tokens.json",
			path:    "mytokens.json",
			want:    false,
		},
		{
			name:    "substring false positive prevention - contains",
			pattern: "**/config.json",
			path:    "my-config.json",
			want:    false,
		},
		{
			name:    "substring false positive prevention - suffix in path",
			pattern: "**/*.json",
			path:    "file.json.backup",
			want:    false,
		},

		// Multiple extensions
		{
			name:    "multiple extensions exact",
			pattern: "**/design-tokens.json",
			path:    "src/design-tokens.json",
			want:    true,
		},
		{
			name:    "multiple extensions with dots",
			pattern: "**/*.tokens.json",
			path:    "colors.tokens.json",
			want:    true,
		},

		// YAML patterns
		{
			name:    "yaml extension",
			pattern: "**/*.yaml",
			path:    "src/tokens.yaml",
			want:    true,
		},
		{
			name:    "yml extension",
			pattern: "**/*.yml",
			path:    "src/tokens.yml",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := matchGlobPattern(tt.pattern, tt.path)
			require.NoError(t, err, "matchGlobPattern should not error for valid patterns")
			assert.Equal(t, tt.want, got, "pattern=%s path=%s", tt.pattern, tt.path)
		})
	}
}

// TestMatchGlobPattern_InvalidPatterns tests error handling for invalid patterns
func TestMatchGlobPattern_InvalidPatterns(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
	}{
		{
			name:    "invalid pattern - unclosed bracket",
			pattern: "tokens[.json",
			path:    "tokens.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := matchGlobPattern(tt.pattern, tt.path)
			assert.Error(t, err, "matchGlobPattern should error for invalid pattern")
		})
	}
}
