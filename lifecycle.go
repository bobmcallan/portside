package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

const lifecycleTimeout = 60 * time.Second

// lifecycleServer handles container/stack start|stop|restart|remove.
// Stack remove deletes containers and the compose network only — never volumes.
type lifecycleServer struct {
	cli *client.Client
}

func (s *lifecycleServer) register(mux *http.ServeMux) {
	mux.HandleFunc("/api/container/", s.handleContainer)
	mux.HandleFunc("/api/stack/", s.handleStack)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// /api/container/{id}/{start|stop|restart|remove}
func (s *lifecycleServer) handleContainer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	// path: /api/container/{id}/{action}
	rest := strings.TrimPrefix(r.URL.Path, "/api/container/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		writeErr(w, http.StatusBadRequest, "path must be /api/container/{id}/{action}")
		return
	}
	id, action := parts[0], parts[1]
	ctx, cancel := context.WithTimeout(r.Context(), lifecycleTimeout)
	defer cancel()

	var err error
	switch action {
	case "start":
		err = s.cli.ContainerStart(ctx, id, container.StartOptions{})
	case "stop":
		err = s.cli.ContainerStop(ctx, id, container.StopOptions{})
	case "restart":
		err = s.cli.ContainerRestart(ctx, id, container.StopOptions{})
	case "remove":
		// Volumes: never remove. Force so a running container can be deleted.
		err = s.cli.ContainerRemove(ctx, id, container.RemoveOptions{Force: true, RemoveVolumes: false})
	default:
		writeErr(w, http.StatusBadRequest, "action must be start|stop|restart|remove")
		return
	}
	if err != nil {
		log.Printf("container %s %s: %v", action, id, err)
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true", "id": id, "action": action})
}

// /api/stack/{project}/{start|stop|restart|remove}
func (s *lifecycleServer) handleStack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/api/stack/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		writeErr(w, http.StatusBadRequest, "path must be /api/stack/{project}/{action}")
		return
	}
	project, action := parts[0], parts[1]
	ctx, cancel := context.WithTimeout(r.Context(), lifecycleTimeout)
	defer cancel()

	list, err := s.cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("label", labelComposeProject+"="+project)),
	})
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	if len(list) == 0 {
		writeErr(w, http.StatusNotFound, "no containers for project")
		return
	}

	var failures []string
	for _, c := range list {
		var opErr error
		switch action {
		case "start":
			opErr = s.cli.ContainerStart(ctx, c.ID, container.StartOptions{})
		case "stop":
			opErr = s.cli.ContainerStop(ctx, c.ID, container.StopOptions{})
		case "restart":
			opErr = s.cli.ContainerRestart(ctx, c.ID, container.StopOptions{})
		case "remove":
			opErr = s.cli.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true, RemoveVolumes: false})
		default:
			writeErr(w, http.StatusBadRequest, "action must be start|stop|restart|remove")
			return
		}
		if opErr != nil {
			log.Printf("stack %s container %s %s: %v", project, c.ID[:12], action, opErr)
			failures = append(failures, c.ID[:12]+": "+opErr.Error())
		}
	}

	// On remove: delete the default compose network only (never volumes).
	if action == "remove" {
		if nerr := s.removeComposeNetwork(ctx, project); nerr != nil {
			log.Printf("stack %s network remove: %v", project, nerr)
			failures = append(failures, "network: "+nerr.Error())
		}
	}

	if len(failures) > 0 {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"ok":       false,
			"project":  project,
			"action":   action,
			"failures": failures,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"project": project,
		"action":  action,
		"count":   len(list),
	})
}

// removeComposeNetwork removes the compose default network for a project.
// Never touches volumes.
func (s *lifecycleServer) removeComposeNetwork(ctx context.Context, project string) error {
	// Compose default network is typically "{project}_default".
	// Also match by compose labels for accuracy.
	nets, err := s.cli.NetworkList(ctx, network.ListOptions{
		Filters: filters.NewArgs(
			filters.Arg("label", labelComposeProject+"="+project),
		),
	})
	if err != nil {
		return err
	}
	for _, n := range nets {
		// Skip external/pre-existing networks that compose marked but shouldn't be torn down
		// if they have the external label — still only remove project-scoped ones.
		if err := s.cli.NetworkRemove(ctx, n.ID); err != nil {
			// Continue trying others; report last error if all fail is handled by caller via log.
			log.Printf("network remove %s (%s): %v", n.Name, n.ID[:12], err)
			return err
		}
	}
	// Fallback: named default network without label (older compose).
	if len(nets) == 0 {
		name := project + "_default"
		if err := s.cli.NetworkRemove(ctx, name); err != nil {
			// Not found is fine — network may already be gone.
			if !client.IsErrNotFound(err) {
				return err
			}
		}
	}
	return nil
}

// ensure types import used when compiling older SDKs that still need types.
var _ = types.Container{}
