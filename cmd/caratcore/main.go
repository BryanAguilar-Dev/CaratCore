// Command caratcore is the CaratCore command line interface.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/BryanAguilar-Dev/CaratCore/internal/doctor"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		printUsage(os.Stdout)
		return 0
	}

	switch args[0] {
	case "doctor":
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		report := doctor.Run(ctx, doctor.DefaultEnv())
		doctor.Render(os.Stdout, report)
		if report.Worst() == doctor.StatusFail {
			return 1
		}
		return 0
	case "install":
		return runInstall(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", args[0])
		printUsage(os.Stderr)
		return 2
	}
}

func printUsage(w *os.File) {
	fmt.Fprintln(w, "Usage: caratcore <command>")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  doctor   check whether this machine can run CaratCore services (read-only)")
	fmt.Fprintln(w, "  install docker --dry-run [--json]")
	fmt.Fprintln(w, "           show the plan to install Docker and Docker Compose")
	fmt.Fprintln(w, "  help     show this help")
}
