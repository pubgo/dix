# Scale-Safe Dependency Visualization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make dependency visualization correct and usable at approximately 100 modules, 200 providers, and 400 objects by introducing registration-aware data and bounded module-first views.

**Architecture:** Add a stable registration identity to provider registrations, correlate runtime state by provider/output identity, stop source-line aggregation from creating false edges, and make the new graph UI module-first with explicit rendering budgets. Existing legacy APIs remain compatible while the new UI becomes the supported scale path.

**Tech Stack:** Go 1.24+, net/http/httptest, reflection-based Dix graph, vendored vanilla JS, vis-network, hash routing.

**Spec:** `docs/superpowers/specs/2026-09-05-scale-graph-usability-design.md`

## Global Constraints

- Do not introduce third-party Go or JavaScript dependencies.
- Preserve all existing public Dix APIs.
- Preserve legacy `/api/dependencies` response compatibility except additive fields and corrected edges/identities.
- Use TDD for every behavior change; run focused tests before implementation and after implementation.
- Keep graph phase-one budgets at module map 100 nodes/300 edges, module detail 150 nodes/400 edges, ego maximum depth 5.
- Do not show an empty default ego graph; `/next#/graph` defaults to module map.
- Every commit must leave `go test -race ./...` green in the root module.

---

### Task 1: Add Logical Provider Registration Identity

**Files:**
- Modify: `dixinternal/provider.go`
- Modify: `dixinternal/dix.go:1119-1220`
- Modify: `dixinternal/dix.go:1350-1370`
- Test: `dixinternal/provider_identity_test.go`

**Interfaces:**
- Consumes: private `providerFn`, `Dix.handleProvide`, and `Dix.provide`.
- Produces: `providerFn.registrationID uint64`; `handleProvide(fnVal reflect.Value, outType reflect.Type, inputs []*providerInputType, registrationID uint64) error`.

- [x] **Step 1: Write the failing identity tests**

Create `dixinternal/provider_identity_test.go`:

```go
package dixinternal

import (
	"reflect"
	"testing"
)

type identityA struct{}
type identityB struct{}
type identityAggregate struct {
	A *identityA
	B *identityB
}

func TestStructProviderOutputsShareRegistrationID(t *testing.T) {
	di := New()
	di.Provide(func() identityAggregate {
		return identityAggregate{A: &identityA{}, B: &identityB{}}
	})

	ids := make(map[string]uint64)
	for _, provider := range di.providers[reflect.TypeOf(&identityA{})] {
		ids["A"] = provider.registrationID
	}
	for _, provider := range di.providers[reflect.TypeOf(&identityB{})] {
		ids["B"] = provider.registrationID
	}
	if ids["A"] == 0 || ids["A"] != ids["B"] {
		t.Fatalf("struct outputs should share one registration ID: %#v", ids)
	}
}

func TestDistinctClosureRegistrationsHaveDistinctRegistrationIDs(t *testing.T) {
	di := New()
	for _, name := range []string{"first", "second"} {
		value := name
		di.Provide(func() *identityA { return &identityA{} })
		if value == "" {
			t.Fatal("closure registration setup unexpectedly empty")
		}
	}

	providers := di.providers[reflect.TypeOf(&identityA{})]
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(providers))
	}
	if providers[0].registrationID == providers[1].registrationID {
		t.Fatal("distinct registrations must not share a registration ID")
	}
}
```

- [x] **Step 2: Run the focused test and verify RED**

Run:

```bash
unset GOROOT; export PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin; go test ./dixinternal -run Test.*(RegistrationID|RegistrationIDs) -count=1
```

Expected: compilation fails because `registrationID` and the new `handleProvide` signature do not exist.

- [x] **Step 3: Implement registration identity**

In `dixinternal/provider.go`, add the field:

```go
type providerFn struct {
	fn             reflect.Value
	inputList      []*providerInputType
	output         *providerOutputType
	hasError       bool
	registrationID uint64
}
```

