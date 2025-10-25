package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <coverage1.out> <coverage2.out> [coverage3.out...]\n", os.Args[0])
		os.Exit(1)
	}

	// Map of file:line -> count
	coverage := make(map[string]int)
	mode := "set"

	// Read all coverage files
	for _, file := range os.Args[1:] {
		f, err := os.Open(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening %s: %v\n", file, err)
			continue
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "mode:") {
				// Extract mode from first file
				parts := strings.Fields(line)
				if len(parts) == 2 && mode == "set" {
					mode = parts[1]
				}
				continue
			}

			// Parse coverage line: path.go:line.col,line.col statements count
			parts := strings.Fields(line)
			if len(parts) < 3 {
				continue
			}

			key := strings.Join(parts[:2], " ")
			count := 0
			fmt.Sscanf(parts[2], "%d", &count)

			// Merge: if any file has coverage, mark as covered
			if count > 0 {
				coverage[key] = 1
			} else if _, exists := coverage[key]; !exists {
				coverage[key] = 0
			}
		}
	}

	// Output merged coverage
	fmt.Printf("mode: %s\n", mode)
	for key, count := range coverage {
		fmt.Printf("%s %d\n", key, count)
	}
}
