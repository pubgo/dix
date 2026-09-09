# Legacy-First Architecture Visualization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the legacy `/` Dix UI readable for large DI architecture review (module/provider/type/group lenses), port useful `/next` graph helpers into it, then delete `/next`.

**Architecture:** Keep Alpine + `template.html` + `legacy/app.js` as the only UI. Load shared `graph_state.mjs` / selected workbench helpers onto `window` for legacy `renderGraph`. Add module-map view, density banner (lens prompt → hubs → short labels → layout → Top-K last), then remove `/next` route and five-view shell.

**Tech Stack:** Go embed static FS, Alpine.js, vis-network, Mermaid (already vendored), ES modules for shared helpers, `node --test` for pure JS.

## Global Constraints

- Single UI entry: `/` only after Task 7; no parallel five-view shell.
- Preserve all existing legacy capabilities (package sidebar, group rules, prefix, depth, layouts, SVG/Mermaid, diagnostic modals, Trace).
- Default view remains `providers` unless user switches.
- Objects are not default architecture canvas nodes (providers/types/modules/groups only).
- Public Dix APIs: additive fields only; no new third-party JS/Go deps.
- Crowding degradation order is fixed: prompt coarser lens → semantic collapse hint → hubs → short labels → layout/camera → Top-K last.
- Do not silently switch the user’s current view; one-click apply from banner is OK.
- Spec: `docs/superpowers/specs/2026-09-06-legacy-first-architecture-viz-design.md`

## File Map

| File | Responsibility |
|---|---|
| `dixhttp/static/js/graph_state.mjs` | Shared labels, budget, hubs, layout/camera, star positions (keep; drop next-only defaults later) |
| `dixhttp/static/js/graph_workbench.mjs` | Optional: group/filter helpers if legacy chooses to call them; otherwise leave until needed |
| `dixhttp/static/js/graph_view.test.mjs` | Node tests for shared helpers; remove next-hash-only cases when APIs removed |
| `dixhttp/template.html` | Load mjs; module map button; density banner DOM |
| `dixhttp/static/js/legacy/app.js` | Wire module map, labels, budget, banner, layout into `renderGraph` |
| `dixhttp/server.go` | Remove `/next` handler |
| `dixhttp/http_scale_e2e_test.go` | Assert `/` embeds graph helpers, not `/next` |
| `dixhttp/scripts/graph_layout_e2e.mjs` | Point health check at `/` |
| `dixhttp/README.md`, `README_zh.md` | Single-UI docs |
| Delete | `static/index.html`, `static/js/main.js`, `static/js/views/*`, `static/js/api.js` if only used by next |

---

### Task 1: Load shared graph helpers in legacy HTML

**Files:**
- Modify: `dixhttp/template.html` (script tags near bottom)
- Test: manual + later Go test in Task 7

**Interfaces:**
- Consumes: `graph_state.mjs` already assigns `window.DIXGraphState`
- Produces: legacy page can call `window.DIXGraphState.shortGraphLabel` after load

- [ ] **Step 1: Add module scripts before `legacy/app.js`**

In `dixhttp/template.html`, before the `legacy/app.js` script tag, insert:

```html
<script type="module" src="__DIX_BASE_PATH__/static/js/graph_state.mjs?v=legacy-arch1"></script>
<script type="module" src="__DIX_BASE_PATH__/static/js/graph_workbench.mjs?v=legacy-arch1"></script>
<script src="__DIX_BASE_PATH__/static/js/legacy/app.js?v=legacy-arch1"></script>
```

Bump `legacy/app.js` query from `legacy1` to `legacy-arch1` so caches refresh.

- [ ] **Step 2: Guard helper access in `app.js`**

Near the top of the Alpine `app()` return object methods area (or a small helper method), add:

```javascript
graphHelpers() {
    return window.DIXGraphState || null;
},
```

Do not call helpers during synchronous script evaluate; only from `init` / `renderGraph` (after modules ran).

- [ ] **Step 3: Smoke in browser or curl**

Run demo server, open `/`, in DevTools: `typeof window.DIXGraphState.shortGraphLabel === 'function'`.

- [ ] **Step 4: Commit**

```bash
git add dixhttp/template.html dixhttp/static/js/legacy/app.js
git commit -m "$(cat <<'EOF'
chore(dixhttp): load shared graph helpers in legacy UI

EOF
)"
```

---