In `dixinternal/dix.go`, add a container counter and change the private signature:

```go
func (dix *Dix) handleProvide(fnVal reflect.Value, outType reflect.Type, inputs []*providerInputType, registrationID uint64) error {
	// existing implementation, but construct:
	provider := &providerFn{fn: fnVal, inputList: inputs, hasError: hasError, registrationID: registrationID}
```

Pass the same `registrationID` into the recursive struct-field call. At the top-level call in `provide`, allocate a new identity:

```go
dix.registrationSeq++
if err := dix.handleProvide(fnVal, typ.Out(0), inputs, dix.registrationSeq); err != nil {
```

Add `registrationSeq uint64` beside `graph` in the private `Dix` struct. Container writes are documented single-threaded, so no atomic is required.

- [x] **Step 4: Run the focused test and verify GREEN**

Run:

```bash
unset GOROOT; export PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin; go test ./dixinternal -run Test.*(RegistrationID|RegistrationIDs) -count=1
```

Expected: both tests pass.

- [x] **Step 5: Run the root race suite**

Run:

```bash
unset GOROOT; export PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin; go test -race ./...
```

Expected: all packages pass.

- [x] **Step 6: Commit**

```bash
git add dixinternal/provider.go dixinternal/dix.go dixinternal/provider_identity_test.go
git commit -m "feat(dixinternal): add provider registration identity"
```

---

### Task 2: Expose Stable Provider IDs and Fix Dependency Aggregation

**Files:**
- Modify: `dixinternal/api.go:200-260`
- Modify: `dixhttp/server.go:650-820`
- Test: `dixhttp/server_provider_identity_test.go`

**Interfaces:**
- Consumes: `providerFn.registrationID` from Task 1.
- Produces: `ProviderDetails.ProviderID string`, `ProviderInfo.ProviderIDs []string`, and `providerAggregateKey(detail dixinternal.ProviderDetails) string`.

- [x] **Step 1: Write the failing API projection test**

Create `dixhttp/server_provider_identity_test.go`:

```go
package dixhttp

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	dix "github.com/pubgo/dix/v2"
	"github.com/pubgo/dix/v2/dixinternal"
)

type projectionInput struct{}
type projectionOutputA struct{}
type projectionOutputB struct{}
type projectionAggregate struct {
	A *projectionOutputA
	B *projectionOutputB
}

func TestDependenciesPreserveRegistrationAndOutputIdentity(t *testing.T) {
	container := dix.New()
	dix.Provide(container, func() projectionAggregate {
		return projectionAggregate{A: &projectionOutputA{}, B: &projectionOutputB{}}
	})

	server := NewServer(container)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest("GET", "/api/dependencies", nil))
	if recorder.Code != 200 {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var response DependencyData
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}

	registrationIDs := map[uint64]bool{}
	providerIDs := map[string]bool{}
	for _, provider := range response.Providers {
		if provider.OutputType != "*dixhttp.projectionOutputA" && provider.OutputType != "*dixhttp.projectionOutputB" {
			continue
		}
		if provider.RegistrationID == 0 || len(provider.ProviderIDs) == 0 {
			t.Fatalf("provider identities must not be empty: %+v", provider)
		}
		registrationIDs[provider.RegistrationID] = true
		for _, providerID := range provider.ProviderIDs {
			providerIDs[providerID] = true
		}
	}
	if len(registrationIDs) != 1 {
		t.Fatalf("one struct registration should produce one registration_id, got %v", registrationIDs)
	}
	if len(providerIDs) != 2 {
		t.Fatalf("two outputs should produce two provider IDs, got %v", providerIDs)
	}
}

func TestDistinctSameLineRegistrationsDoNotCartesianProductEdges(t *testing.T) {
	container := dix.New()
	for i := 0; i < 2; i++ {
		dix.Provide(container, func() *projectionOutputA { return &projectionOutputA{} })
	}

	server := NewServer(container)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest("GET", "/api/dependencies", nil))

	var response DependencyData
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	targetCount := 0
	for _, provider := range response.Providers {
		if provider.OutputType == "*dixhttp.projectionOutputA" {
			targetCount++
		}
	}
	if targetCount != 2 {
		t.Fatalf("expected distinct provider nodes, got %d", targetCount)
	}
	for _, provider := range response.Providers {
		if provider.OutputType != "*dixhttp.projectionOutputA" {
			continue
		}
		if len(provider.InputTypes)*len(provider.OutputTypes) > 1 {
			t.Fatalf("unrelated input/output pairs created a Cartesian product: %+v", provider)
		}
	}
}
```

