package main

import (
	"fmt"
	"log"
	"os"

	"bennypowers.dev/dtls/lsp"
)

func main() {
	// Create and run the LSP server
	server, err := lsp.NewServer()
	if err != nil {
		log.Fatalf("Failed to create LSP server: %v", err)
	}

	// Run with stdio transport (for VSCode and other editors)
	if err := server.RunStdio(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
