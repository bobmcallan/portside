package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

const (
	statsInterval   = 1500 * time.Millisecond
	ringCapacity    = 40 // ~60s at 1.5s ticks
	subscriberBuf   = 4
	statsHTTPTimeout = 8 * time.Second
)

// HostStats is the host slice of an SSE snapshot.
type HostStats struct {
	CPUPct   float64 `json:"cpuPct"`
	MemUsed  uint64  `json:"memUsed"`
	MemTotal uint64  `json:"memTotal"`
}

// ContainerStats is one container row in an SSE snapshot.
type ContainerStats struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Stack   string  `json:"stack"`
	Service string  `json:"service"`
	State   string  `json:"state"`
	CPUPct  float64 `json:"cpuPct"`
	MemMB   float64 `json:"memMB"`
	MemPct  float64 `json:"memPct"`
}

// Snapshot is the JSON frame pushed on /api/stream.
type Snapshot struct {
	Host       HostStats        `json:"host"`
	Containers []ContainerStats `json:"containers"`
	Sparkline  []float64        `json:"sparkline"` // host CPU history for UI
	TS         int64            `json:"ts"`
}

type collector struct {
	cli *client.Client

	mu          sync.RWMutex
	latest      Snapshot
	ring        []float64
	subscribers map[chan Snapshot]struct{}

	prevHostIdle  uint64
	prevHostTotal uint64
	hasPrevHost   bool
}

func newCollector(cli *client.Client) *collector {
	return &collector{
		cli:         cli,
		ring:        make([]float64, 0, ringCapacity),
		subscribers: make(map[chan Snapshot]struct{}),
	}
}

func (c *collector) run(ctx context.Context) {
	// Prime first sample immediately so SSE clients don't wait a full tick.
	c.tick(ctx)
	t := time.NewTicker(statsInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.tick(ctx)
		}
	}
}

func (c *collector) tick(ctx context.Context) {
	snap, err := c.sample(ctx)
	if err != nil {
		log.Printf("stats sample: %v", err)
		return
	}
	c.mu.Lock()
	c.ring = ringPush(c.ring, ringCapacity, snap.Host.CPUPct)
	snap.Sparkline = append([]float64(nil), c.ring...)
	c.latest = snap
	subs := make([]chan Snapshot, 0, len(c.subscribers))
	for ch := range c.subscribers {
		subs = append(subs, ch)
	}
	c.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- snap:
		default:
			// Drop oldest for a lagging client.
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- snap:
			default:
			}
		}
	}
}

func (c *collector) sample(ctx context.Context) (Snapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, statsHTTPTimeout)
	defer cancel()

	list, err := c.cli.ContainerList(ctx, container.ListOptions{All: false})
	if err != nil {
		return Snapshot{}, err
	}

	host, err := c.sampleHost()
	if err != nil {
		// Host sample failure is non-fatal; keep zeros.
		log.Printf("host sample: %v", err)
		host = HostStats{}
	}

	type result struct {
		s   ContainerStats
		err error
	}
	ch := make(chan result, len(list))
	var wg sync.WaitGroup
	for _, ctr := range list {
		wg.Add(1)
		go func(ctr types.Container) {
			defer wg.Done()
			cs, err := c.sampleContainer(ctx, ctr)
			ch <- result{s: cs, err: err}
		}(ctr)
	}
	wg.Wait()
	close(ch)

	containers := make([]ContainerStats, 0, len(list))
	for r := range ch {
		if r.err != nil {
			log.Printf("container stats: %v", r.err)
			continue
		}
		containers = append(containers, r.s)
	}

	return Snapshot{
		Host:       host,
		Containers: containers,
		TS:         time.Now().UnixMilli(),
	}, nil
}

func (c *collector) sampleContainer(ctx context.Context, ctr types.Container) (ContainerStats, error) {
	name := ctr.ID
	if len(ctr.Names) > 0 {
		name = strings.TrimPrefix(ctr.Names[0], "/")
	}
	stack := ctr.Labels[labelComposeProject]
	if stack == "" {
		stack = noProjectLabel
	}
	out := ContainerStats{
		ID:      ctr.ID,
		Name:    name,
		Stack:   stack,
		Service: ctr.Labels[labelComposeService],
		State:   ctr.State,
	}

	resp, err := c.cli.ContainerStats(ctx, ctr.ID, false)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()

var st container.StatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		// Drain leftover if any.
		_, _ = io.Copy(io.Discard, resp.Body)
		return out, err
	}

	out.CPUPct = calcCPUPercent(st)
	used, limit := usedMemory(st)
	out.MemMB = float64(used) / (1024 * 1024)
	if limit > 0 {
		out.MemPct = float64(used) / float64(limit) * 100
	}
	return out, nil
}