- [x] **Step 2: Run the focused HTTP test and verify RED**

Run:

```bash
unset GOROOT; export PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin; go test ./dixhttp -run Test(DependenciesPreserve|DistinctSameLine) -count=1
```

Expected: compilation fails because `RegistrationID` and `ProviderID` do not exist.

- [x] **Step 3: Project and aggregate by registration identity**

In `dixinternal.ProviderDetails`, add:

```go
RegistrationID uint64 `json:"registration_id"`
ProviderID     string `json:"provider_id"`
```

In `GetProviderDetails`, construct IDs:

```go
registrationID := providerFn.registrationID
providerID := fmt.Sprintf("provider_%d_%s", registrationID, outputType.String())
```

In `dixhttp.ProviderInfo`, add:

```go
RegistrationID uint64   `json:"registration_id"`
ProviderIDs    []string `json:"provider_ids"`
```

A logical registration with multiple outputs must expose one concrete `provider_id` per output in `ProviderIDs`. It cannot use a single `provider_id` without losing runtime-state correlation.

Change the aggregate bucket key to:

```go
func providerAggregateKey(detail dixinternal.ProviderDetails) string {
	if detail.RegistrationID != 0 {
		return fmt.Sprintf("registration_%d", detail.RegistrationID)
	}
	// Keep the legacy fallback only for externally constructed fixtures.
	if detail.FunctionFile != "" && detail.FunctionLine > 0 {
		return fmt.Sprintf("%s:%d", detail.FunctionFile, detail.FunctionLine)
	}
	if detail.FunctionName != "" {
		return detail.FunctionName
	}
	if detail.OutputType != "" {
		return detail.OutputType
	}
	return "unknown"
}
```

Copy `RegistrationID` into `ProviderInfo`, and collect every output-specific `detail.ProviderID` into `ProviderIDs`. Edge generation must iterate actual detail input/output pairs while accumulating bucket metadata; it must not regenerate from merged `InputTypes × OutputTypes`.

Refactor the bucket to retain raw `dixinternal.ProviderDetails` values so edge building can preserve exact relationships.

- [x] **Step 4: Run the focused HTTP test and verify GREEN**

Run:

```bash
unset GOROOT; export PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin; go test ./dixhttp -run Test(DependenciesPreserve|DistinctSameLine) -count=1
```

Expected: both tests pass.

- [x] **Step 5: Run existing dependency regression tests**

Run:

```bash
unset GOROOT; export PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin; go test ./dixhttp -run TestHandleDependencies -count=1
```

Expected: pass, including the existing multi-output aggregation test.

- [x] **Step 6: Commit**

```bash
git add dixinternal/api.go dixhttp/server.go dixhttp/server_provider_identity_test.go
git commit -m "fix(dixhttp): preserve provider identity and dependency edges"
```

---

### Task 3: Correlate Runtime Stats by Provider Identity

**Files:**
- Modify: `dixinternal/api.go:170-310`
- Modify: `dixhttp/static/js/views/graph.js:42-55`
- Modify: `dixhttp/static/js/views/graph.js:213-240`
- Test: `dixinternal/runtime_stats_identity_test.go`

