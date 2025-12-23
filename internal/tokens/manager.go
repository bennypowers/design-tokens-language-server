package tokens

import (
	"fmt"
	"strings"
	"sync"

	"bennypowers.dev/dtls/internal/schema"
)

// Manager manages design tokens loaded from various sources
type Manager struct {
	// Composite key: "filePath:tokenName" for multi-schema support
	// This allows tokens with same name from different files to coexist
	tokens map[string]*Token
	mu     sync.RWMutex
}

// NewManager creates a new token manager
func NewManager() *Manager {
	return &Manager{
		tokens: make(map[string]*Token),
	}
}

// makeKey creates a composite key for token storage
func makeKey(filePath, tokenName string) string {
	if filePath == "" {
		// Legacy tokens without file path
		return tokenName
	}
	return filePath + ":" + tokenName
}

// Add adds or updates a token in the manager
func (m *Manager) Add(token *Token) error {
	if token == nil {
		return fmt.Errorf("token cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	key := makeKey(token.FilePath, token.Name)
	m.tokens[key] = token
	return nil
}

// Get retrieves a token by name or CSS variable name
// Returns the first matching token if multiple exist across files
// Supports:
// - "color-primary" (token name with hyphens)
// - "color.primary" (DTCG path with dots)
// - "--color-primary" (CSS variable without prefix)
// - "--prefix-color-primary" (CSS variable with prefix)
func (m *Manager) Get(nameOrVar string) *Token {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try direct lookup first (legacy single-token case)
	if token, exists := m.tokens[nameOrVar]; exists {
		return token
	}

	// Convert dots to hyphens if needed
	searchName := nameOrVar
	if strings.Contains(nameOrVar, ".") {
		searchName = strings.ReplaceAll(nameOrVar, ".", "-")
	}

	// Strip -- prefix if present
	if strings.HasPrefix(searchName, "--") {
		searchName = strings.TrimPrefix(searchName, "--")
	}

	// Search across all files for token with matching name
	for key, token := range m.tokens {
		// Check if key ends with :tokenName
		if strings.HasSuffix(key, ":"+searchName) || key == searchName {
			return token
		}

		// Also check CSS variable name match
		if token.CSSVariableName() == nameOrVar {
			return token
		}
	}

	return nil
}

// GetAll returns all tokens
func (m *Manager) GetAll() []*Token {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tokens := make([]*Token, 0, len(m.tokens))
	for _, token := range m.tokens {
		tokens = append(tokens, token)
	}
	return tokens
}

// Remove removes a token by name
func (m *Manager) Remove(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tokens[name]; !exists {
		return fmt.Errorf("token not found: %s", name)
	}

	delete(m.tokens, name)
	return nil
}

// Clear removes all tokens
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tokens = make(map[string]*Token)
}

// FindByPrefix returns all tokens whose names start with the given prefix
func (m *Manager) FindByPrefix(prefix string) []*Token {
	m.mu.RLock()
	defer m.mu.RUnlock()

	matches := []*Token{}
	for name, token := range m.tokens {
		if strings.HasPrefix(name, prefix) {
			matches = append(matches, token)
		}
	}
	return matches
}

// Count returns the number of tokens
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.tokens)
}

// GetBySchemaVersion returns all tokens for a specific schema version
func (m *Manager) GetBySchemaVersion(version schema.SchemaVersion) []*Token {
	m.mu.RLock()
	defer m.mu.RUnlock()

	matches := []*Token{}
	for _, token := range m.tokens {
		if token.SchemaVersion == version {
			matches = append(matches, token)
		}
	}
	return matches
}

// GetBySourceFile returns all tokens from a specific source file
func (m *Manager) GetBySourceFile(filePath string) []*Token {
	m.mu.RLock()
	defer m.mu.RUnlock()

	matches := []*Token{}
	for _, token := range m.tokens {
		if token.FilePath == filePath {
			matches = append(matches, token)
		}
	}
	return matches
}

// GetQualified retrieves a token by name and file path
// This allows resolving ambiguous token names when multiple files define the same token
func (m *Manager) GetQualified(tokenName, filePath string) *Token {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := makeKey(filePath, tokenName)
	return m.tokens[key]
}

// RemoveBySourceFile removes all tokens from a specific source file
// Returns the number of tokens removed
func (m *Manager) RemoveBySourceFile(filePath string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	removed := 0
	for key, token := range m.tokens {
		if token.FilePath == filePath {
			delete(m.tokens, key)
			removed++
		}
	}
	return removed
}

// GetSourceFiles returns a list of all unique source files that have tokens loaded
func (m *Manager) GetSourceFiles() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	files := make(map[string]bool)
	for _, token := range m.tokens {
		if token.FilePath != "" {
			files[token.FilePath] = true
		}
	}

	result := make([]string, 0, len(files))
	for file := range files {
		result = append(result, file)
	}
	return result
}

// GetSchemaVersionForFile returns the schema version used by a specific file
// Returns Unknown if the file has no tokens or doesn't exist
func (m *Manager) GetSchemaVersionForFile(filePath string) schema.SchemaVersion {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, token := range m.tokens {
		if token.FilePath == filePath {
			return token.SchemaVersion
		}
	}
	return schema.Unknown
}
