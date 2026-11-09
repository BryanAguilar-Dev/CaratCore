package docker

import (
	"reflect"
	"strings"
	"testing"
)

func freshDebian() Host {
	return Host{OSID: "debian", VersionID: "13", Codename: "trixie", Arch: "amd64"}
}

func TestInstallPlanFreshDebian(t *testing.T) {
	p, err := InstallPlan(freshDebian())
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("generated plan is invalid: %v", err)
	}
	if len(p.Steps) == 0 {
		t.Fatal("expected steps on a fresh Debian")
	}

	var repo string
	for _, s := range p.Steps {
		if s.File != nil && s.File.Path == sourcesPath {
			repo = s.File.Content
		}
	}
	for _, want := range []string{"Suites: trixie\n", "Architectures: amd64\n", "Signed-By: /etc/apt/keyrings/docker.asc\n"} {
		if !strings.Contains(repo, want) {
			t.Errorf("repository file is missing %q:\n%s", want, repo)
		}
	}

	last := p.Steps[len(p.Steps)-1].Command
	if !strings.Contains(strings.Join(last, " "), "docker-compose-plugin") {
		t.Errorf("last step should install the Compose plugin, got %v", last)
	}
}

func TestInstallPlanCases(t *testing.T) {
	tests := []struct {
		name      string
		change    func(*Host)
		wantErr   string // empty: no error
		wantSteps bool
	}{
		{"already installed", func(h *Host) { h.DockerFound, h.ComposeFound = true, true }, "", false},
		{"docker without compose", func(h *Host) { h.DockerFound = true }, "Compose is not", false},
		{"conflicts", func(h *Host) { h.Conflicts = []string{"containerd", "runc"} }, "containerd, runc", false},
		{"debian 12", func(h *Host) { h.VersionID = "12" }, "unsupported platform", false},
		{"arm64", func(h *Host) { h.Arch = "arm64" }, "unsupported platform", false},
		{"windows", func(h *Host) { *h = Host{OSID: "windows", Arch: "amd64"} }, "unsupported platform", false},
		{"no codename", func(h *Host) { h.Codename = "" }, "VERSION_CODENAME", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := freshDebian()
			tt.change(&h)
			p, err := InstallPlan(h)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("got %v, want error containing %q", err, tt.wantErr)
			}
			if got := len(p.Steps) > 0; got != tt.wantSteps {
				t.Fatalf("got %d steps", len(p.Steps))
			}
		})
	}
}

func TestParseInstalled(t *testing.T) {
	out := "containerd ii \nrunc un \ndocker.io rc \npodman-docker ii \n"
	got := parseInstalled(out)
	want := []string{"containerd", "podman-docker"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
