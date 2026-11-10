// Package docker builds the plan that installs Docker Engine and the Compose
// plugin on a host, following Docker's official instructions for Debian:
// https://docs.docker.com/engine/install/debian/
package docker

import (
	_ "embed"
	"errors"
	"fmt"
	"strings"

	"github.com/BryanAguilar-Dev/CaratCore/internal/plan"
)

// dockerKey is Docker's apt signing key, downloaded from
// https://download.docker.com/linux/debian/gpg and checked against
// keyFingerprint, the fingerprint in Docker's guide. It ships inside the binary
// instead of being downloaded at install time, so the key can be reviewed in
// the source and a test proves its fingerprint (D26).
//
//go:embed docker.asc
var dockerKey string

const keyFingerprint = "9DC8 5822 9FC7 DD38 854A  E2D8 8D81 803C 0EBF CD88"

const (
	repoURL     = "https://download.docker.com/linux/debian"
	keyringDir  = "/etc/apt/keyrings"
	keyPath     = keyringDir + "/docker.asc"
	sourcesPath = "/etc/apt/sources.list.d/docker.sources"
)

// conflictingPackages are distribution packages that Docker's packages
// replace. Docker's guide asks to remove them first; CaratCore does not remove
// anything on its own, so it stops and tells the user instead.
var conflictingPackages = []string{
	"docker.io", "docker-compose", "docker-doc", "podman-docker", "containerd", "runc",
}

// dockerPackages are the packages Docker's guide installs.
var dockerPackages = []string{
	"docker-ce", "docker-ce-cli", "containerd.io", "docker-buildx-plugin", "docker-compose-plugin",
}

// Host is what the install plan needs to know about the machine.
type Host struct {
	OSID      string // ID in /etc/os-release, e.g. "debian"
	VersionID string // VERSION_ID, e.g. "13"
	Codename  string // VERSION_CODENAME, e.g. "trixie"
	Arch      string // Debian architecture name, e.g. "amd64"

	DockerFound  bool     // the docker command exists
	ComposeFound bool     // `docker compose` works
	Conflicts    []string // installed packages from conflictingPackages
}

// InstallPlan returns the steps that install Docker on h. The plan is empty
// when Docker and Compose are already there. It returns an error when
// CaratCore should not touch the system: unsupported platform, conflicting
// packages, or an existing Docker without Compose.
func InstallPlan(h Host) (plan.Plan, error) {
	p := plan.Plan{Title: "Install Docker Engine and Docker Compose"}

	if h.OSID != "debian" || h.VersionID != "13" || h.Arch != "amd64" {
		return p, fmt.Errorf("unsupported platform %q %q %q: only Debian 13 amd64 is supported", h.OSID, h.VersionID, h.Arch)
	}
	if h.Codename == "" {
		return p, errors.New("VERSION_CODENAME is missing in /etc/os-release")
	}
	if h.DockerFound && h.ComposeFound {
		return p, nil
	}
	if h.DockerFound {
		return p, errors.New("Docker is installed but Docker Compose is not: CaratCore does not modify an existing Docker installation; install the Compose plugin yourself")
	}
	if len(h.Conflicts) > 0 {
		return p, fmt.Errorf("these packages conflict with Docker's packages and must be removed first: %s (review them: sudo apt-get remove %s)",
			strings.Join(h.Conflicts, ", "), strings.Join(h.Conflicts, " "))
	}

	p.Steps = []plan.Step{
		{
			ID:     "apt-update",
			Title:  "Refresh the package lists",
			Reason: "apt must know the current versions before installing the prerequisites",
			Learn: "apt keeps a local copy of the list of packages each repository offers. " +
				"`apt-get update` downloads those lists again; it does not install or upgrade anything",
			Command: []string{"apt-get", "update"},
			Root:    true,
		},
		{
			ID:     "install-prerequisites",
			Title:  "Install ca-certificates",
			Reason: "apt needs it to verify the HTTPS connection to Docker's repository",
			Learn: "ca-certificates is the list of certificate authorities this system trusts. " +
				"With it, apt can check that the server it talks to really is download.docker.com",
			Command: []string{"apt-get", "install", "-y", "ca-certificates"},
			Root:    true,
		},
		{
			ID:     "create-keyrings-dir",
			Title:  "Create " + keyringDir,
			Reason: "it is the standard place for third-party repository signing keys",
			Learn: "A key stored here is trusted only by the repositories that name it (with Signed-By), " +
				"unlike the old `apt-key`, which trusted every key for every repository",
			Command: []string{"install", "-m", "0755", "-d", keyringDir},
			Root:    true,
		},
		{
			ID:    "write-docker-key",
			Title: "Write Docker's signing key",
			Reason: "apt uses this key to check that Docker's packages really come from Docker. " +
				"It is built into CaratCore, not downloaded; its fingerprint is " + keyFingerprint +
				" (check it with: gpg --show-keys " + keyPath + ")",
			Learn: "This is the public half of a key pair. Docker signs its repository with the private half, " +
				"which only Docker has, and apt checks the signature with this public half: if anyone altered the packages " +
				"on the way, the check fails and apt refuses them. The fingerprint is a short summary of the key; " +
				"if it matches the one Docker publishes, it is the same key",
			File: &plan.File{
				Path:    keyPath,
				Content: dockerKey,
				Mode:    0o644, // apt reads it as an unprivileged user
			},
			Root: true,
		},
		{
			ID:     "add-docker-repository",
			Title:  "Add Docker's apt repository",
			Reason: "Debian's own docker.io package is older and does not ship the Compose plugin Docker documents",
			Learn: "A repository is a server apt downloads packages from. This file uses the deb822 format, " +
				"one `Field: value` per line. Signed-By ties the repository to Docker's key only: " +
				"no other key on the system can sign packages for it",
			File: &plan.File{
				Path:    sourcesPath,
				Content: sourcesFile(h.Codename, h.Arch),
				Mode:    0o644,
			},
			Root: true,
		},
		{
			ID:      "apt-update-docker",
			Title:   "Refresh the package lists again",
			Reason:  "apt must read the new Docker repository before installing from it",
			Command: []string{"apt-get", "update"},
			Root:    true,
		},
		{
			ID:     "install-docker",
			Title:  "Install Docker Engine, CLI and plugins",
			Reason: "Engine runs the containers; the Compose plugin starts the services CaratCore will deploy",
			Learn: "docker-ce is the Engine, a background service (dockerd) that manages containers; " +
				"docker-ce-cli is the `docker` command you type; containerd.io is the layer below that starts and stops them. " +
				"The buildx plugin builds images, and the compose plugin adds `docker compose`, " +
				"which starts containers described in a file",
			Command: append([]string{"apt-get", "install", "-y"}, dockerPackages...),
			Root:    true,
		},
	}
	return p, nil
}

// sourcesFile returns the apt source in deb822 format, as in Docker's guide.
func sourcesFile(codename, arch string) string {
	return "Types: deb\n" +
		"URIs: " + repoURL + "\n" +
		"Suites: " + codename + "\n" +
		"Components: stable\n" +
		"Architectures: " + arch + "\n" +
		"Signed-By: " + keyPath + "\n"
}
