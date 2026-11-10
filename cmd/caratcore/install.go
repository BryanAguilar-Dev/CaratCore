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
)

const installUsage = "usage: caratcore install <docker|adguard-home|uptime-kuma> [--dry-run [--json]]"

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
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if *asJSON && !*dryRun {
		fmt.Fprintln(os.Stderr, "--json only works with --dry-run: the plan to execute is shown as text")
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
			nextSteps = s.NextSteps
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
	return execute(p, nextSteps, "")
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
// See confirm for word.
func execute(p plan.Plan, nextSteps []string, word string) int {
	if !confirm(os.Stdin, os.Stdout, len(p.Steps), word) {
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
		fmt.Println("\nDone. Next:")
		for _, step := range nextSteps {
			fmt.Println("  - " + step)
		}
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

// confirm asks for explicit approval. With word == "", only "yes" or "y"
// approve. Otherwise the plan cannot be undone and only word itself approves,
// so a reflex "yes" is not enough. Anything else (including end of input)
// means no.
func confirm(in io.Reader, out io.Writer, steps int, word string) bool {
	if word == "" {
		fmt.Fprintf(out, "\nRun these %d step(s) on this machine? Type yes to continue: ", steps)
	} else {
		fmt.Fprintf(out, "\nRun these %d step(s) on this machine? This deletes data and cannot be undone. Type %s to continue: ", steps, word)
	}
	answer, _ := bufio.NewReader(in).ReadString('\n')
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
