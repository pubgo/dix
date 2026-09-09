# Legacy-First DI Architecture Visualization Design

> Status: approved for planning · Approach: enhance legacy `/`, delete `/next`  
> Date: 2026-09-06 · Branch context: `codex/scale-graph-usability`  
> Supersedes (for product direction): `2026-09-05-dependency-visualization-unified-design.md`  
> Related: `2026-09-05-scale-graph-usability-design.md`, `2026-09-04-graph-trace-redesign-design.md`

## Problem

Dix containers with tens to hundreds of providers make the legacy `/` canvas feel like an unreadable blob: hierarchical “bead strings,” long package labels, and mixed concerns on one graph.

A parallel `/next` five-view shell added module maps, budgets, and layout helpers, but:

1. Maintenance cost of two UIs is too high.
2. `/next` did not replace the legacy workstation users still need.
3. Generic “make the graph prettier” tactics miss the real job.

The primary job is **dependency-injection architecture review**: see whether project wiring is organized sanely and where to reorganize. Debugging inject failures matters less here because call-chain Trace already covers it.

Providers (who produces what) and objects (what has been instantiated) are different questions and must not share the same default canvas treatment.

## Goals

1. Keep a **single UI**: legacy `/` (Alpine + existing tools).
2. Preserve **all existing legacy capabilities** (package sidebar, group rules, prefix, depth, layouts, SVG/Mermaid, diagnostic modals, Trace, etc.).
3. Make large graphs readable for **architecture organization**, not only for pixel density.
4. Support four architectural lenses: **package/module**, **providers**, **types**, **business groups**.
5. Port useful `/next` pieces into legacy; **delete** the `/next` shell and route.
6. Keep public Dix APIs stable (additive fields only); no new third-party dependencies.

## Non-Goals

- Rebuilding a second five-view product shell.
- Making Trace/diagnosis the main graph workflow.
- Drawing every object as a default architecture node.
- Replacing vis-network or Alpine in this iteration.
- Silently changing the user’s current view mode without an explicit action.

## Product Approach

**Chosen:** Approach A — enhance legacy `/`, embed useful `/next` rendering helpers, then delete `/next`.

Rejected:

- Dual-track `/` + `/next` (maintenance cost).
- Full rewrite of the workstation chrome.

**Default entry:** global Providers graph may remain the default (existing habit). Crowding is handled by semantic degradation and clearer lenses, not by removing the Providers entry.

## Jobs and Lenses

### Primary job

Answer: *Is the DI dependency structure organized reasonably? Where should we optimize boundaries?*

### Secondary job

Inject/runtime failures — Trace modals remain the main path; graph only keeps helpful jumps.

### Four lenses (all required)

| Lens | Question | Canvas nodes | Edges | Objects |
|---|---|---|---|---|
| Package / module | Are cross-boundary deps sane? | Packages or modules (prefer `/api/modules`) | Cross-package/module deps | Counts/badges only |
| Providers | Who produces what? | Providers + I/O types (existing) | Produce / depend edges (existing) | Detail/status only |
| Types | Is type coupling too dense? | Type nodes (existing) | Type→type edges (existing) | Detail/status only |
| Business groups | How do domains couple? | Group nodes via existing aggregate rules | Inter-group edges; expand in-group | Detail/status only |

### UI mapping (no capability loss)

- Toolbar: keep **Providers** / **Types**; add **Module map** (ported from `/next`).
- **Business groups**: keep **按分组聚合** (not a fifth mutually exclusive mode that fights Providers/Types). Under module map, aggregation means further collapsing modules by group rules when matched.
- Package sidebar: clicking a package sets **architectural scope** (prefix) and redraws — framed as choosing a slice, not only “filter noise.”

## Crowding Solution (DI-semantic, not generic layout-only)

Crowding happens when an **architecture question** is answered with an **inventory-level canvas** (all providers × types × objects).

Fixed degradation order when the current canvas exceeds a readable budget:

1. **Prompt a coarser lens** — banner suggesting Module map or enabling group aggregation; one-click apply allowed; do not silently switch the user’s view.
2. **Semantic collapse** — if aggregation is on, prefer inter-group edges; if off, recommend turning it on.
3. **Mark coupling hubs** — highest-degree types/packages/groups (architecture smells); click focuses or sets prefix scope.
4. **Short labels** — shorten package/type display names; full name on hover/detail (from `/next`).
5. **Layout / camera** — avoid thin hierarchical bead-strings on large graphs; use readable layout + fit/focus (from `/next` strategies), wired into legacy `renderGraph`.
6. **Top-K truncation last** — keep highest-connectivity nodes, declare truncation, show hub table and narrow-scope actions.

**Objects never enter the architecture node pool by default** — they amplify false density.

Soft budget starting points (tunable): canvas readability soft cap ~40–60 structural nodes; providers/types composition may use ~150/400 before soft truncation — aligned with prior scale work, applied inside legacy.

## Port from `/next` / Delete `/next`

### Port into legacy

- Module map rendering (`/api/modules`, readable/star layout).
- `shortGraphLabel`, layout/camera helpers, hub ranking, budget helpers.
- Optional bounded drill-downs already exposed by API (`/api/module`, `/api/ego`) where they improve scoped reading without removing full-graph modes.
- Stable issue/error → graph/Trace link helpers if diagnostic modals still lack them.

Delivered as helpers consumed by Alpine `renderGraph` / toolbar — **not** as a second shell.

### Delete

- `GET /next` and `HandleNextIndex`.
- Five-view shell: `static/index.html`, `static/js/main.js`, `static/js/views/*` used only by `/next`.
- Next-only docs/recommendations; README states `/` is the only UI.
- Next-shell-only frontend tests; keep/repurpose pure logic tests onto shared modules used by legacy.

### Keep as shared logic

- Testable pieces of `graph_state.mjs` / `graph_workbench.mjs` (labels, budget, hubs, group aggregate helpers) referenced by legacy; strip next-only wiring.

## Implementation Boundaries

### Touch

- `dixhttp/template.html` — module map control; density banner/hub table; sidebar copy for scope.
- `dixhttp/static/js/legacy/app.js` — render pipeline hooks for labels, degradation, module map, camera.
- Shared mjs helpers + node tests.
- `dixhttp/server.go` — remove `/next`.
- README / superseded design notes.

### Do not touch (this iteration)

- Public API semantics (additive only).
- Trace/diagnostic modal product rewrite.
- New JS/Go third-party dependencies.
- Forced default-view change away from Providers unless user later asks.

### Risks

- Large `app.js`: change render in small steps; regress group/prefix/SVG/Mermaid/depth.
- Banner vs silent mode switch: prompt + optional one-click only.

## Acceptance Criteria

1. Example-scale app (~10 modules, ~100 providers): module map shows clear boundaries; global Providers over budget shows banner + hubs instead of a silent bead-string.
2. Group aggregate, package prefix, Providers/Types/Module switching behave as today plus module map — no removed tools.
3. Objects are not default architecture canvas nodes.
4. `/next` removed; `/` is the only visualization UI entry.
5. `node --test` for shared graph helpers passes; `go test ./dixhttp/...` passes.

## Open Parameters (planning may tune, not reopen product direction)

- Exact soft-cap numbers and when auto-layout overrides hierarchical.
- Whether module map uses package list identity vs `/api/modules` labels when they diverge — prefer `/api/modules` for cross-module edges, keep package sidebar for scope.
