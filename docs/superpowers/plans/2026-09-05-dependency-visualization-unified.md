# Dependency Visualization Unified Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Finish the unified `/next` dependency workflow so large containers stay readable (bounded module-first graphs) and failures are diagnosable (Issue → bounded graph → Trace) without breaking legacy APIs or the five-view shell.

**Architecture:** Evolve existing dixhttp endpoints and `/next` views in place. Keep `/api/modules`, `/api/module`, `/api/ego`, `/api/issues`, `/api/trace`, and `/api/trace-tree` as the primary path. Default graph modes never eagerly load `/api/dependencies`. Cancel stale view requests with AbortController + a sequence load guard. Correlate diagnostics through `provider_id`, with function name only as fallback.

**Tech Stack:** Go 1.24+, net/http/httptest, vendored vanilla JS, vis-network, hash routing, Node `node:test` for pure JS helpers.

**Spec:** `docs/superpowers/specs/2026-09-05-dependency-visualization-unified-design.md`

## Global Constraints

- Do not introduce third-party Go or JavaScript dependencies.
- Preserve all existing public Dix APIs.
- Preserve legacy `/` UI and `/api/dependencies` compatibility; additive fields only.
- Keep `/next` five views; do not rebuild a three-pane shell.
- Graph budgets: module map 100/300, module drill-down 150/400, ego 100/300, global advanced 150/400.
- Issue severity values stay `error` and `slow` (existing API); UI may style `slow` as warn.
- Every commit must leave `go test -race ./...` green in the root module.
- JS pure helpers must pass `node --test dixhttp/static/js/graph_view.test.mjs`.

## File Map

| File | Responsibility |
|---|---|
| `dixhttp/static/js/api.js` | `DIX.get(path, params, options)` with optional `AbortSignal` |
| `dixhttp/static/js/graph_state.mjs` | Mode resolution, budgets, load guard, issue hashes, hub ranking, trace filter helpers |
| `dixhttp/static/js/graph_view.test.mjs` | Pure JS contract tests |
| `dixhttp/static/js/views/graph.js` | Bounded graph modes, cancellation, density warning + hubs, drawer → Trace |
| `dixhttp/static/js/views/trace.js` | Hash prefilter + optional auto-open tree |
| `dixhttp/static/js/views/overview.js` | Issues feed (already wired; verify after hash/API changes) |
| `dixhttp/server.go` | `IssueInfo` + `buildIssues` (+ optional `trace_id`) |
| `dixinternal/dix.go` / `dixinternal/api.go` | Optional TraceID on recent errors |
| `dixhttp/README.md` / `dixhttp/README_zh.md` | Document default workflow |

---

### Task 1: Cancel Stale Graph Requests (Correctly)

**Files:**
- Modify: `dixhttp/static/js/api.js`
- Modify: `dixhttp/static/js/graph_state.mjs`
- Modify: `dixhttp/static/js/views/graph.js`
- Test: `dixhttp/static/js/graph_view.test.mjs`

**Interfaces:**
- Consumes: existing `DIX.get`, `window.DIXGraphState`
- Produces: `DIX.get(path, params, options = {})` with `options.signal`; `createLoadGuard()` → `{ begin(): { seq:number }, isCurrent(seq:number): boolean }`

**Bug to avoid:** the previous attempt passed `{ seq }` objects into `isCurrent`. Always pass numeric `seq`.

- [ ] **Step 1: Write the failing load-guard test**

Append to `dixhttp/static/js/graph_view.test.mjs`:

```js
test("stale graph loads are rejected by sequence", async () => {
  const { createLoadGuard } = await import("./graph_state.mjs");
  const guard = createLoadGuard();
  const first = guard.begin();
  const second = guard.begin();
  assert.equal(guard.isCurrent(first.seq), false);
  assert.equal(guard.isCurrent(second.seq), true);
});
```

- [ ] **Step 2: Run the focused JS test and verify RED**

Run:

```bash
node --test dixhttp/static/js/graph_view.test.mjs
```

Expected: FAIL because `createLoadGuard` is not exported.

- [ ] **Step 3: Implement API signal support and load guard**

In `dixhttp/static/js/api.js`, change `DIX.get` to:

```js
DIX.get = async function (path, params, options = {}) {
  let url = base + path;
  if (params) {
    const qs = new URLSearchParams();
    for (const [k, v] of Object.entries(params)) {
      if (v !== undefined && v !== null && v !== "") qs.set(k, v);
    }
    const q = qs.toString();
    if (q) url += "?" + q;
  }
  const resp = await fetch(url, { signal: options.signal });
  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(resp.status + " " + text.slice(0, 200));
  }
  return resp.json();
};
```

