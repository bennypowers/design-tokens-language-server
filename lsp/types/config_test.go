package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, "", config.Prefix)
	assert.Empty(t, config.TokensFiles)
	assert.Equal(t, []string{"_", "@", "DEFAULT"}, config.GroupMarkers)
}
