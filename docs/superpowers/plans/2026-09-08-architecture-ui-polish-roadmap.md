# Architecture UI Polish Roadmap

**Date:** 2026-09-08  
**Status:** Living roadmap (not a single sprint)  
**Scope:** Legacy `/` Providers · Types · Module map — readability, hide/filter, pyramid depth UX  
**Related:** `docs/superpowers/specs/2026-09-08-hide-node-downstream-design.md`

## Principles

- Ship small vertical slices; each phase should be demoable in the http example.
- Prefer canvas/session UX over server API changes unless data identity is wrong.
- Do not block unrelated dix/core work on this roadmap.

---

## Phase 0 — Done (baseline)

| Item | Notes |
|------|--------|
| Hide node + downstream | Right-click / Shift+click / toolbar / inventory |
| Session-shared hide seeds | Providers · Types · Module map |
| Toast + undo / restore list | |
| Label disambiguation | `billing/handler.Handler`, `auth.Worker` |
| Pkg-aware provider edges | Avoid false cross-domain fan-out |
| Microservice example layout | domain layers + bootstrap + plugins |

---

## Phase 1 — Clarity & fewer surprises (near-term)

**Goal:** Users understand what the graph is showing without asking “为什么只有 N 级 / 藏错了吗”.  
**Target window:** next 1–2 focused sessions when touching dixhttp UI.

| # | Item | Why | Acceptance |
|---|------|-----|------------|
| 1.1 | [x] Depth select shows **actual max level** (disable or omit empty 5/10) | Stops “深度 10 只有 4 级” confusion | Depth options ≤ computed max; label e.g. `全部 (4 级)` |
| 1.2 | [x] Hide **preview highlight** before commit (context menu open) | Prevents accidental truncation | Downstream nodes outline while menu open; cancel clears |
| 1.3 | [x] Auto-fit after hide / restore | Graph doesn’t stay zoomed on empty space | After hide/unhide, camera fit once |
| 1.4 | [x] Inventory marks hidden rows | Completeness channel stays honest | Hidden providers grey +「已隐」; 复 restores |

**Exit:** Phase 1 checklist green on `example/http` demo; one short note in changelog / commit message.

---

## Phase 2 — Faster “main graph” workflows (next)

**Goal:** Reach a clean business-only pyramid in ≤2 actions.

| # | Item | Why | Acceptance |
|---|------|-----|------------|
| 2.1 | [x] Context menu: **hide package / subtree** (e.g. `…/plugins`) | Scale fixtures & plugin forests | One action hides all matching packagePath prefix + downstream |
| 2.2 | [x] Preset: **只留业务入口** (hide Dix + diag Timeout chain + plugins) | Default noise off the tip | One toolbar action; reversible via 已隐藏 |
| 2.3 | [x] Keyboard: `H` hide selected, `U` undo last, `Esc` close menu | Power users | Hint on context menu |
| 2.4 | [x] Module map right-click = same menu as Providers | Consistency | Module containment / package hide works via shared bindGraphInteractions |

**Exit:** Can demo “hide plugins + diag → clean Application chain” in under 10 seconds.

---

## Phase 3 — Shareable & durable state (later)

**Goal:** Same filtered view across reloads / teammates without teaching click paths.

| # | Item | Why | Acceptance |
|---|------|-----|------------|
| 3.1 | [x] URL sync for hide seeds (+ optional depth/package) | Share “the graph I’m looking at” | Reload restores; link opens same cut |
| 3.2 | [x] Hide history stack (multi-undo) | Continuous trimming sessions | Undo >1 step; history capped (e.g. 20) |
| 3.3 | [x] Toast stacking / placement vs density tip | Less UI collision | Density tip left, hide toast right; no overlap |

**Exit:** Shared URL reproduces hide set; multi-undo feels safe.

---

## Phase 4 — Structural / data (opportunistic)

**Goal:** Fix root causes of crowding, not only filters.

| # | Item | Why | When |
|---|------|-----|------|
| 4.1 | [x] Stronger type identity in API if still collapsing | Fewer bogus edges | `input_pkgs` + pkg\\x00type dedupe in GetProviderDetails / aggregate / graph edges |
| 4.2 | [x] Example scale via real domains, not pads | Meaningful module map | `example/http` microservice layout (no ScaleFixture) |
| 4.3 | [x] Soft guidance when entry count > N | Onboarding | Density tip CTA「只留业务入口」when entries > 12, unscoped |

---

## Suggested cadence

| Cadence | Focus |
|---------|--------|
| **Whenever shipping a dixhttp UI fix** | Prefer pulling **one** Phase 1 item into the same PR |
| **Dedicated polish half-day** | Finish remaining Phase 1, start 2.1–2.2 |
| **After Phase 2 feels good** | Phase 3 URL/history |
| **Opportunistic** | Phase 4 with related core/example work |

Do **not** schedule Phase 3–4 as blockers for other features.

---

## Tracking

- Keep this file as the queue; check items off in place when done (`[x]`).
- New ideas: append under the right phase or a `## Backlog` section below — don’t start a parallel doc unless scope splits.

## Backlog (unsorted)

- Merge consecutive hide toasts into one line  
- “Hide everything except selected neighborhood” invert mode  
- Persist hide seeds beyond session (opt-in localStorage)  
- Further canvas LOD / label density tweaks after layout-v1 bake-in  

---

## Layout chrome (2026-09-08)

Bold IA pass (`arch-layout-v1` / `legacy-arch29`): compact header, primary toolbar +「更多」, grouped package list, right inspector tabs, density tip only when actionable, no floating Trace FAB.

## Next concrete pick

Architecture polish phases 0–4 are green. Remaining work is **Backlog** (merge hide toasts, invert-hide mode, opt-in localStorage) or unrelated dix/core features.
