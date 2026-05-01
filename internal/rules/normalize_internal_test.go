package rules

import "testing"

func TestNormalizeDescription(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"  hello   world  ", "hello world"},
		{"HELLO WORLD", "hello world"},
		{"café  BISTRO", "café bistro"},
		{"normal", "normal"},
		{"", ""},
	}
	for _, tc := range cases {
		got := normalizeDescription(tc.input)
		if got != tc.want {
			t.Errorf("normalizeDescription(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