In `dixhttp/static/js/graph_state.mjs`, add and export:

```js
export function createLoadGuard() {
  let current = 0;
  return {
    begin() {
      current += 1;
      return { seq: current };
    },
    isCurrent(seq) {
      return seq === current;
    },
  };
}
```

Expose on `window.DIXGraphState` together with existing helpers.

- [ ] **Step 4: Wire graph.js cancellation**

In `dixhttp/static/js/views/graph.js` state, add `loadGuard: null, abortController: null`.

Add helpers:

```js
function beginLoad() {
  state.loadGuard ||= window.DIXGraphState.createLoadGuard();
  state.abortController?.abort();
  const { seq } = state.loadGuard.begin();
  state.abortController = new AbortController();
  return { seq, signal: state.abortController.signal };
}

function isStale(seq) {
  return !state.loadGuard.isCurrent(seq);
}

function isAbortError(err) {
  return err && (err.name === "AbortError" || /aborted/i.test(String(err.message || err)));
}
```

Update `loadRuntimeStats`, `loadProviderData`, `loadModuleData`, `buildModules`, and ego/`DIX.get` calls inside `redraw` to accept/pass `signal`. At each await boundary in `redraw`, return early when `isStale(seq)`. In `catch`, ignore abort errors:

```js
} catch (err) {
  if (isAbortError(err)) return;
  DIX.renderError(canvas, err);
}
```

- [ ] **Step 5: Run JS tests GREEN**

```bash
node --test dixhttp/static/js/graph_view.test.mjs
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add dixhttp/static/js/api.js dixhttp/static/js/graph_state.mjs dixhttp/static/js/views/graph.js dixhttp/static/js/graph_view.test.mjs
git commit -m "$(cat <<'EOF'
fix(dixhttp): cancel stale graph requests safely

EOF
)"
```

---

### Task 2: Align Module Budget and Lock Issue Navigation Hashes

**Files:**
- Modify: `dixhttp/static/js/views/graph.js`
- Modify: `dixhttp/static/js/graph_state.mjs` (hashes already exist; keep behavior)
- Test: `dixhttp/static/js/graph_view.test.mjs`

**Interfaces:**
- Consumes: `issueGraphHash`, `issueTraceHash`
- Produces: `budgets.module = { nodes: 150, edges: 400 }`; `loadModuleData` uses `limit: 150, edge_limit: 400`

- [ ] **Step 1: Write failing navigation + budget tests**

Append:

```js
test("issues have deterministic graph and trace links", async () => {
  const { issueGraphHash, issueTraceHash } = await import("./graph_state.mjs");
  const issue = {
    output_type: "*app.Service",
    provider: "app.NewService",
    module: "app/service",
  };
  assert.equal(issueGraphHash(issue), "#/graph?mode=ego&center=*app.Service");
  assert.equal(issueGraphHash({ ...issue, output_type: "" }), "#/graph?mode=module&module=app%2Fservice");
  assert.equal(issueTraceHash(issue), "#/trace?provider=app.NewService&output_type=*app.Service&status=error");
  assert.equal(
    issueTraceHash({ ...issue, severity: "slow" }),
    "#/trace?provider=app.NewService&output_type=*app.Service&status=slow"
  );
});

test("resolveGraphMode accepts module drilldown", async () => {
  const { resolveGraphMode } = await import("./graph_state.mjs");
  assert.equal(resolveGraphMode(new URLSearchParams("mode=module")), "module");
});
```

Update `issueTraceHash` if needed so `severity: "slow"` yields `status=slow`; default remains `error`.

- [ ] **Step 2: Run JS tests; fix hash helper if RED**

```bash
node --test dixhttp/static/js/graph_view.test.mjs
```

- [ ] **Step 3: Align module budgets in graph.js**

Change:

```js
const budgets = {
  modules: { nodes: 100, edges: 300 },
  module: { nodes: 150, edges: 400 },
  ego: { nodes: 100, edges: 300 },
  providers: { nodes: 150, edges: 400 },
  types: { nodes: 150, edges: 400 },
};
```

And:

```js
async function loadModuleData(module, signal) {
  return DIX.get("/api/module", { name: module, limit: 150, edge_limit: 400 }, { signal });
}
```

(If Task 1 did not yet add `signal`, keep the third argument optional.)

- [ ] **Step 4: Run JS + focused Go module API test**

