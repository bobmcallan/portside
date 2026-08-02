---
name: step
type: workflow
scope: project
tags: [type:workflow]
description: portside's step catalogue — the steps and always-on gates a route selects from by obligation. A step declares what it provides, what it requires, who performs it and the reviewers gating ENTRY to it. Half of a derived route; done.md is the other half. Converted from the retired portside-workflow DOT graph.
---

# Step catalogue — portside

The **one dispatched step is `plan`**: allocated to `agent: planner` (Claude Opus
in `.satelle/agents.toml`), it reads the story and repo and attaches an
implementation plan covering every acceptance criterion. Everything after it
carries `agent: executor` and is performed **in-loop** by the driving session —
no isolated sub-process codes, commits, pushes or deploys.

<!-- HOW TO READ THIS FILE (and its other half). -->
<!-- -->
<!-- A `##` heading is a STAGE NAME — the status an item holds while in that -->
<!-- step — NOT the step's identity. Headings repeat on purpose: `done` appears -->
<!-- three times below and `in_progress` twice, because different lanes reach the -->
<!-- same status by discharging different obligations. -->
<!-- -->
<!-- A step's identity is its `provides:` key. done.md lists obligations; the -->
<!-- binary selects the steps whose `provides:` matches and orders them by -->
<!-- `requires:`. -->
<!-- -->
<!-- TOPOLOGY IS NOT AUTHORED. Every `-> cancelled`, `-> blocked` and recovery -->
<!-- back-edge the retired graph drew by hand is SYNTHESISED by the binary from -->
<!-- done.md's `park:` / `cancel:` / `recover:` keys. None appear here as steps. -->
<!-- -->
<!-- A gate belongs to the step it ADMITS, not to the edge it came from: the -->
<!-- graph's `plan -> in_progress [prompt="@skill:satelle-story-plan-review"]` is -->
<!-- `reviewers:` on `## in_progress` below, never on `## plan`. -->

<!-- ================================================================ -->
<!-- THE WORKING LANE — selected by done.md `## *` -->
<!-- raised -> planned -> coded -> integrated -> deployed -> closed -->
<!-- ================================================================ -->
## backlog
<!-- provides raised — the entry state for EVERY section: the working lane, both -->
<!-- container sections, and both task-run sections all open here. -->
<!-- Carries the graph's `backlog [shape=Mdiamond]`. -->
start: true
provides: raised

## plan
<!-- provides planned — selected by `## *`. The single DISPATCHED step: the -->
<!-- planner agent attaches a plan artifact covering every AC, so the in-loop -->
<!-- implementer works from a self-contained plan. -->
<!-- Carries the graph's `plan [agent=planner, prompt="@skill:plan"]`; the -->
<!-- prompt becomes `skills:`. Entry is UNGATED — the graph's `backlog -> plan` -->
<!-- edge carried no reviewer. -->
agent: planner
skills: plan
provides: planned
requires: raised

## in_progress
<!-- provides coded — selected by `## *`. Implement the slice and its tests -->
<!-- IN-LOOP. Its reviewer is the graph's gate on the `plan -> in_progress` edge, -->
<!-- re-homed onto the step that edge admits. -->
agent: executor
reviewers: satelle-story-plan-review
reviewer_agent: reviewer
provides: coded
requires: planned

## integration
<!-- provides integrated — selected by `## *`. COMMIT AND PUSH the slice, in-loop. -->
<!-- Entry is ungated: the graph's `in_progress -> integration` edge carried no -->
<!-- reviewer. A reject here returns to in_progress via done.md's `recover:`. -->
agent: executor
provides: integrated
requires: coded

## deploy
<!-- provides deployed — selected by `## *`. LOCAL DOCKER DEPLOY/UPDATE, in-loop: -->
<!-- build the image from the committed tree, bring the compose stack up on -->
<!-- 127.0.0.1:8888 and probe it. Entry is ungated, as in the graph. -->
agent: executor
provides: deployed
requires: integrated

## done
<!-- provides closed — selected by `## *`. -->
<!-- -->
<!-- DECISION, recorded deliberately: the retired graph closed UNGATED — its -->
<!-- `deploy -> done` edge carried no reviewer_skill and its `done` node carried -->
<!-- no reviewer prompt. That was judged an oversight rather than an intent, so -->
<!-- the close is gated here, matching the constitution's requirement that every -->
<!-- numbered acceptance criterion be satisfied before done. -->
reviewers: satelle-story-done-review
reviewer_agent: reviewer
terminal: true
provides: closed
requires: deployed

<!-- ================================================================ -->
<!-- THE CONTAINER CLOSE — selected by `## epic-parent` and `## parent` -->
<!-- raised -> children-resolved -->
<!-- ================================================================ -->
## done
<!-- provides children-resolved — carried forward from the shipped default. -->
<!-- Nothing performs on this lane, so `agent: reviewer` marks it as judging -->
<!-- only; the gate is handed a snapshot of the children to judge from. It -->
<!-- requires just `raised`, which is why a container route is two steps long. -->
agent: reviewer
reviewers: satelle-story-done-review
terminal: true
provides: children-resolved
requires: raised

<!-- ================================================================ -->
<!-- THE TASK RUN — selected by `## execution` and `## task` -->
<!-- raised -> run -> run-verified -->
<!-- ================================================================ -->
## in_progress
<!-- provides run — carried forward from the shipped default. Its gate validates -->
<!-- the run BEFORE it begins. -->
agent: executor
reviewers: satelle-task-validate-before-review
reviewer_agent: reviewer
provides: run
requires: raised

## done
<!-- provides run-verified — carried forward from the shipped default. Its gate -->
<!-- validates the run AFTER it has happened, which is what makes a run verified -->
<!-- rather than merely finished. -->
reviewers: satelle-task-validate-after-review
reviewer_agent: reviewer
terminal: true
provides: run-verified
requires: run

<!-- ================================================================ -->
<!-- ALWAYS-ON GATES -->
<!-- -->
<!-- These are the graph's two edge-less nodes. A gate occupies no stage; it -->
<!-- fires on entry to steps. Its three scoping keys are different axes: -->
<!-- -->
<!--   on:         which STEPS it fires on (omitted = every step) -->
<!--   for:        which done.md SECTIONS it belongs to. `for: *` means the -->
<!--               WILDCARD SECTION — the working lane — not "everything". -->
<!--   applies_to: which ITEMS, by tag. -->
<!-- ================================================================ -->
## gate satelle-step-summary
<!-- The graph's `step [agent=reviewer, prompt="@skill:satelle-step-summary", -->
<!-- mandatory=true]`. No `on:` — it fires on entry to every step, as it did. -->
<!-- -->
<!-- DECISION, not omission: `for:` does not name the task-run sections. The -->
<!-- retired graph claimed applies_to ["*"] and so summarised task runs too; the -->
<!-- shipped scoping is adopted along with those re-declared lanes, because a run -->
<!-- has its own before/after validators as its record. -->
agent: reviewer
mandatory: true
for: *, epic-parent, parent

## gate satelle-estimate-actual-review
<!-- The graph's `estimate [agent=reviewer, -->
<!-- prompt="@skill:satelle-estimate-actual-review", on="in_progress,done"]` — -->
<!-- fences the estimate on entry to in_progress and the actual on entry to done. -->
<!-- -->
<!-- DECISION, not omission: `for: *` only. Containers perform no work to -->
<!-- estimate, and a run is not estimated. -->
agent: reviewer
on: in_progress, done
for: *
