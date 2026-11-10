package main

import (
	"bufio"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/BryanAguilar-Dev/CaratCore/internal/plan"
	"github.com/BryanAguilar-Dev/CaratCore/internal/term"
)

func TestGuide(t *testing.T) {
	step := plan.Step{ID: "s", Title: "S", Reason: "r", Command: []string{"true"}}
	tests := []struct {
		input   string
		wantErr error
	}{
		{"\n", nil},
		{"  \n", nil},
		{"q\n", errStopped},
		{"Q\n", errStopped},
		{"what?\n\n", nil}, // anything else asks again
		{"what?\nq\n", errStopped},
		{"", errStopped}, // end of input
	}
	for _, tt := range tests {
		before := guide(bufio.NewReader(strings.NewReader(tt.input)), io.Discard, term.Style{})
		if err := before(context.Background(), 1, 1, step); !errors.Is(err, tt.wantErr) {
			t.Errorf("input %q: got %v, want %v", tt.input, err, tt.wantErr)
		}
	}
}

// Ctrl+C while waiting must stop without anyone pressing Enter.
func TestGuideCancelled(t *testing.T) {
	r, _ := io.Pipe() // never delivers input
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	before := guide(bufio.NewReader(r), io.Discard, term.Style{})
	step := plan.Step{ID: "s", Title: "S", Reason: "r", Command: []string{"true"}}
	if err := before(ctx, 1, 1, step); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}
