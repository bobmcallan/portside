---
name: portside-workflow
scope: project
type: workflow
tags: [type:workflow]
applies_to: ["*"]
description: portside's delivery lifecycle, authored in DOT. A story moves backlog → in_progress → integration → deploy → done, with a cancelled exit and a blocked park. It is reviewer-only for execution: the driving session performs in_progress (implement + test), integration (commit + push), and deploy (build the image and bring the local compose stack up on 127.0.0.1) IN-LOOP, so no context is lost to a dispatched worker. Order matches the repo's rule — code first, then commit/push, then the local docker deploy/update. Recovery edges return integration and deploy to in_progress on a reject.
---

# portside workflow — in-loop, commit/push then local docker deploy

The lifecycle is the **DOT graph** below — read it as the authority; this prose
only orients and must not restate it. Each node is a step carrying an `agent`.
This workflow is **reviewer-only for execution**: `in_progress`, `integration`,
and `deploy` carry `agent=executor`, so the **in-loop driving session** performs
them with full context and the session's own permissions — no isolated
sub-process is spawned for coding, commit, push, or deploy. A **reviewer** node
only gates *entry* (read-only — it judges, never mutates).

The three performing phases, in the repo's required order:

- `in_progress` — implement the slice and its tests in-loop.
- `integration` — **commit and push** the slice.
- `deploy` — **local docker deploy/update**: build the image from the committed
  tree, bring the compose stack up on `127.0.0.1:8888`, and probe it before
  `done`.

`integration -> in_progress` and `deploy -> in_progress` are recovery edges: a
reject returns the story to work to fix and re-traverse, never bypass. `blocked`
is a park state (world-not-ready, same ACs on resume). `done`/`cancelled` are
terminal.

```dot
digraph portside_workflow {
  graph [goal="Drive a portside story to done — implemented in-loop with tests, committed and pushed, then deployed and probed on local docker at 127.0.0.1:8888, every acceptance criterion met", vars="story"]
  rankdir=LR

  backlog     [shape=Mdiamond]
  in_progress [agent=executor]                             // in-loop: implement + test
  integration [agent=executor]                             // in-loop: commit + push
  deploy      [agent=executor]                             // in-loop: build image + local compose up + probe
  done        [shape=Msquare]                              // terminal
  cancelled   [agent=reviewer, prompt="@skill:satelle-story-cancel-review"]
  blocked     [agent=reviewer, prompt="@skill:satelle-story-blocked-review"]

  // edge-less declarations
  step        [agent=reviewer, prompt="@skill:satelle-step-summary", mandatory=true]
  estimate    [agent=reviewer, prompt="@skill:satelle-estimate-actual-review", on="in_progress,done"]

  backlog     -> in_progress
  in_progress -> integration
  integration -> deploy
  deploy      -> done

  integration -> in_progress  // recovery
  deploy      -> in_progress  // recovery

  in_progress -> blocked
  blocked     -> in_progress

  backlog     -> cancelled
  in_progress -> cancelled
  integration -> cancelled
  deploy      -> cancelled
}
```

## Skill resolution

Every gate/skill this workflow names resolves through the doc-index, project
scope (`.satelle/skills`) layered over the embedded system defaults. This lean
workflow references only skills that resolve from the **embedded system
defaults**, so there is no dangling reference and a story drives to a terminal
state without a missing-skill block:

- `satelle-story-cancel-review` — the cancel gate.
- `satelle-story-blocked-review` — the park gate on `in_progress → blocked`.
- `satelle-step-summary` — per-transition step summaries.
- `satelle-estimate-actual-review` — gates begin-work + close.

## Environment

```yaml
guardrails:
  always:
    - Drive an engaged item to a terminal state (done or cancelled) — don't leave work open indefinitely.
    - Give a story numbered acceptance criteria before starting, and satisfy them before moving to done.
    - Perform in_progress, integration, and deploy IN-LOOP as the driving session; do not dispatch an isolated sub-process for coding, commit, push, or deploy.
    - Order is fixed: implement in_progress, then commit/push in integration, then the local docker deploy/update in deploy.
    - Prove the local docker deploy (build the image, compose up on 127.0.0.1:8888, probe it) before done.
  ask_first: []
  never:
    - Place any state after done — done is always the terminal success state.
    - Bind or expose the service on anything but 127.0.0.1.
    - Delete a stack's volumes — delete removes containers and the compose network only.
    - Mark an item done with unmet acceptance criteria.
```
