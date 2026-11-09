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
	"github.com/BryanAguilar-Dev/CaratCore/internal/service"
)

// checkPorts reports whether the ports of the catalog services are free. A
// busy port of a service CaratCore deployed (its compose.yaml exists) is
// expected, so it is OK. Any other busy port is a WARN, not a FAIL: it only
// blocks deploying that service.
//
// doctor cannot confirm that the program on the port is that service's
// container: asking Docker needs sudo (D17), and doctor never uses it.
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
	var busy, deployed []string
	seen := map[string]bool{}
	for _, s := range services {
		_, err := env.ReadFile(service.ComposePath(s))
		isDeployed := err == nil
		var mine []string
		for _, p := range s.Ports {
			if n := strconv.Itoa(p.Port); !seen[n] {
				seen[n] = true
				numbers = append(numbers, n)
			}
			key := ports.Key(p.Port, p.Protocol)
			switch {
			case !inUse[key]:
			case isDeployed:
				mine = append(mine, key)
			default:
				busy = append(busy, fmt.Sprintf("%s (%s)", key, s.Name))
			}
		}
		if len(mine) > 0 {
			deployed = append(deployed, fmt.Sprintf("%s (%s)", s.Name, strings.Join(mine, ", ")))
		}
	}
	switch {
	case len(busy) > 0:
		return newCheck("Ports", StatusWarn, "in use by another program: %s", strings.Join(busy, ", "))
	case len(deployed) > 0:
		return newCheck("Ports", StatusOK, "in use by services CaratCore deployed: %s", strings.Join(deployed, "; "))
	default:
		return newCheck("Ports", StatusOK, "%s are free", strings.Join(numbers, ", "))
	}
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