func (c *collector) sampleHost() (HostStats, error) {
	idle, total, err := readProcStatCPU()
	if err != nil {
		return HostStats{}, err
	}
	memTotal, memAvail, err := readMemInfo()
	if err != nil {
		return HostStats{}, err
	}

	h := HostStats{
		MemTotal: memTotal,
		MemUsed:  memTotal - memAvail,
	}
	if c.hasPrevHost {
		idleDelta := idle - c.prevHostIdle
		totalDelta := total - c.prevHostTotal
		if totalDelta > 0 {
			h.CPUPct = 100 * (1 - float64(idleDelta)/float64(totalDelta))
			if h.CPUPct < 0 {
				h.CPUPct = 0
			}
		}
	}
	c.prevHostIdle = idle
	c.prevHostTotal = total
	c.hasPrevHost = true
	return h, nil
}

func (c *collector) serveSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := make(chan Snapshot, subscriberBuf)
	c.mu.Lock()
	c.subscribers[ch] = struct{}{}
	latest := c.latest
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.subscribers, ch)
		c.mu.Unlock()
		close(ch)
	}()

	writeSnap := func(s Snapshot) error {
		b, err := json.Marshal(s)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", b); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	// Immediate snapshot so the page paints without waiting a tick.
	if latest.TS != 0 {
		if err := writeSnap(latest); err != nil {
			return
		}
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case s := <-ch:
			if err := writeSnap(s); err != nil {
				return
			}
		}
	}
}

// calcCPUPercent computes container CPU% from Docker stats deltas (pre/current).
func calcCPUPercent(st container.StatsResponse) float64 {
	cpuDelta := float64(st.CPUStats.CPUUsage.TotalUsage) - float64(st.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(st.CPUStats.SystemUsage) - float64(st.PreCPUStats.SystemUsage)
	if cpuDelta <= 0 || sysDelta <= 0 {
		return 0
	}
	nCPU := float64(st.CPUStats.OnlineCPUs)
	if nCPU == 0 {
		nCPU = float64(len(st.CPUStats.CPUUsage.PercpuUsage))
	}
	if nCPU == 0 {
		nCPU = 1
	}
	return (cpuDelta / sysDelta) * nCPU * 100
}

// usedMemory returns working-set bytes and the memory limit.
func usedMemory(st container.StatsResponse) (used, limit uint64) {
	limit = st.MemoryStats.Limit
	used = st.MemoryStats.Usage
	if st.MemoryStats.Stats != nil {
		if v, ok := st.MemoryStats.Stats["inactive_file"]; ok && used > v {
			used -= v
		} else if v, ok := st.MemoryStats.Stats["total_inactive_file"]; ok && used > v {
			used -= v
		} else if v, ok := st.MemoryStats.Stats["cache"]; ok && used > v {
			used -= v
		}
	}
	return used, limit
}

func ringPush(ring []float64, cap int, v float64) []float64 {
	if len(ring) < cap {
		return append(ring, v)
	}
	copy(ring, ring[1:])
	ring[cap-1] = v
	return ring
}

func readProcStatCPU() (idle, total uint64, err error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		return 0, 0, fmt.Errorf("empty /proc/stat")
	}
	return parseProcStatCPU(sc.Text())
}

// parseProcStatCPU parses the aggregate "cpu " line from /proc/stat.
func parseProcStatCPU(line string) (idle, total uint64, err error) {
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, 0, fmt.Errorf("bad cpu line: %q", line)
	}
	var vals []uint64
	for _, f := range fields[1:] {
		n, e := strconv.ParseUint(f, 10, 64)
		if e != nil {
			return 0, 0, e
		}
		vals = append(vals, n)
		total += n
	}
	// idle = idle + iowait when present
	idle = vals[3]
	if len(vals) > 4 {
		idle += vals[4]
	}
	return idle, total, nil
}

func readMemInfo() (total, available uint64, err error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	return parseMemInfo(f)
}

// parseMemInfo reads MemTotal and MemAvailable (kB) and returns bytes.
func parseMemInfo(r io.Reader) (total, available uint64, err error) {
	sc := bufio.NewScanner(r)
	var gotTotal, gotAvail bool
	for sc.Scan() {
		line := sc.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		kb, e := strconv.ParseUint(fields[1], 10, 64)
		if e != nil {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			total = kb * 1024
			gotTotal = true
		case "MemAvailable:":
			available = kb * 1024
			gotAvail = true
		}
		if gotTotal && gotAvail {
			return total, available, nil
		}
	}
	if err := sc.Err(); err != nil {
		return 0, 0, err
	}
	if !gotTotal {
		return 0, 0, fmt.Errorf("MemTotal missing")
	}
	return total, available, nil
}
