package tokens

import (
	"fmt"
	"testing"

	"bennypowers.dev/dtls/internal/schema"
)

// BenchmarkManager_Add benchmarks adding tokens to the manager
func BenchmarkManager_Add(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("tokens=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				m := NewManager()
				for j := 0; j < size; j++ {
					token := &Token{
						Name:          fmt.Sprintf("color-token-%d", j),
						Value:         "#FF6B35",
						Type:          "color",
						FilePath:      "/test/tokens.json",
						SchemaVersion: schema.V2025_10,
					}
					_ = m.Add(token)
				}
			}
		})
	}
}

// BenchmarkManager_Get benchmarks token retrieval
func BenchmarkManager_Get(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("tokens=%d", size), func(b *testing.B) {
			// Setup: create manager with tokens
			m := NewManager()
			for j := 0; j < size; j++ {
				token := &Token{
					Name:          fmt.Sprintf("color-token-%d", j),
					Value:         "#FF6B35",
					Type:          "color",
					FilePath:      "/test/tokens.json",
					SchemaVersion: schema.V2025_10,
				}
				_ = m.Add(token)
			}

			b.ReportAllocs()
			b.ResetTimer()

			// Benchmark: lookup tokens
			for i := 0; i < b.N; i++ {
				// Test lookup at beginning, middle, and end
				_ = m.Get(fmt.Sprintf("color-token-%d", 0))
				_ = m.Get(fmt.Sprintf("color-token-%d", size/2))
				_ = m.Get(fmt.Sprintf("color-token-%d", size-1))
			}
		})
	}
}

// BenchmarkManager_GetWithPrefix benchmarks CSS variable name lookup
func BenchmarkManager_GetWithPrefix(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("tokens=%d", size), func(b *testing.B) {
			// Setup
			m := NewManager()
			for j := 0; j < size; j++ {
				token := &Token{
					Name:          fmt.Sprintf("color-token-%d", j),
					Value:         "#FF6B35",
					Type:          "color",
					FilePath:      "/test/tokens.json",
					Prefix:        "ds",
					SchemaVersion: schema.V2025_10,
				}
				_ = m.Add(token)
			}

			b.ReportAllocs()
			b.ResetTimer()

			// Benchmark CSS variable lookup (requires iteration)
			for i := 0; i < b.N; i++ {
				_ = m.Get(fmt.Sprintf("--ds-color-token-%d", size/2))
			}
		})
	}
}

// BenchmarkManager_ListAll benchmarks retrieving all tokens
func BenchmarkManager_ListAll(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("tokens=%d", size), func(b *testing.B) {
			// Setup
			m := NewManager()
			for j := 0; j < size; j++ {
				token := &Token{
					Name:          fmt.Sprintf("color-token-%d", j),
					Value:         "#FF6B35",
					Type:          "color",
					FilePath:      "/test/tokens.json",
					SchemaVersion: schema.V2025_10,
				}
				_ = m.Add(token)
			}

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = m.GetAll()
			}
		})
	}
}

// BenchmarkManager_MultiFile benchmarks multi-file token management
func BenchmarkManager_MultiFile(b *testing.B) {
	numFiles := []int{5, 10, 20}
	tokensPerFile := 500

	for _, files := range numFiles {
		b.Run(fmt.Sprintf("files=%d_tokens=%d", files, files*tokensPerFile), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				m := NewManager()

				// Add tokens from multiple files
				for fileIdx := 0; fileIdx < files; fileIdx++ {
					filePath := fmt.Sprintf("/test/tokens-%d.json", fileIdx)
					schemaVer := schema.Draft
					if fileIdx%2 == 0 {
						schemaVer = schema.V2025_10
					}

					for tokenIdx := 0; tokenIdx < tokensPerFile; tokenIdx++ {
						token := &Token{
							Name:          fmt.Sprintf("color-token-%d", tokenIdx),
							Value:         "#FF6B35",
							Type:          "color",
							FilePath:      filePath,
							SchemaVersion: schemaVer,
						}
						_ = m.Add(token)
					}
				}

				// Verify we have all tokens
				if len(m.tokens) != files*tokensPerFile {
					b.Fatalf("Expected %d tokens, got %d", files*tokensPerFile, len(m.tokens))
				}
			}
		})
	}
}

// BenchmarkManager_ConcurrentAccess benchmarks concurrent read/write operations
func BenchmarkManager_ConcurrentAccess(b *testing.B) {
	m := NewManager()

	// Pre-populate with some tokens
	for i := 0; i < 1000; i++ {
		token := &Token{
			Name:          fmt.Sprintf("color-token-%d", i),
			Value:         "#FF6B35",
			Type:          "color",
			FilePath:      "/test/tokens.json",
			SchemaVersion: schema.V2025_10,
		}
		_ = m.Add(token)
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// Mix of reads and writes (80% reads, 20% writes)
			if i%5 == 0 {
				token := &Token{
					Name:          fmt.Sprintf("color-new-%d", i),
					Value:         "#00FF00",
					Type:          "color",
					FilePath:      "/test/tokens.json",
					SchemaVersion: schema.V2025_10,
				}
				_ = m.Add(token)
			} else {
				_ = m.Get(fmt.Sprintf("color-token-%d", i%1000))
			}
			i++
		}
	})
}
