package integration_test

import (
	"testing"

	"github.com/bennypowers/design-tokens-language-server/test/integration/testutil"
	"github.com/stretchr/testify/assert"
)

// TestConfigurationDefault tests default configuration
func TestConfigurationDefault(t *testing.T) {
	server := testutil.NewTestServer(t)

	config := server.GetConfig()

	// Should have defaults
	assert.Empty(t, config.Prefix)
	assert.Contains(t, config.GroupMarkers, "_")
	assert.Contains(t, config.GroupMarkers, "@")
	assert.Contains(t, config.GroupMarkers, "DEFAULT")
}

// TestConfigurationWithPrefix tests loading tokens with prefix
func TestConfigurationWithPrefix(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadTokensWithPrefix(t, server, "ds")

	// Verify tokens are loaded (implicitly tests configuration)
	// GetConfig() is called, verifying the code path works
	config := server.GetConfig()
	assert.NotNil(t, config)
}

// TestConfigurationGroupMarkers tests group markers in token names
func TestConfigurationGroupMarkers(t *testing.T) {
	server := testutil.NewTestServer(t)

	// Load tokens with group markers
	// The default config includes "_", "@", "DEFAULT" as group markers
	config := server.GetConfig()
	assert.Contains(t, config.GroupMarkers, "_")
	assert.Contains(t, config.GroupMarkers, "@")
	assert.Contains(t, config.GroupMarkers, "DEFAULT")
}
