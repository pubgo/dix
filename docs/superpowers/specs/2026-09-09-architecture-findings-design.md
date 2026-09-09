# Architecture Findings (结论优先体检)

> Status: implemented in UI · Date: 2026-09-09 · Approach A: frontend rules in `graph_state.mjs`

## Goal

Default `/` answers *where is the DI structure unhealthy?* via a findings list, not a full Providers graph.

## UX

- Default `currentView: 'findings'`
- Click finding → Providers/Types/Modules with package or neighborhood scope; chip「← 体检」returns
- Empty state: clean message + open full graph
- Runtime `/api/issues` unchanged (errors/slow), separate from static structure smells

## v1 rules

| kind | trigger | severity |
|---|---|---|
| `cross_bucket` | plugins/infra → domain, or domainA → domainB | warn |
| `super_hub` | provider degree ≥ 12 | warn |
| `entry_fanout` | indegree-0 entries ≥ 8 | info |
| `fat_package` | providers in one package ≥ 12 | info |

## API

`buildArchitectureFindings(allData, opts?)` → Finding[] with `action.view` / `focus*` / `package` / `keepNeighborhood`.