**Interfaces:**
- Consumes: `providerFn.registrationID`.
- Produces: `ProviderRuntimeStats.RegistrationID uint64`, `ProviderRuntimeStats.ProviderID string`, and `statForProvider(provider ProviderInfo) ProviderRuntimeStats | null`.

- [x] **Step 1: Write the failing runtime identity test**

Create `dixinternal/runtime_stats_identity_test.go`:

```go
package dixinternal

import (
	"reflect"
	"testing"
)

type runtimeTarget struct{ Name string }

func TestRuntimeStatsIncludeConcreteProviderIdentity(t *testing.T) {
	di := New()
	di.Provide(func() *runtimeTarget { return &runtimeTarget{Name: "ready"} })
	_ = di.TryInject(func(*runtimeTarget) {})

	stats := di.GetProviderRuntimeStats()
	for _, stat := range stats {
		if stat.OutputType != "*dixinternal.runtimeTarget" {
			continue
		}
		if stat.RegistrationID == 0 {
			t.Fatal("expected non-zero registration ID")
		}
		if stat.ProviderID == "" {
			t.Fatal("expected provider ID")
		}
		if stat.CallCount != 1 {
			t.Fatalf("call count = %d, want 1", stat.CallCount)
		}
		return
	}
	t.Fatalf("target stat not found in %+v", stats)
}

func TestRuntimeStatsDoNotDeduplicateDistinctClosures(t *testing.T) {
	di := New()
	for i := 0; i < 2; i++ {
		di.Provide(func() *runtimeTarget { return &runtimeTarget{} })
	}
	stats := di.GetProviderRuntimeStats()
	count := 0
	for _, stat := range stats {
		if stat.OutputType == reflect.TypeOf(&runtimeTarget{}).String() {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("provider stats = %d, want 2", count)
	}
}
```

- [x] **Step 2: Run the focused runtime test and verify RED**

Run:

```bash
unset GOROOT; export PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin; go test ./dixinternal -run TestRuntimeStats -count=1
```

Expected: compilation fails because identity fields do not exist.

- [x] **Step 3: Emit concrete identity fields**

Add fields to `ProviderRuntimeStats`:

```go
RegistrationID uint64 `json:"registration_id"`
ProviderID     string `json:"provider_id"`
```

In `GetProviderRuntimeStats`, replace function-name-only deduplication with concrete provider/output identity:

```go
seen := make(map[string]bool)
identity := fmt.Sprintf("%d:%s", p.registrationID, outputType)
if seen[identity] {
	continue
}
seen[identity] = true
item.RegistrationID = p.registrationID
item.ProviderID = fmt.Sprintf("provider_%d_%s", p.registrationID, outputType)
```

Update the graph JS stat lookup to use:

```js
function statForProvider(provider) {
  const stats = state.runtimeStats || [];
  for (const providerID of provider.provider_ids || []) {
    const exact = stats.find(s => s.provider_id === providerID);
    if (exact) return exact;
  }
  return stats.find(s => !s.provider_id && s.function_name === provider.function_name) || null;
}
```

Use this helper in provider detail and error coloring. Keep the fallback only for stale cached responses.

- [x] **Step 4: Run the focused runtime test and verify GREEN**

Run:

```bash
unset GOROOT; export PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin; go test ./dixinternal -run TestRuntimeStats -count=1
```

Expected: both tests pass.

- [x] **Step 5: Run race tests**

Run:

```bash
unset GOROOT; export PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin; go test -race ./...
```

Expected: all packages pass.

- [x] **Step 6: Commit**

```bash
git add dixinternal/api.go dixhttp/static/js/views/graph.js dixinternal/runtime_stats_identity_test.go
git commit -m "fix(dixinternal): identify runtime stats by provider output"
```

---

### Task 4: Correct Module, Ego State, and Resolved Aggregation

