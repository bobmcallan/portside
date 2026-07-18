package main

import (
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
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

// With a stopped container mixed in, default view hides it.
	list = append(list, types.Container{
		ID:     "eee",
		Names:  []string{"/alpha-old-1"},
		State:  "exited",
		Status: "Exited (0)",
		Labels: map[string]string{labelComposeProject: "alpha"},
	})
	stacks := groupByProject(list, false)
	if len(stacks) != 3 {
		t.Fatalf("got %d stacks, want 3", len(stacks))
	}
	if len(stacks[1].Containers) != 2 {
		t.Fatalf("running-only alpha should have 2, got %d", len(stacks[1].Containers))
	}
	withStopped := groupByProject(list, true)
	if len(withStopped[1].Containers) != 3 {
		t.Fatalf("show-stopped alpha should have 3, got %d", len(withStopped[1].Containers))
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

func TestCalcCPUPercent(t *testing.T) {
	st := container.StatsResponse{
		Stats: container.Stats{
			CPUStats: container.CPUStats{
				CPUUsage: container.CPUUsage{
					TotalUsage:  2000,
					PercpuUsage: []uint64{1000, 1000},
				},
				SystemUsage: 10000,
				OnlineCPUs:  2,
			},
			PreCPUStats: container.CPUStats{
				CPUUsage: container.CPUUsage{
					TotalUsage: 1000,
				},
				SystemUsage: 5000,
			},
		},
	}
	// cpuDelta=1000, sysDelta=5000, nCPU=2 → (1000/5000)*2*100 = 40
	got := calcCPUPercent(st)
	if got < 39.9 || got > 40.1 {
		t.Fatalf("calcCPUPercent = %v, want ~40", got)
	}
	// Zero deltas → 0
	st.CPUStats = st.PreCPUStats
	if calcCPUPercent(st) != 0 {
		t.Fatalf("zero delta should be 0")
	}
}

func TestUsedMemory(t *testing.T) {
	st := container.StatsResponse{
		Stats: container.Stats{
			MemoryStats: container.MemoryStats{
				Usage: 1000,
				Limit: 4000,
				Stats: map[string]uint64{"inactive_file": 200},
			},
		},
	}
	used, limit := usedMemory(st)
	if used != 800 || limit != 4000 {
		t.Fatalf("usedMemory = %d/%d, want 800/4000", used, limit)
	}
}

func TestParseProcStatCPU(t *testing.T) {
	// user nice system idle iowait
	idle, total, err := parseProcStatCPU("cpu  10 20 30 40 10 0 0 0")
	if err != nil {
		t.Fatal(err)
	}
	if idle != 50 { // 40+10
		t.Fatalf("idle=%d want 50", idle)
	}
	if total != 110 {
		t.Fatalf("total=%d want 110", total)
	}
}

func TestParseMemInfo(t *testing.T) {
	in := "MemTotal:       2048 kB\nMemFree:        100 kB\nMemAvailable:   1024 kB\n"
	total, avail, err := parseMemInfo(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if total != 2048*1024 || avail != 1024*1024 {
		t.Fatalf("got %d/%d", total, avail)
	}
}

func TestRingPush(t *testing.T) {
	var r []float64
	for i := 0; i < 5; i++ {
		r = ringPush(r, 3, float64(i))
	}
	if len(r) != 3 || r[0] != 2 || r[2] != 4 {
		t.Fatalf("ring = %v", r)
	}
}
