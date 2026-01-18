package common_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"bennypowers.dev/dtls/internal/parser/common"
	"bennypowers.dev/dtls/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRootToken(t *testing.T) {
	t.Run("recognize $root in 2025.10 schema", func(t *testing.T) {
		isRoot := common.IsRootToken("$root", schema.V2025_10, []string{"_", "@", "DEFAULT"})
		assert.True(t, isRoot)
	})

	t.Run("recognize group markers in draft schema", func(t *testing.T) {
		markers := []string{"_", "@", "DEFAULT"}

		assert.True(t, common.IsRootToken("_", schema.Draft, markers))
		assert.True(t, common.IsRootToken("@", schema.Draft, markers))
		assert.True(t, common.IsRootToken("DEFAULT", schema.Draft, markers))
		assert.False(t, common.IsRootToken("normal", schema.Draft, markers))
	})

	t.Run("$root not recognized in draft schema without groupMarkers", func(t *testing.T) {
		// $root is not a reserved name in draft, so it's treated as normal token
		// unless it's in the groupMarkers list
		isRoot := common.IsRootToken("$root", schema.Draft, []string{"_"})
		assert.False(t, isRoot, "$root is not reserved in draft schema")
	})

	t.Run("group markers not recognized in 2025.10 schema", func(t *testing.T) {
		// groupMarkers are draft-only, should not work in 2025.10
		isRoot := common.IsRootToken("_", schema.V2025_10, []string{"_"})
		assert.False(t, isRoot, "groupMarkers should not work in 2025.10 schema")
	})
}

func TestGenerateRootTokenPath(t *testing.T) {
	t.Run("generate path for $root in 2025.10", func(t *testing.T) {
		// $root at color.primary.$root should become color.primary
		groupPath := []string{"color", "primary"}

		tokenPath := common.GenerateRootTokenPath(groupPath, "$root", schema.V2025_10)
		assert.Equal(t, []string{"color", "primary"}, tokenPath)
	})

	t.Run("generate path for group marker in draft", func(t *testing.T) {
		// _ at color.primary._ should become color.primary
		groupPath := []string{"color", "primary"}

		tokenPath := common.GenerateRootTokenPath(groupPath, "_", schema.Draft)
		assert.Equal(t, []string{"color", "primary"}, tokenPath)
	})

	t.Run("same path generation for both schemas", func(t *testing.T) {
		groupPath := []string{"color", "primary"}

		pathDraft := common.GenerateRootTokenPath(groupPath, "_", schema.Draft)
		path2025 := common.GenerateRootTokenPath(groupPath, "$root", schema.V2025_10)

		assert.Equal(t, pathDraft, path2025, "both should produce same token path")
	})
}

func TestRootTokensFromFixture(t *testing.T) {
	t.Run("parse draft group markers from fixture", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "test", "fixtures", "root", "draft-markers.json"))
		require.NoError(t, err)

		var data map[string]interface{}
		require.NoError(t, json.Unmarshal(content, &data))

		colors := data["color"].(map[string]interface{})
		markers := []string{"_", "@", "DEFAULT"}

		// Check "primary" group with "_" marker
		primary := colors["primary"].(map[string]interface{})
		assert.Contains(t, primary, "_")
		assert.True(t, common.IsRootToken("_", schema.Draft, markers))

		// Check "secondary" group with "@" marker
		secondary := colors["secondary"].(map[string]interface{})
		assert.Contains(t, secondary, "@")
		assert.True(t, common.IsRootToken("@", schema.Draft, markers))

		// Check "tertiary" group with "DEFAULT" marker
		tertiary := colors["tertiary"].(map[string]interface{})
		assert.Contains(t, tertiary, "DEFAULT")
		assert.True(t, common.IsRootToken("DEFAULT", schema.Draft, markers))
	})

	t.Run("parse 2025.10 $root from fixture", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "test", "fixtures", "root", "2025-root.json"))
		require.NoError(t, err)

		var data map[string]interface{}
		require.NoError(t, json.Unmarshal(content, &data))

		colors := data["color"].(map[string]interface{})

		// Check "primary" group with "$root"
		primary := colors["primary"].(map[string]interface{})
		assert.Contains(t, primary, "$root")
		assert.True(t, common.IsRootToken("$root", schema.V2025_10, nil))
	})
}

func TestCSSVariableConsistency(t *testing.T) {
	t.Run("CSS variables match across schemas", func(t *testing.T) {
		// Both color.primary.$root and color.primary._ should produce "--color-primary"
		groupPath := []string{"color", "primary"}

		pathDraft := common.GenerateRootTokenPath(groupPath, "_", schema.Draft)
		path2025 := common.GenerateRootTokenPath(groupPath, "$root", schema.V2025_10)

		// Convert to CSS variable name (simplified - actual implementation in tokens package)
		cssNameDraft := "--" + pathDraft[0] + "-" + pathDraft[1]
		cssName2025 := "--" + path2025[0] + "-" + path2025[1]

		assert.Equal(t, "--color-primary", cssNameDraft)
		assert.Equal(t, "--color-primary", cssName2025)
		assert.Equal(t, cssNameDraft, cssName2025)
	})
}
