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
principles, skills). Work is satelle-gated: the lifecycle is the **derived route**
in `.satelle/workflows/done.md` + `step.md` — `plan` (dispatched to the Opus
planner), implement in-loop (`in_progress`), **commit/push** (`integration`),
**local docker deploy/update** (`deploy`), then `done`. Give every story numbered
acceptance criteria and satisfy them before `done`.

## Delivery intent

The route describes states, obligations and gates; it never described intent.
What follows is the operator intent carried over from the retired
`portside-workflow` DOT graph — its `goal=` and `guardrails:` block — because the
route grammar has no home for it and nothing warns when it goes missing.

**Goal of a portside story:** drive it to done — planned against the ACs,
implemented in-loop with tests, committed and pushed, then deployed and probed on
local docker at `127.0.0.1:8888`, every acceptance criterion met.

**Always:**

- Drive an engaged item to a terminal state (`done` or `cancelled`) — don't leave
  work open indefinitely.
- Give a story numbered acceptance criteria before starting, and satisfy them
  before moving to `done`.
- Enter `plan` before `in_progress`; the planner attaches a plan that covers every
  AC; do not skip plan-review.
- Perform `in_progress`, `integration` and `deploy` **in-loop** as the driving
  session; do not dispatch an isolated sub-process for coding, commit, push or
  deploy.
- Order is fixed: plan, then implement in `in_progress`, then commit/push in
  `integration`, then the local docker deploy/update in `deploy`.
- Prove the local docker deploy by running `./scripts/deploy.sh` (build + compose
  up on `127.0.0.1:8888` + probe) before `done`.

**Never:**

- Place any state after `done` — `done` is always the terminal success state.
- Bind or expose the service on anything but `127.0.0.1` (ground rule 2).
- Delete a stack's volumes — delete removes containers and the compose network
  only (ground rule 3).
- Mark an item `done` with unmet acceptance criteria.
