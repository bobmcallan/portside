# portside

A minimal, single-image Docker container & stack monitor/controller — the small subset of Docker Desktop you actually reach for, on a WSL/Linux host.

Server-side rendered (SSR) HTML, a Go backend for realtime stats, one static binary in one small image. No database, no auth, no agents, no alerting.

## What it does

- **Running containers by default**, toggle to show stopped — no phantom exited rows.
- **Grouped by compose stack** (via the `com.docker.compose.project` label).
- **Realtime CPU% + memory**, host-overall and per-container (~1–2s refresh).
- **Published ports** shown per container.
- **Lifecycle control**: start / stop / restart / delete — per container *and* per whole stack.

## What it deliberately is NOT

No historical database, no multi-host/agents, no RBAC/OIDC/auth, no notifications, no compose-file editing. Single service, localhost-only.

## Design decisions

| Area | Decision |
|------|----------|
| Language / shape | Go + official Docker SDK, single static binary, page embedded via `go:embed` — no Node build step |
| Rendering | SSR HTML; realtime stats pushed to the page via SSE |
| Stats source | Server polls the Docker stats API on a ~1–2s ticker, computes CPU%, pushes one snapshot to all clients |
| Frontend | Vanilla JS + CSS; stacks as collapsible groups, CPU/mem bars, port chips, action buttons |
| "Delete stack" | Removes containers + the compose network only — **never volumes** (data is protected by design) |
| Tiny history | ~60s in-memory ring for sparklines; no persistence |
| Deploy | Runs as a container, mounts `docker.sock` (rw, needed for actions), bound to `127.0.0.1:8888` |
| Logs | Out of v1 (easy to add later as a stream endpoint) |

## Security note

Performing start/stop/delete requires the Docker socket read-write, which is effectively root on the host. The mitigation is binding to `127.0.0.1` only — no off-box access. Appropriate for a local WSL tool.

## HTTP surface (planned)

```
GET  /                       → the page (SSR, embedded)
GET  /api/stream             → SSE: { host:{cpuPct,memUsed,memTotal}, containers:[{id,name,stack,service,state,ports[],cpuPct,memMB,memPct}] }
POST /api/container/{id}/{start|stop|restart|remove}
POST /api/stack/{project}/{start|stop|restart|remove}
```

## Status

MVP shipped. `./scripts/deploy.sh` → http://127.0.0.1:8888

## Run

```bash
./scripts/deploy.sh
# open http://127.0.0.1:8888
```
