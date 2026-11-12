package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/BryanAguilar-Dev/CaratCore/internal/catalog"
	"github.com/BryanAguilar-Dev/CaratCore/internal/plan"
)

// BackupDir returns the folder where an update copies the service folder
// before changing it, e.g. /opt/caratcore/uptime-kuma.bak-2026-11-12-153000.
// It sits next to the service folder, not inside, so the copy does not
// contain itself and the service folder stays as it was.
func BackupDir(s catalog.Service, t time.Time) string {
	return Dir(s) + ".bak-" + t.Format("2006-01-02-150405")
}

// UpdatePlan returns the steps that move a deployed service to the
// compose.yaml and image of this CaratCore version (D33). now names the copy
// of the data. version is recorded in the state file.
func UpdatePlan(s catalog.Service, h Host, version string, now time.Time) (plan.Plan, error) {
	p := plan.Plan{Title: "Update " + s.Name + " to " + s.Image}

	if !h.Linux {
		return p, errors.New("unsupported platform: services are updated on Linux only")
	}
	if !h.DockerFound || !h.ComposeFound {
		return p, errors.New("Docker and Docker Compose are needed: run `caratcore install docker`")
	}
	compose, err := ComposeFile(s)
	if err != nil {
		return p, err
	}
	state, err := stateFile(State{CaratCoreVersion: version, Image: s.Image, ComposeSHA256: hashCompose(compose)})
	if err != nil {
		return p, fmt.Errorf("generating %s: %w", StatePath(s), err)
	}
	writeState := plan.Step{
		ID:    "write-state",
		Title: "Write " + StatePath(s),
		Reason: "records which CaratCore version wrote compose.yaml and a fingerprint of it, " +
			"so CaratCore can tell its own file from one you edited",
		File: &plan.File{Path: StatePath(s), Content: state, Mode: 0o644},
		Root: true,
	}
	start := plan.Step{
		ID:     "start",
		Title:  "Start " + s.Name + " with " + s.Image,
		Reason: "Compose replaces the container when its image changed and leaves it alone otherwise; the data folders are kept",
		Learn: "`up --detach` compares the running container with compose.yaml: if the image changed, it removes the old " +
			"container and creates a new one from the new image. The data lives in bind-mounted folders, so it is not lost",
		Command: composeCmd(s, "up", "--detach"),
		Root:    true,
	}

	switch composeOrigin(h, compose) {
	case originNone:
		return p, fmt.Errorf("%s is not installed: run `caratcore install %s`", s.Name, s.ID)
	case originUnknown:
		return p, fmt.Errorf("CaratCore cannot vouch for %s: %s. Updating would overwrite your changes, so CaratCore does not change it. Review it, or move it away and run `caratcore install %s`",
			ComposePath(s), unknownOriginReason(h, s), s.ID)
	case originThisVersion:
		// Already this version's file. Making sure the container runs it is
		// what is left after an update interrupted between writing
		// compose.yaml and starting, and it is harmless otherwise.
		if h.State == nil || h.State.ComposeSHA256 != hashCompose(compose) {
			p.Steps = append(p.Steps, writeState)
		}
		p.Steps = append(p.Steps, start)
		return p, nil
	}

	backup := BackupDir(s, now)
	stopReason := "a service that is writing its data while it is copied can leave a broken copy (half a database)"
	if s.RemoveWarning != "" {
		stopReason += ". Until the last step starts it again: " + s.RemoveWarning
	}
	p.Steps = append(p.Steps,
		plan.Step{
			ID:    "pull-image",
			Title: "Download the image " + s.Image,
			Reason: "downloading first means a network problem stops here, while " + s.Name + " still uses " +
				h.State.Image,
			Learn:   "`docker pull` only downloads: the running container keeps using its own image until it is replaced",
			Command: []string{"docker", "pull", s.Image},
			Root:    true,
		},
		plan.Step{
			ID:     "stop",
			Title:  "Stop " + s.Name,
			Reason: stopReason,
			Learn: "`docker compose stop` stops the containers but keeps them; `down` would also remove them. " +
				"Here the old compose.yaml is still in place, so Compose stops exactly what is running",
			Command: composeCmd(s, "stop"),
			Root:    true,
		},
		plan.Step{
			ID:    "copy-data",
			Title: "Copy " + Dir(s) + " to " + backup,
			Reason: "a new image may convert the data to a format the old one cannot read; " +
				"with this copy you can go back. It takes as much disk space as the service folder",
			Learn: "`cp -a` copies a folder with everything inside and keeps owners, permissions and dates, " +
				"so the copy can replace the original as is. It is a copy on the same disk, not a backup: " +
				"if the disk breaks, both are lost",
			Command: []string{"cp", "-a", "--", Dir(s), backup},
			Root:    true,
		},
		plan.Step{
			ID:     "write-compose",
			Title:  "Write " + ComposePath(s),
			Reason: "replaces the file CaratCore " + h.State.CaratCoreVersion + " wrote (image " + h.State.Image + ") with the one this version writes (image " + s.Image + ")",
			File:   &plan.File{Path: ComposePath(s), Content: compose, Mode: 0o644},
			Root:   true,
		},
		writeState,
		start,
	)
	return p, nil
}

// UpdateNextSteps tells the user how to check the update and, when the plan
// copied the service folder to backup, how to go back to it. backup is ""
// when no copy was made.
func UpdateNextSteps(s catalog.Service, backup string) []string {
	next := []string{"Open " + s.Name + " and check that your data is there."}
	if backup == "" {
		return next
	}
	next = append(next,
		"The previous version is copied in "+backup+". To go back to it: "+
			"sudo docker compose --file "+ComposePath(s)+" down; "+
			"sudo mv "+Dir(s)+" "+Dir(s)+".failed; "+
			"sudo mv "+backup+" "+Dir(s)+"; "+
			"sudo docker compose --file "+ComposePath(s)+" up --detach",
		"When you are sure everything works, free the space: sudo rm -r "+backup,
	)
	return next
}
