package main

import (
	"testing"

	"github.com/docker/docker/api/types"
)

func TestGroupByProject(t *testing.T) {
	list := []types.Container{
		{
			ID:     "aaa",
			Names:  []string{"/alpha-web-1"},
			State:  "running",
			Status: "Up 1m",
			Labels: map[string]string{
				labelComposeProject: "alpha",
				labelComposeService: "web",
			},
			Ports: []types.Port{{IP: "0.0.0.0", PublicPort: 8080, PrivatePort: 80, Type: "tcp"}},
		},
		{
			ID:     "bbb",
			Names:  []string{"/alpha-db-1"},
			State:  "running",
			Status: "Up 1m",
			Labels: map[string]string{
				labelComposeProject: "alpha",
				labelComposeService: "db",
			},
		},
		{
			ID:     "ccc",
			Names:  []string{"/beta-api-1"},
			State:  "running",
			Status: "Up 2m",
			Labels: map[string]string{
				labelComposeProject: "beta",
				labelComposeService: "api",
			},
		},
		{
			ID:     "ddd",
			Names:  []string{"/orphan"},
			State:  "running",
			Status: "Up 3m",
			Labels: map[string]string{},
		},
	}

	stacks := groupByProject(list)
	if len(stacks) != 3 {
		t.Fatalf("got %d stacks, want 3", len(stacks))
	}
	// Sorted by project name: (no project), alpha, beta
	if stacks[0].Name != noProjectLabel {
		t.Errorf("first stack = %q, want %q", stacks[0].Name, noProjectLabel)
	}
	if stacks[1].Name != "alpha" || len(stacks[1].Containers) != 2 {
		t.Errorf("alpha stack = %+v", stacks[1])
	}
	if stacks[1].Containers[0].Name != "alpha-db-1" {
		t.Errorf("alpha rows not sorted: %v", stacks[1].Containers)
	}
	if stacks[2].Name != "beta" || stacks[2].Containers[0].Service != "api" {
		t.Errorf("beta stack = %+v", stacks[2])
	}
	if len(stacks[1].Containers[1].Ports) != 1 || stacks[1].Containers[1].Ports[0].HostPort != "8080" {
		t.Errorf("ports = %+v", stacks[1].Containers[1].Ports)
	}
}
