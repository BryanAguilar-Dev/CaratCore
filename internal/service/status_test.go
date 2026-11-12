package service

import (
	"strings"
	"testing"
)

func TestStatusOf(t *testing.T) {
	s := mustGet(t, "uptime-kuma")
	compose, err := ComposeFile(s)
	if err != nil {
		t.Fatal(err)
	}
	current := deployedHost(t, "uptime-kuma")
	current.State = &State{CaratCoreVersion: "v0.3.0", Image: s.Image, ComposeSHA256: hashCompose(compose)}
	leftover := readyHost()
	leftover.DirExists = true
	edited := deployedHost(t, "uptime-kuma")
	edited.ExistingCompose += "# my change\n"

	tests := []struct {
		name       string
		host       Host
		wantName   string
		wantDetail string // must be contained; "" means empty
	}{
		{"nothing", readyHost(), "not installed", ""},
		{"leftover folder", leftover, "not installed", "--purge"},
		{"current", current, "up to date", s.Image},
		{"current, no state", deployedHost(t, "uptime-kuma"), "up to date", "no caratcore.json"},
		{"older version", otherVersionHost(t, "uptime-kuma"), "update available", "example/old:1.0 -> " + s.Image},
		{"edited", edited, "edited", "does not change it"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StatusOf(s, tt.host)
			if err != nil {
				t.Fatal(err)
			}
			if got.Name != tt.wantName {
				t.Errorf("name: got %q, want %q", got.Name, tt.wantName)
			}
			if tt.wantDetail == "" && got.Detail != "" || !strings.Contains(got.Detail, tt.wantDetail) {
				t.Errorf("detail: got %q, want %q", got.Detail, tt.wantDetail)
			}
		})
	}
}
