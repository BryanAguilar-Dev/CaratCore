package doctor

import (
	"context"
	"fmt"
	"io"
)

// Run performs every check and returns the report.
func Run(ctx context.Context, env Env) Report {
	host, err := env.Hostname()
	if err != nil {
		host = "unknown"
	}

	checks := []Check{
		checkPlatform(env),
		checkCPU(env),
		checkMemory(env),
		checkDisk(env),
		checkPrivileges(env),
	}
	checks = append(checks, checkDocker(ctx, env)...)
	checks = append(checks,
		checkPorts(env),
		checkInternet(ctx, env),
		checkTailscale(ctx, env),
	)
	return Report{Host: host, Checks: checks}
}

func checkPrivileges(env Env) Check {
	const name = "Privileges"
	if env.Euid == 0 {
		return newCheck(name, StatusOK, "running as root")
	}
	if _, err := env.LookPath("sudo"); err == nil {
		return newCheck(name, StatusInfo, "sudo is available (not verified that this user may use it)")
	}
	return newCheck(name, StatusWarn, "not root and sudo not found: installing packages will not be possible")
}

// Render writes the report as plain text.
func Render(w io.Writer, r Report) {
	fmt.Fprintln(w, "CaratCore doctor")
	fmt.Fprintf(w, "Host: %s\n\n", r.Host)
	for _, c := range r.Checks {
		fmt.Fprintf(w, "[%-4s] %-15s %s\n", c.Status, c.Name, c.Detail)
	}
	fmt.Fprintf(w, "\nVerdict: %s\n", r.Verdict())
}
