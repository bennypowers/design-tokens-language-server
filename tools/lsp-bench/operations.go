package main

import (
	"sort"
	"time"
)

func benchmarkInitialization(client *LSPClient) OperationResult {
	// Only run once
	start := time.Now()
	client.Initialize()
	elapsed := time.Since(start)

	return OperationResult{
		Name:       "initialize",
		AvgLatency: elapsed,
		MinLatency: elapsed,
		MaxLatency: elapsed,
		P50Latency: elapsed,
		P95Latency: elapsed,
		P99Latency: elapsed,
		Iterations: 1,
	}
}

func benchmarkHover(client *LSPClient, uri string, iterations int) OperationResult {
	latencies := make([]time.Duration, iterations)

	for i := 0; i < iterations; i++ {
		start := time.Now()
		// Hover over --color-primary on line 2, character 4
		client.Hover(uri, 2, 4)
		latencies[i] = time.Since(start)
	}

	return computeStats("hover", latencies)
}

func benchmarkCompletion(client *LSPClient, uri string, iterations int) OperationResult {
	latencies := make([]time.Duration, iterations)

	for i := 0; i < iterations; i++ {
		start := time.Now()
		// Completion at var(-- position on line 8
		client.Completion(uri, 8, 16)
		latencies[i] = time.Since(start)
	}

	return computeStats("completion", latencies)
}

func benchmarkDiagnostics(client *LSPClient, uri string, iterations int) OperationResult {
	latencies := make([]time.Duration, iterations)

	for i := 0; i < iterations; i++ {
		start := time.Now()
		client.Diagnostic(uri)
		latencies[i] = time.Since(start)
	}

	return computeStats("diagnostics", latencies)
}

func benchmarkDefinition(client *LSPClient, uri string, iterations int) OperationResult {
	latencies := make([]time.Duration, iterations)

	for i := 0; i < iterations; i++ {
		start := time.Now()
		// Definition for --color-primary reference on line 8
		client.Definition(uri, 8, 16)
		latencies[i] = time.Since(start)
	}

	return computeStats("definition", latencies)
}

func computeStats(name string, latencies []time.Duration) OperationResult {
	if len(latencies) == 0 {
		return OperationResult{Name: name}
	}

	// Sort for percentile calculations
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// Calculate average
	var sum time.Duration
	for _, d := range latencies {
		sum += d
	}
	avg := sum / time.Duration(len(latencies))

	// Calculate percentiles
	p50 := sorted[len(sorted)*50/100]
	p95 := sorted[len(sorted)*95/100]
	p99 := sorted[len(sorted)*99/100]

	return OperationResult{
		Name:       name,
		AvgLatency: avg,
		MinLatency: sorted[0],
		MaxLatency: sorted[len(sorted)-1],
		P50Latency: p50,
		P95Latency: p95,
		P99Latency: p99,
		Iterations: len(latencies),
	}
}

func getMemoryStats(client *LSPClient) MemoryStats {
	idle, _ := client.GetProcessMemory()

	// Do some operations to load the server
	for i := 0; i < 10; i++ {
		client.Hover("file:///test/benchmark.css", 2, 4)
	}

	underLoad, _ := client.GetProcessMemory()

	return MemoryStats{
		Idle:      idle,
		UnderLoad: underLoad,
	}
}
