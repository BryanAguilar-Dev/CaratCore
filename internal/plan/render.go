package plan

import (
	"fmt"
	"io"
	"strings"
)

// Render writes the plan as plain text for the user to review. It only prints:
// nothing is executed.
func Render(w io.Writer, p Plan) {
	fmt.Fprintf(w, "Plan: %s\n", p.Title)
	if len(p.Steps) == 0 {
		fmt.Fprintln(w, "\nNothing to do.")
		return
	}
	fmt.Fprintf(w, "%d step(s). Nothing has been executed yet.\n", len(p.Steps))
	for i, s := range p.Steps {
		fmt.Fprintf(w, "\n%d. %s\n", i+1, s.Title)
		fmt.Fprintf(w, "   Why: %s\n", s.Reason)
		run := "Run"
		if s.Root {
			run = "Run as root"
		}
		fmt.Fprintf(w, "   %s: %s\n", run, ShellJoin(s.Command))
	}
}

// ShellJoin formats a command so it can be read and, if the user wants,
// copied into a POSIX shell. Arguments with special characters are quoted.
func ShellJoin(args []string) string {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = shellQuote(a)
	}
	return strings.Join(parts, " ")
}

func shellQuote(s string) string {
	if s != "" && !strings.ContainsAny(s, " \t\n'\"\\$`|&;<>()*?[]#~!{}") {
		return s
	}
	// Inside single quotes nothing is special except the quote itself,
	// which is written as: close quote, escaped quote, reopen quote.
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
