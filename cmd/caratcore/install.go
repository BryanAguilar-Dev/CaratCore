package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/BryanAguilar-Dev/CaratCore/internal/docker"
	"github.com/BryanAguilar-Dev/CaratCore/internal/plan"
)

// runInstall handles `caratcore install <target> [flags]`.
func runInstall(args []string) int {
	if len(args) == 0 || args[0] != "docker" {
		fmt.Fprintln(os.Stderr, "usage: caratcore install docker --dry-run [--json]")
		return 2
	}

	flags := flag.NewFlagSet("install docker", flag.ContinueOnError)
	dryRun := flags.Bool("dry-run", false, "show the plan without executing anything")
	asJSON := flags.Bool("json", false, "print the plan as JSON")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	host, err := docker.Detect(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	p, err := docker.InstallPlan(host)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot plan the Docker installation: %v\n", err)
		return 1
	}
	if err := p.Validate(); err != nil {
		// A bug in CaratCore, not in the user's system.
		fmt.Fprintf(os.Stderr, "internal error: invalid plan: %v\n", err)
		return 1
	}

	if err := printPlan(os.Stdout, p, *asJSON); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	if !*dryRun && len(p.Steps) > 0 {
		fmt.Fprintln(os.Stderr, "\nExecuting plans is not implemented yet. Use --dry-run to review the plan.")
		return 1
	}
	return 0
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
