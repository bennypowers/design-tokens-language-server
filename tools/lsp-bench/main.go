package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"
)

type BenchmarkResults struct {
	Timestamp   time.Time         `json:"timestamp"`
	Server      string            `json:"server"`
	Iterations  int               `json:"iterations"`
	Operations  []OperationResult `json:"operations"`
	MemoryUsage MemoryStats       `json:"memory_usage"`
}

type OperationResult struct {
	Name       string        `json:"name"`
	AvgLatency time.Duration `json:"avg_latency_ns"`
	MinLatency time.Duration `json:"min_latency_ns"`
	MaxLatency time.Duration `json:"max_latency_ns"`
	P50Latency time.Duration `json:"p50_latency_ns"`
	P95Latency time.Duration `json:"p95_latency_ns"`
	P99Latency time.Duration `json:"p99_latency_ns"`
	Iterations int           `json:"iterations"`
}

type MemoryStats struct {
	Idle      uint64 `json:"idle_bytes"`
	UnderLoad uint64 `json:"under_load_bytes"`
}

func main() {
	serverCmd := flag.String("server", "", "Server command to benchmark (e.g., 'deno run --allow-all src/server/server.ts')")
	testdataDir := flag.String("testdata", "./test/testdata", "Path to testdata directory")
	iterations := flag.Int("iterations", 100, "Number of iterations per operation")
	outputFile := flag.String("output", "benchmark-results.json", "Output file for results")
	flag.Parse()

	if *serverCmd == "" {
		fmt.Fprintf(os.Stderr, "Error: --server flag is required\n")
		flag.Usage()
		os.Exit(1)
	}

	fmt.Printf("🔬 LSP Benchmark Harness\n")
	fmt.Printf("Server: %s\n", *serverCmd)
	fmt.Printf("Testdata: %s\n", *testdataDir)
	fmt.Printf("Iterations: %d\n\n", *iterations)

	client, err := NewLSPClient(*serverCmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start LSP server: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	results := BenchmarkResults{
		Timestamp:  time.Now(),
		Server:     *serverCmd,
		Iterations: *iterations,
		Operations: []OperationResult{},
	}

	// Benchmark initialization
	fmt.Printf("⏱️  Benchmarking initialization...\n")
	initResult := benchmarkInitialization(client)
	results.Operations = append(results.Operations, initResult)
	fmt.Printf("   ✓ Avg: %v, Min: %v, Max: %v\n", initResult.AvgLatency, initResult.MinLatency, initResult.MaxLatency)

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)

	// Open test document
	testCSS := `
:root {
  --color-primary: #0000ff;
  --color-secondary: #ff0000;
  --spacing-small: 8px;
}

.button {
  color: var(--color-primary, #000);
  padding: var(--spacing-small);
}
`
	uri := "file:///test/benchmark.css"
	if err := client.DidOpen(uri, "css", testCSS); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open document: %v\n", err)
		os.Exit(1)
	}

	// Benchmark hover
	fmt.Printf("⏱️  Benchmarking hover (%d iterations)...\n", *iterations)
	hoverResult := benchmarkHover(client, uri, *iterations)
	results.Operations = append(results.Operations, hoverResult)
	fmt.Printf("   ✓ Avg: %v, P95: %v, P99: %v\n", hoverResult.AvgLatency, hoverResult.P95Latency, hoverResult.P99Latency)

	// Benchmark completion
	fmt.Printf("⏱️  Benchmarking completion (%d iterations)...\n", *iterations)
	completionResult := benchmarkCompletion(client, uri, *iterations)
	results.Operations = append(results.Operations, completionResult)
	fmt.Printf("   ✓ Avg: %v, P95: %v, P99: %v\n", completionResult.AvgLatency, completionResult.P95Latency, completionResult.P99Latency)

	// Benchmark diagnostics
	fmt.Printf("⏱️  Benchmarking diagnostics (%d iterations)...\n", *iterations)
	diagnosticsResult := benchmarkDiagnostics(client, uri, *iterations)
	results.Operations = append(results.Operations, diagnosticsResult)
	fmt.Printf("   ✓ Avg: %v, P95: %v, P99: %v\n", diagnosticsResult.AvgLatency, diagnosticsResult.P95Latency, diagnosticsResult.P99Latency)

	// Benchmark definition
	fmt.Printf("⏱️  Benchmarking definition (%d iterations)...\n", *iterations)
	definitionResult := benchmarkDefinition(client, uri, *iterations)
	results.Operations = append(results.Operations, definitionResult)
	fmt.Printf("   ✓ Avg: %v, P95: %v, P99: %v\n", definitionResult.AvgLatency, definitionResult.P95Latency, definitionResult.P99Latency)

	// Get memory stats (estimated)
	results.MemoryUsage = getMemoryStats(client)

	// Write results to file
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal results: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outputFile, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write results: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✅ Results saved to %s\n", *outputFile)
	printSummary(results)
}

func printSummary(results BenchmarkResults) {
	fmt.Printf("\n📊 Summary\n")
	fmt.Printf("==================\n")
	for _, op := range results.Operations {
		fmt.Printf("%-20s: avg=%10v  p95=%10v  p99=%10v\n",
			op.Name,
			op.AvgLatency,
			op.P95Latency,
			op.P99Latency,
		)
	}
	fmt.Printf("\nMemory Usage:\n")
	fmt.Printf("  Idle:       %d bytes (%.2f MB)\n", results.MemoryUsage.Idle, float64(results.MemoryUsage.Idle)/1024/1024)
	fmt.Printf("  Under Load: %d bytes (%.2f MB)\n", results.MemoryUsage.UnderLoad, float64(results.MemoryUsage.UnderLoad)/1024/1024)
}
