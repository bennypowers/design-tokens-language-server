package semantictokens_test

import (
	"testing"

	semantictokens "bennypowers.dev/dtls/lsp/methods/textDocument/semanticTokens"
	"github.com/stretchr/testify/assert"
)

func TestComputeDelta_NoChange(t *testing.T) {
	oldData := []uint32{0, 0, 5, 0, 0, 1, 2, 3, 1, 0}
	newData := []uint32{0, 0, 5, 0, 0, 1, 2, 3, 1, 0}

	edits := semantictokens.ComputeDelta(oldData, newData)

	assert.Empty(t, edits, "No change should produce no edits")
}

func TestComputeDelta_AppendTokens(t *testing.T) {
	oldData := []uint32{0, 0, 5, 0, 0}
	newData := []uint32{0, 0, 5, 0, 0, 1, 2, 3, 1, 0}

	edits := semantictokens.ComputeDelta(oldData, newData)

	assert.Len(t, edits, 1)
	assert.Equal(t, uint32(5), edits[0].Start)
	assert.Equal(t, uint32(0), edits[0].DeleteCount)
	assert.Equal(t, []uint32{1, 2, 3, 1, 0}, edits[0].Data)
}

func TestComputeDelta_PrependTokens(t *testing.T) {
	oldData := []uint32{1, 2, 3, 1, 0}
	newData := []uint32{0, 0, 5, 0, 0, 1, 2, 3, 1, 0}

	edits := semantictokens.ComputeDelta(oldData, newData)

	assert.Len(t, edits, 1)
	assert.Equal(t, uint32(0), edits[0].Start)
	assert.Equal(t, uint32(0), edits[0].DeleteCount)
	assert.Equal(t, []uint32{0, 0, 5, 0, 0}, edits[0].Data)
}

func TestComputeDelta_RemoveTokens(t *testing.T) {
	oldData := []uint32{0, 0, 5, 0, 0, 1, 2, 3, 1, 0}
	newData := []uint32{0, 0, 5, 0, 0}

	edits := semantictokens.ComputeDelta(oldData, newData)

	assert.Len(t, edits, 1)
	assert.Equal(t, uint32(5), edits[0].Start)
	assert.Equal(t, uint32(5), edits[0].DeleteCount)
	assert.Empty(t, edits[0].Data)
}

func TestComputeDelta_ReplaceTokens(t *testing.T) {
	oldData := []uint32{0, 0, 5, 0, 0}
	newData := []uint32{1, 1, 6, 1, 1}

	edits := semantictokens.ComputeDelta(oldData, newData)

	assert.Len(t, edits, 1)
	assert.Equal(t, uint32(0), edits[0].Start)
	assert.Equal(t, uint32(5), edits[0].DeleteCount)
	assert.Equal(t, []uint32{1, 1, 6, 1, 1}, edits[0].Data)
}

func TestComputeDelta_ReplaceMiddle(t *testing.T) {
	// Same prefix (first token), different middle, same suffix (last token)
	oldData := []uint32{0, 0, 5, 0, 0, 1, 2, 3, 1, 0, 2, 4, 6, 2, 0}
	newData := []uint32{0, 0, 5, 0, 0, 9, 9, 9, 9, 9, 2, 4, 6, 2, 0}

	edits := semantictokens.ComputeDelta(oldData, newData)

	assert.Len(t, edits, 1)
	assert.Equal(t, uint32(5), edits[0].Start)
	assert.Equal(t, uint32(5), edits[0].DeleteCount)
	assert.Equal(t, []uint32{9, 9, 9, 9, 9}, edits[0].Data)
}

