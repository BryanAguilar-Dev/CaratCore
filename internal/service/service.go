// Package service builds the plans that deploy a catalog service with Docker
// Compose (create its data folders, write its compose.yaml, download the image
// and start it) and that remove it.
package service

import (
	"bytes"
	"errors"
	"fmt"
	"path"
	"strings"
	"text/template"

	"github.com/BryanAguilar-Dev/CaratCore/internal/catalog"
	"github.com/BryanAguilar-Dev/CaratCore/internal/plan"
	"github.com/BryanAguilar-Dev/CaratCore/internal/ports"
)

// BaseDir holds one folder per deployed service: its compose.yaml and its
// data, all in one place the user can find, read and back up.
const BaseDir = "/opt/caratcore"

// Dir returns the folder of a service, e.g. /opt/caratcore/uptime-kuma.
func Dir(s catalog.Service) string { return path.Join(BaseDir, s.ID) }

// ComposePath returns where the service's compose.yaml lives.
func ComposePath(s catalog.Service) string { return path.Join(Dir(s), "compose.yaml") }

// composeCmd returns a `docker compose` command on the service's compose.yaml.
func composeCmd(s catalog.Service, args ...string) []string {
	return append([]string{"docker", "compose", "--file", ComposePath(s)}, args...)
}

// Host is what the deploy plan needs to know about the machine.
type Host struct {
	Linux        bool
	DockerFound  bool
	ComposeFound bool
	// PortsInUse has keys like "53/udp" (see package ports).
	PortsInUse map[string]bool
	// ExistingCompose is the content of the service's compose.yaml, or ""
	// when it does not exist.
	ExistingCompose string
	// DirExists is true when the service folder exists, even without a
	// compose.yaml (for example, a deployment interrupted after step 1).
	DirExists bool
	// State is the service's caratcore.json, or nil when it does not exist
	// or cannot be read.
	State *State
	// Backups are the copies of the service folder made by update.
	Backups []string
}

// DeployPlan returns the steps that deploy s on h. version is the running
// CaratCore version, recorded in the state file. It returns an error when
// CaratCore should not touch the system: no Docker, ports taken by something
// else, or a compose.yaml that this version did not write.
func DeployPlan(s catalog.Service, h Host, version string) (plan.Plan, error) {
	p := plan.Plan{Title: "Deploy " + s.Name}

	if !h.Linux {
		return p, errors.New("unsupported platform: services are deployed on Linux only")
	}
	if !h.DockerFound || !h.ComposeFound {
		return p, errors.New("Docker and Docker Compose are needed first: run `caratcore install docker`")
	}

	compose, err := ComposeFile(s)
	if err != nil {
		return p, err
	}
	switch composeOrigin(h, compose) {
	case originNone:
		// First deployment: the ports must be free.
		if busy := busyPorts(s, h.PortsInUse); len(busy) > 0 {
			return p, fmt.Errorf("ports already in use by another program: %s; free them and try again", strings.Join(busy, ", "))
		}
	case originThisVersion:
		// Same file CaratCore would write: a previous run was interrupted or
		// the service is already running. Every step is safe to repeat, and
		// the ports are probably taken by the service itself.
	case originOtherVersion:
		return p, fmt.Errorf("%s was written by CaratCore %s with the image %s; this version deploys %s. `install` does not replace it; `caratcore update %s` does, copying the service folder first",
			ComposePath(s), h.State.CaratCoreVersion, h.State.Image, s.Image, s.ID)
	default:
		return p, fmt.Errorf("%s exists and is not the file this CaratCore version would write; %s. CaratCore does not change it. Review it or move it away to deploy again",
			ComposePath(s), unknownOriginReason(h, s))
	}
	state, err := stateFile(State{CaratCoreVersion: version, Image: s.Image, ComposeSHA256: hashCompose(compose)})
	if err != nil {
		return p, fmt.Errorf("generating %s: %w", StatePath(s), err)
	}

	dataDirs := make([]string, len(s.Volumes))
	for i, v := range s.Volumes {
		dataDirs[i] = path.Join(Dir(s), v.Name)
	}
	if len(dataDirs) > 0 {
		p.Steps = append(p.Steps, plan.Step{
			ID:    "create-data-dirs",
			Title: "Create the data folders in " + Dir(s),
			Reason: "the service keeps its data on this machine, outside the container, so it survives updates and can be backed up; " +
				"only root can read it because it may hold password hashes",
			Learn: "Files written inside a container are lost when the container is replaced, for example to update it. " +
				"A bind mount links a folder of this machine into the container, so what the service saves there stays here. " +
				"Mode 0700 means only the owner (root) can open the folder",
			Command: append([]string{"install", "-d", "-m", "0700"}, dataDirs...),
			Root:    true,
		})
	}
	p.Steps = append(p.Steps,
		plan.Step{
			ID:     "write-compose",
			Title:  "Write " + ComposePath(s),
			Reason: "it describes the container (pinned image version, ports and data folders) in a file you can read and edit",
			Learn: "Docker Compose describes containers in a YAML file instead of a long `docker run` command. " +
				"In `ports`, \"3000:3000\" means host port : container port; in `volumes`, host folder : container folder",
			File: &plan.File{Path: ComposePath(s), Content: compose, Mode: 0o644},
			Root: true,
		},
		plan.Step{
			ID:    "write-state",
			Title: "Write " + StatePath(s),
			Reason: "records which CaratCore version wrote compose.yaml and a fingerprint of it, " +
				"so a later version can tell its own file from one you edited and update it safely",
			Learn: "A SHA-256 hash is a short fingerprint of a file: changing a single character changes it completely. " +
				"Comparing fingerprints tells whether compose.yaml changed without keeping a second copy of it",
			File: &plan.File{Path: StatePath(s), Content: state, Mode: 0o644},
			Root: true,
		},
		plan.Step{
			ID:     "pull-image",
			Title:  "Download the image " + s.Image,
			Reason: "downloading first means a network problem stops here, before anything is started",
			Learn: "An image is a read-only package with the program and everything it needs; a container is a running copy of it. " +
				"The text after `:` is the tag (the version): CaratCore pins it, so it always installs the same software",
			Command: composeCmd(s, "pull"),
			Root:    true,
		},
		plan.Step{
			ID:     "start",
			Title:  "Start " + s.Name,
			Reason: "runs the container in the background; it restarts by itself after a reboot or a crash",
			Learn: "`--detach` returns the terminal to you while the container keeps running. " +
				"`restart: unless-stopped` in compose.yaml tells Docker to start it again after a reboot or a crash, " +
				"unless you stopped it yourself",
			Command: composeCmd(s, "up", "--detach"),
			Root:    true,
		},
	)
	return p, nil
}

