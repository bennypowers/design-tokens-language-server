package helpers

import (
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// RangesIntersect checks if two LSP ranges intersect.
// Ranges are treated as half-open intervals [start, end) where the end position is exclusive.
//
// Returns true if the ranges overlap, false otherwise.
//
// Examples:
//   - [0:0, 0:5) and [0:3, 0:7) -> true (overlap from 0:3 to 0:5)
//   - [0:0, 0:5) and [0:5, 0:10) -> false (adjacent but not overlapping)
//   - [0:0, 1:0) and [0:5, 0:10) -> true (first range includes line 0:5)
func RangesIntersect(a, b protocol.Range) bool {
	// Check if a ends before or at the start of b (no intersection)
	if a.End.Line < b.Start.Line {
		return false
	}
	if a.End.Line == b.Start.Line && a.End.Character <= b.Start.Character {
		return false
	}

	// Check if b ends before or at the start of a (no intersection)
	if b.End.Line < a.Start.Line {
		return false
	}
	if b.End.Line == a.Start.Line && b.End.Character <= a.Start.Character {
		return false
	}

	return true
}