**Files:**
- Modify: `dixinternal/graph_query.go:40-260`
- Modify: `dixinternal/graph_query.go:265-360`
- Modify: `dixhttp/server.go:414-540`
- Test: `dixinternal/graph_query_projection_test.go`
- Test: `dixhttp/server_packages_test.go`

**Interfaces:**
- Consumes: `Graph` nodes/edges and cached provider details.
- Produces: accurate `ModuleGraph`, `EgoGraph` instantiated state, `ResolvedTopN`, and `PackageInfo`.

- [ ] **Step 1: Write failing projection tests**

Create `dixinternal/graph_query_projection_test.go`:

```go
package dixinternal

import (
	"reflect"
	"testing"
)

type packageTarget struct{}

func TestEgoInstantiatedUsesObjectNodes(t *testing.T) {
	di := New()
	view := di.EgoGraph("*dixinternal.packageTarget", 1, "both")
	for _, node := range view.Nodes {
		if node.Label == "*dixinternal.packageTarget" && node.State == "instantiated" {
			t.Fatal("missing provider must not be reported instantiated")
		}
	}

	di.Provide(func() *packageTarget { return &packageTarget{} })
	di.addObject(reflect.TypeOf(&packageTarget{}), "")
	view = di.EgoGraph("*dixinternal.packageTarget", 1, "both")
	instantiated := false
	for _, node := range view.Nodes {
		if node.Label == "*dixinternal.packageTarget" && node.State == "instantiated" {
			instantiated = true
		}
	}
	if !instantiated {
		t.Fatal("object-bearing ego node should be instantiated")
	}
}

func TestResolvedTopNAggregatesByType(t *testing.T) {
	di := New()
	di.Provide(func() *packageTarget { return &packageTarget{} })
	for i := 0; i < 3; i++ {
		_, _ = di.TryInject(func(*packageTarget) {})
	}

	rows := di.ResolvedTopN(10)
	found := 0
	for _, row := range rows {
		if row.Type == "*dixinternal.packageTarget" {
			found++
			if row.Count != 3 {
				t.Fatalf("count = %d, want 3", row.Count)
			}
		}
	}
	if found != 1 {
		t.Fatalf("type rows = %d, want 1", found)
	}
}
```

Create `dixhttp/server_packages_test.go`:

```go
package dixhttp

import "testing"

func TestPackageInfoUsesResolvedOutputPackage(t *testing.T) {
	details := []dixinternal.ProviderDetails{
		{OutputType: "*main.Plugin[main.RoleReader]", OutputPkg: "main"},
		{OutputType: "*billing.Client", OutputPkg: "example/billing"},
	}
	packages := buildPackageInfos(details)
	if len(packages) != 2 {
		t.Fatalf("packages = %+v", packages)
	}
	if packages[0].Name != "main" || packages[1].Name != "example/billing" {
		t.Fatalf("malformed generic package path retained: %+v", packages)
	}
}
```

Add the required import `github.com/pubgo/dix/v2/dixinternal` to the HTTP test.

- [ ] **Step 2: Run focused projection tests and verify RED**

Run:

```bash
unset GOROOT; export PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin; go test ./dixinternal -run Test(EgoInstantiated|ResolvedTopNAggregates) -count=1; go test ./dixhttp -run TestPackageInfoUses -count=1
```

Expected: ego/state and aggregation/package helper failures.

- [ ] **Step 3: Fix the three projections**

In `EgoGraph`, build an instantiated set from object nodes before creating nodes:

```go
instantiated := make(map[reflect.Type]bool)
for key := range g.nIndex {
	if key.kind == NodeObject {
		instantiated[key.typ] = true
	}
}
```

Set node state to `"instantiated"` only when `instantiated[t]`; otherwise omit state.

In `ResolvedTopN`, accumulate counts by type before sorting:

