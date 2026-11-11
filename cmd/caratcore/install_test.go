package main

import (
	"bufio"
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
		if got := confirm(bufio.NewReader(strings.NewReader(tt.input)), io.Discard, 3, ""); got != tt.want {
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
		if got := confirm(bufio.NewReader(strings.NewReader(tt.input)), io.Discard, 2, "uptime-kuma"); got != tt.want {
			t.Errorf("confirm(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestStepRange(t *testing.T) {
	for _, c := range []struct {
		first, last int
		want        string
	}{
		{1, 1, "Step 1 was"},
		{4, 7, "Steps 4-7 were"},
	} {
		if got := stepRange(c.first, c.last); got != c.want {
			t.Errorf("stepRange(%d, %d) = %q, want %q", c.first, c.last, got, c.want)
		}
	}
}
