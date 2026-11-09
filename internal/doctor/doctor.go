package doctor

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
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
	if _, err := env.LookPath("sudo"); err != nil {
		return newCheck(name, StatusWarn, "not root and sudo is not installed: run CaratCore as root, or install sudo as root (apt install sudo) and add this user to the sudo group")
	}
	switch inSudoGroup(env) {
	case groupYes:
		return newCheck(name, StatusOK, "this user is in the sudo group (sudo will ask for your password)")
	case groupNo:
		return newCheck(name, StatusWarn, "sudo is installed but this user is not in the sudo group: run CaratCore as root, or as root run `adduser <user> sudo` and log in again")
	default:
		return newCheck(name, StatusInfo, "sudo is available (not verified that this user may use it)")
	}
}

type groupAnswer int

const (
	groupUnknown groupAnswer = iota
	groupYes
	groupNo
)

// inSudoGroup tells whether this process belongs to the "sudo" group, which
// Debian's default sudoers file allows to use sudo. It is a hint, not proof:
// sudoers can be configured otherwise. Asking sudo itself (sudo -n) would mean
// running sudo, which doctor never does. The process's groups are the ones of
// the login session, as sudo sees them: after adduser, the user must log in
// again.
func inSudoGroup(env Env) groupAnswer {
	if env.Groups == nil {
		return groupUnknown
	}
	data, err := env.ReadFile("/etc/group")
	if err != nil {
		return groupUnknown
	}
	sudoGID := -1
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Split(line, ":")
		if len(fields) >= 3 && fields[0] == "sudo" {
			if gid, err := strconv.Atoi(fields[2]); err == nil {
				sudoGID = gid
			}
		}
	}
	if sudoGID < 0 {
		return groupUnknown
	}
	groups, err := env.Groups()
	if err != nil {
		return groupUnknown
	}
	for _, g := range groups {
		if g == sudoGID {
			return groupYes
		}
	}
	return groupNo
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
