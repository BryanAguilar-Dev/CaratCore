// Package doctor inspects the host and reports whether it can run CaratCore
// services. It only reads information: it never installs or changes anything.
package doctor

import "fmt"

// Status is the outcome of a single check. A higher value is more severe.
type Status int

const (
	StatusOK   Status = iota // everything fine
	StatusInfo               // nothing wrong, just information
	StatusWarn               // works, but something deserves attention
	StatusFail               // CaratCore cannot work on this host
)

func (s Status) String() string {
	switch s {
	case StatusOK:
		return "OK"
	case StatusInfo:
		return "INFO"
	case StatusWarn:
		return "WARN"
	case StatusFail:
		return "FAIL"
	}
	return "?"
}

// Check is the result of inspecting one aspect of the host.
type Check struct {
	Name   string
	Status Status
	Detail string
}

// Report is the full result of a doctor run.
type Report struct {
	Host   string
	Checks []Check
}

// Worst returns the most severe status among all checks.
func (r Report) Worst() Status {
	worst := StatusOK
	for _, c := range r.Checks {
		if c.Status > worst {
			worst = c.Status
		}
	}
	return worst
}

// Verdict summarizes the report in one sentence.
func (r Report) Verdict() string {
	switch r.Worst() {
	case StatusFail:
		return "not ready: fix the FAIL items first"
	case StatusWarn:
		return "ready with warnings: review the WARN items"
	default:
		return "ready"
	}
}

func newCheck(name string, status Status, format string, args ...any) Check {
	return Check{Name: name, Status: status, Detail: fmt.Sprintf(format, args...)}
}
