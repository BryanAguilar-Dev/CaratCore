// Package plan describes what CaratCore would change on a host, step by step,
// and why. A plan is plain data: building or printing it never touches the
// system. Running the steps is the job of another package.
package plan

import (
	"errors"
	"fmt"
)

// Step is one action CaratCore would take.
type Step struct {
	// ID identifies the step inside its plan, e.g. "docker-apt-update".
	// It must be unique so logs and future state can refer to it.
	ID string `json:"id"`
	// Title says what the step does, in a few words.
	Title string `json:"title"`
	// Reason explains why the step is needed. Every step must have one:
	// CaratCore never asks the user to approve something unexplained.
	Reason string `json:"reason"`
	// Command is the program and its arguments, run directly without a shell.
	Command []string `json:"command"`
	// Root is true when the command needs administrator privileges.
	Root bool `json:"root"`
}

// Plan is an ordered list of steps. An empty plan is valid and means there is
// nothing to do (for example, Docker is already installed).
type Plan struct {
	Title string `json:"title"`
	Steps []Step `json:"steps"`
}

// Validate reports the first problem that would make the plan unsafe or
// impossible to explain to the user.
func (p Plan) Validate() error {
	if p.Title == "" {
		return errors.New("plan has no title")
	}
	seen := make(map[string]bool, len(p.Steps))
	for i, s := range p.Steps {
		pos := i + 1
		switch {
		case s.ID == "":
			return fmt.Errorf("step %d: missing id", pos)
		case seen[s.ID]:
			return fmt.Errorf("step %d: duplicate id %q", pos, s.ID)
		case s.Title == "":
			return fmt.Errorf("step %d (%s): missing title", pos, s.ID)
		case s.Reason == "":
			return fmt.Errorf("step %d (%s): missing reason", pos, s.ID)
		case len(s.Command) == 0 || s.Command[0] == "":
			return fmt.Errorf("step %d (%s): missing command", pos, s.ID)
		}
		seen[s.ID] = true
	}
	return nil
}
