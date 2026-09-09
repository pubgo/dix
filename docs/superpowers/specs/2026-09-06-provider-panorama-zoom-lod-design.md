# Provider Panorama + Zoom LOD Design

> **Status: CANCELLED (2026-09-07)** — 全景 layout removed from product. Completeness stays via right-rail Provider inventory + package scope / 模块地图; canvas layouts remain 层级 / 力导向 only.  
> Parent: `2026-09-06-legacy-first-architecture-viz-design.md`  
> Branch context: `codex/scale-graph-usability`

## Historical note

Earlier drafts (Approach B grid + LOD, Approach C structure panorama + inventory) are retained below for context only. Do not re-introduce a `panorama` layout option without a new approved design.

---

## Approach C (locked 2026-09-07) — superseded by removal

**Dual channel — do not put completeness and readability on the same canvas encoding:**

| Channel | Job | Encoding |
|---|---|---|
| **Structure panorama** | Perceive organization / cross-boundary coupling | Package-level nodes + inter-package edges (default when 布局=全景 and scope=全部) |
| **Provider inventory** | Guarantee every provider is reachable | Scrollable list (right rail); search; click → detail / scoped graph |

- Package is an **abstraction tier for the structure graph**, not merely a filter that “gives up” on panorama.
- Drill-down: click/double-click a package node (or pick from left sidebar) → scoped Providers canvas may still show that package’s providers.
- Hierarchical / force layouts stay literal provider+type graphs for users who want them; tip steers large global graphs to 全景结构 + 清单.

### Non-goals (Approach C)

- Making every provider label simultaneously readable on one fitted canvas.
- Treating package scope as the only completeness mechanism.

---

## Problem (original)

Global Providers with hierarchical layout becomes a thin horizontal strip. Fit-to-screen makes every node a speck. Auto-swapping hierarchical → physics was the wrong fix (hijacked user layout choice). Top-K truncation conflicts with the product need for a **full panorama of every provider**.

## Goals

1. **All providers remain reachable** via inventory (Approach C); structure canvas uses package abstraction by default.
2. **Same canvas, zoom-stratified reading** where useful on scoped provider maps.
3. Explicit **全景 (panorama)** layout = structure map at global scope.
4. **层级 / 力导向 stay honest**: selecting them uses that algorithm; never silently rewrite the user’s choice.
5. Keep existing tools (package scope, group aggregate, edge declutter, module map, details, Trace).

## Non-Goals

- Separate minimap widget (user chose same-canvas zoom LOD).
- Forcing package selection before drawing.
- Replacing hierarchical with physics under the hood.
- Drawing objects as default architecture nodes.
- Perfect simultaneous readability of every label at overview zoom (impossible at scale; LOD handles this).

## Product Decisions (locked)

| Decision | Choice |
|---|---|
| Default lens when opening `/` | Unchanged unless noted below: Providers may remain default |
| Full inventory | Panorama shows **all** providers in scope (global or selected package) |
| Dual-layer UX | **Same canvas zoom LOD** (not minimap, not mode toggle) |
| Layout honesty | Explicit **全景** option; hierarchical/force never auto-swapped |
| Providers default layout | **全景** when opening Providers at global scope; user can switch to 层级/力导向 |

## Design

### 1. Layout option: 全景

Toolbar `布局` gains:

- **全景** (new, recommended for large Providers)
- 层级布局 (unchanged semantics)
- 力导向 (unchanged semantics)

**Panorama algorithm (client-side, deterministic):**

1. Group provider nodes by package key (`output_pkg` / `function_pkg` / node `packagePath`; fallback `"_"`).
2. Sort packages by name; sort providers within package by id/name.
3. Place packages on a **grid of columns** (column count ≈ `ceil(sqrt(packageCount))`), each package a vertical stack of provider boxes with fixed cell size.
4. Package header is not a separate graph node; at overview zoom the **visible label** for a region is the package short name (via LOD), not an extra inventory node.
5. Cross-package edges kept; same-package edges optional via existing 边降噪.
6. Disable vis hierarchical layout and physics for panorama; set `{x,y}` (and optionally `fixed.x/y` during initial paint) so positions stay stable.

Types view may keep current layouts for this iteration; panorama is required for **Providers** first. Module map stays its own view.

### 2. Zoom LOD (same canvas)

Three bands (tunable constants):

| Band | Approx. scale | Labels shown |
|---|---|---|
| Overview | `< ~0.55` | Package short names on a representative node per package (or hub-only if single package); other nodes show `·` or empty |
| Mid | `~0.55–0.9` | Hubs + package reps; more provider short labels appear by degree |
| Detail | `> ~0.9` | All provider short labels in viewport; full name on hover/title |

Rules:

- **Nodes and edges are never removed by LOD** — only label visibility/text changes.
- Existing `labelLodVisibleIds` / `shortGraphLabel` are extended for package-band behavior; do not invent a second shell.
- Initial camera for panorama: **fit** the full map (structure panorama). User zooms for names. Density tip may say: “缩放过小看结构；放大读 Provider 名”.
- 「适应全图」 = fit panorama. 「聚焦枢纽」 = focus highest-degree provider at detail scale (~1.05).

### 3. Crowding interaction (revised for panorama)

When Providers + 全景 + global scope:

1. Prefer panorama packing over hierarchical strip.
2. Edge declutter may still hide same-package edges (toggle remains).
3. Group aggregate still available; if on, panorama packs **group nodes** (or expanded members) consistently — do not drop providers from the dataset unless user enabled aggregation.
4. **No auto layout swap.**
5. Soft Top-K budget is **off for panorama** (all providers). Soft budget may still apply to hierarchical/force if those paths remain dense — declare in UI if truncated. Panorama path must not truncate.

Package sidebar still scopes the panorama to one package when selected (fewer columns; detail labels appear earlier).

### 4. Honesty / messaging

- If user selects 层级 and the graph is wide, tip suggests switching to **全景** or selecting a package — **do not** change the dropdown for them.
- When auto-defaulting Providers → 全景 on first load of a large global graph, set the layout control to 全景 so UI matches reality.

## Acceptance

1. Global Providers + 全景: every provider in current API payload appears as a node; fit shows a 2D package map (not a 1-row strip).
2. Zoom out: package-oriented labels dominate; zoom in: provider names become readable without removing nodes.
3. Switching to 层级 uses hierarchical layout; switching back to 全景 restores panorama positions; no silent overrides.
4. Package scope + 全景 still shows all providers in that package.
5. Existing module map / aggregate / edge declutter / details / Trace still work.
6. Unit tests cover panorama position determinism and LOD band label selection.

## Implementation Boundaries

### Touch

- `dixhttp/template.html` — layout select option 全景; tip copy.
- `dixhttp/static/js/legacy/app.js` — Providers render path for panorama positions + camera; persist layout choice.
- `dixhttp/static/js/graph_state.mjs` (+ tests) — `layoutPanoramaPositions`, LOD band helpers.
- Cache bump on static assets.

### Do not touch

- Server dependency APIs (unless a bug blocks package keys).
- Reintroducing `/next`.
- Minimap DOM widget.

## Open questions (resolved)

- Minimap vs zoom LOD → **zoom LOD**.
- Truncate for readability → **no on panorama path**.
- Hijack hierarchical → **no**; add explicit 全景 instead.
