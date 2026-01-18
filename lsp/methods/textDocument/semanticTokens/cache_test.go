package semantictokens_test

import (
	"sync"
	"testing"

	semantictokens "bennypowers.dev/dtls/lsp/methods/textDocument/semanticTokens"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Verify TokenCache implements SemanticTokenCacher interface
var _ types.SemanticTokenCacher = (*semantictokens.TokenCache)(nil)

func TestTokenCache_StoreAndGet(t *testing.T) {
	cache := semantictokens.NewTokenCache()
	uri := "file:///test.json"
	data := []uint32{0, 0, 5, 0, 0, 1, 2, 3, 1, 0}
	version := 1

	resultID := cache.Store(uri, data, version)
	require.NotEmpty(t, resultID)

	entry := cache.Get(resultID)
	require.NotNil(t, entry)
	assert.Equal(t, resultID, entry.ResultID)
	assert.Equal(t, data, entry.Data)
	assert.Equal(t, version, entry.Version)
}

func TestTokenCache_GetByURI(t *testing.T) {
	cache := semantictokens.NewTokenCache()
	uri := "file:///test.json"
	data := []uint32{0, 0, 5, 0, 0}
	version := 1

	resultID := cache.Store(uri, data, version)

	entry := cache.GetByURI(uri)
	require.NotNil(t, entry)
	assert.Equal(t, resultID, entry.ResultID)
	assert.Equal(t, data, entry.Data)
	assert.Equal(t, version, entry.Version)
}

func TestTokenCache_InvalidateByURI(t *testing.T) {
	cache := semantictokens.NewTokenCache()
	uri := "file:///test.json"
	data := []uint32{0, 0, 5, 0, 0}
	version := 1

	resultID := cache.Store(uri, data, version)

	// Verify it exists
	require.NotNil(t, cache.Get(resultID))
	require.NotNil(t, cache.GetByURI(uri))

	// Invalidate
	cache.Invalidate(uri)

	// Verify both lookups return nil
	assert.Nil(t, cache.Get(resultID))
	assert.Nil(t, cache.GetByURI(uri))
}

func TestTokenCache_OverwritesPreviousEntry(t *testing.T) {
	cache := semantictokens.NewTokenCache()
	uri := "file:///test.json"
	data1 := []uint32{0, 0, 5, 0, 0}
	data2 := []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	version1 := 1
	version2 := 2

	resultID1 := cache.Store(uri, data1, version1)
	resultID2 := cache.Store(uri, data2, version2)

	// New resultID should be different
	assert.NotEqual(t, resultID1, resultID2)

	// Old resultID should not work
	assert.Nil(t, cache.Get(resultID1))

	// New resultID should work
	entry := cache.Get(resultID2)
	require.NotNil(t, entry)
	assert.Equal(t, data2, entry.Data)
	assert.Equal(t, version2, entry.Version)

	// GetByURI should return latest
	byURI := cache.GetByURI(uri)
	require.NotNil(t, byURI)
	assert.Equal(t, resultID2, byURI.ResultID)
}

func TestTokenCache_GeneratesUniqueResultIDs(t *testing.T) {
	cache := semantictokens.NewTokenCache()
	seen := make(map[string]bool)

	// Generate 1000 result IDs and verify uniqueness
	for i := range 1000 {
		uri := "file:///test.json"
		resultID := cache.Store(uri, []uint32{0, 0, 5, 0, 0}, i)
		if seen[resultID] {
			t.Fatalf("Duplicate resultID generated: %s", resultID)
		}
		seen[resultID] = true
	}
}

func TestTokenCache_ThreadSafety(t *testing.T) {
	cache := semantictokens.NewTokenCache()
	numGoroutines := 100
	numOps := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // Store, Get, Invalidate goroutines

	// Store goroutines
	for i := range numGoroutines {
		go func(idx int) {
			defer wg.Done()
			for j := range numOps {
				uri := "file:///test.json"
				cache.Store(uri, []uint32{uint32(idx), uint32(j)}, j)
			}
		}(i)
	}

	// Get goroutines
	for range numGoroutines {
		go func() {
			defer wg.Done()
			for range numOps {
				cache.GetByURI("file:///test.json")
			}
		}()
	}

	// Invalidate goroutines
	for range numGoroutines {
		go func() {
			defer wg.Done()
			for range numOps {
				cache.Invalidate("file:///test.json")
			}
		}()
	}

	wg.Wait()
	// If we get here without panic/race, test passes
}

func TestTokenCache_GetNonExistentResultID(t *testing.T) {
	cache := semantictokens.NewTokenCache()
	entry := cache.Get("non-existent-id")
	assert.Nil(t, entry)
}

func TestTokenCache_GetByURINonExistent(t *testing.T) {
	cache := semantictokens.NewTokenCache()
	entry := cache.GetByURI("file:///non-existent.json")
	assert.Nil(t, entry)
}

func TestTokenCache_MultipleDocuments(t *testing.T) {
	cache := semantictokens.NewTokenCache()
	uri1 := "file:///test1.json"
	uri2 := "file:///test2.json"
	data1 := []uint32{1, 2, 3}
	data2 := []uint32{4, 5, 6}

	resultID1 := cache.Store(uri1, data1, 1)
	resultID2 := cache.Store(uri2, data2, 1)

	// Both should exist independently
	entry1 := cache.Get(resultID1)
	entry2 := cache.Get(resultID2)
	require.NotNil(t, entry1)
	require.NotNil(t, entry2)
	assert.Equal(t, data1, entry1.Data)
	assert.Equal(t, data2, entry2.Data)

	// Invalidate one should not affect the other
	cache.Invalidate(uri1)
	assert.Nil(t, cache.Get(resultID1))
	assert.NotNil(t, cache.Get(resultID2))
}