func TestComputeDelta_ComplexChange(t *testing.T) {
	// Change in middle and add at end
	// Common prefix: [0, 0, 5, 0, 0] (5 elements)
	// Common suffix: [0] (1 element - both end with 0)
	// So we replace oldData[5:9] with newData[5:14]
	oldData := []uint32{0, 0, 5, 0, 0, 1, 2, 3, 1, 0}
	newData := []uint32{0, 0, 5, 0, 0, 9, 9, 9, 9, 9, 2, 4, 6, 2, 0}

	edits := semantictokens.ComputeDelta(oldData, newData)

	// Should produce a single edit that replaces the changed portion
	assert.Len(t, edits, 1)
	assert.Equal(t, uint32(5), edits[0].Start)
	assert.Equal(t, uint32(4), edits[0].DeleteCount) // Delete 4 elements: [1, 2, 3, 1]
	assert.Equal(t, []uint32{9, 9, 9, 9, 9, 2, 4, 6, 2}, edits[0].Data)
}

func TestComputeDelta_EmptyOld(t *testing.T) {
	oldData := []uint32{}
	newData := []uint32{0, 0, 5, 0, 0}

	edits := semantictokens.ComputeDelta(oldData, newData)

	assert.Len(t, edits, 1)
	assert.Equal(t, uint32(0), edits[0].Start)
	assert.Equal(t, uint32(0), edits[0].DeleteCount)
	assert.Equal(t, []uint32{0, 0, 5, 0, 0}, edits[0].Data)
}

func TestComputeDelta_EmptyNew(t *testing.T) {
	oldData := []uint32{0, 0, 5, 0, 0}
	newData := []uint32{}

	edits := semantictokens.ComputeDelta(oldData, newData)

	assert.Len(t, edits, 1)
	assert.Equal(t, uint32(0), edits[0].Start)
	assert.Equal(t, uint32(5), edits[0].DeleteCount)
	assert.Empty(t, edits[0].Data)
}

func TestComputeDelta_BothEmpty(t *testing.T) {
	oldData := []uint32{}
	newData := []uint32{}

	edits := semantictokens.ComputeDelta(oldData, newData)

	assert.Empty(t, edits)
}

func TestApplyEdits_Roundtrip(t *testing.T) {
	tests := []struct {
		name    string
		oldData []uint32
		newData []uint32
	}{
		{
			name:    "no change",
			oldData: []uint32{0, 0, 5, 0, 0},
			newData: []uint32{0, 0, 5, 0, 0},
		},
		{
			name:    "append",
			oldData: []uint32{0, 0, 5, 0, 0},
			newData: []uint32{0, 0, 5, 0, 0, 1, 2, 3, 1, 0},
		},
		{
			name:    "prepend",
			oldData: []uint32{1, 2, 3, 1, 0},
			newData: []uint32{0, 0, 5, 0, 0, 1, 2, 3, 1, 0},
		},
		{
			name:    "remove",
			oldData: []uint32{0, 0, 5, 0, 0, 1, 2, 3, 1, 0},
			newData: []uint32{0, 0, 5, 0, 0},
		},
		{
			name:    "replace",
			oldData: []uint32{0, 0, 5, 0, 0},
			newData: []uint32{1, 1, 6, 1, 1},
		},
		{
			name:    "complex",
			oldData: []uint32{0, 0, 5, 0, 0, 1, 2, 3, 1, 0},
			newData: []uint32{0, 0, 5, 0, 0, 9, 9, 9, 9, 9, 2, 4, 6, 2, 0},
		},
		{
			name:    "empty to non-empty",
			oldData: []uint32{},
			newData: []uint32{0, 0, 5, 0, 0},
		},
		{
			name:    "non-empty to empty",
			oldData: []uint32{0, 0, 5, 0, 0},
			newData: []uint32{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edits := semantictokens.ComputeDelta(tt.oldData, tt.newData)
			result := semantictokens.ApplyEdits(tt.oldData, edits)
			assert.Equal(t, tt.newData, result)
		})
	}
}

func TestApplyEdits_EmptyEdits(t *testing.T) {
	oldData := []uint32{0, 0, 5, 0, 0}
	edits := semantictokens.ComputeDelta(oldData, oldData)

	result := semantictokens.ApplyEdits(oldData, edits)
	assert.Equal(t, oldData, result)
}
