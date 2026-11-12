package service

import "github.com/BryanAguilar-Dev/CaratCore/internal/catalog"

// Status is what CaratCore finds on the machine for a catalog service. It
// comes from the files in /opt/caratcore only: CaratCore does not ask Docker
// whether the container is running, which would need sudo (D17).
type Status struct {
	// Name is one of "not installed", "up to date", "update available" or
	// "edited".
	Name string
	// Detail says what is there and, when something can be done, the
	// command to do it.
	Detail string
}

// StatusOf summarizes h for s, for `caratcore list` (D32).
func StatusOf(s catalog.Service, h Host) (Status, error) {
	compose, err := ComposeFile(s)
	if err != nil {
		return Status{}, err
	}
	switch composeOrigin(h, compose) {
	case originNone:
		if h.DirExists {
			return Status{"not installed", "the folder " + Dir(s) + " is left without compose.yaml; `caratcore install " + s.ID + "` uses it, `caratcore uninstall " + s.ID + " --purge` deletes it"}, nil
		}
		return Status{"not installed", ""}, nil
	case originThisVersion:
		if h.State == nil || h.State.ComposeSHA256 != hashCompose(compose) {
			return Status{"up to date", s.Image + "; no caratcore.json yet: `caratcore update " + s.ID + "` writes it"}, nil
		}
		return Status{"up to date", s.Image}, nil
	case originOtherVersion:
		return Status{"update available", h.State.Image + " -> " + s.Image + ": `caratcore update " + s.ID + "`"}, nil
	default:
		return Status{"edited", "CaratCore cannot vouch for " + ComposePath(s) + " and does not change it"}, nil
	}
}
