package json

import (
	"encoding/json"
	"fmt"
	"testing"

	"bennypowers.dev/dtls/internal/parser/asimonim"
	"bennypowers.dev/dtls/internal/schema"
)

// BenchmarkParseWithSchemaVersion_Asimonim benchmarks parsing JSON token files using asimonim
func BenchmarkParseWithSchemaVersion_Asimonim(b *testing.B) {
	sizes := []int{100, 500, 1000, 5000}
	schemas := []schema.SchemaVersion{schema.Draft, schema.V2025_10}

	for _, size := range sizes {
		for _, schemaVer := range schemas {
			b.Run(fmt.Sprintf("tokens=%d_schema=%s", size, schemaVer.String()), func(b *testing.B) {
				data := generateLargeTokenFile(size, schemaVer)

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					_, err := asimonim.ParseWithSchemaVersion(data, "test", schemaVer, []string{})
					if err != nil {
						b.Fatalf("Parse failed: %v", err)
					}
				}

				b.ReportMetric(float64(len(data)), "bytes")
			})
		}
	}
}

// BenchmarkParseWithSchemaVersion_AsimonimFast benchmarks asimonim with SkipSort for max performance
func BenchmarkParseWithSchemaVersion_AsimonimFast(b *testing.B) {
	sizes := []int{100, 500, 1000, 5000}
	schemas := []schema.SchemaVersion{schema.Draft, schema.V2025_10}

	for _, size := range sizes {
		for _, schemaVer := range schemas {
			b.Run(fmt.Sprintf("tokens=%d_schema=%s", size, schemaVer.String()), func(b *testing.B) {
				data := generateLargeTokenFile(size, schemaVer)

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					_, err := asimonim.ParseWithOptions(data, "test", schemaVer, []string{}, true)
					if err != nil {
						b.Fatalf("Parse failed: %v", err)
					}
				}

				b.ReportMetric(float64(len(data)), "bytes")
			})
		}
	}
}

// BenchmarkParseWithSchemaVersion_Asimonim_WithAliases benchmarks parsing with alias resolution using asimonim
func BenchmarkParseWithSchemaVersion_Asimonim_WithAliases(b *testing.B) {
	// Create token file with aliases (reuse the same generation logic as original)
	createFileWithAliases := func(numTokens int, schemaVer schema.SchemaVersion) []byte {
		return generateFileWithAliases(numTokens, schemaVer)
	}

	sizes := []int{100, 500, 1000}
	schemas := []schema.SchemaVersion{schema.Draft, schema.V2025_10}

	for _, size := range sizes {
		for _, schemaVer := range schemas {
			b.Run(fmt.Sprintf("tokens=%d_schema=%s", size, schemaVer.String()), func(b *testing.B) {
				data := createFileWithAliases(size, schemaVer)

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					_, err := asimonim.ParseWithSchemaVersion(data, "test", schemaVer, []string{})
					if err != nil {
						b.Fatalf("Parse failed: %v", err)
					}
				}
			})
		}
	}
}

// BenchmarkParseWithSchemaVersion_Asimonim_DeeplyNested benchmarks parsing deeply nested structures using asimonim
func BenchmarkParseWithSchemaVersion_Asimonim_DeeplyNested(b *testing.B) {
	depths := []int{5, 10, 15}

	for _, depth := range depths {
		b.Run(fmt.Sprintf("depth=%d", depth), func(b *testing.B) {
			data := generateDeeplyNestedFile(depth)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := asimonim.ParseWithSchemaVersion(data, "test", schema.V2025_10, []string{})
				if err != nil {
					b.Fatalf("Parse failed: %v", err)
				}
			}
		})
	}
}

// generateFileWithAliases creates a token file with aliases for benchmarking
func generateFileWithAliases(numTokens int, schemaVer schema.SchemaVersion) []byte {
	tokens := make(map[string]interface{})
	schemaURL := ""
	if schemaVer == schema.V2025_10 {
		schemaURL = "https://www.designtokens.org/schemas/2025.10.json"
	} else {
		schemaURL = "https://www.designtokens.org/schemas/draft.json"
	}
	tokens["$schema"] = schemaURL

	colors := make(map[string]interface{})

	// Create base tokens
	for i := 0; i < numTokens/2; i++ {
		if schemaVer == schema.V2025_10 {
			colors[fmt.Sprintf("base-%d", i)] = map[string]interface{}{
				"$type": "color",
				"$value": map[string]interface{}{
					"colorSpace": "srgb",
					"components": []float64{
						float64(i%256) / 255.0,
						0.5,
						0.5,
					},
					"alpha": 1.0,
				},
			}
		} else {
			colors[fmt.Sprintf("base-%d", i)] = map[string]interface{}{
				"$type":  "color",
				"$value": fmt.Sprintf("#%02X8080", i%256),
			}
		}
	}

	// Create alias tokens
	for i := 0; i < numTokens/2; i++ {
		baseIdx := i % (numTokens / 2)
		if schemaVer == schema.V2025_10 {
			colors[fmt.Sprintf("alias-%d", i)] = map[string]interface{}{
				"$type": "color",
				"$ref":  fmt.Sprintf("#/color/base-%d", baseIdx),
			}
		} else {
			colors[fmt.Sprintf("alias-%d", i)] = map[string]interface{}{
				"$type":  "color",
				"$value": fmt.Sprintf("{color.base-%d}", baseIdx),
			}
		}
	}

	tokens["color"] = colors
	data, err := json.Marshal(tokens)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal tokens: %v", err))
	}
	return data
}

// generateDeeplyNestedFile creates a deeply nested token file for benchmarking
func generateDeeplyNestedFile(depth int) []byte {
	tokens := make(map[string]interface{})
	tokens["$schema"] = "https://www.designtokens.org/schemas/2025.10.json"

	current := tokens
	for i := 0; i < depth; i++ {
		nested := make(map[string]interface{})
		current[fmt.Sprintf("level-%d", i)] = nested
		current = nested
	}

	// Add tokens at the deepest level
	for i := 0; i < 100; i++ {
		current[fmt.Sprintf("token-%d", i)] = map[string]interface{}{
			"$type": "color",
			"$value": map[string]interface{}{
				"colorSpace": "srgb",
				"components": []float64{0.5, 0.5, 0.5},
				"alpha":      1.0,
			},
		}
	}

	data, err := json.Marshal(tokens)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal tokens: %v", err))
	}
	return data
}
