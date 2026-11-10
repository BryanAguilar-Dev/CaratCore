package service

import (
	"reflect"
	"strings"
	"testing"
)

func deployedHost(t *testing.T, id string) Host {
	t.Helper()
	compose, err := ComposeFile(mustGet(t, id))
	if err != nil {
		t.Fatal(err)
	}
	h := readyHost()
	h.ExistingCompose = compose
	h.DirExists = true
	return h
}

func stepIDs(t *testing.T, h Host, id string, purge bool) []string {
	t.Helper()
	p, err := RemovePlan(mustGet(t, id), h, purge)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("invalid plan: %v", err)
	}
	ids := []string{}
	for _, step := range p.Steps {
		if !step.Root {
			t.Errorf("step %s must run as root (D17)", step.ID)
		}
		ids = append(ids, step.ID)
	}
	return ids
}

func TestRemovePlan(t *testing.T) {
	notDeployed := readyHost()
	leftover := readyHost() // deployment interrupted before compose.yaml
	leftover.DirExists = true
	edited := deployedHost(t, "uptime-kuma")
	edited.ExistingCompose += "# my change\n"

	tests := []struct {
		name  string
		host  Host
		purge bool
		want  []string
	}{
		{"deployed", deployedHost(t, "uptime-kuma"), false, []string{"stop"}},
		{"deployed, purge", deployedHost(t, "uptime-kuma"), true, []string{"stop", "delete-data"}},
		{"not deployed", notDeployed, false, []string{}},
		{"not deployed, purge", notDeployed, true, []string{}},
		{"leftover folder", leftover, false, []string{}},
		{"leftover folder, purge", leftover, true, []string{"delete-data"}},
		{"edited compose, no purge", edited, false, []string{"stop"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stepIDs(t, tt.host, "uptime-kuma", tt.purge); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got steps %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRemovePlanCommands(t *testing.T) {
	p, err := RemovePlan(mustGet(t, "adguard-home"), deployedHost(t, "adguard-home"), true)
	if err != nil {
		t.Fatal(err)
	}
	wantStop := []string{"docker", "compose", "--file", "/opt/caratcore/adguard-home/compose.yaml", "down"}
	if !reflect.DeepEqual(p.Steps[0].Command, wantStop) {
		t.Errorf("stop: got %q, want %q", p.Steps[0].Command, wantStop)
	}
	if !strings.Contains(p.Steps[0].Reason, "DNS") {
		t.Errorf("stopping AdGuard Home must warn about DNS: %q", p.Steps[0].Reason)
	}
	wantDelete := []string{"rm", "-r", "--", "/opt/caratcore/adguard-home"}
	if !reflect.DeepEqual(p.Steps[1].Command, wantDelete) {
		t.Errorf("delete: got %q, want %q", p.Steps[1].Command, wantDelete)
	}
}

func TestRemovePlanRefuses(t *testing.T) {
	notLinux := deployedHost(t, "uptime-kuma")
	notLinux.Linux = false
	noCompose := deployedHost(t, "uptime-kuma")
	noCompose.ComposeFound = false
	edited := deployedHost(t, "uptime-kuma")
	edited.ExistingCompose += "# my change\n"

	tests := []struct {
		name    string
		host    Host
		purge   bool
		wantErr string
	}{
		{"not linux", notLinux, false, "unsupported platform"},
		{"no compose", noCompose, false, "not available"},
		{"edited compose, purge", edited, true, "without --purge"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RemovePlan(mustGet(t, "uptime-kuma"), tt.host, tt.purge)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("got %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}
