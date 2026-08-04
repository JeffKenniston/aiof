package main

import "testing"

func TestReverseString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "olleh"},
		{"world", "dlrow"},
		{"Hello, 世界", "界世 ,olleH"},
		{"", ""},
		{"a", "a"},
	}

	for _, tt := range tests {
		result := ReverseString(tt.input)
		if result != tt.expected {
			t.Errorf("ReverseString(%q) = %q; expected %q", tt.input, result, tt.expected)
		}
	}
}
