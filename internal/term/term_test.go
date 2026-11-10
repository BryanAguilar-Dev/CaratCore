package term

import (
	"os"
	"testing"
)

func TestZeroValueIsPlain(t *testing.T) {
	if got := (Style{}).Green("ok"); got != "ok" {
		t.Errorf("got %q, want plain text", got)
	}
}

func TestOn(t *testing.T) {
	if got := (Style{on: true}).Red("x"); got != "\x1b[31mx\x1b[0m" {
		t.Errorf("got %q", got)
	}
}

func TestForFileIsPlain(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if For(f).on {
		t.Error("a regular file is not a terminal: no colour")
	}
}

func TestNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if For(os.Stdout).on {
		t.Error("NO_COLOR must disable colour")
	}
}
