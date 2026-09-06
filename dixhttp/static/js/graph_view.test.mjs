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
  const edges = Array.from({ length: 301 }, (_, i) => ({ from: String(i % 100), to: String((i + 1) % 100) }));
  const result = applyGraphBudget(nodes, edges, { nodes: 100, edges: 300 });
  assert.equal(result.nodes.length, 100);
  assert.equal(result.edges.length, 300);
  assert.equal(result.degraded, true);
});

test("stale graph loads are rejected by sequence", async () => {
  const { createLoadGuard } = await import("./graph_state.mjs");
  const guard = createLoadGuard();
  const first = guard.begin();
  const second = guard.begin();
  assert.equal(guard.isCurrent(first.seq), false);
  assert.equal(guard.isCurrent(second.seq), true);
});

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
  assert.equal(
    issueTraceHash({ ...issue, trace_id: "abc123" }),
    "#/trace?trace_id=abc123&provider=app.NewService&output_type=*app.Service&status=error"
  );
});

test("resolveGraphMode accepts module drilldown", async () => {
  const { resolveGraphMode } = await import("./graph_state.mjs");
  assert.equal(resolveGraphMode(new URLSearchParams("mode=module")), "module");
});

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

test("provider drawer can build trace hash from node identity", async () => {
  const { issueTraceHash } = await import("./graph_state.mjs");
  assert.equal(
    issueTraceHash({ provider: "app.NewService", output_type: "*app.Service", severity: "error" }),
    "#/trace?provider=app.NewService&output_type=*app.Service&status=error"
  );
});

test("large graphs switch hierarchical preference to physics", async () => {
  const { resolveEffectiveLayout, READABLE_NODE_CAP } = await import("./graph_state.mjs");
  assert.equal(resolveEffectiveLayout("hierarchical", READABLE_NODE_CAP), "hierarchical");
  assert.equal(resolveEffectiveLayout("hierarchical", READABLE_NODE_CAP + 1), "physics");
  assert.equal(resolveEffectiveLayout("physics", 5), "physics");
  assert.equal(resolveEffectiveLayout("auto", 10), "hierarchical");
  assert.equal(resolveEffectiveLayout("auto", READABLE_NODE_CAP + 5), "physics");
});

test("display soft-cap keeps graphs readable", async () => {
  const { applyGraphBudget, READABLE_NODE_CAP } = await import("./graph_state.mjs");
  const nodes = Array.from({ length: 80 }, (_, i) => ({ id: String(i) }));
  const edges = Array.from({ length: 120 }, (_, i) => ({ from: String(i % 80), to: String((i + 1) % 80) }));
  const result = applyGraphBudget(nodes, edges, { nodes: READABLE_NODE_CAP, edges: READABLE_NODE_CAP * 3 });
  assert.ok(result.nodes.length <= READABLE_NODE_CAP);
  assert.equal(result.degraded, true);
});

test("pickFocusNodeId prefers explicit center then hub", async () => {
  const { pickFocusNodeId } = await import("./graph_state.mjs");
  const nodes = [{ id: "a" }, { id: "b" }, { id: "c" }];
  const edges = [
    { from: "a", to: "b" },
    { from: "a", to: "c" },
  ];
  assert.equal(pickFocusNodeId(nodes, edges, "c"), "c");
  assert.equal(pickFocusNodeId(nodes, edges, ""), "a");
});

test("shortGraphLabel shortens package paths and types", async () => {
  const { shortGraphLabel } = await import("./graph_state.mjs");
  assert.equal(shortGraphLabel("github.com/pubgo/dix/example/http/domain/billing"), "domain/billing");
  assert.equal(shortGraphLabel("*billing.Config"), "Config");
  assert.equal(shortGraphLabel("main"), "main");
});

test("module maps prefer fit camera; dense graphs prefer focus", async () => {
  const { resolveCameraStrategy } = await import("./graph_state.mjs");
  assert.equal(resolveCameraStrategy("modules", 11), "fit");
  assert.equal(resolveCameraStrategy("ego", 40), "focus");
  assert.equal(resolveCameraStrategy("providers", 8), "fit");
});

