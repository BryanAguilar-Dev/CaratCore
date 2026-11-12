package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/BryanAguilar-Dev/CaratCore/internal/catalog"
	"github.com/BryanAguilar-Dev/CaratCore/internal/plan"
	"github.com/BryanAguilar-Dev/CaratCore/internal/service"
)

const updateUsage = "usage: caratcore update <adguard-home|uptime-kuma> [--guided] [--dry-run [--json]]"

// runUpdate handles `caratcore update <service> [flags]`.
func runUpdate(args []string) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(os.Stderr, updateUsage)
		return 2
	}
	target := args[0]

	flags := flag.NewFlagSet("update "+target, flag.ContinueOnError)
	dryRun := flags.Bool("dry-run", false, "show the plan without executing anything")
	asJSON := flags.Bool("json", false, "print the plan as JSON (only with --dry-run)")
	guided := flags.Bool("guided", false, "explain each step and wait for Enter before running it")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if !validFlags(*dryRun, *asJSON, *guided) {
		return 2
	}
	if target == "docker" {
		fmt.Fprintln(os.Stderr, "CaratCore does not update Docker: it comes from Docker's apt repository, so `sudo apt update && sudo apt upgrade` updates it with the rest of the system.")
		return 1
	}

	s, err := catalog.Get(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n%s\n", err, updateUsage)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	host, err := service.Detect(ctx, s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	now := time.Now()
	p, err := service.UpdatePlan(s, host, version, now)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot plan the update of %s: %v\n", s.Name, err)
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
	backup := ""
	if hasStep(p, "copy-data") {
		backup = service.BackupDir(s, now)
	}
	return execute(p, service.UpdateNextSteps(s, backup), "", *guided)
}

func hasStep(p plan.Plan, id string) bool {
	for _, step := range p.Steps {
		if step.ID == id {
			return true
		}
	}
	return false
}
