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

	"github.com/BryanAguilar-Dev/CaratCore/internal/catalog"
	"github.com/BryanAguilar-Dev/CaratCore/internal/docker"
	"github.com/BryanAguilar-Dev/CaratCore/internal/plan"
	"github.com/BryanAguilar-Dev/CaratCore/internal/runner"
	"github.com/BryanAguilar-Dev/CaratCore/internal/service"
	"github.com/BryanAguilar-Dev/CaratCore/internal/term"
)

const installUsage = "usage: caratcore install <docker|adguard-home|uptime-kuma> [--guided] [--dry-run [--json]]"

// runInstall handles `caratcore install <target> [flags]`.
func runInstall(args []string) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(os.Stderr, installUsage)
		return 2
	}
	target := args[0]

	flags := flag.NewFlagSet("install "+target, flag.ContinueOnError)
	dryRun := flags.Bool("dry-run", false, "show the plan without executing anything")
	asJSON := flags.Bool("json", false, "print the plan as JSON (only with --dry-run)")
	guided := flags.Bool("guided", false, "explain each step and wait for Enter before running it")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if !validFlags(*dryRun, *asJSON, *guided) {
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var p plan.Plan
	var nextSteps []string
	var err error
	if target == "docker" {
		p, err = dockerPlan(ctx)
		nextSteps = []string{"Check the result with: caratcore doctor"}
	} else {
		var s catalog.Service
		if s, err = catalog.Get(target); err == nil {
			p, err = servicePlan(ctx, s)
			nextSteps = service.DeployNextSteps(s)
		}
	}
	if err == nil {
		err = p.Validate()
		if err != nil {
			// A bug in CaratCore, not in the user's system.
			err = fmt.Errorf("internal error: invalid plan: %w", err)
		}
	}
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
	return execute(p, nextSteps, "", *guided)
}

// validFlags reports (and explains) combinations that make no sense.
func validFlags(dryRun, asJSON, guided bool) bool {
	switch {
	case asJSON && !dryRun:
		fmt.Fprintln(os.Stderr, "--json only works with --dry-run: the plan to execute is shown as text")
		return false
	case guided && dryRun:
		fmt.Fprintln(os.Stderr, "--guided paces a real run; --dry-run only shows the plan")
		return false
	}
	return true
}

func dockerPlan(ctx context.Context) (plan.Plan, error) {
	host, err := docker.Detect(ctx)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("error: %w", err)
	}
	p, err := docker.InstallPlan(host)
	if err != nil {
		return p, fmt.Errorf("cannot plan the Docker installation: %w", err)
	}
	return p, nil
}

func servicePlan(ctx context.Context, s catalog.Service) (plan.Plan, error) {
	host, err := service.Detect(ctx, s)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("error: %w", err)
	}
	p, err := service.DeployPlan(s, host)
	if err != nil {
		return p, fmt.Errorf("cannot plan the deployment of %s: %w", s.Name, err)
	}
	return p, nil
}

// execute asks the user to approve p, runs it and then says what to do next.
// See confirm for word. guided pauses before each step (D31); the approval is
// the same either way.
func execute(p plan.Plan, nextSteps []string, word string, guided bool) int {
	// One reader for all answers, so a line typed ahead is not lost between
	// the approval and the guided pauses.
	in := bufio.NewReader(os.Stdin)
	if !confirm(in, os.Stdout, len(p.Steps), word) {
		fmt.Println("Cancelled. Nothing was executed.")
		return 1
	}

	// Ctrl+C cancels ctx instead of killing CaratCore, so it can say where it
	// stopped. The running command gets Ctrl+C from the terminal itself.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	r := runner.New()
	if guided {
		fmt.Println("\nGuided mode: CaratCore stops before each step to explain it.")
		r.Before = guide(in, os.Stdout, r.Style)
	}
	errStyle := term.For(os.Stderr)
	err := r.Run(ctx, p)
	var stepErr *runner.StepError
	switch {
	case err == nil:
		fmt.Printf("\n%s\n", r.Style.Green(fmt.Sprintf("✓ %s: all %d step(s) completed.", p.Title, len(p.Steps))))
		fmt.Println("Next:")
		for _, step := range nextSteps {
			fmt.Println("  - " + step)
		}
		return 0
	case errors.As(err, &stepErr):
		fmt.Fprintf(os.Stderr, "\n%s\n", errStyle.Red("✗ "+err.Error()))
		if done := stepErr.Number - 1; done > 0 {
			fmt.Fprintf(os.Stderr, "Steps 1-%d were completed and were not undone.\n", done)
		}
		firstNotRun := stepErr.Number + 1
		if stepErr.NotStarted {
			firstNotRun = stepErr.Number
		}
		if firstNotRun <= stepErr.Total {
			fmt.Fprintf(os.Stderr, "Steps %d-%d were not run.\n", firstNotRun, stepErr.Total)
		}
		if stepErr.NotStarted {
			fmt.Fprintln(os.Stderr, "Run the same command again to continue: it plans from the current state.")
		} else {
			fmt.Fprintln(os.Stderr, "Fix the problem and run the same command again: it plans from the current state.")
		}
		return 1
	default:
		fmt.Fprintf(os.Stderr, "\n%s\n", errStyle.Red("✗ "+err.Error()))
		return 1
	}
}

// confirm asks for explicit approval. With word == "", only "yes" or "y"
// approve. Otherwise the plan cannot be undone and only word itself approves,
// so a reflex "yes" is not enough. Anything else (including end of input)
// means no.
func confirm(in *bufio.Reader, out io.Writer, steps int, word string) bool {
	if word == "" {
		fmt.Fprintf(out, "\nRun these %d step(s) on this machine? Type yes to continue: ", steps)
	} else {
		fmt.Fprintf(out, "\nRun these %d step(s) on this machine? This deletes data and cannot be undone. Type %s to continue: ", steps, word)
	}
	answer, _ := in.ReadString('\n')
	answer = strings.TrimSpace(answer)
	if word != "" {
		return answer == word
	}
	answer = strings.ToLower(answer)
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
