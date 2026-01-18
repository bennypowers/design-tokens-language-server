package semantictokens

import (
	"fmt"
	"sync"
	"sync/atomic"

	"bennypowers.dev/dtls/lsp/types"
)

// internalEntry wraps types.SemanticTokenCacheEntry with internal fields
type internalEntry struct {
	types.SemanticTokenCacheEntry
	uri string // internal, for reverse lookup invalidation
}

// TokenCache stores semantic tokens by resultID and document URI.
// It implements types.SemanticTokenCacher interface.
type TokenCache struct {
	mu         sync.RWMutex
	byResultID map[string]*internalEntry // resultID -> entry
	byURI      map[string]*internalEntry // uri -> entry
	counter    uint64
}

// NewTokenCache creates a new TokenCache
func NewTokenCache() *TokenCache {
	return &TokenCache{
		byResultID: make(map[string]*internalEntry),
		byURI:      make(map[string]*internalEntry),
	}
}

// Store stores semantic tokens for a document and returns a unique resultID.
// If there was a previous entry for this URI, it is replaced.
func (c *TokenCache) Store(uri string, data []uint32, version int) string {
	// Generate unique resultID using atomic counter
	id := atomic.AddUint64(&c.counter, 1)
	resultID := fmt.Sprintf("st-%d", id)

	// Make a copy of the data to prevent mutations
	dataCopy := make([]uint32, len(data))
	copy(dataCopy, data)

	entry := &internalEntry{
		SemanticTokenCacheEntry: types.SemanticTokenCacheEntry{
			ResultID: resultID,
			Data:     dataCopy,
			Version:  version,
		},
		uri: uri,
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove old entry if exists (by resultID)
	if oldEntry, exists := c.byURI[uri]; exists {
		delete(c.byResultID, oldEntry.ResultID)
	}

	// Store new entry
	c.byResultID[resultID] = entry
	c.byURI[uri] = entry

	return resultID
}

// Get retrieves a cache entry by resultID
func (c *TokenCache) Get(resultID string) *types.SemanticTokenCacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if entry, ok := c.byResultID[resultID]; ok {
		return &entry.SemanticTokenCacheEntry
	}
	return nil
}

// GetByURI retrieves a cache entry by document URI
func (c *TokenCache) GetByURI(uri string) *types.SemanticTokenCacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if entry, ok := c.byURI[uri]; ok {
		return &entry.SemanticTokenCacheEntry
	}
	return nil
}

// Invalidate removes the cache entry for a document URI
func (c *TokenCache) Invalidate(uri string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, exists := c.byURI[uri]; exists {
		delete(c.byResultID, entry.ResultID)
		delete(c.byURI, uri)
	}
}
