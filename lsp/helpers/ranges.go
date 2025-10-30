package helpers

import (
	"fmt"
	"math"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
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

// PositionToUTF16 converts a tree-sitter Point (which uses byte offsets for Column)
// to LSP Position (which uses UTF-16 code units for Character).
//
// Returns an error if the position exceeds uint32 limits (LSP protocol limitation).
// This should never happen with normal text files, but guards against parser corruption.
func PositionToUTF16(source string, point sitter.Point) (protocol.Position, error) {
	// Check for uint32 overflow (LSP protocol limitation)
	if point.Row > math.MaxUint32 || point.Column > math.MaxUint32 {
		return protocol.Position{}, fmt.Errorf("position overflow: row=%d, col=%d exceeds uint32 limit", point.Row, point.Column)
	}

	lines := strings.Split(source, "\n")
	if point.Row >= uint(len(lines)) {
		return protocol.Position{Line: uint32(point.Row), Character: uint32(point.Column)}, nil
	}

	line := lines[point.Row]
	// point.Column is a byte offset within the line
	// Convert it to UTF-16 code units
	if point.Column > uint(len(line)) {
		point.Column = uint(len(line))
	}

	utf16Count := uint32(0)
	for _, r := range line[:point.Column] {
		if r <= 0xFFFF {
			utf16Count++
		} else {
			utf16Count += 2 // Surrogate pair
		}
	}

	return protocol.Position{
		Line:      uint32(point.Row),
		Character: utf16Count,
	}, nil
}