// unknownOriginReason explains why CaratCore cannot vouch for a compose.yaml.
func unknownOriginReason(h Host, s catalog.Service) string {
	if h.State == nil {
		return "there is no valid " + StatePath(s) + " to tell which CaratCore version wrote it, so it may have been edited by hand"
	}
	return "it was edited after CaratCore " + h.State.CaratCoreVersion + " wrote it"
}

func busyPorts(s catalog.Service, inUse map[string]bool) []string {
	var busy []string
	for _, p := range s.Ports {
		if key := ports.Key(p.Port, p.Protocol); inUse[key] {
			busy = append(busy, key)
		}
	}
	return busy
}

// composeTemplate writes host paths in full so the file reads on its own.
// Catalog validation guarantees values cannot break the YAML.
var composeTemplate = template.Must(template.New("compose").Parse(`# Generated by CaratCore. Do not edit if you want CaratCore to manage it.
# {{.Service.Name}}: {{.Service.Description}}
name: caratcore-{{.Service.ID}}
services:
  {{.Service.ID}}:
    image: {{.Service.Image}}
    container_name: caratcore-{{.Service.ID}}
    restart: unless-stopped
    ports:
{{- range .Service.Ports}}
      - "{{.Port}}:{{.Port}}/{{.Protocol}}" # {{.Purpose}}
{{- end}}
{{- if .Service.Volumes}}
    volumes:
{{- range .Service.Volumes}}
      - {{$.Dir}}/{{.Name}}:{{.Target}} # {{.Purpose}}
{{- end}}
{{- end}}
`))

// ComposeFile returns the compose.yaml for s.
func ComposeFile(s catalog.Service) (string, error) {
	var buf bytes.Buffer
	err := composeTemplate.Execute(&buf, struct {
		Service catalog.Service
		Dir     string
	}{s, Dir(s)})
	if err != nil {
		return "", fmt.Errorf("generating compose.yaml: %w", err)
	}
	return buf.String(), nil
}
