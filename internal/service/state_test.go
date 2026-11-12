package service

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestDeployPlanWritesState(t *testing.T) {
	s := mustGet(t, "uptime-kuma")
	compose, err := ComposeFile(s)
	if err != nil {
		t.Fatal(err)
	}
	p, err := DeployPlan(s, readyHost(), "v0.3.0")
	if err != nil {
		t.Fatal(err)
	}

	var ids []string
	for _, step := range p.Steps {
		ids = append(ids, step.ID)
	}
	want := []string{"create-data-dirs", "write-compose", "write-state", "pull-image", "start"}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("got steps %q, want %q", ids, want)
	}

	file := p.Steps[2].File
	if file.Path != "/opt/caratcore/uptime-kuma/caratcore.json" {
		t.Errorf("state path: got %q", file.Path)
	}
	var got State
	if err := json.Unmarshal([]byte(file.Content), &got); err != nil {
		t.Fatalf("state is not valid JSON: %v\n%s", err, file.Content)
	}
	wantState := State{CaratCoreVersion: "v0.3.0", Image: s.Image, ComposeSHA256: hashCompose(compose)}
	if got != wantState {
		t.Errorf("got state %+v, want %+v", got, wantState)
	}
}

// otherVersionHost has a compose.yaml written by an older CaratCore with
// another image, unchanged since: its state file matches it.
func otherVersionHost(t *testing.T, id string) Host {
	t.Helper()
	h := deployedHost(t, id)
	h.ExistingCompose = strings.Replace(h.ExistingCompose, mustGet(t, id).Image, "example/old:1.0", 1)
	h.State = &State{CaratCoreVersion: "v0.2.0", Image: "example/old:1.0", ComposeSHA256: hashCompose(h.ExistingCompose)}
	return h
}

func TestDeployPlanComposeOrigin(t *testing.T) {
	s := mustGet(t, "uptime-kuma")

	// Deployed before state files existed (v0.1, v0.2): same compose.yaml,
	// no caratcore.json. Deploying again adopts it.
	adopted := deployedHost(t, "uptime-kuma")

	editedNoState := deployedHost(t, "uptime-kuma")
	editedNoState.ExistingCompose += "# my change\n"

	editedAfter := otherVersionHost(t, "uptime-kuma")
	editedAfter.ExistingCompose += "# my change\n"

	tests := []struct {
		name    string
		host    Host
		wantErr string // "" means the plan must be allowed
	}{
		{"no state, same file", adopted, ""},
		{"other version, unchanged", otherVersionHost(t, "uptime-kuma"), "written by CaratCore v0.2.0 with the image example/old:1.0"},
		{"edited, no state", editedNoState, "no valid /opt/caratcore/uptime-kuma/caratcore.json"},
		{"edited after deploy", editedAfter, "edited after CaratCore v0.2.0 wrote it"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DeployPlan(s, tt.host, "v0.3.0")
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("must be allowed: %v", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Fatalf("got %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestRemovePlanPurgeOtherVersion(t *testing.T) {
	got := stepIDs(t, otherVersionHost(t, "uptime-kuma"), "uptime-kuma", true)
	if want := []string{"stop", "delete-data"}; !reflect.DeepEqual(got, want) {
		t.Errorf("--purge must delete a file another CaratCore version wrote: got steps %q, want %q", got, want)
	}

	edited := otherVersionHost(t, "uptime-kuma")
	edited.ExistingCompose += "# my change\n"
	_, err := RemovePlan(mustGet(t, "uptime-kuma"), edited, true)
	if err == nil || !strings.Contains(err.Error(), "without --purge") {
		t.Fatalf("got %v, want a refusal to purge an edited file", err)
	}
}

func TestParseState(t *testing.T) {
	for _, data := range []string{"", "not json", `{"caratcore_version": "v0.3.0"}`} {
		if st := parseState([]byte(data)); st != nil {
			t.Errorf("parseState(%q) = %+v, want nil", data, st)
		}
	}
	st := parseState([]byte(`{"caratcore_version": "v0.3.0", "image": "a/b:1", "compose_sha256": "abc"}`))
	if st == nil || st.ComposeSHA256 != "abc" {
		t.Errorf("valid state not parsed: %+v", st)
	}
}
