package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"bennypowers.dev/asimonim/lsp"
)

func version() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == "bennypowers.dev/asimonim" {
				return dep.Version
			}
		}
	}
	return "dev"
}

func main() {
	server, err := lsp.NewServer(lsp.WithVersion(version()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create LSP server: %v\n", err)
		os.Exit(1)
	}

	if err := server.RunStdio(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
