package lsp

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestIsTokenFile(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		configFiles    []any
		rootPath       string
		expectedResult bool
	}{
		{
			name:           "Explicit token file - JSON",
			path:           "/workspace/tokens.json",
			rootPath:       "/workspace",
			configFiles:    []any{"tokens.json"},
			expectedResult: true,
		},
		{
			name:           "Explicit token file - absolute path",
			path:           "/workspace/design-system/tokens.json",
			rootPath:       "/workspace",
			configFiles:    []any{"/workspace/design-system/tokens.json"},
			expectedResult: true,
		},
		{
			name:     "Explicit token file - relative path",
			path:     "/workspace/design-system/tokens.json",
			rootPath: "/workspace",
			configFiles: []any{
				map[string]any{
					"path": "design-system/tokens.json",
				},
			},
			expectedResult: true,
		},
		{
			name:           "Non-token file",
			path:           "/workspace/package.json",
			rootPath:       "/workspace",
			configFiles:    []any{"tokens.json"},
			expectedResult: false,
		},
		{
			name:           "Auto-discover - tokens.json",
			path:           "/workspace/tokens.json",
			rootPath:       "/workspace",
			configFiles:    []any{}, // Empty = auto-discover
			expectedResult: true,
		},
		{
			name:           "Auto-discover - design-tokens.json",
			path:           "/workspace/design-tokens.json",
			rootPath:       "/workspace",
			configFiles:    []any{},
			expectedResult: true,
		},
		{
			name:           "Auto-discover - custom.tokens.json",
			path:           "/workspace/custom.tokens.json",
			rootPath:       "/workspace",
			configFiles:    []any{},
			expectedResult: true,
		},
		{
			name:           "Auto-discover - YAML",
			path:           "/workspace/tokens.yaml",
			rootPath:       "/workspace",
			configFiles:    []any{},
			expectedResult: true,
		},
		{
			name:           "Auto-discover - non-token file",
			path:           "/workspace/package.json",
			rootPath:       "/workspace",
			configFiles:    []any{},
			expectedResult: false,
		},
		{
			name:           "Non-JSON/YAML file",
			path:           "/workspace/tokens.txt",
			rootPath:       "/workspace",
			configFiles:    []any{},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewServer()
			if err != nil {
				t.Fatalf("Failed to create server: %v", err)
			}

			s.rootPath = tt.rootPath
			s.config.TokensFiles = tt.configFiles

			result := s.isTokenFile(tt.path)
			if result != tt.expectedResult {
				t.Errorf("Expected %v, got %v", tt.expectedResult, result)
			}
		})
	}
}

func TestHandleDidChangeWatchedFiles(t *testing.T) {
	s, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Set up a simple configuration
	s.rootPath = "/workspace"
	s.config.TokensFiles = []any{"tokens.json"}

	// Create a change event
	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{
				URI:  "file:///workspace/tokens.json",
				Type: protocol.FileChangeTypeChanged,
			},
		},
	}

	// Handle the change
	err = s.handleDidChangeWatchedFiles(nil, params)
	if err != nil {
		t.Errorf("handleDidChangeWatchedFiles failed: %v", err)
	}
}

func TestHandleDidChangeWatchedFiles_MultipleChanges(t *testing.T) {
	s, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	s.rootPath = "/workspace"
	s.config.TokensFiles = []any{"tokens.json", "design-tokens.json"}

	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{
				URI:  "file:///workspace/tokens.json",
				Type: protocol.FileChangeTypeChanged,
			},
			{
				URI:  "file:///workspace/design-tokens.json",
				Type: protocol.FileChangeTypeChanged,
			},
			{
				URI:  "file:///workspace/package.json", // Not a token file
				Type: protocol.FileChangeTypeChanged,
			},
		},
	}

	err = s.handleDidChangeWatchedFiles(nil, params)
	if err != nil {
		t.Errorf("handleDidChangeWatchedFiles failed: %v", err)
	}
}

func TestHandleDidChangeWatchedFiles_DeletedFile(t *testing.T) {
	s, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	s.rootPath = "/workspace"
	s.config.TokensFiles = []any{"tokens.json"}

	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{
				URI:  "file:///workspace/tokens.json",
				Type: protocol.FileChangeTypeDeleted,
			},
		},
	}

	err = s.handleDidChangeWatchedFiles(nil, params)
	if err != nil {
		t.Errorf("handleDidChangeWatchedFiles failed: %v", err)
	}
}

func TestHandleDidChangeWatchedFiles_NonTokenFile(t *testing.T) {
	s, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	s.rootPath = "/workspace"
	s.config.TokensFiles = []any{"tokens.json"}

	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{
				URI:  "file:///workspace/package.json", // Not a token file
				Type: protocol.FileChangeTypeChanged,
			},
		},
	}

	// Should not trigger a reload since it's not a token file
	err = s.handleDidChangeWatchedFiles(nil, params)
	if err != nil {
		t.Errorf("handleDidChangeWatchedFiles failed: %v", err)
	}
}
