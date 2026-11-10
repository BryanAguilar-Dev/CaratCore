package service

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"runtime"

	"github.com/BryanAguilar-Dev/CaratCore/internal/catalog"
	"github.com/BryanAguilar-Dev/CaratCore/internal/ports"
)

// Detect inspects the real machine for s. It only reads: `docker compose
// version` does not contact the Docker daemon, so it works without sudo.
func Detect(ctx context.Context, s catalog.Service) (Host, error) {
	h := Host{Linux: runtime.GOOS == "linux"}
	if !h.Linux {
		return h, nil // DeployPlan reports the platform as unsupported.
	}

	if _, err := exec.LookPath("docker"); err == nil {
		h.DockerFound = true
		h.ComposeFound = exec.CommandContext(ctx, "docker", "compose", "version").Run() == nil
	}

	var err error
	if h.PortsInUse, err = ports.InUse(os.ReadFile); err != nil {
		return h, err
	}

	switch _, err := os.Stat(Dir(s)); {
	case err == nil:
		h.DirExists = true
	case !errors.Is(err, fs.ErrNotExist):
		return h, fmt.Errorf("checking %s: %w", Dir(s), err)
	}

	data, err := os.ReadFile(ComposePath(s))
	switch {
	case err == nil:
		h.ExistingCompose = string(data)
	case !errors.Is(err, fs.ErrNotExist):
		return h, fmt.Errorf("reading %s: %w", ComposePath(s), err)
	}
	return h, nil
}
