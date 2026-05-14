package utils

// ReverseString reverses a given string, correctly handling
// multi-byte UTF-8 characters by operating on runes.
//
// Note: This function does not preserve Unicode combining characters
// or grapheme clusters (e.g., "é" composed as U+0065 + U+0301 may
// produce incorrect results after reversal). For grapheme-correct
// reversal, consider using golang.org/x/text/unicode/norm.
func ReverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