```go
countsByType := make(map[string]int64)
for _, e := range g.eIndex {
	if e.Kind == EdgeResolved && e.Count > 0 {
		countsByType[g.nodes[e.To].Type.String()] += e.Count
	}
}
```

Then sort and truncate the materialized rows.

Extract package construction from `HandlePackages` into:

```go
func buildPackageInfos(details []dixinternal.ProviderDetails) []PackageInfo
```

Use `detail.OutputPkg` when non-empty and `(anonymous)` when empty. Do not call `extractPackage(detail.OutputType)` for provider package grouping. Return rows sorted by `Name` so API output and tests are deterministic.

- [ ] **Step 4: Run focused projection tests and verify GREEN**

Run:

```bash
unset GOROOT; export PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin; go test ./dixinternal -run Test(EgoInstantiated|ResolvedTopNAggregates) -count=1; go test ./dixhttp -run TestPackageInfoUses -count=1
```

Expected: all focused tests pass.

- [ ] **Step 5: Commit**

```bash
git add dixinternal/graph_query.go dixhttp/server.go dixinternal/graph_query_projection_test.go dixhttp/server_packages_test.go
git commit -m "fix(dixhttp): correct graph projection state and modules"
```

---

### Task 5: Make the New Graph Default Module-First and Bounded

**Files:**
- Modify: `dixhttp/static/js/views/graph.js:1-420`
- Test: `dixhttp/static/js/graph_view.test.mjs`

**Interfaces:**
- Consumes: `/api/modules`, `/api/ego`, and identity-aware `ProviderInfo`.
- Produces: `resolveGraphMode(query)`, `applyGraphBudget(nodes, edges, budget)`, and module-first graph state.

- [ ] **Step 1: Write failing pure JS view tests**

Create `dixhttp/static/js/graph_view.test.mjs`:

```js
import test from "node:test";
import assert from "node:assert/strict";

test("graph defaults to module map instead of empty ego", async () => {
  const { resolveGraphMode } = await import("./graph_state.mjs");
  assert.equal(resolveGraphMode(new URLSearchParams()), "modules");
  assert.equal(resolveGraphMode(new URLSearchParams("mode=ego")), "ego");
});

test("graphs degrade to table metadata above explicit budgets", async () => {
  const { applyGraphBudget } = await import("./graph_state.mjs");
  const nodes = Array.from({ length: 101 }, (_, i) => ({ id: String(i) }));
  const edges = Array.from({ length: 301 }, (_, i) => ({ from: "0", to: String(i + 1) }));
  const result = applyGraphBudget(nodes, edges, { nodes: 100, edges: 300 });
  assert.equal(result.nodes.length, 100);
  assert.equal(result.edges.length, 300);
  assert.equal(result.degraded, true);
});
```

- [ ] **Step 2: Run JS tests and verify RED**

Run:

```bash
node --test dixhttp/static/js/graph_view.test.mjs
```

Expected: module resolution fails because `graph_state.mjs` does not exist.

- [ ] **Step 3: Extract and implement pure graph state helpers**

Create `dixhttp/static/js/graph_state.mjs`:

```js
export function resolveGraphMode(query) {
  const mode = (query.get("mode") || "").trim();
  return ["modules", "ego", "providers", "types"].includes(mode) ? mode : "modules";
}

export function applyGraphBudget(nodes, edges, budget = { nodes: 100, edges: 300 }) {
  if (nodes.length <= budget.nodes && edges.length <= budget.edges) {
    return { nodes, edges, degraded: false };
  }
  const degree = new Map();
  for (const edge of edges) {
    degree.set(edge.from, (degree.get(edge.from) || 0) + 1);
    degree.set(edge.to, (degree.get(edge.to) || 0) + 1);
  }
  const keptNodes = new Set([...nodes].sort((a, b) =>
    (degree.get(b.id) || 0) - (degree.get(a.id) || 0) ||
    String(a.id).localeCompare(String(b.id))
  ).slice(0, budget.nodes).map(node => node.id));
  const keptEdges = edges.filter(edge =>
    keptNodes.has(edge.from) && keptNodes.has(edge.to)
  ).slice(0, budget.edges);
  return {
    nodes: nodes.filter(node => keptNodes.has(node.id)),
    edges: keptEdges,
    degraded: true,
  };
}
```

