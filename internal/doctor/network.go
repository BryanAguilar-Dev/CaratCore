package doctor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// requiredPorts are the ports the planned services need: DNS for AdGuard Home,
// its first-run setup page, and Uptime Kuma.
var requiredPorts = []int{53, 3000, 3001}

func checkPorts(env Env) Check {
	sources := []struct {
		path  string
		udp   bool
		proto string
	}{
		{"/proc/net/tcp", false, "tcp"},
		{"/proc/net/tcp6", false, "tcp"},
		{"/proc/net/udp", true, "udp"},
		{"/proc/net/udp6", true, "udp"},
	}

	inUse := map[string]bool{}
	readAny := false
	for _, src := range sources {
		data, err := env.ReadFile(src.path)
		if err != nil {
			continue
		}
		readAny = true
		listening := parseListeningPorts(string(data), src.udp)
		for _, port := range requiredPorts {
			if listening[port] {
				inUse[fmt.Sprintf("%d/%s", port, src.proto)] = true
			}
		}
	}
	if !readAny {
		return newCheck("Ports", StatusWarn, "could not read /proc/net to check ports")
	}
	if len(inUse) == 0 {
		return newCheck("Ports", StatusOK, "%s are free", portList())
	}
	names := make([]string, 0, len(inUse))
	for name := range inUse {
		names = append(names, name)
	}
	sort.Strings(names)
	return newCheck("Ports", StatusWarn, "in use: %s (may conflict with AdGuard Home or Uptime Kuma)", strings.Join(names, ", "))
}

func portList() string {
	parts := make([]string, len(requiredPorts))
	for i, port := range requiredPorts {
		parts[i] = strconv.Itoa(port)
	}
	return strings.Join(parts, ", ")
}

// parseListeningPorts returns the local ports that are listening (TCP, state
// 0A) or bound (UDP, state 07) in a /proc/net/{tcp,udp}[6] file. Reading /proc
// needs no root, unlike `ss -p`.
func parseListeningPorts(data string, udp bool) map[int]bool {
	wantState := "0A"
	if udp {
		wantState = "07"
	}
	ports := map[int]bool{}
	for _, line := range strings.Split(data, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 || fields[3] != wantState {
			continue
		}
		_, hexPort, found := strings.Cut(fields[1], ":")
		if !found {
			continue
		}
		port, err := strconv.ParseUint(hexPort, 16, 16)
		if err != nil {
			continue
		}
		ports[int(port)] = true
	}
	return ports
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
