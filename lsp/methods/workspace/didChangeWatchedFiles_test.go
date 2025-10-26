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
