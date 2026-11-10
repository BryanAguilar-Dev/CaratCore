package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/BryanAguilar-Dev/CaratCore/internal/catalog"
	"github.com/BryanAguilar-Dev/CaratCore/internal/service"
)

const uninstallUsage = "usage: caratcore uninstall <adguard-home|uptime-kuma> [--purge] [--dry-run [--json]]"

// runUninstall handles `caratcore uninstall <service> [flags]`.
func runUninstall(args []string) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(os.Stderr, uninstallUsage)
		return 2
	}
	target := args[0]

	flags := flag.NewFlagSet("uninstall "+target, flag.ContinueOnError)
	purge := flags.Bool("purge", false, "also delete the service folder and its data (cannot be undone)")
	dryRun := flags.Bool("dry-run", false, "show the plan without executing anything")
	asJSON := flags.Bool("json", false, "print the plan as JSON (only with --dry-run)")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if *asJSON && !*dryRun {
		fmt.Fprintln(os.Stderr, "--json only works with --dry-run: the plan to execute is shown as text")
		return 2
	}
	if target == "docker" {
		fmt.Fprintln(os.Stderr, "CaratCore does not uninstall Docker: other programs on this machine may use it. Docker's guide for Debian explains how: https://docs.docker.com/engine/install/debian/")
		return 1
	}

	s, err := catalog.Get(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n%s\n", err, uninstallUsage)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	host, err := service.Detect(ctx, s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	p, err := service.RemovePlan(s, host, *purge)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot plan the removal of %s: %v\n", s.Name, err)
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
	if *dryRun || len(p.Steps) == 0 {
		return 0
	}
	word := ""
	if *purge {
		word = s.ID
	}
	return execute(p, service.RemoveNextSteps(s, *purge), word)
}
