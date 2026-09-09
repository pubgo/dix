# Hide Node And Downstream Design

**Date:** 2026-09-08  
**Status:** Implemented  
**Scope:** Legacy architecture UI — hide selected canvas nodes and their depends-on descendants across Providers / Types / Module map.

## Behavior

- **Primary:** right-click node →「隐藏此节点及下游 (N)」
- **Fast path:** Shift+click node; or select then toolbar「隐藏及下游」
- **Inventory:** hover Provider row →「隐」
- Remove the seed and all downstream nodes (edge direction: consumer → dependency)
- Upstream consumers remain; edges into hidden set are dropped
- Seeds in `sessionStorage`; shared by Providers / Types / Module map
- Toast with undo; dropdown「已隐藏 N」to restore

## Non-goals

- Package-path blacklist UI
- Objects canvas
- Persist hide beyond session without URL (opt-in localStorage is backlog)

## Persistence (Phase 3)

- `sessionStorage` + URL query: `hide` (pipe-separated seeds / `pkg:prefix`), optional `depth`, `pkg`
- URL `hide` overrides session on load; changes `replaceState` the address bar
- Multi-step undo via in-memory snapshot stack (cap 20); `U` / toast 撤销
