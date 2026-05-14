package utils

import "testing"

func TestReverseString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty string", input: "", expected: ""},
		{name: "single character", input: "a", expected: "a"},
		{name: "ascii string", input: "hello", expected: "olleh"},
		{name: "palindrome", input: "racecar", expected: "racecar"},
		{name: "multi-byte UTF-8", input: "Hello, 世界", expected: "界世 ,olleH"},
		{name: "emoji characters", input: "Go🚀", expected: "🚀oG"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ReverseString(tc.input)
			if got != tc.expected {
				t.Errorf("ReverseString(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}