### Task 2: Short labels on architecture nodes (TDD on helper already exists)

**Files:**
- Modify: `dixhttp/static/js/legacy/app.js` (`formatTypeName`, provider display labels)
- Test: `dixhttp/static/js/graph_view.test.mjs` (existing `shortGraphLabel` test must still pass)

**Interfaces:**
- Consumes: `DIXGraphState.shortGraphLabel(name: string): string`
- Produces: node `label` short; `title` keeps full name

- [ ] **Step 1: Confirm existing test still passes**

Run: `node --test dixhttp/static/js/graph_view.test.mjs --test-name-pattern shortGraphLabel`  
Expected: PASS

- [ ] **Step 2: Wire `formatTypeName` through short label**

Replace body of `formatTypeName` to keep full name in tooltip path but shorten display:

```javascript
formatTypeName(typeName) {
    const full = String(typeName || '');
    const helpers = this.graphHelpers();
    if (helpers && helpers.shortGraphLabel) {
        return helpers.shortGraphLabel(full);
    }
    // existing fallback truncation if any
    return full;
},
```

Ensure wherever nodes set `title: '类型: ' + outType` they still use the **full** type string (already true in `renderGraph`).

- [ ] **Step 3: Shorten provider box labels similarly**

In `providerNodeLabel`, if the label is a long function path, prefer:

```javascript
providerNodeLabel(provider) {
    const raw = /* existing computation */;
    const helpers = this.graphHelpers();
    if (helpers && helpers.shortGraphLabel) {
        return helpers.shortGraphLabel(raw);
    }
    return raw;
},
```

Keep full identity in `title` / `buildProviderTooltip`.

- [ ] **Step 4: Manual check**

Open `/`, Providers view: labels short; hover title shows full path.

- [ ] **Step 5: Commit**

```bash
git add dixhttp/static/js/legacy/app.js
git commit -m "$(cat <<'EOF'
feat(dixhttp): shorten legacy graph labels via shared helper

EOF
)"
```

---

### Task 3: Module map view in legacy toolbar

**Files:**
- Modify: `dixhttp/template.html` (view buttons)
- Modify: `dixhttp/static/js/legacy/app.js` (`switchView`, `renderGraph`, fetch modules)
- Test: add node test for building module nodes/edges from API-shaped data (pure function preferred)

**Interfaces:**
- Consumes: `GET /api/modules` → `[{ name, provider_count, object_count, type_count, depends_on }]`
- Consumes: `DIXGraphState.layoutStarPositions`, `shortGraphLabel`
- Produces: `currentView === 'modules'` renders module graph; objects only as count in label text, not separate nodes

- [ ] **Step 1: Write failing test for module graph builder**

Add to `dixhttp/static/js/graph_state.mjs` (or a tiny `legacy_graph_build.mjs` if you want isolation):

```javascript
export function buildModuleMapGraph(modules = []) {
  const nodes = modules.map((m) => ({
    id: m.name,
    label: `${shortGraphLabel(m.name)}\n(${m.provider_count || 0}p/${m.object_count || 0}o)`,
    shape: 'box',
    data: { type: 'module', module: m, packagePath: m.name },
  }));
  const edges = [];
  for (const m of modules) {
    for (const dep of m.depends_on || []) {
      edges.push({ from: m.name, to: dep, arrows: 'to' });
    }
  }
  return { nodes, edges };
}
```

Test in `graph_view.test.mjs`:

```javascript
test("buildModuleMapGraph uses modules not objects as nodes", async () => {
  const { buildModuleMapGraph } = await import("./graph_state.mjs");
  const { nodes, edges } = buildModuleMapGraph([
    { name: "app/a", provider_count: 2, object_count: 5, depends_on: ["app/b"] },
    { name: "app/b", provider_count: 1, object_count: 1, depends_on: [] },
  ]);
  assert.equal(nodes.length, 2);
  assert.equal(edges.length, 1);
  assert.equal(nodes[0].data.type, "module");
});
```

- [ ] **Step 2: Run test — expect FAIL then implement — expect PASS**

Run: `node --test dixhttp/static/js/graph_view.test.mjs --test-name-pattern buildModuleMapGraph`

- [ ] **Step 3: Add toolbar button**

In `template.html` next to Providers / Types:

