package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
)

// CoverageLine represents a single coverage entry
type CoverageLine struct {
	file       string
	blocks     string
	statements string
	count      int
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <unit-coverage.out> <integration-coverage.out> [output.out]\n", os.Args[0])
		os.Exit(1)
	}

	unitFile := os.Args[1]
	integrationFile := os.Args[2]
	outputFile := "coverage-merged.out"
	if len(os.Args) > 3 {
		outputFile = os.Args[3]
	}

	// Parse both coverage files
	coverage := make(map[string]*CoverageLine)

	// Read unit test coverage
	if err := readCoverage(unitFile, coverage); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading unit coverage: %v\n", err)
		os.Exit(1)
	}

	// Read integration test coverage
	if err := readCoverage(integrationFile, coverage); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading integration coverage: %v\n", err)
		os.Exit(1)
	}

	// Write merged coverage
	if err := writeCoverage(outputFile, coverage); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing merged coverage: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Merged coverage written to %s\n", outputFile)
}

func readCoverage(filename string, coverage map[string]*CoverageLine) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip mode line
		if strings.HasPrefix(line, "mode:") {
			continue
		}

		// Parse coverage line: path.go:line.col,line.col statements count
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		fileBlocks := parts[0]   // "file.go:10.2,12.3"
		statements := parts[1]   // "2"
		countStr := parts[2]     // "1"
		
		var count int
		fmt.Sscanf(countStr, "%d", &count)

		// Create key from file and blocks
		key := fileBlocks

		if existing, ok := coverage[key]; ok {
			// Merge: if either has coverage (count > 0), mark as covered
			if count > 0 || existing.count > 0 {
				existing.count = 1
			}
		} else {
			// Convert count to 0 or 1 (set mode)
			if count > 0 {
				count = 1
			}
			coverage[key] = &CoverageLine{
				file:       fileBlocks,
				blocks:     fileBlocks,
				statements: statements,
				count:      count,
			}
		}
	}

	return scanner.Err()
}

func writeCoverage(filename string, coverage map[string]*CoverageLine) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write mode
	fmt.Fprintln(f, "mode: set")

	// Sort keys for consistent output
	keys := make([]string, 0, len(coverage))
	for k := range coverage {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Write coverage lines
	for _, key := range keys {
		line := coverage[key]
		fmt.Fprintf(f, "%s %s %d\n", line.blocks, line.statements, line.count)
	}

	return nil
}
