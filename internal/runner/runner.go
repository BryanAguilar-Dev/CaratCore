// Package runner executes a plan on the real machine, one step at a time.
// It stops at the first step that fails and never undoes anything: it reports
// what was done and what was not, so the user knows the state of the system.
package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/BryanAguilar-Dev/CaratCore/internal/plan"
	"github.com/BryanAguilar-Dev/CaratCore/internal/term"
)

// Runner executes plans. Its zero value is not usable: create it with New
// (tests fill the fields themselves).
type Runner struct {
	// Sudo prefixes steps that need root with `sudo`. It is false when
	// CaratCore already runs as root.
	Sudo bool
	// Out receives CaratCore's progress messages and the commands' output.
	Out io.Writer
	// Style colours the progress messages; its zero value prints plain text.
	Style term.Style
	// Before, optional, is called before each step, after its title is
	// printed. Returning an error stops the plan there, like Ctrl+C: that
	// step and the rest are not run. The CLI's guided mode uses it to explain
	// the step and wait for the user.
	Before func(ctx context.Context, number, total int, s plan.Step) error
	// exec runs one command connected to the terminal. Tests replace it.
	exec func(args []string) error
}

// New returns a Runner wired to the real terminal.
func New() Runner {
	return Runner{
		Sudo:  os.Geteuid() != 0,
		Out:   os.Stdout,
		Style: term.For(os.Stdout),
		exec:  runTerminal,
	}
}

// StepError says at which step the plan stopped. Steps before it were
// completed and were not undone; steps after it were not run.
type StepError struct {
	Number int // 1-based position of the step
	Total  int // steps in the plan
	Step   plan.Step
	// NotStarted is true when the step itself did not run (Ctrl+C, or Before
	// said stop), false when it ran and failed.
	NotStarted bool
	Err        error
}

func (e *StepError) Error() string {
	what := "failed"
	if e.NotStarted {
		what = "not started"
	}
	return fmt.Sprintf("step %d/%d (%s) %s: %v", e.Number, e.Total, e.Step.ID, what, e.Err)
}

func (e *StepError) Unwrap() error { return e.Err }

// interrupted gives context.Canceled a text a user understands. Unwrap keeps
// errors.Is(err, context.Canceled) working.
type interrupted struct{ cause error }

func (e interrupted) Error() string { return "interrupted (Ctrl+C)" }
func (e interrupted) Unwrap() error { return e.cause }

// Run executes the steps of p in order. ctx is checked between steps: when it
// is cancelled (Ctrl+C) no further step starts. A running command is not
// killed by CaratCore; it receives Ctrl+C from the terminal itself, so
// programs like apt can stop cleanly.
func (r Runner) Run(ctx context.Context, p plan.Plan) error {
	if err := p.Validate(); err != nil {
		return fmt.Errorf("invalid plan: %w", err)
	}
	if r.Sudo && needsRoot(p) {
		if err := r.checkSudo(); err != nil {
			return err
		}
	}

	total := len(p.Steps)
	for i, s := range p.Steps {
		notStarted := func(err error) error {
			if errors.Is(err, context.Canceled) {
				err = interrupted{err}
			}
			return &StepError{Number: i + 1, Total: total, Step: s, NotStarted: true, Err: err}
		}
		if err := ctx.Err(); err != nil {
			return notStarted(err)
		}
		fmt.Fprintf(r.Out, "\n%s\n", r.Style.Bold(fmt.Sprintf("[%d/%d] %s", i+1, total, s.Title)))
		if r.Before != nil {
			if err := r.Before(ctx, i+1, total, s); err != nil {
				return notStarted(err)
			}
			// Ctrl+C may have arrived while Before was waiting.
			if err := ctx.Err(); err != nil {
				return notStarted(err)
			}
		}
		if err := r.runStep(s); err != nil {
			fmt.Fprintln(r.Out, r.Style.Red("✗ failed"))
			return &StepError{Number: i + 1, Total: total, Step: s, Err: err}
		}
		fmt.Fprintln(r.Out, r.Style.Green("✓ done"))
	}
	return nil
}

func (r Runner) runStep(s plan.Step) error {
	args := s.Command
	if s.File != nil {
		tmp, err := writeTemp(s.File.Content)
		if err != nil {
			return err
		}
		defer os.Remove(tmp)
		args = installArgs(tmp, s.File)
	}
	if s.Root && r.Sudo {
		args = append([]string{"sudo", "--"}, args...)
	}
	fmt.Fprintln(r.Out, r.Style.Dim("$ "+plan.ShellJoin(args)))
	return r.exec(args)
}

// checkSudo asks sudo for the user's password once, before any step runs, so
// a user without sudo rights finds out before the system is half changed.
// The password goes to sudo, never to CaratCore.
func (r Runner) checkSudo() error {
	fmt.Fprintln(r.Out, "Some steps need root. sudo may ask for your password.")
	if err := r.exec([]string{"sudo", "-v"}); err != nil {
		return fmt.Errorf("cannot use sudo (%v): run CaratCore as root or check that this user may use sudo; nothing was executed", err)
	}
	return nil
}

func needsRoot(p plan.Plan) bool {
	for _, s := range p.Steps {
		if s.Root {
			return true
		}
	}
	return false
}

// installArgs copies the temporary file to its destination with the right
// mode. `install` does it in one command, without a shell, and creates the
// file owned by whoever runs it (root when run with sudo).
func installArgs(tmp string, f *plan.File) []string {
	return []string{"install", "-m", fmt.Sprintf("%04o", uint32(f.Mode.Perm())), "--", tmp, f.Path}
}

// writeTemp saves content to a new temporary file readable only by this user.
func writeTemp(content string) (string, error) {
	f, err := os.CreateTemp("", "caratcore-*")
	if err != nil {
		return "", fmt.Errorf("creating temporary file: %w", err)
	}
	_, werr := f.WriteString(content)
	cerr := f.Close()
	if err := errors.Join(werr, cerr); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("writing temporary file: %w", err)
	}
	return f.Name(), nil
}

// runTerminal runs a command attached to the terminal, so the user sees its
// output live and sudo can ask for a password. It does not use a context on
// purpose: see Run.
func runTerminal(args []string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}