```html
<button @click="switchView('modules')"
    :class="{'bg-primary-500 text-white border-primary-500': currentView === 'modules', 'bg-white hover:bg-gray-50': currentView !== 'modules'}"
    class="px-3 py-1.5 text-sm border rounded-lg transition flex items-center gap-1">
    <span>📦</span> 模块地图
</button>
```

- [ ] **Step 4: Implement `renderModulesGraph` path in `app.js`**

- Add `modulesData: null` to state.
- In `switchView('modules')`, fetch `/api/modules` if needed, set `currentView`, call `renderGraph`.
- At start of `renderGraph`, if `currentView === 'modules'`:

```javascript
if (this.currentView === 'modules') {
    const built = window.DIXGraphState.buildModuleMapGraph(this.modulesData || []);
    // apply prefix filter if set; skip provider/type construction
    // then aggregateByGroups only if meaningful for modules; apply budget; draw with star layout
    ...
    return;
}
```

Use `layoutStarPositions` to set `x`/`y` on nodes when `modules` view and `nodes.length >= 2`. Disable physics for that draw (match `/next` star behavior).

- [ ] **Step 5: Double-click module → set `filterPrefix` to module name and `switchView('providers')`**

```javascript
if (node.data.type === 'module') {
    this.filterPrefix = node.data.module.name;
    this.switchView('providers');
}
```

- [ ] **Step 6: Manual check on example app**

Module map shows ~10 boxes with clear separation; double-click drills to providers scoped by prefix.

- [ ] **Step 7: Commit**

```bash
git add dixhttp/template.html dixhttp/static/js/legacy/app.js dixhttp/static/js/graph_state.mjs dixhttp/static/js/graph_view.test.mjs
git commit -m "$(cat <<'EOF'
feat(dixhttp): add module map lens to legacy graph UI

EOF
)"
```

---

### Task 4: Density banner, hubs, and Top-K last

**Files:**
- Modify: `dixhttp/template.html` (banner container above `#network`)
- Modify: `dixhttp/static/js/legacy/app.js` (`renderGraph` post-filter pipeline)
- Test: existing `applyGraphBudget` / `rankHubNodes` tests

**Interfaces:**
- Consumes: `applyGraphBudget(nodes, edges, { nodes, edges })`, `rankHubNodes(nodes, edges, limit)`, `READABLE_NODE_CAP`
- Produces: `densityWarning` state `{ show, message, hubs[], suggestModules, suggestAggregate }`

- [ ] **Step 1: Add banner markup**

Above the network canvas in `template.html`:

```html
<div x-show="densityWarning.show" x-cloak
     class="mx-4 mt-2 p-3 rounded-lg border border-amber-200 bg-amber-50 text-sm text-amber-900">
  <p x-text="densityWarning.message"></p>
  <div class="mt-2 flex flex-wrap gap-2">
    <button type="button" class="px-2 py-1 text-xs border rounded bg-white"
            x-show="densityWarning.suggestModules"
            @click="switchView('modules')">切换到模块地图</button>
    <button type="button" class="px-2 py-1 text-xs border rounded bg-white"
            x-show="densityWarning.suggestAggregate && !aggregateGroups"
            @click="aggregateGroups = true; saveLocalState(); renderGraph()">开启按分组聚合</button>
  </div>
  <table class="mt-2 w-full text-xs" x-show="densityWarning.hubs.length">
    <tr><th class="text-left">耦合枢纽</th><th class="text-right">度数</th></tr>
    <template x-for="h in densityWarning.hubs" :key="h.id">
      <tr>
        <td class="font-mono truncate">
          <button type="button" class="underline" @click="filterPrefix = h.id; renderGraph()" x-text="h.id"></button>
        </td>
        <td class="text-right" x-text="h.degree"></td>
      </tr>
    </template>
  </table>
</div>
```

Initialize `densityWarning: { show: false, message: '', hubs: [], suggestModules: false, suggestAggregate: false }`.

- [ ] **Step 2: Insert pipeline after aggregate + prefix, before `vis.Network`**

Order must be:

1. Build nodes/edges for current view  
2. `aggregateByGroups`  
3. `filterByPrefix`  
4. If `nodes.length > READABLE_NODE_CAP` (or budget): set banner (`suggestModules` if view is providers/types; `suggestAggregate` if `!aggregateGroups && groupRules.length`)  
5. `rankHubNodes(..., 8)` into banner  
6. `applyGraphBudget(..., { nodes: READABLE_NODE_CAP, edges: READABLE_NODE_CAP * 3 })` **last**  
7. Create network from bounded nodes  

