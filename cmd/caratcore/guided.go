package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/BryanAguilar-Dev/CaratCore/internal/plan"
)

// errStopped means the user chose to stop between steps.
var errStopped = errors.New("stopped by the user")

// guide returns the runner.Before of the guided mode (D31). The plan was
// already approved as a whole; this only sets the pace: before each step it
// shows why it runs, what it teaches and what it does, then waits. Enter runs
// the step; q, Ctrl+C or end of input stop.
func guide(in *bufio.Reader, out io.Writer) func(context.Context, int, int, plan.Step) error {
	return func(ctx context.Context, _, _ int, s plan.Step) error {
		fmt.Fprintf(out, "Why: %s\n", s.Reason)
		if s.Learn != "" {
			fmt.Fprintf(out, "Learn: %s\n", s.Learn)
		}
		fmt.Fprintf(out, "Does: %s\n", describe(s))
		for {
			fmt.Fprint(out, "Press Enter to run this step, or type q to stop: ")
			answer, err := readLine(ctx, in)
			switch {
			case err != nil:
				return err
			case answer == "":
				return nil
			case strings.EqualFold(answer, "q"):
				return errStopped
			}
		}
	}
}

// describe says in one line what a step does, as the plan shows it.
func describe(s plan.Step) string {
	asRoot := ""
	if s.Root {
		asRoot = " (as root)"
	}
	if s.File != nil {
		return fmt.Sprintf("write %s, mode %04o%s", s.File.Path, uint32(s.File.Mode), asRoot)
	}
	return plan.ShellJoin(s.Command) + asRoot
}

// readLine reads one trimmed line, but returns as soon as ctx is cancelled:
// Ctrl+C must stop the wait without the user also pressing Enter. The
// reading goroutine is left behind then, which is fine: CaratCore stops
// reading and exits.
func readLine(ctx context.Context, in *bufio.Reader) (string, error) {
	type result struct {
		line string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		line, err := in.ReadString('\n')
		ch <- result{strings.TrimSpace(line), err}
	}()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case r := <-ch:
		if r.err != nil && r.line == "" {
			return "", errStopped // end of input: nobody can press Enter
		}
		return r.line, nil
	}
}
