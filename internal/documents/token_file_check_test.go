package documents

import (
	"os"
	"testing"
)

func TestIsDesignTokensSchema(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		expected bool
	}{
		{
			name:     "valid draft schema",
			fixture:  "testdata/token_file_check/valid-draft-schema.json",
			expected: true,
		},
		{
			name:     "valid 2025 schema",
			fixture:  "testdata/token_file_check/valid-2025-schema.json",
			expected: true,
		},
		{
			name:     "non-token file (no schema)",
			fixture:  "testdata/token_file_check/non-token-file.json",
			expected: false,
		},
		{
			name:     "non-token schema (json-schema.org)",
			fixture:  "testdata/token_file_check/non-token-schema.json",
			expected: false,
		},
		{
			name:     "YAML with valid schema",
			fixture:  "testdata/token_file_check/yaml-with-schema.yaml",
			expected: true,
		},
		{
			name:     "YAML without schema",
			fixture:  "testdata/token_file_check/yaml-without-schema.yaml",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := os.ReadFile(tt.fixture)
			if err != nil {
				t.Fatalf("failed to read fixture %s: %v", tt.fixture, err)
			}

			result := IsDesignTokensSchema(string(content))
			if result != tt.expected {
				t.Errorf("IsDesignTokensSchema() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestIsDesignTokensSchema_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "empty string",
			content:  "",
			expected: false,
		},
		{
			name:     "schema with single quotes",
			content:  "$schema: 'https://www.designtokens.org/schemas/draft.json'",
			expected: true,
		},
		{
			name:     "schema in nested path",
			content:  `{"nested": {"$schema": "https://www.designtokens.org/schemas/draft.json"}}`,
			expected: false, // only top-level schemas should match
		},
		{
			name:     "schema with subdirectory",
			content:  `{
  "$schema": "https://www.designtokens.org/schemas/dtcg/2025.10.json"
}`,
			expected: true,
		},
		{
			name:     "schema without .json extension",
			content:  `{"$schema": "https://www.designtokens.org/schemas/draft"}`,
			expected: false,
		},
		{
			name:     "similar domain but not designtokens.org",
			content:  `{"$schema": "https://www.fakedesigntokens.org/schemas/draft.json"}`,
			expected: false,
		},
		{
			name:     "http instead of https",
			content:  `{"$schema": "http://www.designtokens.org/schemas/draft.json"}`,
			expected: false, // require https
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDesignTokensSchema(tt.content)
			if result != tt.expected {
				t.Errorf("IsDesignTokensSchema() = %v, expected %v for content: %s", result, tt.expected, tt.content)
			}
		})
	}
}