```javascript
const helpers = this.graphHelpers();
const cap = helpers.READABLE_NODE_CAP;
let ns = filteredByPrefix.nodes;
let es = filteredByPrefix.edges;
const over = ns.length > cap || es.length > cap * 3;
this.densityWarning = {
  show: over,
  message: over
    ? `图规模过大（${ns.length} 节点 / ${es.length} 边）。建议先看模块地图或按分组聚合审查组织；下列为耦合枢纽。`
    : '',
  hubs: over ? helpers.rankHubNodes(ns, es, 8) : [],
  suggestModules: over && this.currentView !== 'modules',
  suggestAggregate: over && !this.aggregateGroups && (this.groupRules || []).length > 0,
};
if (over) {
  const bounded = helpers.applyGraphBudget(ns, es, { nodes: cap, edges: cap * 3 });
  ns = bounded.nodes;
  es = bounded.edges;
}
```

- [ ] **Step 3: Run unit tests**

Run: `node --test dixhttp/static/js/graph_view.test.mjs`  
Expected: all PASS

- [ ] **Step 4: Manual check**

Providers view on example app shows amber banner + hubs when over cap; clicking「模块地图」switches without losing package sidebar tools.

- [ ] **Step 5: Commit**

```bash
git add dixhttp/template.html dixhttp/static/js/legacy/app.js
git commit -m "$(cat <<'EOF'
feat(dixhttp): density banner and hub-first budget on legacy graph

EOF
)"
```

---

### Task 5: Readable layout/camera for large legacy graphs

**Files:**
- Modify: `dixhttp/static/js/legacy/app.js` (`getNetworkOptions`, post-create camera)
- Test: existing `resolveEffectiveLayout` / `resolveCameraStrategy` / `layoutStarPositions` tests

**Interfaces:**
- Consumes: `resolveEffectiveLayout(preferred, nodeCount)`, `resolveCameraStrategy(mode, nodeCount)`, `pickFocusNodeId`

- [ ] **Step 1: Map legacy layout select to helper**

Legacy uses `currentLayout: 'hierarchical' | 'force'`. Map:

```javascript
const preferred = this.currentLayout === 'force' ? 'physics' : 'hierarchical';
const effective = helpers.resolveEffectiveLayout(preferred, nodeCount);
```

When `effective === 'physics'`, use forceAtlas-style options already used for `force`; when hierarchical and over cap, helper returns physics — honor it even if select says hierarchical (banner can note「已自动改用分散布局」).

- [ ] **Step 2: Camera after draw**

```javascript
const mode = this.currentView === 'modules' ? 'modules' : 'providers';
const camera = helpers.resolveCameraStrategy(mode, ns.length);
const focusId = helpers.pickFocusNodeId(ns, es, this.filterPrefix || '');
if (camera === 'fit' || this.currentView === 'modules') {
  this.network.fit({ animation: false, padding: 48 });
} else if (focusId) {
  this.network.focus(focusId, { scale: 1.1, animation: false });
}
```

For modules + star positions, physics off (Task 3).

- [ ] **Step 3: Manual check**

Large Providers graph is not a single vertical bead-string; module map fits in view.

- [ ] **Step 4: Commit**

```bash
git add dixhttp/static/js/legacy/app.js
git commit -m "$(cat <<'EOF'
feat(dixhttp): apply readable layout and camera on dense legacy graphs

EOF
)"
```

---

### Task 6: Package sidebar as architectural scope (copy + behavior check)

**Files:**
- Modify: `dixhttp/template.html` (sidebar heading/placeholder text)
- Modify: `dixhttp/static/js/legacy/app.js` only if click handler does not already set prefix + redraw

**Interfaces:**
- Produces: unchanged filtering semantics; clearer UX framing

- [ ] **Step 1: Update sidebar copy**

Change labels to emphasize scope, e.g. heading「包范围」and placeholder「选择包以缩小架构切片…」.

- [ ] **Step 2: Verify click sets `filterPrefix` and calls `renderGraph`**

If existing handler only highlights, align to: set prefix, keep current view, redraw (existing behavior preferred).

- [ ] **Step 3: Commit**

```bash
git add dixhttp/template.html dixhttp/static/js/legacy/app.js
git commit -m "$(cat <<'EOF'
docs(dixhttp): frame package sidebar as architecture scope

EOF
)"
```

---

### Task 7: Delete `/next` and retarget tests/docs

