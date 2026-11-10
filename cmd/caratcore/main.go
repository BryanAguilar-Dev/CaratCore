// Command caratcore is the CaratCore command line interface.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/BryanAguilar-Dev/CaratCore/internal/doctor"
)

// version is set at build time by the release workflow with
// -ldflags "-X main.version=v0.1.0". Local builds report "dev".
var version = "dev"

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
	case "uninstall":
		return runUninstall(args[1:])
	case "version", "--version":
		fmt.Println("caratcore", version)
		return 0
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
	fmt.Fprintln(w, "  install <docker|adguard-home|uptime-kuma> [--dry-run [--json]]")
	fmt.Fprintln(w, "           show the plan, ask for approval and run it;")
	fmt.Fprintln(w, "           --dry-run only shows the plan")
	fmt.Fprintln(w, "  uninstall <adguard-home|uptime-kuma> [--purge] [--dry-run [--json]]")
	fmt.Fprintln(w, "           stop and remove a service; its data stays unless --purge")
	fmt.Fprintln(w, "  version  show the CaratCore version")
	fmt.Fprintln(w, "  help     show this help")
}