```bash
node --test dixhttp/static/js/graph_view.test.mjs
unset GOROOT; export PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin
go test ./dixhttp -run TestHandleModuleReturnsBoundedTopology -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add dixhttp/static/js/views/graph.js dixhttp/static/js/graph_state.mjs dixhttp/static/js/graph_view.test.mjs
git commit -m "$(cat <<'EOF'
fix(dixhttp): align module graph budget and issue links

EOF
)"
```

---

### Task 3: Density Warning Offers Hubs Table

**Files:**
- Modify: `dixhttp/static/js/graph_state.mjs`
- Modify: `dixhttp/static/js/views/graph.js`
- Test: `dixhttp/static/js/graph_view.test.mjs`

**Interfaces:**
- Consumes: `applyGraphBudget`
- Produces: `rankHubNodes(nodes, edges, limit = 10) → [{ id, degree }]`

- [ ] **Step 1: Write failing hub-rank test**

```js
test("rankHubNodes returns highest-degree nodes first", async () => {
  const { rankHubNodes } = await import("./graph_state.mjs");
  const nodes = [{ id: "a" }, { id: "b" }, { id: "c" }];
  const edges = [
    { from: "a", to: "b" },
    { from: "a", to: "c" },
    { from: "b", to: "c" },
  ];
  assert.deepEqual(rankHubNodes(nodes, edges, 2).map((h) => h.id), ["a", "b"]);
});
```

- [ ] **Step 2: Run test RED**

```bash
node --test dixhttp/static/js/graph_view.test.mjs
```

- [ ] **Step 3: Implement rankHubNodes**

```js
export function rankHubNodes(nodes, edges, limit = 10) {
  const degree = new Map();
  for (const node of nodes) degree.set(node.id, 0);
  for (const edge of edges) {
    degree.set(edge.from, (degree.get(edge.from) || 0) + 1);
    degree.set(edge.to, (degree.get(edge.to) || 0) + 1);
  }
  return [...nodes]
    .map((node) => ({ id: node.id, degree: degree.get(node.id) || 0 }))
    .sort((a, b) => b.degree - a.degree || String(a.id).localeCompare(String(b.id)))
    .slice(0, limit);
}
```

Export on `window.DIXGraphState`.

- [ ] **Step 4: Enrich density warning UI in graph.js**

When `bounded.degraded || graph.truncated`, set `#g-budget` innerHTML (not only textContent) to:

1. One-line warning with node/edge counts
2. A compact hubs table from `rankHubNodes(bounded.nodes, bounded.edges, 8)`
3. Hint chips/actions already implied by copy: reduce module / depth / use search

Keep styling minimal with existing `.tbl` / `.muted` classes.

- [ ] **Step 5: Run JS tests GREEN**

```bash
node --test dixhttp/static/js/graph_view.test.mjs
```

- [ ] **Step 6: Commit**

```bash
git add dixhttp/static/js/graph_state.mjs dixhttp/static/js/views/graph.js dixhttp/static/js/graph_view.test.mjs
git commit -m "$(cat <<'EOF'
feat(dixhttp): show hub table when graph budget truncates

EOF
)"
```

---

### Task 4: Trace View Hash Prefilter

**Files:**
- Modify: `dixhttp/static/js/graph_state.mjs`
- Modify: `dixhttp/static/js/views/trace.js`
- Test: `dixhttp/static/js/graph_view.test.mjs`

**Interfaces:**
- Consumes: hash query `provider`, `output_type`, `status`, `trace_id`
- Produces: `matchTraceRecord(rec, filter) → boolean`; `filterTraceRecords(records, filter) → records`

Trace API records expose fields such as `provider` / `provider_function`, `output_type`, `status`, `trace_id` (match whatever `/api/trace` already returns; inspect one fixture response while implementing).

- [ ] **Step 1: Write failing filter tests**

```js
test("filterTraceRecords keeps matching provider and status", async () => {
  const { filterTraceRecords } = await import("./graph_state.mjs");
  const records = [
    { trace_id: "aa", provider_function: "app.NewA", output_type: "*A", status: "error" },
    { trace_id: "bb", provider_function: "app.NewB", output_type: "*B", status: "ok" },
    { trace_id: "cc", provider_function: "app.NewA", output_type: "*A", status: "ok" },
  ];
  const filtered = filterTraceRecords(records, {
    provider: "app.NewA",
    output_type: "*A",
    status: "error",
  });
  assert.deepEqual(filtered.map((r) => r.trace_id), ["aa"]);
});
```