**Files:**
- Modify: `dixhttp/server.go` (remove route + `HandleNextIndex`)
- Modify: `dixhttp/http_scale_e2e_test.go`
- Modify: `dixhttp/scripts/graph_layout_e2e.mjs`
- Modify: `dixhttp/README.md`, `dixhttp/README_zh.md`
- Delete: `dixhttp/static/index.html`, `dixhttp/static/js/main.js`, `dixhttp/static/js/api.js`, `dixhttp/static/js/views/overview.js`, `graph.js`, `search.js`, `trace.js`, `diag.js` (all next-only)
- Modify: `dixhttp/static/js/graph_view.test.mjs` — remove tests that only exist for next hash routing (`issueGraphHash` may stay if still used by legacy later; if unused, keep helpers but drop next-only assertions that require five-view hashes, or keep helpers for future Trace jumps)
- Modify: `docs/superpowers/specs/2026-09-05-dependency-visualization-unified-design.md` — add status line `Superseded by 2026-09-06-legacy-first-architecture-viz-design.md`

**Interfaces:**
- Produces: `GET /next` → 404; `GET /` includes `graph_state.mjs` and module map button markup

- [ ] **Step 1: Rewrite Go e2e test**

```go
func TestLegacyGraphEmbedsSharedHelpers(t *testing.T) {
	container := dix.New()
	server := NewServer(container)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest("GET", "/", nil))
	if recorder.Code != 200 {
		t.Fatalf("status = %d", recorder.Code)
	}
	body := recorder.Body.String()
	for _, asset := range []string{"static/js/graph_state.mjs", "static/js/legacy/app.js", "模块地图"} {
		if !strings.Contains(body, asset) {
			t.Fatalf("missing %s", asset)
		}
	}
	if strings.Contains(body, "static/js/views/graph.js") {
		t.Fatalf("legacy index must not reference next graph.js")
	}
}
```

- [ ] **Step 2: Run test — FAIL (next still present / assertion)** then remove `/next` handler and delete next static files — PASS

Also add a quick test that `/next` returns 404 if desired:

```go
server.ServeHTTP(rec, httptest.NewRequest("GET", "/next", nil))
if rec.Code != 404 { t.Fatalf(...) }
```

- [ ] **Step 3: Update `graph_layout_e2e.mjs`**

Health check `fetch(`${BASE}/`)` and assert `graph_state.mjs` + `legacy/app.js` in HTML (not `graph.js`).

- [ ] **Step 4: Update README / README_zh**

State clearly: visualization UI is `/` only; remove “alongside `/next`” and next workbench bullets; document module map + density banner briefly.

- [ ] **Step 5: Run full verification**

```bash
node --test dixhttp/static/js/graph_view.test.mjs
go test -race ./dixhttp/...
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add -A dixhttp docs/superpowers/specs
git commit -m "$(cat <<'EOF'
chore(dixhttp): remove /next shell; legacy UI is sole entry

EOF
)"
```

---

### Task 8: End-to-end acceptance against the spec

**Files:** none new (verification only)

- [ ] **Step 1: Run demo**

`DIX_HTTP_ADDR=127.0.0.1:18099 go run -C example ./http`

- [ ] **Step 2: Checklist**

1. `/next` 404  
2. `/` loads; Providers default  
3. 模块地图 clear boundaries  
4. Dense Providers shows banner + hubs; one-click to modules / aggregate  
5. Group rules, prefix, SVG, Mermaid, Trace modal still work  
6. No object-only nodes on canvas  

- [ ] **Step 3: Commit only if doc tweaks needed**

```bash
git commit -m "$(cat <<'EOF'
docs(dixhttp): note legacy-first acceptance for architecture viz

EOF
)"
```

---

## Spec coverage self-check

| Spec requirement | Task |
|---|---|
| Single UI `/`, delete `/next` | 7 |
| Preserve legacy capabilities | 3–6 (additive), 8 checklist |
| Four lenses (module/provider/type/group) | 3 + existing providers/types + aggregate |
| Crowding order (prompt → aggregate → hubs → labels → layout → Top-K) | 2,4,5 |
| Objects not default canvas nodes | 3 builder + 8 |
| Port helpers from next | 1–5 |
| API stable / no new deps | Global + all tasks |
| Acceptance example-scale | 8 |

## Placeholder scan

No TBD steps; commands and code sketches included for each task.
