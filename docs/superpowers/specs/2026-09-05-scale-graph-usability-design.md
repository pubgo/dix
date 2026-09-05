# Scale-Safe Dependency Visualization Design

## Problem

The dependency UI is usable for small containers, but it does not scale to the target shape of approximately 100 modules, 200 providers, and 400 objects. The issue is not primarily browser capacity. The UI answers several different questions with one global graph, and derived provider data can multiply edges and collapse distinct provider instances.

An example container with 10 business modules, 200 provider graph nodes, and 192 objects produced 72 aggregated providers and 3,860 edges from `/api/dependencies`. This happened because providers from the same generic factory source line were merged and their inputs and outputs were later cross-producted.

## Goals

1. Preserve provider identity so generic factories, multi-output providers, runtime metrics, errors, traces, and graph nodes can be correlated correctly.
2. Make dependency edges reflect actual declared provider relationships instead of Cartesian products.
3. Make the first graph experience module-centric, bounded, and actionable.
4. Keep objects as state and detail rather than requiring every object to be a graph node.
5. Guarantee bounded views: default module map, module drill-down, and ego graph each have explicit node and edge limits.
6. Keep legacy endpoints and views compatible while the new model becomes the default.

## Non-Goals

This iteration does not replace `vis-network`, introduce WebGL, remove the legacy UI, or expose every object as a graph node. Those are follow-ups only if bounded module and ego views still fail usability targets.

## Data Model

Provider registration identity is the correlation key. A logical registration may produce multiple graph outputs for a struct provider, but two calls to the same generic factory are distinct registrations. Runtime state remains attached to the concrete provider/output pair.

The visualization provider shape gains:

- `registration_id`: stable identity for one logical provider registration within the current process.
- `provider_id`: stable identity for the registration/output pair used by API clients.

Dependency projections must group outputs by registration identity, not source file and line. The full projection must emit one declared edge per real input/output relationship; it must never synthesize all `inputs × outputs` relationships after aggregation.

Module data uses `reflect.Type.PkgPath()`-derived module identity. Generic type names must not be parsed as package paths.

## API Behavior

Phase one evolves existing endpoints:

- `/api/dependencies` uses registration-aware provider identity and emits real edges.
- `/api/runtime-stats` includes registration and provider identities.
- `/api/modules` remains the bounded module aggregate.
- `/api/ego` reports instantiated state only when an object node exists.
- `/api/stats` aggregates resolved counts by type and reports internally consistent module/type/provider counts.
- `/api/packages` uses provider output package data and never displays malformed generic package names such as `main.Plugin[main`.

The next major endpoint group will be `/api/graph/summary`, `/api/graph/modules`, `/api/graph/module`, `/api/graph/ego`, `/api/graph/node`, and `/api/issues`. Those endpoints should serve progressive disclosure and issue-centric debugging.

## Frontend Behavior

The default graph mode is module map. Selecting a module opens a bounded module view; selecting a type opens an ego graph; selecting a provider opens provider details and runtime state.

Rendering budgets are explicit:

- Module map: at most 100 nodes and 300 edges.
- Module detail: at most 150 nodes and 400 edges.
- Ego graph: default depth 2, maximum depth 5, and an edge cap.

When data exceeds a budget, the UI shows a density warning and offers tables, Top-K hubs, or a narrower selection instead of rendering an unreadable graph.

The graph canvas must retain a usable minimum size. Narrow layouts move details into a drawer or bottom sheet instead of shrinking the canvas to an unusable area.

State and filters are encoded in the hash query so views can be refreshed and shared. Requests use cancellation when a view changes. Runtime metrics join by provider identity and output type, not function name alone.

## Correctness Rules

1. Two registrations from the same source line can have different identities.
2. Outputs from one struct-producing registration can share a registration identity.
3. Runtime stats are keyed by concrete provider/output, not only function name.
4. An ego node is instantiated only when an object node exists.
5. Package/module names must be valid package paths or `(anonymous)`.
6. A graph may be hidden or degraded, but it must not silently show false relationships.

## Acceptance

- Main and example tests pass with race detection.
- A generic provider fixture produces distinct provider identities and one edge per actual input/output pair.
- Runtime stats correlate to the correct provider/output pair.
- `/next#/graph` defaults to the module map and renders useful content without first loading `/api/dependencies`.
- Module map respects the phase-one node/edge budgets.
- Graph canvas remains usable at narrow viewports.
- Diagnostics can navigate from an issue to the provider graph or trace tree in at most three interactions.
