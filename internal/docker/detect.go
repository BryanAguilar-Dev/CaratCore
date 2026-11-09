package docker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/BryanAguilar-Dev/CaratCore/internal/osrelease"
)

// Detect inspects the real machine and fills a Host. It only reads: it runs
// `docker compose version` and `dpkg-query`, which change nothing.
func Detect(ctx context.Context) (Host, error) {
	// Go's names for amd64 and arm64 match Debian's.
	h := Host{OSID: runtime.GOOS, Arch: runtime.GOARCH}
	if runtime.GOOS != "linux" {
		return h, nil // InstallPlan reports the platform as unsupported.
	}

	data, err := os.ReadFile(osrelease.Path)
	if err != nil {
		return h, fmt.Errorf("reading %s: %w", osrelease.Path, err)
	}
	release := osrelease.Parse(string(data))
	h.OSID = release["ID"]
	h.VersionID = release["VERSION_ID"]
	h.Codename = release["VERSION_CODENAME"]

	if _, err := exec.LookPath("docker"); err == nil {
		h.DockerFound = true
		h.ComposeFound = exec.CommandContext(ctx, "docker", "compose", "version").Run() == nil
	}

	if h.OSID == "debian" {
		h.Conflicts, err = installedPackages(ctx, conflictingPackages)
		if err != nil {
			return h, err
		}
	}
	return h, nil
}

// installedPackages returns which of names are installed, according to dpkg.
func installedPackages(ctx context.Context, names []string) ([]string, error) {
	args := append([]string{"-W", "-f=${Package} ${db:Status-Abbrev}\n"}, names...)
	out, err := exec.CommandContext(ctx, "dpkg-query", args...).Output()
	// dpkg-query exits with 1 when some names are unknown to dpkg, which is
	// normal here. Any other failure means we cannot trust the answer.
	var exitErr *exec.ExitError
	if err != nil && !(errors.As(err, &exitErr) && exitErr.ExitCode() == 1) {
		return nil, fmt.Errorf("checking installed packages with dpkg-query: %w", err)
	}
	return parseInstalled(string(out)), nil
}

// parseInstalled reads "name status" lines from dpkg-query. Status "ii" means
// the package is installed.
func parseInstalled(out string) []string {
	var installed []string
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == "ii" {
			installed = append(installed, fields[0])
		}
	}
	return installed
}