Create a browser adapter `dixhttp/static/js/graph_state.js` that exposes the same functions on `window.DIXGraphState`, either by duplicating the small pure logic or via a generated embedded bundle only if the build already supports one.

Update `views/graph.js` to import/use these helpers, default the select control to `modules`, and render a visible density warning when `degraded` is true.

- [ ] **Step 4: Remove eager full dependencies loading**

Change `loadData()` so only providers/types detail views request `/api/dependencies`; module and ego views must not preload it. Request runtime stats in parallel and cache by provider identity. The graph draw path must use `/api/modules` for module mode and `/api/ego` for ego mode.

- [ ] **Step 5: Run JS tests**

Run:

```bash
node --test dixhttp/static/js/graph_view.test.mjs
```

Expected: both tests pass.

- [ ] **Step 6: Run Go tests**

Run:

```bash
unset GOROOT; export PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin; go test -race ./...
```

Expected: all packages pass.

- [ ] **Step 7: Commit**

```bash
git add dixhttp/static/js/graph_state.mjs dixhttp/static/js/graph_state.js dixhttp/static/js/views/graph.js dixhttp/static/js/graph_view.test.mjs
git commit -m "feat(dixhttp): default graph to bounded module map"
```

---

### Task 6: Responsive Graph Layout and Manual Scale Verification

**Files:**
- Modify: `dixhttp/static/css/app.css:100-150`
- Modify: `dixhttp/static/js/views/graph.js:15-40`
- Test: `dixhttp/http_scale_e2e_test.go`

**Interfaces:**
- Consumes: module-first UI and graph budgets.
- Produces: stable graph canvas sizing and an HTTP smoke test proving `/next` defaults to module assets.

- [ ] **Step 1: Write the failing HTTP/UI smoke test**

Create `dixhttp/http_scale_e2e_test.go`:

```go
package dixhttp

import (
	"net/http/httptest"
	"strings"
	"testing"

	dix "github.com/pubgo/dix/v2"
)

func TestNextGraphDefaultsToModuleFirstAssets(t *testing.T) {
	container := dix.New()
	server := NewServer(container)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest("GET", "/next", nil))

	if recorder.Code != 200 {
		t.Fatalf("status = %d", recorder.Code)
	}
	body := recorder.Body.String()
	for _, asset := range []string{"static/js/views/graph.js", "static/js/main.js"} {
		if !strings.Contains(body, asset) {
			t.Fatalf("missing asset %s", asset)
		}
	}
}
```

- [ ] **Step 2: Run smoke test and verify RED or baseline**

Run:

```bash
unset GOROOT; export PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin; go test ./dixhttp -run TestNextGraphDefaults -count=1
```

Expected: this route already serves the assets and should pass; if it fails, fix routing before UI CSS work.

- [ ] **Step 3: Lock canvas geometry**

Update `app.css`:

```css
.graph-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(280px, 320px);
  align-items: stretch;
}

#graph-canvas {
  height: clamp(420px, calc(100vh - 220px), 900px);
  min-width: 300px;
}

@media (max-width: 900px) {
  .graph-layout { grid-template-columns: minmax(0, 1fr); }
  #graph-canvas { height: 70vh; min-height: 420px; }
  #g-detail { position: static; max-height: 45vh; overflow: auto; }
}
```

Apply `class="graph-layout"` to the graph grid and use `minmax(0, 1fr)` for all toolbar inputs that can overflow.

- [ ] **Step 4: Run automated checks**

Run:

