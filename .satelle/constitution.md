---
type: constitution
title: Project constitution
description: Order-zero context for portside — what an agent must know before touching this repo.
---

# portside — constitution

**What this repo is:** a minimal, single-image Docker container & stack
monitor/controller for a local WSL/Linux host — the small slice of Docker Desktop
you actually reach for. Server-side-rendered (SSR) HTML with a **Go** backend for
realtime stats, shipped as **one small Docker image**. No database, no auth, no
agents, no alerting, no historical persistence.

**Ground rules (non-negotiable):**

1. **Single Go service, single image.** Go + the official Docker SDK
   (`github.com/docker/docker/client`), one static binary, the page embedded via
   `go:embed`. No Node build step; SSR + a little vanilla JS for the live view.
2. **Localhost-only.** It mounts the Docker socket read-write (required to
   start/stop/delete), which is effectively host root. The sole mitigation is
   binding to `127.0.0.1` — never expose it off-box, never add `0.0.0.0`.
3. **Deleting never destroys data.** "Delete stack" removes containers and the
   compose network **only** — never volumes. A hard safety rule, not a default.
4. **Realtime, not historical.** Stats are pushed live (SSE, ~1–2s). At most a
   ~60s in-memory ring for sparklines. No database, ever.
5. **Design comes from the Satelle Design System.** The SSR frontend takes its
   tokens/components from the "Satelle Design System" claude.ai design project —
   don't invent a parallel visual language.

**Where the process lives:** authored substrate under `.satelle/` (workflows,
principles, skills). Work is satelle-gated: stories bind to
`workflow:portside-workflow` — implement in-loop (`in_progress`), **commit/push**
(`integration`), **local docker deploy/update** (`deploy`), then `done`. Give
every story numbered acceptance criteria and satisfy them before `done`.
