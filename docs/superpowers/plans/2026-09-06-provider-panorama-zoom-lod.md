# Provider Panorama + Zoom LOD Implementation Plan

> **Status: CANCELLED (2026-09-07)** — 全景 layout removed from product; do not continue this plan.

**Goal (historical):** Add an explicit Providers「全景」layout that places every provider on a 2D package map, with same-canvas zoom LOD (structure far / names near), without hijacking 层级/力导向.

**Architecture:** Pure helpers in `graph_state.mjs` compute panorama `{x,y}` and LOD label maps; legacy `renderGraph` applies them when `currentLayout === 'panorama'`. Hierarchical and force paths stay unchanged. Panorama disables vis hierarchical + physics and does not apply Top-K budget.

**Tech Stack:** Alpine legacy UI, vis-network, shared ESM helpers + `node --test`.

**Spec:** `docs/superpowers/specs/2026-09-06-provider-panorama-zoom-lod-design.md`

## Global Constraints

- Panorama path: **all** providers in scope stay on canvas (no Top-K).
- Never silently swap 层级 ↔ 力导向 ↔ 全景.
- LOD changes labels only, never removes nodes/edges.
- Providers global default layout: **全景**.
- Cache bump static assets after UI wiring.

## File map

| File | Responsibility |
|---|---|
| `dixhttp/static/js/graph_state.mjs` | `layoutPanoramaPositions`, `resolveLodBand`, `labelLodByBand` |
| `dixhttp/static/js/graph_view.test.mjs` | Unit tests for helpers |
| `dixhttp/static/js/legacy/app.js` | Apply panorama in `renderGraph`; skip budget; camera; persist layout |
| `dixhttp/template.html` | Layout `<option value="panorama">全景</option>`; tip copy |

---

### Task 1: Panorama positions helper (TDD)

**Files:**
- Modify: `dixhttp/static/js/graph_state.mjs`
- Test: `dixhttp/static/js/graph_view.test.mjs`

**Interfaces:**
- Produces: `layoutPanoramaPositions(nodes, opts?) → { [id]: { x, y } }`
- Uses: `nodePackageKey(node)` for grouping; deterministic sort by package then id

- [x] **Step 1: Write failing tests** for multi-package grid, deterministic coords, single-package stack
- [x] **Step 2: Run tests — expect FAIL**
- [x] **Step 3: Implement `layoutPanoramaPositions`**
- [x] **Step 4: Run tests — expect PASS**

---

### Task 2: Zoom LOD bands (TDD)

**Files:**
- Modify: `dixhttp/static/js/graph_state.mjs`
- Test: `dixhttp/static/js/graph_view.test.mjs`

**Interfaces:**
- Produces: `resolveLodBand(scale) → 'overview' | 'mid' | 'detail'`
- Produces: `labelLodByBand(nodes, edges, scale, opts?) → Map<id, string>` (overview: package short name on one rep per package; mid: hubs+reps; detail: all short labels)

- [x] **Step 1: Write failing tests** for bands and overview package reps
- [x] **Step 2: Implement helpers; export on `window.DIXGraphState`**
- [x] **Step 3: Tests PASS**

---

### Task 3: Wire panorama into legacy Providers render

**Files:**
- Modify: `dixhttp/static/js/legacy/app.js`
- Modify: `dixhttp/template.html`

- [x] Default `currentLayout: 'panorama'`
- [x] Add layout option 全景 in template
- [x] In `renderGraph` when layout is panorama (providers/modules-compatible nodes): skip `applyGraphBudget`; compute positions; `getNetworkOptions` with hierarchical off + physics off; assign x/y/fixed; use `labelLodByBand` in `applyLabelLod`
- [x] Persist `currentLayout` in localStorage with other prefs
- [x] Density tip for hierarchical+wide: suggest 全景 (do not auto-switch)
- [x] Panorama tip: “缩放过小看结构；放大读 Provider 名”
- [x] Bump `?v=legacy-arch8`

- [x] **Verify:** `node --test dixhttp/static/js/graph_view.test.mjs` && `go test ./dixhttp/...`

---

### Task 4: Smoke acceptance

- [ ] Manual: global Providers + 全景 → 2D package clusters, all nodes present
- [ ] Zoom out → package labels; zoom in → provider names
- [ ] Switch to 层级 → hierarchical; back to 全景 → panorama again

---

## Done when

Acceptance criteria in the spec §Acceptance all hold; helpers covered by unit tests.