test("assessLayoutMetrics rejects thin-strip layouts", async () => {
  const { assessLayoutMetrics } = await import("./graph_state.mjs");
  const thin = {};
  for (let i = 0; i < 10; i++) thin["n" + i] = { x: i * 40, y: 0 };
  assert.equal(assessLayoutMetrics(thin).ok, false);
  assert.equal(assessLayoutMetrics(thin).thin, true);

  const spread = {
    a: { x: 0, y: 0 },
    b: { x: 200, y: 0 },
    c: { x: 0, y: 200 },
    d: { x: 200, y: 200 },
    e: { x: 100, y: 100 },
  };
  assert.equal(assessLayoutMetrics(spread).ok, true);
});

test("buildModuleMapGraph uses modules not objects as nodes", async () => {
  const { buildModuleMapGraph } = await import("./graph_state.mjs");
  const { nodes, edges } = buildModuleMapGraph([
    { name: "app/a", provider_count: 2, object_count: 5, depends_on: ["app/b"] },
    { name: "app/b", provider_count: 1, object_count: 1, depends_on: [] },
  ]);
  assert.equal(nodes.length, 2);
  assert.equal(edges.length, 1);
  assert.equal(nodes[0].data.type, "module");
  assert.match(nodes[0].label, /2p\/5o/);
});

test("layoutStarPositions spreads modules around a hub", async () => {
  const { layoutStarPositions, assessLayoutMetrics } = await import("./graph_state.mjs");
  const nodes = [
    { id: "main" },
    { id: "a" }, { id: "b" }, { id: "c" }, { id: "d" }, { id: "e" },
  ];
  const edges = [
    { from: "main", to: "a" },
    { from: "main", to: "b" },
    { from: "main", to: "c" },
    { from: "main", to: "d" },
    { from: "main", to: "e" },
  ];
  const positions = layoutStarPositions(nodes, edges);
  assert.deepEqual(positions.main, { x: 0, y: 0 });
  assert.equal(Object.keys(positions).length, 6);
  assert.equal(assessLayoutMetrics(positions).ok, true);
});

test("aggregateByGroups collapses matching providers", async () => {
  const { aggregateByGroups, matchGroup, filterByPrefix, buildMermaidSource } = await import("./graph_workbench.mjs");
  const rules = [{ name: "billing", prefixes: ["billing"] }];
  assert.equal(matchGroup("billing.Service", "example/billing", rules), "billing");
  const nodes = [
    { id: "p1", data: { kind: "provider", packagePath: "example/billing" } },
    { id: "p2", data: { kind: "provider", packagePath: "example/billing" } },
    { id: "t1", data: { kind: "type", packagePath: "example/other" } },
  ];
  const edges = [
    { from: "p1", to: "t1" },
    { from: "p2", to: "t1" },
  ];
  const agg = aggregateByGroups(nodes, edges, { enabled: true, groupRules: rules });
  assert.equal(agg.nodes.some((n) => n.id === "group:billing"), true);
  assert.equal(agg.nodes.some((n) => n.id === "p1"), false);
  const filtered = filterByPrefix(nodes, edges, "billing");
  assert.deepEqual(filtered.nodes.map((n) => n.id).sort(), ["p1", "p2"]);
  assert.match(buildMermaidSource(agg.nodes, agg.edges), /flowchart TD/);
});

test("group rules persistence round-trips through storage", async () => {
  const { loadGroupRulesState, saveGroupRulesState, mergeServerGroupRules } = await import("./graph_workbench.mjs");
  const mem = new Map();
  const storage = {
    getItem: (k) => (mem.has(k) ? mem.get(k) : null),
    setItem: (k, v) => mem.set(k, String(v)),
  };
  saveGroupRulesState({
    aggregateGroups: true,
    groupRules: [{ name: "core", prefixes: ["github.com/x"] }],
  }, storage);
  const loaded = loadGroupRulesState(storage);
  assert.equal(loaded.aggregateGroups, true);
  assert.deepEqual(loaded.groupRules, [{ name: "core", prefixes: ["github.com/x"] }]);
  assert.deepEqual(
    mergeServerGroupRules([], [{ name: "api", prefixes: ["api/"] }]),
    [{ name: "api", prefixes: ["api/"] }],
  );
  assert.deepEqual(
    mergeServerGroupRules([{ name: "local", prefixes: ["l"] }], [{ name: "api", prefixes: ["api/"] }]),
    [{ name: "local", prefixes: ["l"] }],
  );
});
