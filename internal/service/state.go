package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path"

	"github.com/BryanAguilar-Dev/CaratCore/internal/catalog"
)

// State records what CaratCore wrote when it deployed a service (D32). It
// keeps a hash of compose.yaml, not a copy: the hash is enough to tell
// whether the file changed since, and a copy could drift from the original.
type State struct {
	// CaratCoreVersion is the version that wrote the files, e.g. "v0.3.0".
	CaratCoreVersion string `json:"caratcore_version"`
	// Image is the image compose.yaml uses.
	Image string `json:"image"`
	// ComposeSHA256 is the SHA-256 of compose.yaml, in hexadecimal.
	ComposeSHA256 string `json:"compose_sha256"`
}

// StatePath returns where the service's state file lives.
func StatePath(s catalog.Service) string { return path.Join(Dir(s), "caratcore.json") }

func hashCompose(compose string) string {
	sum := sha256.Sum256([]byte(compose))
	return hex.EncodeToString(sum[:])
}

func stateFile(st State) (string, error) {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}

// parseState returns nil when data is not a valid state file: an unreadable
// state proves nothing, so the compose.yaml is judged as if it had none.
func parseState(data []byte) *State {
	var st State
	if err := json.Unmarshal(data, &st); err != nil || st.ComposeSHA256 == "" {
		return nil
	}
	return &st
}

// origin says who wrote the compose.yaml found on the host.
type origin int

const (
	originNone         origin = iota // there is no compose.yaml
	originThisVersion                // identical to what this version writes
	originOtherVersion               // written by another CaratCore version, unchanged since
	originUnknown                    // edited after CaratCore wrote it, or no state to tell
)

func composeOrigin(h Host, want string) origin {
	switch {
	case h.ExistingCompose == "":
		return originNone
	case h.ExistingCompose == want:
		return originThisVersion
	case h.State != nil && h.State.ComposeSHA256 == hashCompose(h.ExistingCompose):
		return originOtherVersion
	default:
		return originUnknown
	}
}
