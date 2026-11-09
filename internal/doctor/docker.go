package doctor

import (
	"context"
	"strings"
)

func checkDocker(ctx context.Context, env Env) []Check {
	if _, err := env.LookPath("docker"); err != nil {
		return []Check{
			newCheck("Docker CLI", StatusInfo, "not installed (install it with: caratcore install docker)"),
			newCheck("Docker Engine", StatusInfo, "not checked: needs the Docker CLI"),
			newCheck("Docker Compose", StatusInfo, "not checked: needs the Docker CLI"),
		}
	}
	return []Check{
		checkDockerCLI(ctx, env),
		checkDockerEngine(ctx, env),
		checkDockerCompose(ctx, env),
	}
}

func checkDockerCLI(ctx context.Context, env Env) Check {
	out, err := env.Run(ctx, "docker", "--version")
	if err != nil {
		return newCheck("Docker CLI", StatusWarn, "found, but `docker --version` failed: %v", err)
	}
	if version := parseDockerVersion(out); version != "" {
		return newCheck("Docker CLI", StatusOK, "available (version %s)", version)
	}
	return newCheck("Docker CLI", StatusOK, "available")
}

func checkDockerEngine(ctx context.Context, env Env) Check {
	out, err := env.Run(ctx, "docker", "version", "--format", "{{.Server.Version}}")
	if err != nil {
		if strings.Contains(strings.ToLower(out), "permission denied") {
			// Expected: CaratCore uses Docker through sudo and does not add
			// users to the docker group, which would give them root (D17).
			return newCheck("Docker Engine", StatusInfo, "not checked: this user cannot use Docker without sudo (CaratCore uses Docker with sudo)")
		}
		return newCheck("Docker Engine", StatusWarn, "the Engine does not respond (is the docker service running?)")
	}
	return newCheck("Docker Engine", StatusOK, "ready (version %s)", strings.TrimSpace(out))
}

func checkDockerCompose(ctx context.Context, env Env) Check {
	out, err := env.Run(ctx, "docker", "compose", "version", "--short")
	if err != nil {
		return newCheck("Docker Compose", StatusInfo, "plugin not found")
	}
	return newCheck("Docker Compose", StatusOK, "available (version %s)", strings.TrimSpace(out))
}

// parseDockerVersion extracts "29.9.1" from "Docker version 29.9.1, build abc".
func parseDockerVersion(out string) string {
	_, after, found := strings.Cut(out, "version ")
	if !found {
		return ""
	}
	version, _, _ := strings.Cut(after, ",")
	return strings.TrimSpace(version)
}
