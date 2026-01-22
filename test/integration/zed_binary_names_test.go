package integration_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestZedBinaryNamingConsistency verifies that the Zed extension references
// the correct binary names that match the go-release-workflows output.
func TestZedBinaryNamingConsistency(t *testing.T) {
	// Read Zed extension source
	zedSource, err := os.ReadFile("../../extensions/zed/src/dtls.rs")
	require.NoError(t, err)

	sourceStr := string(zedSource)

	// Assert on actual Rust mapping literals to catch logic regressions
	expectedArchMappings := []string{
		`zed::Architecture::Aarch64 => "arm64"`,
		`zed::Architecture::X8664 => "x64"`,
	}
	for _, mapping := range expectedArchMappings {
		assert.Contains(t, sourceStr, mapping,
			"Zed extension should map architecture via %q", mapping)
	}

	expectedOsMappings := []string{
		`zed::Os::Mac => ("darwin", "")`,
		`zed::Os::Linux => ("linux", "")`,
		`zed::Os::Windows => ("win32", ".exe")`,
	}
	for _, mapping := range expectedOsMappings {
		assert.Contains(t, sourceStr, mapping,
			"Zed extension should map OS via %q", mapping)
	}

	// Ensure old target triple names are NOT present
	oldPatterns := []string{
		"aarch64-apple-darwin",
		"x86_64-apple-darwin",
		"aarch64-unknown-linux-gnu",
		"x86_64-unknown-linux-gnu",
	}
	for _, pattern := range oldPatterns {
		assert.NotContains(t, sourceStr, pattern,
			"Zed extension should NOT reference old pattern %q", pattern)
	}
}
