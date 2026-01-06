package main

import (
	"os"

	"bennypowers.dev/dtls/internal/log"
	"bennypowers.dev/dtls/lsp"
)

func main() {
	// Create and run the LSP server
	server, err := lsp.NewServer()
	if err != nil {
		log.Error("Failed to create LSP server: %v", err)
		os.Exit(1)
	}

	// Run with stdio transport (for VSCode and other editors)
	if err := server.RunStdio(); err != nil {
		log.Error("Server error: %v", err)
		os.Exit(1)
	}
}
