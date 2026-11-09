package catalog

import (
	"strings"
	"testing"
)

// The embedded catalog must always be valid: this test is what stops a bad
// edit of services.json from reaching a release.
func TestEmbeddedCatalogIsValid(t *testing.T) {
	services, err := All()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"adguard-home", "uptime-kuma"} {
		if _, err := Get(id); err != nil {
			t.Errorf("%s: %v", id, err)
		}
	}
	if len(services) != 2 {
		t.Errorf("v1 has exactly 2 services (D20), got %d", len(services))
	}
}

func TestGetUnknown(t *testing.T) {
	if _, err := Get("nextcloud"); err == nil {
		t.Fatal("want an error for an unknown service")
	}
}

func TestParseRejects(t *testing.T) {
	valid := `{"id":"x","name":"X","description":"d","image":"a/b:1.0",
		"ports":[{"port":80,"protocol":"tcp","purpose":"web"}],
		"volumes":[{"name":"data","target":"/data","purpose":"p"}],"next_steps":["s"]}`
	tests := []struct {
		name, json, wantErr string
	}{
		{"valid", "[" + valid + "]", ""},
		{"latest image", "[" + strings.Replace(valid, "a/b:1.0", "a/b:latest", 1) + "]", "pinned"},
		{"unpinned image", "[" + strings.Replace(valid, "a/b:1.0", "a/b", 1) + "]", "pinned"},
		{"bad id", "[" + strings.Replace(valid, `"id":"x"`, `"id":"X Y"`, 1) + "]", "id must be"},
		{"bad protocol", "[" + strings.Replace(valid, "tcp", "sctp", 1) + "]", "invalid port"},
		{"relative target", "[" + strings.Replace(valid, `"/data"`, `"data"`, 1) + "]", "invalid volume"},
		{"newline in purpose", "[" + strings.Replace(valid, `"web"`, `"web\nx: y"`, 1) + "]", "invalid port"},
		{"no next steps", "[" + strings.Replace(valid, `["s"]`, `[]`, 1) + "]", "next steps"},
		{"duplicate", "[" + valid + "," + valid + "]", "duplicate"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parse([]byte(tt.json))
			if tt.wantErr == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("got %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}
