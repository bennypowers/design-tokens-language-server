package tokens

import (
	"fmt"
	"strings"
	"sync"

	"bennypowers.dev/dtls/internal/schema"
)

// Manager manages design tokens loaded from various sources.
//
// Multi-Schema Support:
// The Manager uses a composite key system to support tokens from multiple files
// with different schema versions simultaneously. This is essential for workspaces
// where legacy (draft) and modern (2025.10) token files coexist.
//
// Key Format:
//   - With file path: "filePath:tokenName" (e.g., "/path/tokens.json:color-primary")
//   - Without file path: "tokenName" (legacy support for backward compatibility)
//
// This design allows:
//   - Multiple files to define tokens with the same name without collision
//   - Schema-specific parsing and resolution for each file
//   - Incremental migration from draft to 2025.10 schema
//
// Example:
//
//	// Two files can both define "color-primary"
//	legacy/tokens.json:color-primary  (draft schema)
//	design/tokens.json:color-primary  (2025.10 schema)
type Manager struct {
	// tokens stores design tokens using composite keys.
	// Key format: "filePath:tokenName" for multi-file support,
	// or just "tokenName" for legacy single-file scenarios.
	tokens map[string]*Token
	mu     sync.RWMutex
}

// NewManager creates a new token manager with an empty token registry.
func NewManager() *Manager {
	return &Manager{
		tokens: make(map[string]*Token),
	}
}

// makeKey creates a composite key for token storage.
//
// The key format enables multi-schema workspace support by including the file path:
//   - If filePath is provided: returns "filePath:tokenName"
//   - If filePath is empty: returns "tokenName" (for backward compatibility)
//
// This allows tokens with identical names from different files to coexist without
// collision, which is critical when supporting both draft and 2025.10 schemas in
// the same workspace.
//
// Parameters:
//   - filePath: The absolute path to the token file, or empty string for legacy support
//   - tokenName: The hyphenated token name (e.g., "color-primary")
//
// Returns:
//   - A composite key string used for internal storage
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
// For multi-file scenarios, use RemoveBySourceFile or provide the full composite key
func (m *Manager) Remove(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Try direct lookup first (legacy or composite key)
	if _, exists := m.tokens[name]; !exists {
		// Search across all files for matching token
		for key, token := range m.tokens {
			if strings.HasSuffix(key, ":"+name) || token.Name == name {
				delete(m.tokens, key)
				return nil
			}
		}
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
	for _, token := range m.tokens {
		if strings.HasPrefix(token.Name, prefix) {
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
// Returns Unknown if the file has no tokens, doesn't exist, or has inconsistent schema versions.
// All tokens from the same file should have the same schema version; if they don't,
// this indicates a parsing bug and Unknown is returned.
func (m *Manager) GetSchemaVersionForFile(filePath string) schema.SchemaVersion {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var version schema.SchemaVersion
	found := false

	for _, token := range m.tokens {
		if token.FilePath == filePath {
			if !found {
				// First token from this file
				version = token.SchemaVersion
				found = true
			} else if token.SchemaVersion != version {
				// Inconsistency detected - this should not happen
				// Return Unknown to signal an error condition
				return schema.Unknown
			}
		}
	}

	if !found {
		return schema.Unknown
	}
	return version
}
