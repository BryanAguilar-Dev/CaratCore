package main

import (
	"io"
	"strings"
	"testing"
)

func TestConfirm(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"yes\n", true},
		{"y\n", true},
		{"  YES  \n", true},
		{"yes", true}, // no newline before end of input
		{"\n", false},
		{"no\n", false},
		{"yess\n", false},
		{"", false}, // end of input, e.g. stdin closed
	}
	for _, tt := range tests {
		if got := confirm(strings.NewReader(tt.input), io.Discard, 3, ""); got != tt.want {
			t.Errorf("confirm(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// Deleting data needs the exact word: "yes" typed by reflex is not enough.
func TestConfirmWord(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"uptime-kuma\n", true},
		{"  uptime-kuma  \n", true},
		{"yes\n", false},
		{"y\n", false},
		{"Uptime-Kuma\n", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := confirm(strings.NewReader(tt.input), io.Discard, 2, "uptime-kuma"); got != tt.want {
			t.Errorf("confirm(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
