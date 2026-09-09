# Providers Pyramid View Design

> Status: implemented 2026-09-07 · Branch: `codex/scale-graph-usability`

## Goal

Providers 视图只画 Provider，并按依赖金字塔分层：塔尖是少数入口，深度控件裁层，包/Provider 过滤看相关上下游。

## Decisions

| Topic | Choice |
|---|---|
| Nodes | Provider only（Type 压成依赖边） |
| Edge | `A→B` = A 依赖 B（消费 B 的产出） |
| Entry (塔尖) | 未被其它 Provider 消费（子图内 indegree 0） |
| Depth N | 只保留 level ≤ N；`0` = 不截层 |
| Package filter | 包内 Provider 为种子，展开相关上下游，再套深度 |
| Provider filter | 双击或清单点选设种子，同上 |
| Crowding | 深度 + 包/Provider 范围；不再靠 Type 节点或 Top-K 伪装全景 |

## Non-goals

- 不恢复「全景」布局
- Types 视图仍为 type→type 图
- 不在服务端重算层级（当前 client helpers 足够 example 规模）

## Implementation

- Helpers: `dixhttp/static/js/graph_state.mjs` — `buildProviderDependencyGraph`, `assignProviderPyramidLevels`, `truncateProviderPyramid`, `providerRelatedSubgraph`, `buildProvidersPyramidView`
- UI: `dixhttp/static/js/legacy/app.js` Providers `renderGraph` path; `pyramidFocusProviderId`
- Cache: `legacy-arch15`
