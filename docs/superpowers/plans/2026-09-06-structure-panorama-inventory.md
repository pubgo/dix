# Structure Panorama + Inventory (Approach C) Plan

> **Status: CANCELLED (2026-09-07)** — 全景 removed; keep Provider inventory + package scope / 模块地图 instead.

**Goal (historical):** Replace “literal all-provider panorama” with dual channels: readable **package-structure graph** + complete **provider inventory**.

**Architecture:** Panorama with no package scope draws package nodes and cross-package edges from `allData`. Right rail always lists every provider (or scoped list). Click package → `selectPackage`; click inventory row → focus/detail. Hierarchical/force unchanged.

**Tech Stack:** Alpine legacy UI, `graph_state.mjs`, node tests.

**Spec amend:** `docs/superpowers/specs/2026-09-06-provider-panorama-zoom-lod-design.md` → Approach C.

---

### Task 1: `buildPanoramaStructureGraph` + inventory helper (TDD)

- [ ] Tests for package nodes, cross-pkg edges, inventory completeness
- [ ] Implement helpers; export on window

### Task 2: Wire legacy render + sidebar

- [ ] Panorama global → structure graph; panorama + package → provider map
- [ ] Always show inventory; double-click package drills scope
- [ ] Simplify panorama tips; bump `legacy-arch13`

### Task 3: Verify

- [ ] `node --test` + `go test ./dixhttp/...`