```bash
unset GOROOT; export PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin; go test -race ./... && node --test dixhttp/static/js/graph_view.test.mjs
```

Expected: all checks pass.

- [ ] **Step 5: Run the scale fixture manually**

Run:

```bash
unset GOROOT; export PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin; (cd example && DIX_HTTP_ADDR=127.0.0.1:18099 go run ./http)
```

Inspect:

```text
http://127.0.0.1:18099/next#/graph
```

Confirm module map is visible, no full dependency request occurs on first render, and the canvas remains usable at 700px viewport width. Stop the fixture after verification.

- [ ] **Step 6: Commit**

```bash
git add dixhttp/http_scale_e2e_test.go dixhttp/static/css/app.css dixhttp/static/js/views/graph.js
git commit -m "fix(dixhttp): keep scaled graph views usable"
```

---

### Task 7: Documentation, Example Fixture, and Full Verification

**Files:**
- Modify: `dixhttp/README.md`
- Modify: `dixhttp/README_zh.md`
- Modify: `example/http/main.go`
- Test: `example/http/scale_shape_test.go`

**Interfaces:**
- Consumes: completed phase-one behavior.
- Produces: documented scale workflow and a repeatable near-target fixture assertion.

- [ ] **Step 1: Write the failing example shape test**

Create `example/http/scale_shape_test.go`:

```go
package main

import "testing"

func TestScaleFixtureShape(t *testing.T) {
	container := buildContainer()
	modules := container.ModuleGraph()
	providers := container.GetProviderDetails()
	objects := container.GetObjects()

	objectCount := 0
	for _, groups := range objects {
		for _, values := range groups {
			objectCount += len(values)
		}
	}
	if len(modules) < 10 {
		t.Fatalf("modules = %d, want at least 10", len(modules))
	}
	if len(providers) < 190 {
		t.Fatalf("providers = %d, want at least 190", len(providers))
	}
	if objectCount < 180 {
		t.Fatalf("objects = %d, want at least 180", objectCount)
	}
}
```

`buildContainer()` already returns `*dix.Dix` and is in package `main`; call it directly. Do not add another fixture builder unless production initialization changes.

- [ ] **Step 2: Run example shape test and verify RED or baseline**

Run:

```bash
unset GOROOT; export PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin; go test -C example ./http -run TestScaleFixtureShape -count=1
```

Expected: pass if the existing fixture already satisfies the bounds; otherwise extract the helper and make the test pass.

- [ ] **Step 3: Document the module-first workflow**

In both README files, replace claims that the default graph is a full graph. Document:

```text
Default graph = module map
Module click = bounded module detail
Type click/search = ego graph
Object = state shown in type/provider details
Over-budget graph = density warning plus table/Top-K path
```

Document `/api/dependencies` as a compatibility/full-data endpoint, not the first-scale UI data source.

- [ ] **Step 4: Run complete verification**

Run:

```bash
unset GOROOT; export PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin; go test -race ./...; go vet ./...; go build -C example ./...; go test -C example -count=1 ./...
```

Expected: every command succeeds.

- [ ] **Step 5: Commit**

```bash
git add dixhttp/README.md dixhttp/README_zh.md example/http/main.go example/http/scale_shape_test.go
git commit -m "docs(dixhttp): document scale-first dependency workflow"
```

---

## Final Review Checklist

- [ ] Generic registrations do not merge merely because they share source line.
- [ ] Struct multi-output registrations still aggregate as one logical registration.
- [ ] Runtime errors and durations attach to the correct provider/output pair.
- [ ] Ego state does not claim instantiation without an object node.
- [ ] `/next#/graph` renders module map on first load.
- [ ] Module map does not request `/api/dependencies`.
- [ ] Graph budgets and degradation warning are active.
- [ ] Narrow viewport keeps graph canvas usable.
- [ ] Root race tests, vet, example build, and example tests pass.
- [ ] Documentation matches actual default behavior.
