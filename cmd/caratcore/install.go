package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/BryanAguilar-Dev/CaratCore/internal/docker"
	"github.com/BryanAguilar-Dev/CaratCore/internal/plan"
	"github.com/BryanAguilar-Dev/CaratCore/internal/runner"
)

const installUsage = "usage: caratcore install docker [--dry-run [--json]]"

// runInstall handles `caratcore install <target> [flags]`.
func runInstall(args []string) int {
	if len(args) == 0 || args[0] != "docker" {
		fmt.Fprintln(os.Stderr, installUsage)
		return 2
	}

	flags := flag.NewFlagSet("install docker", flag.ContinueOnError)
	dryRun := flags.Bool("dry-run", false, "show the plan without executing anything")
	asJSON := flags.Bool("json", false, "print the plan as JSON (only with --dry-run)")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if *asJSON && !*dryRun {
		fmt.Fprintln(os.Stderr, "--json only works with --dry-run: the plan to execute is shown as text")
		return 2
	}

	p, err := dockerPlan()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	if err := printPlan(os.Stdout, p, *asJSON); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	if *dryRun || len(p.Steps) == 0 {
		return 0
	}
	return execute(p)
}

func dockerPlan() (plan.Plan, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	host, err := docker.Detect(ctx)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("error: %w", err)
	}
	p, err := docker.InstallPlan(host)
	if err != nil {
		return p, fmt.Errorf("cannot plan the Docker installation: %w", err)
	}
	if err := p.Validate(); err != nil {
		// A bug in CaratCore, not in the user's system.
		return p, fmt.Errorf("internal error: invalid plan: %w", err)
	}
	return p, nil
}

// execute asks the user to approve p and runs it.
func execute(p plan.Plan) int {
	if !confirm(os.Stdin, os.Stdout, len(p.Steps)) {
		fmt.Println("Cancelled. Nothing was executed.")
		return 1
	}

	// Ctrl+C cancels ctx instead of killing CaratCore, so it can say where it
	// stopped. The running command gets Ctrl+C from the terminal itself.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	err := runner.New().Run(ctx, p)
	var stepErr *runner.StepError
	switch {
	case err == nil:
		fmt.Println("\nDone. Check the result with: caratcore doctor")
		return 0
	case errors.As(err, &stepErr):
		fmt.Fprintf(os.Stderr, "\n%v\n", err)
		if done := stepErr.Number - 1; done > 0 {
			fmt.Fprintf(os.Stderr, "Steps 1-%d were completed and were not undone.\n", done)
		}
		if stepErr.Number < stepErr.Total {
			fmt.Fprintf(os.Stderr, "Steps %d-%d were not run.\n", stepErr.Number+1, stepErr.Total)
		}
		fmt.Fprintln(os.Stderr, "Fix the problem and run the same command again: it plans from the current state.")
		return 1
	default:
		fmt.Fprintf(os.Stderr, "\n%v\n", err)
		return 1
	}
}

// confirm asks for explicit approval. Anything but "yes" or "y" (including
// end of input) means no.
func confirm(in io.Reader, out io.Writer, steps int) bool {
	fmt.Fprintf(out, "\nRun these %d step(s) on this machine? Type yes to continue: ", steps)
	answer, _ := bufio.NewReader(in).ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "yes" || answer == "y"
}

func printPlan(w io.Writer, p plan.Plan, asJSON bool) error {
	if !asJSON {
		plan.Render(w, p)
		return nil
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(p)
}
