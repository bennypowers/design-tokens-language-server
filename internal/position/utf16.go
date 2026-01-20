package position

import (
	"math"
	"unicode/utf16"
	"unicode/utf8"
)

// UTF16ToByteOffset converts a UTF-16 code unit offset to a byte offset in a string.
// LSP positions use UTF-16 code units, but Go strings are UTF-8 byte sequences.
// This function handles surrogate pairs correctly (characters > U+FFFF count as 2 UTF-16 units).
func UTF16ToByteOffset(s string, utf16Col int) int {
	if utf16Col <= 0 {
		return 0
	}

	units := 0
	byteOffset := 0

	for byteOffset < len(s) && units < utf16Col {
		r, size := utf8.DecodeRuneInString(s[byteOffset:])
		if r == utf8.RuneError && size == 1 {
			// Invalid UTF-8 byte; treat as single unit and advance by 1 byte
			byteOffset++
			units++
			continue
		}

		// Use stdlib utf16.RuneLen to determine UTF-16 length (1 or 2 code units)
		runeUTF16Len := utf16.RuneLen(r)

		// If target falls within a surrogate pair, clamp to the start of the rune
		if runeUTF16Len == 2 && units+1 == utf16Col {
			// Stop here (clamp to code-point boundary)
			break
		}

		units += runeUTF16Len

		byteOffset += size
	}

	return byteOffset
}

// ByteOffsetToUTF16 converts a byte offset to a UTF-16 code unit offset in a string.
// This is the inverse of UTF16ToByteOffset and is useful for converting positions
// from Go's byte-based indexing to LSP's UTF-16 positions.
func ByteOffsetToUTF16(s string, byteOffset int) int {
	if byteOffset <= 0 {
		return 0
	}
	if byteOffset > len(s) {
		byteOffset = len(s)
	}

	utf16Count := 0
	currentOffset := 0

	// Iterate through runes without slicing to avoid partial rune issues
	for currentOffset < byteOffset {
		r, size := utf8.DecodeRuneInString(s[currentOffset:])
		if r == utf8.RuneError && size == 0 {
			break // End of string
		}

		// Stop if decoding this rune would cross the target byteOffset
		if currentOffset+size > byteOffset {
			break
		}

		utf16Count += utf16.RuneLen(r)

		currentOffset += size
	}
	return utf16Count
}

// StringLengthUTF16 returns the length of a string in UTF-16 code units.
// This is useful for calculating ranges and validating positions.
func StringLengthUTF16(s string) int {
	utf16Count := 0
	for _, r := range s {
		utf16Count += utf16.RuneLen(r)
	}
	return utf16Count
}

// ByteOffsetToUTF16Uint32 is like ByteOffsetToUTF16 but returns uint32 for LSP compatibility.
// Clamps the result to valid uint32 range.
func ByteOffsetToUTF16Uint32(s string, byteOffset int) uint32 {
	result := ByteOffsetToUTF16(s, byteOffset)
	if result < 0 {
		return 0
	}
	if result > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(result)
}

// StringLengthUTF16Uint32 is like StringLengthUTF16 but returns uint32 for LSP compatibility.
// Clamps the result to valid uint32 range.
func StringLengthUTF16Uint32(s string) uint32 {
	result := StringLengthUTF16(s)
	if result < 0 {
		return 0
	}
	if result > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(result)
}
