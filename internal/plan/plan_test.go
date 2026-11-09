package plan

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func validStep(id string) Step {
	return Step{
		ID:      id,
		Title:   "Refresh package lists",
		Reason:  "apt needs the latest package index",
		Command: []string{"apt-get", "update"},
		Root:    true,
	}
}

func TestValidate(t *testing.T) {
	noReason := validStep("b")
	noReason.Reason = ""
	noCommand := validStep("b")
	noCommand.Command = nil

	tests := []struct {
		name    string
		plan    Plan
		wantErr string
	}{
		{"valid", Plan{Title: "t", Steps: []Step{validStep("a"), validStep("b")}}, ""},
		{"empty plan is valid", Plan{Title: "t"}, ""},
		{"no title", Plan{Steps: []Step{validStep("a")}}, "no title"},
		{"missing id", Plan{Title: "t", Steps: []Step{validStep("")}}, "missing id"},
		{"duplicate id", Plan{Title: "t", Steps: []Step{validStep("a"), validStep("a")}}, "duplicate id"},
		{"missing reason", Plan{Title: "t", Steps: []Step{validStep("a"), noReason}}, "step 2 (b): missing reason"},
		{"missing command", Plan{Title: "t", Steps: []Step{noCommand}}, "missing command"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.plan.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("got %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestShellJoin(t *testing.T) {
	got := ShellJoin([]string{"echo", "plain", "two words", "it's", "$HOME", ""})
	want := `echo plain 'two words' 'it'\''s' '$HOME' ''`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestRender(t *testing.T) {
	var buf bytes.Buffer
	Render(&buf, Plan{Title: "Install Docker", Steps: []Step{validStep("a")}})
	out := buf.String()
	for _, want := range []string{
		"Plan: Install Docker",
		"Nothing has been executed yet.",
		"1. Refresh package lists",
		"Why: apt needs the latest package index",
		"Run as root: apt-get update",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output is missing %q:\n%s", want, out)
		}
	}

	buf.Reset()
	Render(&buf, Plan{Title: "Install Docker"})
	if !strings.Contains(buf.String(), "Nothing to do.") {
		t.Errorf("empty plan output:\n%s", buf.String())
	}
}

func TestJSONRoundTrip(t *testing.T) {
	in := Plan{Title: "Install Docker", Steps: []Step{validStep("a")}}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out Plan
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("round trip changed the plan:\n in  %+v\n out %+v", in, out)
	}
}
