package service

import (
	"reflect"
	"strings"
	"testing"

	"github.com/BryanAguilar-Dev/CaratCore/internal/catalog"
)

func readyHost() Host {
	return Host{Linux: true, DockerFound: true, ComposeFound: true, PortsInUse: map[string]bool{}}
}

func mustGet(t *testing.T, id string) catalog.Service {
	t.Helper()
	s, err := catalog.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestDeployPlanEveryService(t *testing.T) {
	services, err := catalog.All()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range services {
		t.Run(s.ID, func(t *testing.T) {
			p, err := DeployPlan(s, readyHost(), "v0.3.0")
			if err != nil {
				t.Fatal(err)
			}
			if err := p.Validate(); err != nil {
				t.Fatalf("invalid plan: %v", err)
			}
			for _, step := range p.Steps {
				if !step.Root {
					t.Errorf("step %s must run as root: Docker is used only with sudo (D17)", step.ID)
				}
			}
			last := p.Steps[len(p.Steps)-1].Command
			want := []string{"docker", "compose", "--file", "/opt/caratcore/" + s.ID + "/compose.yaml", "up", "--detach"}
			if !reflect.DeepEqual(last, want) {
				t.Errorf("last step: got %q, want %q", last, want)
			}
		})
	}
}

func TestComposeFileAdGuard(t *testing.T) {
	got, err := ComposeFile(mustGet(t, "adguard-home"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"name: caratcore-adguard-home\n",
		"    image: adguard/adguardhome:v0.107.80\n",
		"    restart: unless-stopped\n",
		`      - "53:53/udp" # DNS` + "\n",
		`      - "3000:3000/tcp" # web interface` + "\n",
		"      - /opt/caratcore/adguard-home/conf:/opt/adguardhome/conf # ",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("compose.yaml is missing %q:\n%s", want, got)
		}
	}
}

func TestDeployPlanRefuses(t *testing.T) {
	s := mustGet(t, "adguard-home")
	compose, err := ComposeFile(s)
	if err != nil {
		t.Fatal(err)
	}

	notLinux := readyHost()
	notLinux.Linux = false
	noDocker := readyHost()
	noDocker.DockerFound = false
	noCompose := readyHost()
	noCompose.ComposeFound = false
	busy := readyHost()
	busy.PortsInUse = map[string]bool{"53/udp": true}
	edited := readyHost()
	edited.ExistingCompose = compose + "# my change\n"

	tests := []struct {
		name    string
		host    Host
		wantErr string
	}{
		{"not linux", notLinux, "unsupported platform"},
		{"no docker", noDocker, "caratcore install docker"},
		{"no compose", noCompose, "caratcore install docker"},
		{"port taken", busy, "53/udp"},
		{"edited compose", edited, "does not change it"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DeployPlan(s, tt.host, "v0.3.0")
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("got %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

// Re-running after an interrupted or finished deployment must work even
// though the service itself now holds the ports.
func TestDeployPlanRepeatable(t *testing.T) {
	s := mustGet(t, "adguard-home")
	compose, err := ComposeFile(s)
	if err != nil {
		t.Fatal(err)
	}
	h := readyHost()
	h.ExistingCompose = compose
	h.PortsInUse = map[string]bool{"53/udp": true, "53/tcp": true, "3000/tcp": true}
	if _, err := DeployPlan(s, h, "v0.3.0"); err != nil {
		t.Fatalf("re-running must be allowed: %v", err)
	}
}
