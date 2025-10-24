package tokens

import (
	"fmt"
	"strings"
	"sync"
)

// Manager manages design tokens loaded from various sources
type Manager struct {
	tokens map[string]*Token // Key is the token name
	mu     sync.RWMutex
}

// NewManager creates a new token manager
func NewManager() *Manager {
	return &Manager{
		tokens: make(map[string]*Token),
	}
}

// Add adds or updates a token in the manager
func (m *Manager) Add(token *Token) error {
	if token == nil {
		return fmt.Errorf("token cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.tokens[token.Name] = token
	return nil
}

// Get retrieves a token by name or CSS variable name
// Supports:
// - "color-primary" (token name)
// - "--color-primary" (CSS variable without prefix)
// - "--prefix-color-primary" (CSS variable with prefix)
func (m *Manager) Get(nameOrVar string) *Token {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try direct lookup first
	if token, exists := m.tokens[nameOrVar]; exists {
		return token
	}

	// If it starts with --, try removing it
	if strings.HasPrefix(nameOrVar, "--") {
		nameWithoutDashes := strings.TrimPrefix(nameOrVar, "--")

		// Try direct lookup without dashes
		if token, exists := m.tokens[nameWithoutDashes]; exists {
			return token
		}

		// Try matching with prefixes
		for _, token := range m.tokens {
			if token.CSSVariableName() == nameOrVar {
				return token
			}
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
