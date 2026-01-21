package semantictokens

import (
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// ComputeDelta computes the minimal edits needed to transform oldData into newData.
// Uses a simple algorithm: find common prefix and suffix, then create a single edit
// for the differing middle portion.
func ComputeDelta(oldData, newData []uint32) []protocol.SemanticTokensEdit {
	oldLen := len(oldData)
	newLen := len(newData)

	// Find common prefix length
	prefixLen := 0
	minLen := min(oldLen, newLen)
	for prefixLen < minLen && oldData[prefixLen] == newData[prefixLen] {
		prefixLen++
	}

	// If both are identical (prefix covers all of oldData and lengths match)
	if prefixLen == oldLen && oldLen == newLen {
		return nil
	}

	// Find common suffix length (avoiding overlap with prefix)
	suffixLen := 0
	for suffixLen < minLen-prefixLen &&
		oldData[oldLen-1-suffixLen] == newData[newLen-1-suffixLen] {
		suffixLen++
	}

	// Calculate the edit range
	start := prefixLen
	deleteCount := oldLen - prefixLen - suffixLen
	insertData := newData[prefixLen : newLen-suffixLen]

	// If no actual change (shouldn't happen given earlier check, but safety)
	if deleteCount == 0 && len(insertData) == 0 {
		return nil
	}

	// Make a copy of insert data
	dataCopy := make([]uint32, len(insertData))
	copy(dataCopy, insertData)

	return []protocol.SemanticTokensEdit{
		{
			Start:       uint32(start),       //nolint:gosec // start is always non-negative
			DeleteCount: uint32(deleteCount), //nolint:gosec // deleteCount is always non-negative
			Data:        dataCopy,
		},
	}
}

// ApplyEdits applies semantic token edits to data, returning the result.
// This is used for testing the roundtrip: applyEdits(old, computeDelta(old, new)) == new
func ApplyEdits(oldData []uint32, edits []protocol.SemanticTokensEdit) []uint32 {
	if len(edits) == 0 {
		// Return a copy to avoid aliasing
		result := make([]uint32, len(oldData))
		copy(result, oldData)
		return result
	}

	// For single edit (our algorithm always produces at most one edit)
	edit := edits[0]
	start := int(edit.Start)
	deleteCount := int(edit.DeleteCount)

	// Calculate new length
	newLen := len(oldData) - deleteCount + len(edit.Data)
	result := make([]uint32, newLen)

	// Copy prefix
	copy(result[:start], oldData[:start])

	// Copy insert data
	copy(result[start:start+len(edit.Data)], edit.Data)

	// Copy suffix
	copy(result[start+len(edit.Data):], oldData[start+deleteCount:])

	return result
}
