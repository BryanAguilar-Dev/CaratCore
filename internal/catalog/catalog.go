// Package catalog lists the services CaratCore can deploy. The list lives in
// services.json, embedded in the binary, so adding a service means editing
// data, not code.
package catalog

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

//go:embed services.json
var servicesJSON []byte

// Service is one deployable service.
type Service struct {
	// ID names the service in commands and paths, e.g. "adguard-home".
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// Image is a container image with a pinned version, never "latest", so
	// the same CaratCore version always deploys the same software.
	Image     string   `json:"image"`
	Ports     []Port   `json:"ports"`
	Volumes   []Volume `json:"volumes"`
	NextSteps []string `json:"next_steps"`
}

// Port is published on the host with the same number as in the container.
type Port struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"` // "tcp" or "udp"
	Purpose  string `json:"purpose"`
}

// Volume is a folder on the host, mounted in the container, where the
// service keeps its data so it survives updates and can be backed up.
type Volume struct {
	Name    string `json:"name"`   // folder name inside the service directory
	Target  string `json:"target"` // absolute path inside the container
	Purpose string `json:"purpose"`
}

// All returns every service in the catalog, in file order.
func All() ([]Service, error) {
	return parse(servicesJSON)
}

// Get returns the service with the given ID.
func Get(id string) (Service, error) {
	services, err := All()
	if err != nil {
		return Service{}, err
	}
	for _, s := range services {
		if s.ID == id {
			return s, nil
		}
	}
	return Service{}, fmt.Errorf("unknown service %q", id)
}

func parse(data []byte) ([]Service, error) {
	var services []Service
	if err := json.Unmarshal(data, &services); err != nil {
		return nil, fmt.Errorf("reading the service catalog: %w", err)
	}
	seen := map[string]bool{}
	for _, s := range services {
		if err := s.validate(); err != nil {
			return nil, fmt.Errorf("service catalog, %q: %w", s.ID, err)
		}
		if seen[s.ID] {
			return nil, fmt.Errorf("service catalog: duplicate id %q", s.ID)
		}
		seen[s.ID] = true
	}
	return services, nil
}

// safeName is what IDs and volume names may contain: they become paths and
// names in compose.yaml, so nothing that a shell or YAML could misread.
var safeName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// safeText rejects characters that would break a line of compose.yaml.
func safeText(s string) bool {
	return s != "" && !strings.ContainsAny(s, "\n\r\"#")
}

func (s Service) validate() error {
	switch {
	case !safeName.MatchString(s.ID):
		return errors.New("id must be lowercase letters, digits and dashes")
	case s.Name == "" || s.Description == "":
		return errors.New("missing name or description")
	case !safeText(s.Image) || strings.ContainsAny(s.Image, " \t"):
		return errors.New("invalid image")
	case !strings.Contains(s.Image, ":") || strings.HasSuffix(s.Image, ":latest"):
		return errors.New("image must have a pinned version, not latest")
	case len(s.Ports) == 0:
		return errors.New("no ports")
	case len(s.NextSteps) == 0:
		return errors.New("no next steps: the user must know what to do after installing")
	}
	for _, p := range s.Ports {
		if p.Port < 1 || p.Port > 65535 || (p.Protocol != "tcp" && p.Protocol != "udp") || !safeText(p.Purpose) {
			return fmt.Errorf("invalid port %d/%s", p.Port, p.Protocol)
		}
	}
	for _, v := range s.Volumes {
		if !safeName.MatchString(v.Name) || !strings.HasPrefix(v.Target, "/") || !safeText(v.Target) || !safeText(v.Purpose) {
			return fmt.Errorf("invalid volume %q", v.Name)
		}
	}
	return nil
}
