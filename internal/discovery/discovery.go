package discovery

import (
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/BryanAguilar-Dev/CaratCore/internal/model"
)

func Scan() model.Computer {

	computer := model.Computer{
		OperatingSystem: runtime.GOOS,
		Architecture:    runtime.GOARCH,
	}
	computer.Distribution, computer.DistributionVersion = readDistribution()
	computer.PlatformSupported = runtime.GOOS == "linux" &&
		runtime.GOARCH == "amd64" &&
		computer.Distribution == "debian" &&
		computer.DistributionVersion == "13"

	hostname, err := os.Hostname()
	if err == nil {
		computer.Hostname = hostname
	} else {
		computer.HostnameError = err.Error()
	}

	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return computer
	}
	computer.DockerCLIAvailable = true

	output, err := exec.Command(dockerPath, "info", "--format", "{{.ServerVersion}}").CombinedOutput()
	if err != nil {
		computer.DockerError = strings.TrimSpace(string(output))
		if computer.DockerError == "" {
			computer.DockerError = err.Error()
		}
		return computer
	}
	computer.DockerEngineAvailable = true
	computer.DockerVersion = strings.TrimSpace(string(output))

	return computer
}

func readDistribution() (string, string) {
	if runtime.GOOS != "linux" {
		return "", ""
	}

	contents, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "", ""
	}

	var distribution, version string
	for _, line := range strings.Split(string(contents), "\n") {
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		value = strings.Trim(value, "\"'")
		switch key {
		case "ID":
			distribution = value
		case "VERSION_ID":
			version = value
		}
	}

	return distribution, version
}
