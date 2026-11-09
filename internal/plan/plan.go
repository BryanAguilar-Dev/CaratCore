// Package plan describes what CaratCore would change on a host, step by step,
// and why. A plan is plain data: building or printing it never touches the
// system. Running the steps is the job of another package.
package plan

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
)

// Step is one action CaratCore would take. It does exactly one thing: either
// run Command or write File.
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
	Command []string `json:"command,omitempty"`
	// File is a file to create or replace with the given content.
	File *File `json:"file,omitempty"`
	// Root is true when the step needs administrator privileges.
	Root bool `json:"root"`
}

// File is the content of a file a step writes. Writing it directly (instead
// of `sh -c "echo ... > file"`) keeps steps free of shell quoting problems.
type File struct {
	// Path is absolute, on the target Linux host.
	Path    string      `json:"path"`
	Content string      `json:"content"`
	Mode    fs.FileMode `json:"mode"`
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
		if err := s.validate(); err != nil {
			return fmt.Errorf("step %d (%s): %w", i+1, s.ID, err)
		}
		if seen[s.ID] {
			return fmt.Errorf("step %d (%s): duplicate id", i+1, s.ID)
		}
		seen[s.ID] = true
	}
	return nil
}

func (s Step) validate() error {
	hasCommand := len(s.Command) > 0
	switch {
	case s.ID == "":
		return errors.New("missing id")
	case s.Title == "":
		return errors.New("missing title")
	case s.Reason == "":
		return errors.New("missing reason")
	case hasCommand == (s.File != nil):
		return errors.New("must have either a command or a file, not both or none")
	case hasCommand && s.Command[0] == "":
		return errors.New("empty program name")
	case s.File != nil && !path.IsAbs(s.File.Path):
		// path (not path/filepath) because the target is always Linux, even
		// when CaratCore is built or tested on Windows.
		return fmt.Errorf("file path %q is not absolute", s.File.Path)
	case s.File != nil && s.File.Mode == 0:
		return errors.New("file mode is not set")
	}
	return nil
}
