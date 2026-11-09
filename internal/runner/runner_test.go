package runner

import (
	"bytes"
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/BryanAguilar-Dev/CaratCore/internal/plan"
)

// fakeRunner records every command instead of running it. failOn makes the
// command whose last argument equals it fail.
func fakeRunner(sudo bool, failOn string) (*Runner, *[][]string) {
	var calls [][]string
	r := &Runner{Sudo: sudo, Out: &bytes.Buffer{}}
	r.exec = func(args []string) error {
		calls = append(calls, args)
		if failOn != "" && args[len(args)-1] == failOn {
			return errors.New("exit status 1")
		}
		return nil
	}
	return r, &calls
}

func testPlan() plan.Plan {
	return plan.Plan{Title: "t", Steps: []plan.Step{
		{ID: "one", Title: "One", Reason: "r", Command: []string{"apt-get", "update"}, Root: true},
		{ID: "two", Title: "Two", Reason: "r", Command: []string{"echo", "two"}},
		{ID: "three", Title: "Three", Reason: "r", Command: []string{"apt-get", "install", "x"}, Root: true},
	}}
}

func TestRunWithSudo(t *testing.T) {
	r, calls := fakeRunner(true, "")
	if err := r.Run(context.Background(), testPlan()); err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"sudo", "-v"},
		{"sudo", "--", "apt-get", "update"},
		{"echo", "two"}, // not a root step: no sudo
		{"sudo", "--", "apt-get", "install", "x"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Errorf("calls:\n got %q\nwant %q", *calls, want)
	}
}

func TestRunAsRootDoesNotUseSudo(t *testing.T) {
	r, calls := fakeRunner(false, "")
	if err := r.Run(context.Background(), testPlan()); err != nil {
		t.Fatal(err)
	}
	for _, c := range *calls {
		if c[0] == "sudo" {
			t.Errorf("unexpected sudo call %q", c)
		}
	}
}

func TestRunStopsAtFirstFailure(t *testing.T) {
	r, calls := fakeRunner(false, "two")
	err := r.Run(context.Background(), testPlan())
	var stepErr *StepError
	if !errors.As(err, &stepErr) {
		t.Fatalf("want *StepError, got %v", err)
	}
	if stepErr.Number != 2 || stepErr.Total != 3 || stepErr.Step.ID != "two" {
		t.Errorf("got step %d/%d %s, want 2/3 two", stepErr.Number, stepErr.Total, stepErr.Step.ID)
	}
	if len(*calls) != 2 {
		t.Errorf("step three must not run; calls: %q", *calls)
	}
}

func TestRunFailsEarlyWithoutSudo(t *testing.T) {
	r, calls := fakeRunner(true, "-v")
	err := r.Run(context.Background(), testPlan())
	if err == nil || !strings.Contains(err.Error(), "nothing was executed") {
		t.Fatalf("got %v", err)
	}
	if len(*calls) != 1 {
		t.Errorf("no step may run when sudo fails; calls: %q", *calls)
	}
}

func TestRunStopsWhenCancelled(t *testing.T) {
	r, calls := fakeRunner(false, "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := r.Run(ctx, testPlan())
	var stepErr *StepError
	if !errors.As(err, &stepErr) || stepErr.Number != 1 || !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if len(*calls) != 0 {
		t.Errorf("nothing may run after cancel; calls: %q", *calls)
	}
}

func TestRunRejectsInvalidPlan(t *testing.T) {
	r, calls := fakeRunner(false, "")
	p := testPlan()
	p.Steps[1].Reason = ""
	if err := r.Run(context.Background(), p); err == nil {
		t.Fatal("want error for a step without reason")
	}
	if len(*calls) != 0 {
		t.Errorf("nothing may run from an invalid plan; calls: %q", *calls)
	}
}

func TestRunWritesFileWithInstall(t *testing.T) {
	content := "Types: deb\n"
	p := plan.Plan{Title: "t", Steps: []plan.Step{{
		ID: "file", Title: "File", Reason: "r", Root: true,
		File: &plan.File{Path: "/etc/apt/sources.list.d/x.sources", Content: content, Mode: 0o644},
	}}}

	r, calls := fakeRunner(true, "")
	var tmp string
	inner := r.exec
	r.exec = func(args []string) error {
		if args[len(args)-1] == "/etc/apt/sources.list.d/x.sources" {
			tmp = args[len(args)-2]
			got, err := os.ReadFile(tmp)
			if err != nil || string(got) != content {
				t.Errorf("temporary file: got %q, %v; want %q", got, err, content)
			}
		}
		return inner(args)
	}
	if err := r.Run(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	want := []string{"sudo", "--", "install", "-m", "0644", "--", tmp, "/etc/apt/sources.list.d/x.sources"}
	if got := (*calls)[1]; !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Errorf("temporary file %s was not removed", tmp)
	}
}
