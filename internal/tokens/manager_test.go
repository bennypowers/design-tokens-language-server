package tokens_test

import (
	"fmt"
	"testing"

	"bennypowers.dev/dtls/internal/schema"
	"bennypowers.dev/dtls/internal/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTokenManagerAddGet tests adding and retrieving tokens
func TestTokenManagerAddGet(t *testing.T) {
	manager := tokens.NewManager()

	// Initially empty
	token := manager.Get("color-primary")
	assert.Nil(t, token, "Token should not exist initially")

	// Add token
	err := manager.Add(&tokens.Token{
		Name:  "color-primary",
		Value: "#0000ff",
		Type:  "color",
	})
	require.NoError(t, err)

	// Should now exist
	token = manager.Get("color-primary")
	require.NotNil(t, token, "Token should exist after adding")
	assert.Equal(t, "color-primary", token.Name)
	assert.Equal(t, "#0000ff", token.Value)
	assert.Equal(t, "color", token.Type)
}

// TestTokenManagerGetByCSSVariable tests getting tokens by CSS variable name
func TestTokenManagerGetByCSSVariable(t *testing.T) {
	manager := tokens.NewManager()

	err := manager.Add(&tokens.Token{
		Name:  "color-primary",
		Value: "#0000ff",
	})
	require.NoError(t, err)

	// Get by CSS variable name (with --)
	token := manager.Get("--color-primary")
	require.NotNil(t, token)
	assert.Equal(t, "color-primary", token.Name)

	// Get by name (without --)
	token = manager.Get("color-primary")
	require.NotNil(t, token)
	assert.Equal(t, "color-primary", token.Name)
}

// TestTokenManagerGetWithPrefix tests tokens with CSS variable prefixes
func TestTokenManagerGetWithPrefix(t *testing.T) {
	manager := tokens.NewManager()

	err := manager.Add(&tokens.Token{
		Name:   "color-primary",
		Value:  "#0000ff",
		Prefix: "my-design-system",
	})
	require.NoError(t, err)

	// Get by full CSS variable name with prefix
	token := manager.Get("--my-design-system-color-primary")
	require.NotNil(t, token)
	assert.Equal(t, "color-primary", token.Name)
	assert.Equal(t, "#0000ff", token.Value)

	// CSS variable name should include prefix
	assert.Equal(t, "--my-design-system-color-primary", token.CSSVariableName())
}

// TestTokenManagerGetAll tests retrieving all tokens
func TestTokenManagerGetAll(t *testing.T) {
	manager := tokens.NewManager()

	// Initially empty
	allTokens := manager.GetAll()
	assert.Empty(t, allTokens)

	// Add multiple tokens
	_ = manager.Add(&tokens.Token{Name: "color-primary", Value: "#0000ff"})
	_ = manager.Add(&tokens.Token{Name: "color-secondary", Value: "#ff0000"})
	_ = manager.Add(&tokens.Token{Name: "spacing-small", Value: "8px"})

	// Should return all tokens
	allTokens = manager.GetAll()
	assert.Len(t, allTokens, 3)

	// Verify all tokens are present
	names := make(map[string]bool)
	for _, token := range allTokens {
		names[token.Name] = true
	}
	assert.True(t, names["color-primary"])
	assert.True(t, names["color-secondary"])
	assert.True(t, names["spacing-small"])
}

// TestTokenManagerRemove tests removing tokens
func TestTokenManagerRemove(t *testing.T) {
	manager := tokens.NewManager()

	_ = manager.Add(&tokens.Token{Name: "color-primary", Value: "#0000ff"})

	// Token should exist
	token := manager.Get("color-primary")
	require.NotNil(t, token)

	// Remove token
	err := manager.Remove("color-primary")
	require.NoError(t, err)

	// Token should no longer exist
	token = manager.Get("color-primary")
	assert.Nil(t, token)

	// Removing non-existent token should error
	err = manager.Remove("nonexistent")
	assert.Error(t, err)
}

// TestTokenManagerClear tests clearing all tokens
func TestTokenManagerClear(t *testing.T) {
	manager := tokens.NewManager()

	_ = manager.Add(&tokens.Token{Name: "token1", Value: "value1"})
	_ = manager.Add(&tokens.Token{Name: "token2", Value: "value2"})

	// Should have tokens
	assert.Len(t, manager.GetAll(), 2)

	// Clear all tokens
	manager.Clear()

	// Should be empty
	assert.Empty(t, manager.GetAll())
}

// TestTokenManagerDuplicateNames tests handling duplicate token names
func TestTokenManagerDuplicateNames(t *testing.T) {
	manager := tokens.NewManager()

	// Add first token
	err := manager.Add(&tokens.Token{Name: "color-primary", Value: "#0000ff"})
	require.NoError(t, err)

	// Adding duplicate should update the existing token
	err = manager.Add(&tokens.Token{Name: "color-primary", Value: "#ff0000"})
	require.NoError(t, err)

	// Should have the updated value
	token := manager.Get("color-primary")
	require.NotNil(t, token)
	assert.Equal(t, "#ff0000", token.Value)

	// Should still only have one token
	assert.Len(t, manager.GetAll(), 1)
}

// TestTokenManagerFindByPrefix tests finding tokens by name prefix
func TestTokenManagerFindByPrefix(t *testing.T) {
	manager := tokens.NewManager()

	_ = manager.Add(&tokens.Token{Name: "color-primary", Value: "#0000ff"})
	_ = manager.Add(&tokens.Token{Name: "color-secondary", Value: "#ff0000"})
	_ = manager.Add(&tokens.Token{Name: "color-accent", Value: "#00ff00"})
	_ = manager.Add(&tokens.Token{Name: "spacing-small", Value: "8px"})

	// Find all color tokens
	colorTokens := manager.FindByPrefix("color")
	assert.Len(t, colorTokens, 3)

	// Verify all are color tokens
	for _, token := range colorTokens {
		assert.Contains(t, token.Name, "color")
	}

	// Find spacing tokens
	spacingTokens := manager.FindByPrefix("spacing")
	assert.Len(t, spacingTokens, 1)
	assert.Equal(t, "spacing-small", spacingTokens[0].Name)
}

// TestTokenManagerDeprecated tests handling deprecated tokens
func TestTokenManagerDeprecated(t *testing.T) {
	manager := tokens.NewManager()

	err := manager.Add(&tokens.Token{
		Name:               "old-color",
		Value:              "#0000ff",
		Deprecated:         true,
		DeprecationMessage: "Use 'color-primary' instead",
	})
	require.NoError(t, err)

	token := manager.Get("old-color")
	require.NotNil(t, token)
	assert.True(t, token.Deprecated)
	assert.Equal(t, "Use 'color-primary' instead", token.DeprecationMessage)
}

// TestTokenManagerConcurrentAccess tests thread-safe operations
func TestTokenManagerConcurrentAccess(t *testing.T) {
	manager := tokens.NewManager()

	// Add initial token
	_ = manager.Add(&tokens.Token{Name: "token1", Value: "value1"})

	// Concurrent reads and writes
	done := make(chan bool, 4)

	// Reader 1
	go func() {
		for i := 0; i < 100; i++ {
			manager.Get("token1")
			manager.GetAll()
		}
		done <- true
	}()

	// Reader 2
	go func() {
		for i := 0; i < 100; i++ {
			manager.Get("token1")
			manager.FindByPrefix("tok")
		}
		done <- true
	}()

	// Writer 1
	go func() {
		for i := 0; i < 100; i++ {
			_ = manager.Add(&tokens.Token{Name: "token2", Value: "value2"})
		}
		done <- true
	}()

	// Writer 2
	go func() {
		for i := 0; i < 100; i++ {
			_ = manager.Add(&tokens.Token{Name: "token3", Value: "value3"})
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}

	// Should have all tokens without crashes
	allTokens := manager.GetAll()
	assert.GreaterOrEqual(t, len(allTokens), 1)
}

// TestManager_MultiFile_CorrectCount verifies that multi-file token management
// correctly stores tokens with composite keys and maintains accurate counts
func TestManager_MultiFile_CorrectCount(t *testing.T) {
	numFiles := 5
	tokensPerFile := 500
	m := tokens.NewManager()

	for fileIdx := 0; fileIdx < numFiles; fileIdx++ {
		filePath := fmt.Sprintf("/test/tokens-%d.json", fileIdx)
		schemaVer := schema.Draft
		if fileIdx%2 == 0 {
			schemaVer = schema.V2025_10
		}

		for tokenIdx := 0; tokenIdx < tokensPerFile; tokenIdx++ {
			token := &tokens.Token{
				Name:          fmt.Sprintf("color-token-%d", tokenIdx),
				Value:         "#FF6B35",
				Type:          "color",
				FilePath:      filePath,
				SchemaVersion: schemaVer,
			}
			err := m.Add(token)
			require.NoError(t, err)
		}
	}

	// Verify we have all tokens
	expectedCount := numFiles * tokensPerFile
	actualCount := m.Count()
	assert.Equal(t, expectedCount, actualCount, "Expected %d tokens, got %d", expectedCount, actualCount)
}
