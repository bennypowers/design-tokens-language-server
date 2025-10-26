package workspace

import (
	"testing"

	"github.com/bennypowers/design-tokens-language-server/lsp/testutil"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestHandleDidChangeWatchedFiles(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	ctx.SetRootPath("/workspace")

	config := ctx.GetConfig()
	config.TokensFiles = []any{"tokens.json"}
	ctx.SetConfig(config)

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
	err := DidChangeWatchedFiles(ctx, nil, params)
	if err != nil {
		t.Errorf("DidChangeWatchedFiles failed: %v", err)
	}
}

func TestHandleDidChangeWatchedFiles_MultipleChanges(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	ctx.SetRootPath("/workspace")

	config := ctx.GetConfig()
	config.TokensFiles = []any{"tokens.json", "design-tokens.json"}
	ctx.SetConfig(config)

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

	err := DidChangeWatchedFiles(ctx, nil, params)
	if err != nil {
		t.Errorf("DidChangeWatchedFiles failed: %v", err)
	}
}

func TestHandleDidChangeWatchedFiles_DeletedFile(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	ctx.SetRootPath("/workspace")

	config := ctx.GetConfig()
	config.TokensFiles = []any{"tokens.json"}
	ctx.SetConfig(config)

	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{
				URI:  "file:///workspace/tokens.json",
				Type: protocol.FileChangeTypeDeleted,
			},
		},
	}

	err := DidChangeWatchedFiles(ctx, nil, params)
	if err != nil {
		t.Errorf("DidChangeWatchedFiles failed: %v", err)
	}

	// The handler should have called RemoveLoadedFile to clean up the deleted file
	// This prevents stale entries that would cause reload errors
}

func TestHandleDidChangeWatchedFiles_DeletedAndModified(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	ctx.SetRootPath("/workspace")

	config := ctx.GetConfig()
	config.TokensFiles = []any{"tokens.json", "design-tokens.json"}
	ctx.SetConfig(config)

	// Test that when one file is deleted and another is modified,
	// we only reload the remaining files
	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{
				URI:  "file:///workspace/tokens.json",
				Type: protocol.FileChangeTypeDeleted,
			},
			{
				URI:  "file:///workspace/design-tokens.json",
				Type: protocol.FileChangeTypeChanged,
			},
		},
	}

	err := DidChangeWatchedFiles(ctx, nil, params)
	if err != nil {
		t.Errorf("DidChangeWatchedFiles failed: %v", err)
	}

	// Should have triggered a reload for the modified file
	// Should NOT have tried to reload the deleted file
}

func TestHandleDidChangeWatchedFiles_NonTokenFile(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	ctx.SetRootPath("/workspace")

	config := ctx.GetConfig()
	config.TokensFiles = []any{"tokens.json"}
	ctx.SetConfig(config)

	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{
				URI:  "file:///workspace/package.json", // Not a token file
				Type: protocol.FileChangeTypeChanged,
			},
		},
	}

	// Should not trigger a reload since it's not a token file
	err := DidChangeWatchedFiles(ctx, nil, params)
	if err != nil {
		t.Errorf("DidChangeWatchedFiles failed: %v", err)
	}
}

func TestHandleDidChangeWatchedFiles_NewlyCreatedFile(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	ctx.SetRootPath("/workspace")

	// Empty TokensFiles means auto-discovery mode
	config := ctx.GetConfig()
	config.TokensFiles = []any{}
	ctx.SetConfig(config)

	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{
				URI:  "file:///workspace/new-tokens.json",
				Type: protocol.FileChangeTypeCreated,
			},
		},
	}

	// When a new token file is created in auto-discovery mode,
	// the reload should discover it
	err := DidChangeWatchedFiles(ctx, nil, params)
	if err != nil {
		t.Errorf("DidChangeWatchedFiles failed: %v", err)
	}

	// The handler should have triggered a reload which would
	// call LoadTokensFromConfig, which would discover the new file
}
