package doctor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/BryanAguilar-Dev/CaratCore/internal/catalog"
	"github.com/BryanAguilar-Dev/CaratCore/internal/ports"
)

// checkPorts reports whether the ports of the catalog services are free. A
// port in use may be a service CaratCore already deployed, so it is a WARN,
// not a FAIL.
func checkPorts(env Env) Check {
	services, err := catalog.All()
	if err != nil {
		return newCheck("Ports", StatusWarn, "%v", err)
	}
	inUse, err := ports.InUse(env.ReadFile)
	if err != nil {
		return newCheck("Ports", StatusWarn, "%v", err)
	}

	var numbers []string
	var busy []string
	seen := map[string]bool{}
	for _, s := range services {
		for _, p := range s.Ports {
			if n := strconv.Itoa(p.Port); !seen[n] {
				seen[n] = true
				numbers = append(numbers, n)
			}
			if key := ports.Key(p.Port, p.Protocol); inUse[key] {
				busy = append(busy, fmt.Sprintf("%s (%s)", key, s.Name))
			}
		}
	}
	if len(busy) == 0 {
		return newCheck("Ports", StatusOK, "%s are free", strings.Join(numbers, ", "))
	}
	return newCheck("Ports", StatusWarn, "in use: %s", strings.Join(busy, ", "))
}

func checkInternet(ctx context.Context, env Env) Check {
	const address = "download.docker.com:443"
	if err := env.Dial(ctx, address); err != nil {
		return newCheck("Internet", StatusWarn, "cannot reach %s (needed to install Docker): %v", address, err)
	}
	return newCheck("Internet", StatusOK, "%s is reachable", address)
}

func checkTailscale(ctx context.Context, env Env) Check {
	const name = "Tailscale"
	if _, err := env.LookPath("tailscale"); err != nil {
		return newCheck(name, StatusInfo, "not installed (optional: used for remote access)")
	}
	out, runErr := env.Run(ctx, "tailscale", "status", "--json")
	state, err := parseTailscaleState(out)
	if err != nil {
		if runErr != nil {
			return newCheck(name, StatusWarn, "installed, but its status could not be read: %v", runErr)
		}
		return newCheck(name, StatusWarn, "installed, but its status output was not understood")
	}
	if state == "Running" {
		return newCheck(name, StatusOK, "connected")
	}
	return newCheck(name, StatusWarn, "installed but not connected (state: %s)", state)
}

// parseTailscaleState reads only BackendState from `tailscale status --json`.
// The output also holds device names and addresses, which doctor ignores.
func parseTailscaleState(out string) (string, error) {
	start := strings.Index(out, "{")
	end := strings.LastIndex(out, "}")
	if start < 0 || end < start {
		return "", errors.New("no JSON found")
	}
	var status struct {
		BackendState string `json:"BackendState"`
	}
	if err := json.Unmarshal([]byte(out[start:end+1]), &status); err != nil {
		return "", err
	}
	if status.BackendState == "" {
		return "", errors.New("BackendState is missing")
	}
	return status.BackendState, nil
}