- [ ] **Step 2: Run RED, then implement helpers**

```js
export function filterTraceRecords(records = [], filter = {}) {
  return records.filter((rec) => matchTraceRecord(rec, filter));
}

export function matchTraceRecord(rec = {}, filter = {}) {
  if (filter.trace_id && rec.trace_id !== filter.trace_id) return false;
  const provider = rec.provider_function || rec.provider || "";
  if (filter.provider && provider !== filter.provider) return false;
  if (filter.output_type && rec.output_type !== filter.output_type) return false;
  if (filter.status === "error" && rec.status !== "error") return false;
  if (filter.status === "slow") {
    // slow is a feed severity, not always an event status; keep provider/output matches
    return true;
  }
  return true;
}
```

Tune field names against real `/api/trace` JSON before committing.

- [ ] **Step 3: Wire trace.js**

In `DIX.views.trace.render(el, query)` (accept `query` from router like other views; if `main.js` already passes query, use it; otherwise parse `location.hash`):

1. Build `filter` from `provider`, `output_type`, `status`, `trace_id`.
2. Apply `filterTraceRecords` before grouping.
3. Sort groups with errors first, then newest.
4. If `filter.trace_id` is set and present, auto-open that tree once.
5. Show a muted banner when filters are active, with a clear-filter control that resets hash to `#/trace`.

`main.js` already calls `v.render(el, query)` for every view — do not change it unless the signature is broken.

Match filter fields against `dixtrace.Event` JSON: `provider_function`, `output_type`, `status`, `trace_id`.

- [ ] **Step 4: Run JS tests + smoke Go compile**

```bash
node --test dixhttp/static/js/graph_view.test.mjs
unset GOROOT; export PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin
go test ./dixhttp -count=1
```

- [ ] **Step 5: Commit**

```bash
git add dixhttp/static/js/graph_state.mjs dixhttp/static/js/views/trace.js dixhttp/static/js/graph_view.test.mjs
git commit -m "$(cat <<'EOF'
feat(dixhttp): prefilter trace view from issue hash

EOF
)"
```

---

### Task 5: Graph Drawer Jump to Trace

**Files:**
- Modify: `dixhttp/static/js/views/graph.js`
- Modify: `dixhttp/static/js/graph_state.mjs` if a small helper helps
- Test: `dixhttp/static/js/graph_view.test.mjs`

**Interfaces:**
- Consumes: `issueTraceHash({ provider, output_type, severity })`
- Produces: drawer button `#d-trace` setting `location.hash`

- [ ] **Step 1: Add helper test for provider-detail trace hash**

```js
test("provider drawer can build trace hash from node identity", async () => {
  const { issueTraceHash } = await import("./graph_state.mjs");
  assert.equal(
    issueTraceHash({ provider: "app.NewService", output_type: "*app.Service", severity: "error" }),
    "#/trace?provider=app.NewService&output_type=*app.Service&status=error"
  );
});
```

- [ ] **Step 2: Add drawer button in type/provider detail**

In `typeDetail` / provider detail rendering inside `graph.js`, add:

```html
<button class="btn ghost" id="d-trace">查看解析链路</button>
```

Wire:

```js
document.getElementById("d-trace").onclick = () => {
  location.hash = window.DIXGraphState.issueTraceHash({
    provider: providerFnName || "",
    output_type: label,
    severity: "error",
  });
};
```

Use the best available provider function name from node `data` / selected provider chips. If only a type label exists, still jump with `output_type`.

- [ ] **Step 3: Manual sanity via example is optional; automated JS tests must PASS**

```bash
node --test dixhttp/static/js/graph_view.test.mjs
```

- [ ] **Step 4: Commit**

```bash
git add dixhttp/static/js/views/graph.js dixhttp/static/js/graph_view.test.mjs
git commit -m "$(cat <<'EOF'
feat(dixhttp): jump from graph drawer to filtered trace

EOF
)"
```

---

### Task 6: Optional TraceID on Issues

**Files:**
- Modify: `dixinternal/dix.go` (`recentErrorRecord` / `recordRecentErrorWithContext`)
- Modify: `dixinternal/api.go` (`RecentError`)
- Modify: `dixhttp/server.go` (`IssueInfo`, `buildIssues`)
- Test: `dixhttp/server_issues_test.go`
- Test: optional `dixinternal` focused test if TraceID capture needs one

