package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/BryanAguilar-Dev/CaratCore/internal/catalog"
	"github.com/BryanAguilar-Dev/CaratCore/internal/plan"
)

// RemovePlan returns the steps that remove s from h: stop it with
// `docker compose down` and, only with purge, delete its folder and data.
// An empty plan means there is nothing to remove.
func RemovePlan(s catalog.Service, h Host, purge bool) (plan.Plan, error) {
	p := plan.Plan{Title: "Remove " + s.Name}
	if purge {
		p.Title += " and delete its data"
	}

	if !h.Linux {
		return p, errors.New("unsupported platform: services are removed on Linux only")
	}

	if h.ExistingCompose != "" {
		if !h.DockerFound || !h.ComposeFound {
			return p, fmt.Errorf("%s exists but Docker Compose is not available to stop the service", ComposePath(s))
		}
		if purge {
			compose, err := ComposeFile(s)
			if err != nil {
				return p, err
			}
			// A file another CaratCore version wrote and nobody edited is
			// still CaratCore's own, so --purge may delete it.
			if composeOrigin(h, compose) == originUnknown {
				return p, fmt.Errorf("CaratCore cannot vouch for %s: %s. --purge does not delete what CaratCore did not write. Run uninstall without --purge, then review %s and delete it yourself",
					ComposePath(s), unknownOriginReason(h, s), Dir(s))
			}
		}

		reason := "stops and removes the container and its network, so the service no longer runs or starts after a reboot"
		if !purge {
			reason += "; the data stays in " + Dir(s) + ", so installing again picks it up"
		}
		if s.RemoveWarning != "" {
			reason += ". Warning: " + s.RemoveWarning
		}
		p.Steps = append(p.Steps, plan.Step{
			ID:     "stop",
			Title:  "Stop and remove the " + s.Name + " container",
			Reason: reason,
			Learn: "`docker compose down` removes the containers and the network Compose created for them. " +
				"It does not touch bind-mounted folders: they are ordinary folders of this machine, which is why the data survives",
			Command: []string{"docker", "compose", "--file", ComposePath(s), "down"},
			Root:    true,
		})
	}

	if purge && h.DirExists {
		p.Steps = append(p.Steps, plan.Step{
			ID:    "delete-data",
			Title: "Delete " + Dir(s),
			Reason: "you asked for --purge: this deletes compose.yaml and the data (" + volumePurposes(s) + "). " +
				"It cannot be undone; copy the folder first if you may need it",
			Learn: "`rm -r` deletes a folder and everything inside it; on the command line there is no trash to recover it from. " +
				"`--` marks the end of the options, so the path that follows can never be read as one",
			Command: []string{"rm", "-r", "--", Dir(s)},
			Root:    true,
		})
	}
	return p, nil
}

// DeployNextSteps tells the user what to do after deploying s and where it
// lives.
func DeployNextSteps(s catalog.Service) []string {
	next := append([]string{}, s.NextSteps...)
	return append(next, "Its compose.yaml and data are in "+Dir(s)+"; to remove it: caratcore uninstall "+s.ID)
}

// RemoveNextSteps tells the user what is left on the machine after removing s.
func RemoveNextSteps(s catalog.Service, purge bool) []string {
	var next []string
	if !purge {
		next = append(next,
			"The data is still in "+Dir(s)+": `caratcore install "+s.ID+"` uses it again, `caratcore uninstall "+s.ID+" --purge` deletes it.")
	}
	return append(next, "The image is still on disk; to free the space: sudo docker image rm "+s.Image)
}

func volumePurposes(s catalog.Service) string {
	if len(s.Volumes) == 0 {
		return "none besides compose.yaml"
	}
	purposes := make([]string, len(s.Volumes))
	for i, v := range s.Volumes {
		purposes[i] = v.Purpose
	}
	return strings.Join(purposes, "; ")
}
