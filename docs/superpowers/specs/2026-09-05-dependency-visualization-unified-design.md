# Dependency Visualization Unified Design

> Status: approved for planning · Approach: evolve `/next` five-view shell  
> Date: 2026-09-05 · Branch context: `codex/scale-graph-usability`  
> Related: `docs/superpowers/specs/2026-09-04-graph-trace-redesign-design.md`,  
> `docs/superpowers/specs/2026-09-05-scale-graph-usability-design.md`

## Problem

Dix containers at approximately 100 modules, 200 providers, and 400 objects make a single global dependency graph unreadable. Two product jobs fail together today:

1. **Structure readability** — too many providers/objects; dense canvases are hard to browse and reason about.
2. **Failure diagnosis** — when `Provide` / `Inject` misbehaves, call trees, errors, and graph context are not one continuous path.

Identity and module-first work on `codex/scale-graph-usability` fixed false edges and defaulted `/next#/graph` to a module map, but the issue → graph → trace loop and request-cancellation polish are incomplete. This design unifies both jobs without replacing the existing five-view shell or breaking legacy surfaces.

## Goals

1. Keep the default graph always small and actionable (module map → module drill-down → ego neighborhood).
2. Treat objects as state and detail by default, not as required graph nodes.
3. Make diagnostics navigable: from an issue to a bounded graph or trace tree in at most three interactions.
4. Correlate runtime metrics, issues, graph nodes, and traces by stable `provider_id`.
5. Preserve existing public Dix APIs, legacy `/` UI, and existing `/api/*` field compatibility (additive only).

## Non-Goals

- Replace `vis-network` or introduce WebGL.
- Remove legacy `/` UI or five-view `/next` tabs.
- Draw every object as a default graph node.
- Rebuild the shell as a single three-pane workspace (deferred alternative).
- Introduce a parallel full `/api/graph/*` namespace in this iteration (reuse `/api/modules`, `/api/module`, `/api/ego`, `/api/issues`).

## Product Approach

**Chosen:** evolve the existing `/next` five-view information architecture into one workflow:

`Issue → locate → bounded graph → call tree → runtime detail`

Rejected for this iteration:

- **Issue Hub default** — shortest debug path, but weak for structure-first browsing.
- **Single three-pane shell** — best context continuity, highest breakage and layout cost.

## Information Architecture

| View | Primary question | Default content |
|---|---|---|
| Overview | Is the system healthy? | Stats cards + Issues feed + jump actions |
| Graph | How do dependencies connect? | Module map → module drill-down → type ego; full providers/types only as advanced modes |
| Search | Where is the target? | Server search; hits jump to ego/module or detail |
| Trace | How did this inject run? | Trace list (failures first) + TraceTree; prefilter from issue/graph |
| Diag | What is slow or wrong at startup? | Runtime stats, error tables, error-type help |

### Path A — Structure (anti-density)

1. Open Graph → module map.
2. Select module → bounded in-module topology (providers/types; objects not default nodes).
3. Select type → ego neighborhood (depth default 2, max 5).
4. Open global providers/types only as advanced mode, with an explicit density warning.

### Path B — Diagnosis (anti-fragmentation)

1. Overview Issues (or Diag) shows error/slow items.
2. One click to bounded graph (ego or module).
3. One click to Trace tree (prefiltered by provider/output; open tree directly when `trace_id` exists).
4. Graph node drawer exposes declared deps, runtime state, and jump-to-trace.

## Bounded Views and Density Policy

| View | Node cap | Edge cap | Source |
|---|---|---|---|
| Module map | 100 | 300 | `/api/modules` |
| Module drill-down | 150 | 400 | `/api/module` |
| Ego | 100 | 300 | `/api/ego` |
| Global providers/types | 150 | 400 | `/api/dependencies` (advanced only) |

Degradation order:

1. Server truncates with `limit` / `edge_limit` and returns truncation/degraded metadata.
2. Client applies degree Top-K budget (`applyGraphBudget`).
3. UI shows an explicit density warning with current N/M caps.
4. Same screen offers table / Top-K hubs / narrower scope (module, depth, prefix).
5. Objects stay in drawers and diagnostic tables unless the user deliberately drills to instance detail later.