**Interfaces:**
- Consumes: current span TraceID from inject/provider context when available
- Produces: `RecentError.TraceID string \`json:"trace_id,omitempty"\``; `IssueInfo.TraceID string \`json:"trace_id,omitempty"\``
- Updates: `issueTraceHash` prefers `trace_id` query when present

Minimal capture path: in `recordRecentErrorWithContext`, if `dixtrace` has a current span TraceID accessor already used elsewhere in dixinternal, copy it into the record. If no accessor exists without new public API, implement only the additive struct/JSON fields + `buildIssues` projection, leave TraceID empty at runtime, and rely on Task 4 provider/output filters. Do not invent a second TraceID generator.

- [ ] **Step 1: Extend IssueInfo test expectations**

In `dixhttp/server_issues_test.go`, add a unit assertion on `buildIssues` that when `RecentError.TraceID` is set, the projected issue keeps it.

```go
recent := []dixinternal.RecentError{{
	ErrorType: "provider_error",
	Message:   "boom",
	TraceID:   "abc123",
	OutputType: "*app.Service",
	ProviderFunction: "app.NewService",
	OccurredAtUnixNano: 1,
}}
issues := buildIssues(nil, recent, nil, 0, 10)
if issues[0].TraceID != "abc123" {
	t.Fatalf("trace id = %q", issues[0].TraceID)
}
```

- [ ] **Step 2: Run RED (missing field), then add fields and projection**

Add `TraceID` to internal recent-error record, `RecentError`, and `IssueInfo`; copy in `buildIssues`.

- [ ] **Step 3: Prefer trace_id in issueTraceHash**

```js
export function issueTraceHash(issue = {}) {
  const params = new URLSearchParams();
  if (issue.trace_id) params.set("trace_id", issue.trace_id);
  if (issue.provider) params.set("provider", issue.provider);
  if (issue.output_type) params.set("output_type", issue.output_type);
  params.set("status", issue.severity === "slow" ? "slow" : "error");
  return "#/trace?" + params.toString();
}
```

Update JS tests accordingly.

- [ ] **Step 4: Run tests**

```bash
unset GOROOT; export PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin
go test ./dixhttp ./dixinternal -run 'Issue|RecentError|Trace' -count=1
node --test dixhttp/static/js/graph_view.test.mjs
```

- [ ] **Step 5: Commit**

```bash
git add dixinternal/dix.go dixinternal/api.go dixhttp/server.go dixhttp/server_issues_test.go dixhttp/static/js/graph_state.mjs dixhttp/static/js/graph_view.test.mjs
git commit -m "$(cat <<'EOF'
feat(dixhttp): propagate optional trace id on issues

EOF
)"
```

---

### Task 7: Docs and Full Verification

**Files:**
- Modify: `dixhttp/README.md`
- Modify: `dixhttp/README_zh.md`
- Modify: `docs/superpowers/specs/2026-09-05-dependency-visualization-unified-design.md` only if behavior notes need a one-line status flip

**Interfaces:**
- Consumes: delivered behavior from Tasks 1–6
- Produces: README sections describing Issue → Graph → Trace and budgets

- [ ] **Step 1: Update README workflow bullets**

Document:

1. `/next#/graph` defaults to module map and does not load `/api/dependencies`.
2. Module drill-down budget 150/400; density warning includes hubs.
3. Overview Issues jump to ego/module graph and filtered Trace.
4. Graph drawer can open filtered Trace.
5. Legacy `/` and `/api/dependencies` remain available.

- [ ] **Step 2: Full verification**

```bash
unset GOROOT; export PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin
node --test dixhttp/static/js/graph_view.test.mjs
go test -race ./...
go test ./example/http -count=1
```

Expected: all PASS.

- [ ] **Step 3: Commit**

```bash
git add dixhttp/README.md dixhttp/README_zh.md
git commit -m "$(cat <<'EOF'
docs(dixhttp): document unified dependency diagnosis workflow

EOF
)"
```

---

## Final Review Checklist

- [ ] Stale graph requests abort and do not overwrite newer views.
- [ ] Module drill-down uses 150/400 budgets.
- [ ] Issue hash navigation to graph/trace is tested.
- [ ] Density truncation shows hubs table, not silent cut-only.
- [ ] Trace view honors provider/output/status/trace_id filters.
- [ ] Graph drawer can open Trace in ≤1 click from a selected node.
- [ ] Optional issue `trace_id` is additive and omitted when unknown.
- [ ] Default `/next` path still avoids eager `/api/dependencies`.
- [ ] Legacy UI/API remain reachable.
- [ ] `go test -race ./...` and JS helper tests pass.
