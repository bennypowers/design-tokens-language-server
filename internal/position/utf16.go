package position

import "unicode/utf8"

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

		// Characters above U+FFFF (outside BMP) use surrogate pairs in UTF-16
		if r > 0xFFFF {
			units += 2
		} else {
			units++
		}

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
	for _, r := range s[:byteOffset] {
		if r > 0xFFFF {
			utf16Count += 2 // Surrogate pair
		} else {
			utf16Count++
		}
	}
	return utf16Count
}

// StringLengthUTF16 returns the length of a string in UTF-16 code units.
// This is useful for calculating ranges and validating positions.
func StringLengthUTF16(s string) int {
	utf16Count := 0
	for _, r := range s {
		if r > 0xFFFF {
			utf16Count += 2
		} else {
			utf16Count++
		}
	}
	return utf16Count
}