Correctness rules:

1. Distinct registrations keep distinct identities even when they share a source line.
2. Struct multi-output registrations may share one registration identity.
3. Edges are real declared relationships; never synthesize `inputs × outputs` after aggregation.
4. Ego “instantiated” is true only when an object node exists.
5. A graph may hide or degrade; it must not silently show false relationships.
6. In-flight graph/issues/trace requests cancel when the view or hash changes.

Canvas rules:

- Preserve a usable minimum graph height.
- Narrow layouts move details into a drawer/bottom sheet instead of shrinking the canvas.
- Encode `mode` / `module` / `center` / `depth` (and related filters) in the hash query.

## Issue → Graph → Trace Contract

### `/api/issues`

Merge injection failures, provider failures, and slow providers into one actionable feed.

Minimum fields per issue:

- `severity` (`error` | `warn`, optional `info`)
- `title`
- `provider`
- `provider_id` (when known)
- `output_type`
- `module`
- `root_cause`
- optional `trace_id`

Navigation:

- Graph button → `#/graph?mode=ego&center=<output_type>`, else `mode=module&module=...`, else `mode=providers&prefix=<provider>`
- Trace button → `#/trace?provider=...&output_type=...&status=error|slow`, or direct tree when `trace_id` is present
- Graph drawer “view resolve path” uses the same Trace prefilter rules

Correlation key across issues, runtime-stats, graph nodes, and trace attributes: `provider_id`. Function name is display/fallback only.

## API Contract

Primary (enhance in place):

- `GET /api/modules`
- `GET /api/module?name=&limit=&edge_limit=`
- `GET /api/ego?center=&depth=&direction=`
- `GET /api/issues?limit=`
- `GET /api/trace`, `GET /api/trace-tree?trace_id=`
- `GET /api/search`, `/api/stats`, `/api/runtime-stats`, `/api/errors`

Compatibility:

- `GET /api/dependencies` remains available; `/next` default paths must not eagerly load it.
- Legacy `/` UI and existing response fields remain; new fields are additive.
- No new third-party Go or JS dependencies in this iteration.

Deferred:

- Full `/api/graph/{summary,modules,module,ego,node}` namespace rename/consolidation.

## Compatibility Boundaries

- Do not break public Dix APIs.
- Do not remove legacy `/` UI.
- Do not change existing `/api/*` field meanings; additive fields only.
- `/next` defaults and navigation may evolve (module-first, issues entry, request cancellation).
- Do not replace the rendering engine in this iteration.

## Delivery Phases

| Phase | Goal | Outcomes |
|---|---|---|
| P0 Stabilize | Finish half-done branch pieces | Reintroduce reliable request cancellation; align module budget to 150/400; lock Issues navigation contract with tests |
| P1 Density | Structure readability | Default path avoids `/api/dependencies`; density warning + table/Top-K alternatives; narrow-viewport canvas floor |
| P2 Diagnosis loop | Failure navigability | Complete Issues fields; graph drawer → Trace; Trace prefilters; ≤3-interaction acceptance cases |
| P3 Docs & fixture | Regressible | Scale fixture walkthrough; README matches `/next` defaults |

## Acceptance

1. At ~100 providers / hundreds of objects, the default module map stays interactive and labels remain readable.
2. Module drill-down and ego respect budgets; over-budget views degrade explicitly with no false edges.
3. Overview Issue → bounded graph in ≤2 clicks; → Trace tree in ≤3 clicks.
4. Runtime errors/durations attach to the correct node via `provider_id`.
5. `go test -race ./...` and `example/http` tests pass; legacy full-graph entry remains reachable.
6. Documentation matches actual `/next` default behavior.

## Success Signal

A developer can complete both jobs on a large container without relying on an unreadable global graph:

1. understand module-level dependencies and drill to relevant types;
2. jump from a failure or slowdown to the call tree and related dependency context.
