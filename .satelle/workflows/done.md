---
name: done
type: workflow
scope: project
tags: [type:workflow]
description: portside's declaration of done — one section per story category, listing the obligations discharged before work closes, plus the park and cancel states the binary synthesises. Half of a derived route; step.md is the other half. Converted from the retired portside-workflow DOT graph.
---

# Definition of done — portside

The working lane is portside's delivery lifecycle: a story is raised, **planned**
by a dispatched Opus planner, **coded** in-loop, **integrated** (commit + push),
**deployed** to local docker, then closed. The container and task-run lanes are
carried forward unchanged from the shipped order-zero route.

<!-- CONVERTED from .satelle/workflows/portside-workflow.md (DOT), which claimed -->
<!-- applies_to: ["*"] and therefore governed every category. Because an authored -->
<!-- done.md overrides the shipped one WHOLLY, the container and task-run sections -->
<!-- are re-declared here verbatim from the shipped default — without them a repo -->
<!-- with tasks in its store (this one has three) loses task-run routing entirely. -->
<!-- -->
<!-- DECISION, not omission: no `## substrate` section. The retired graph had no -->
<!-- substrate lane, so substrate stories fell through to the working lane and were -->
<!-- deployed like code. That behaviour is carried forward unchanged; authoring a -->
<!-- markdown-only substrate lane would be a process change, not a conversion. -->

<!-- ================================================================ -->
<!-- THE WORKING LANE — governs every category with no section of its own -->
<!-- raised -> planned -> coded -> integrated -> deployed -> closed -->
<!-- ================================================================ -->
## *
<!-- The graph's spine, in the order the repo requires: plan first, then code, -->
<!-- then commit/push, then the local docker deploy/update. Each obligation is -->
<!-- discharged by the step in step.md declaring the matching `provides:`. -->
<!-- -->
<!-- Order here is for the reader only — the binary derives it by topologically -->
<!-- sorting the selected steps on `requires:`. -->
- raised
- planned
- coded
- integrated
- deployed
- closed
park: blocked @satelle-story-blocked-review
cancel: cancelled @satelle-story-cancel-review
recover: in_progress from integration, deploy

<!-- `park:` carries the graph's blocked node (agent=reviewer, -->
<!-- @skill:satelle-story-blocked-review). It names NO advisor: the retired graph -->
<!-- had no on_enter_agent, and the shipped default's blocked-triage advisor was -->
<!-- never part of this repo's lifecycle. -->
<!-- -->
<!-- `recover:` carries the graph's two recovery edges — integration -> in_progress -->
<!-- and deploy -> in_progress — so a reject returns the story to work to fix and -->
<!-- re-traverse, never to bypass. blocked -> in_progress is synthesised from -->
<!-- `park:` and is deliberately not written here. -->

<!-- ================================================================ -->
<!-- CONTAINERS — closed by their children, not by work of their own -->
<!-- raised -> children-resolved -->
<!-- ================================================================ -->
## epic-parent
<!-- Carried forward from the shipped default. Nothing is performed on a -->
<!-- container, so it has no working lane and — deliberately — no `park:`. -->
- raised
- children-resolved
cancel: cancelled @satelle-story-cancel-review

## parent
<!-- Its own section rather than a shared one: a section is selected by exact -->
<!-- category name before `*` is consulted, so both container kinds must name -->
<!-- themselves or they would fall onto the working lane and be asked to deploy. -->
- raised
- children-resolved
cancel: cancelled @satelle-story-cancel-review

<!-- ================================================================ -->
<!-- TASK RUNS — an action, then its verification -->
<!-- raised -> run -> run-verified -->
<!-- ================================================================ -->
## execution
<!-- Carried forward from the shipped default. A run, not a piece of work: it is -->
<!-- executed and then checked, gated by the task-validate reviewers on entry to -->
<!-- each. No `park:`, and `cancel:` names no gate — a run that should stop simply -->
<!-- stops; there is no slice of work to preserve. -->
- raised
- run
- run-verified
cancel: cancelled

## task
<!-- Same shape as execution, kept separate so both kinds resolve by exact name -->
<!-- rather than one of them falling through to `*` — which would put a run on -->
<!-- portside's plan/deploy spine and gate it as though it were code. -->
- raised
- run
- run-verified
cancel: cancelled
