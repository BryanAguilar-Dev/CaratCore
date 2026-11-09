package doctor

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	gib = 1 << 30
	mib = 1 << 20

	// Starting thresholds. They are guesses and must be adjusted by measuring
	// what AdGuard Home, Uptime Kuma and Docker really use.
	warnAvailableMemory = 1 * gib
	failAvailableMemory = 512 * mib
	warnFreeDisk        = 10 * gib
	failFreeDisk        = 3 * gib
)

func checkPlatform(env Env) Check {
	const name = "Platform"
	if env.GOOS != "linux" {
		return newCheck(name, StatusFail, "%s/%s is not supported: CaratCore targets Linux", env.GOOS, env.GOARCH)
	}
	data, err := env.ReadFile("/etc/os-release")
	if err != nil {
		return newCheck(name, StatusWarn, "Linux detected, but /etc/os-release could not be read")
	}
	release := parseOSRelease(string(data))
	pretty := release["PRETTY_NAME"]
	if pretty == "" {
		pretty = "Linux"
	}
	if release["ID"] == "debian" && release["VERSION_ID"] == "13" && env.GOARCH == "amd64" {
		return newCheck(name, StatusOK, "%s %s: supported target", pretty, env.GOARCH)
	}
	return newCheck(name, StatusWarn, "%s %s: not tested (initial target is Debian 13 amd64)", pretty, env.GOARCH)
}

func checkCPU(env Env) Check {
	if env.CPUs < 2 {
		return newCheck("CPU", StatusWarn, "%d logical CPU(s): services may run slowly", env.CPUs)
	}
	return newCheck("CPU", StatusOK, "%d logical CPU(s)", env.CPUs)
}

func checkMemory(env Env) Check {
	data, err := env.ReadFile("/proc/meminfo")
	if err != nil {
		return newCheck("Memory", StatusWarn, "could not read /proc/meminfo: %v", err)
	}
	total, available, err := parseMeminfo(string(data))
	if err != nil {
		return newCheck("Memory", StatusWarn, "could not parse /proc/meminfo: %v", err)
	}
	status := StatusOK
	switch {
	case available < failAvailableMemory:
		status = StatusFail
	case available < warnAvailableMemory:
		status = StatusWarn
	}
	return newCheck("Memory", status, "%s available of %s", formatBytes(available), formatBytes(total))
}

func checkDisk(env Env) Check {
	free, total, err := env.DiskUsage("/")
	if err != nil {
		return newCheck("Disk", StatusWarn, "could not read disk usage of /: %v", err)
	}
	status := StatusOK
	switch {
	case free < failFreeDisk:
		status = StatusFail
	case free < warnFreeDisk:
		status = StatusWarn
	}
	return newCheck("Disk", status, "%s free of %s on /", formatBytes(free), formatBytes(total))
}

// parseOSRelease reads the KEY=value lines of /etc/os-release.
func parseOSRelease(data string) map[string]string {
	values := map[string]string{}
	for _, line := range strings.Split(data, "\n") {
		key, value, found := strings.Cut(strings.TrimSpace(line), "=")
		if !found || strings.HasPrefix(key, "#") {
			continue
		}
		values[key] = strings.Trim(value, `"'`)
	}
	return values
}

// parseMeminfo returns MemTotal and MemAvailable of /proc/meminfo in bytes.
// It reads the file directly instead of parsing the output of `free`, whose
// text depends on the system language.
func parseMeminfo(data string) (total, available uint64, err error) {
	for _, line := range strings.Split(data, "\n") {
		key, rest, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			continue
		}
		kib, convErr := strconv.ParseUint(fields[0], 10, 64)
		if convErr != nil {
			continue
		}
		switch key {
		case "MemTotal":
			total = kib * 1024
		case "MemAvailable":
			available = kib * 1024
		}
	}
	if total == 0 || available == 0 {
		return 0, 0, errors.New("MemTotal or MemAvailable is missing")
	}
	return total, available, nil
}

func formatBytes(n uint64) string {
	return fmt.Sprintf("%.1f GiB", float64(n)/gib)
}
