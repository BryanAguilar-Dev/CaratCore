package service

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

var updateTime = time.Date(2026, 11, 12, 15, 30, 0, 0, time.UTC)

func updateSteps(t *testing.T, h Host) []string {
	t.Helper()
	p, err := UpdatePlan(mustGet(t, "uptime-kuma"), h, "v0.3.0", updateTime)
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

func TestUpdatePlanSteps(t *testing.T) {
	s := mustGet(t, "uptime-kuma")
	compose, err := ComposeFile(s)
	if err != nil {
		t.Fatal(err)
	}
	current := deployedHost(t, "uptime-kuma")
	current.State = &State{CaratCoreVersion: "v0.3.0", Image: s.Image, ComposeSHA256: hashCompose(compose)}

	tests := []struct {
		name string
		host Host
		want []string
	}{
		{"other version", otherVersionHost(t, "uptime-kuma"),
			[]string{"pull-image", "stop", "copy-data", "write-compose", "write-state", "start"}},
		// An update interrupted after writing compose.yaml leaves this
		// version's file: what is left is starting the container.
		{"this version", current, []string{"start"}},
		// Deployed by v0.1 or v0.2: same file, no state yet.
		{"this version, no state", deployedHost(t, "uptime-kuma"), []string{"write-state", "start"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := updateSteps(t, tt.host); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got steps %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUpdatePlanCommands(t *testing.T) {
	s := mustGet(t, "uptime-kuma")
	p, err := UpdatePlan(s, otherVersionHost(t, "uptime-kuma"), "v0.3.0", updateTime)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string][]string{}
	for _, step := range p.Steps {
		byID[step.ID] = step.Command
	}
	want := map[string][]string{
		"pull-image": {"docker", "pull", s.Image},
		"stop":       {"docker", "compose", "--file", "/opt/caratcore/uptime-kuma/compose.yaml", "stop"},
		"copy-data":  {"cp", "-a", "--", "/opt/caratcore/uptime-kuma", "/opt/caratcore/uptime-kuma.bak-2026-11-12-153000"},
		"start":      {"docker", "compose", "--file", "/opt/caratcore/uptime-kuma/compose.yaml", "up", "--detach"},
	}
	for id, cmd := range want {
		if !reflect.DeepEqual(byID[id], cmd) {
			t.Errorf("%s: got %q, want %q", id, byID[id], cmd)
		}
	}

	var st State
	if err := json.Unmarshal([]byte(p.Steps[4].File.Content), &st); err != nil {
		t.Fatal(err)
	}
	if st.CaratCoreVersion != "v0.3.0" || st.Image != s.Image {
		t.Errorf("state after the update: got %+v", st)
	}
	if !strings.Contains(p.Steps[3].Reason, "v0.2.0") || !strings.Contains(p.Steps[3].Reason, "example/old:1.0") {
		t.Errorf("write-compose must say what it replaces: %q", p.Steps[3].Reason)
	}
}

func TestUpdatePlanRefuses(t *testing.T) {
	notLinux := otherVersionHost(t, "uptime-kuma")
	notLinux.Linux = false
	noDocker := otherVersionHost(t, "uptime-kuma")
	noDocker.DockerFound = false
	edited := otherVersionHost(t, "uptime-kuma")
	edited.ExistingCompose += "# my change\n"

	tests := []struct {
		name    string
		host    Host
		wantErr string
	}{
		{"not linux", notLinux, "unsupported platform"},
		{"no docker", noDocker, "caratcore install docker"},
		{"not installed", readyHost(), "caratcore install uptime-kuma"},
		{"edited", edited, "would overwrite your changes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := UpdatePlan(mustGet(t, "uptime-kuma"), tt.host, "v0.3.0", updateTime)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("got %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateNextSteps(t *testing.T) {
	s := mustGet(t, "uptime-kuma")
	if got := UpdateNextSteps(s, ""); len(got) != 1 {
		t.Errorf("without a copy there is nothing to go back to: %q", got)
	}
	backup := BackupDir(s, updateTime)
	got := strings.Join(UpdateNextSteps(s, backup), "\n")
	for _, want := range []string{"sudo mv " + backup + " /opt/caratcore/uptime-kuma", "sudo rm -r " + backup} {
		if !strings.Contains(got, want) {
			t.Errorf("next steps miss %q:\n%s", want, got)
		}
	}
}

func TestRemoveNextStepsMentionsBackups(t *testing.T) {
	h := deployedHost(t, "uptime-kuma")
	h.Backups = []string{"/opt/caratcore/uptime-kuma.bak-2026-11-12-153000"}
	got := strings.Join(RemoveNextSteps(mustGet(t, "uptime-kuma"), h, true), "\n")
	if !strings.Contains(got, h.Backups[0]) {
		t.Errorf("--purge does not delete the copies, so it must say where they are:\n%s", got)
	}
}

// Stopping AdGuard Home to copy its data leaves the network without DNS for
// a moment: the plan must say so, as uninstall does.
func TestUpdatePlanWarnsAboutDNS(t *testing.T) {
	p, err := UpdatePlan(mustGet(t, "adguard-home"), otherVersionHost(t, "adguard-home"), "v0.3.0", updateTime)
	if err != nil {
		t.Fatal(err)
	}
	if p.Steps[1].ID != "stop" || !strings.Contains(p.Steps[1].Reason, "DNS") {
		t.Errorf("stop step must warn about DNS: %+v", p.Steps[1])
	}
}

// --guided shows Learn before each step: every update step explains its
// idea, as the deploy plan does (found in the v0.3.0 VM test, M1).
func TestUpdatePlanLearn(t *testing.T) {
	p, err := UpdatePlan(mustGet(t, "uptime-kuma"), otherVersionHost(t, "uptime-kuma"), "v0.3.0", updateTime)
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range p.Steps {
		if step.Learn == "" {
			t.Errorf("step %s has no Learn", step.ID)
		}
	}
}
