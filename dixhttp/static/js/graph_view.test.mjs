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

test("explicit hierarchical layout is never auto-swapped", async () => {
  const { resolveEffectiveLayout, READABLE_NODE_CAP } = await import("./graph_state.mjs");
  assert.equal(resolveEffectiveLayout("hierarchical", READABLE_NODE_CAP), "hierarchical");
  assert.equal(resolveEffectiveLayout("hierarchical", READABLE_NODE_CAP + 50), "hierarchical");
  assert.equal(resolveEffectiveLayout("hierarchical", 20, { stripRisk: { wide: true } }), "hierarchical");
  assert.equal(resolveEffectiveLayout("physics", 5), "physics");
  assert.equal(resolveEffectiveLayout("auto", 10), "hierarchical");
  assert.equal(resolveEffectiveLayout("auto", READABLE_NODE_CAP + 5), "physics");
});
test("estimateHierarchicalStripRisk flags many roots", async () => {
  const { estimateHierarchicalStripRisk } = await import("./graph_state.mjs");
  const nodes = Array.from({ length: 10 }, (_, i) => ({ id: "n" + i }));
  const edges = []; // all roots / islands
  const risk = estimateHierarchicalStripRisk(nodes, edges);
  assert.equal(risk.wide, true);
  assert.ok(risk.roots >= 6);

  const chainNodes = [
    { id: "a" }, { id: "b" }, { id: "c" }, { id: "d" },
    { id: "e" }, { id: "f" }, { id: "g" }, { id: "h" },
  ];
  const chainEdges = [
    { from: "a", to: "b" }, { from: "b", to: "c" }, { from: "c", to: "d" },
    { from: "d", to: "e" }, { from: "e", to: "f" }, { from: "f", to: "g" },
    { from: "g", to: "h" },
  ];
  assert.equal(estimateHierarchicalStripRisk(chainNodes, chainEdges).wide, false);
});

test("hierarchical soft budget constants are larger than display soft-cap", async () => {
  const { READABLE_NODE_CAP, HIERARCHICAL_NODE_BUDGET, HIERARCHICAL_EDGE_BUDGET } = await import("./graph_state.mjs");
  assert.ok(HIERARCHICAL_NODE_BUDGET > READABLE_NODE_CAP);
  assert.ok(HIERARCHICAL_EDGE_BUDGET >= HIERARCHICAL_NODE_BUDGET);
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
  assert.equal(shortGraphLabel("*billing.Config"), "billing.Config");
  assert.equal(shortGraphLabel("main"), "main");
  assert.equal(
    shortGraphLabel("*main.Worker[github.com/pubgo/dix/example/http.RoleAuthReader]"),
    "Worker[http.RoleAuthReader]"
  );
  assert.equal(
    shortGraphLabel("*github.com/pubgo/dix/example/http/domain/analytics.Handler"),
    "analytics.Handler"
  );
});

test("providerDisplayLabel prefers output type over anonymous func names", async () => {
  const { providerDisplayLabel } = await import("./graph_state.mjs");
  assert.equal(
    providerDisplayLabel({
      function_name: "github.com/pubgo/dix/example/http/domain/analytics.Providers.func6",
      output_type: "*github.com/pubgo/dix/example/http/domain/analytics.Handler",
      output_pkg: "github.com/pubgo/dix/example/http/domain/analytics",
    }),
    "domain/analytics.Handler"
  );
  assert.equal(
    providerDisplayLabel({
      function_name: "main.registerWorker[...].func1",
      output_type: "*main.Worker[github.com/pubgo/dix/example/http.RoleAuthReader]",
    }),
    "Worker[http.RoleAuthReader]"
  );
  assert.equal(
    providerDisplayLabel({ function_name: "main.buildContainer.func4", output_type: "*main.Application" }),
    "main.Application"
  );
});

test("providerDisplayLabel disambiguates shared layer package names", async () => {
  const { providerDisplayLabel } = await import("./graph_state.mjs");
  assert.equal(
    providerDisplayLabel({
      function_name: "handler.Provide.func1",
      output_type: "*handler.Handler",
      output_pkg: "github.com/pubgo/dix/example/http/domain/billing/handler",
    }),
    "billing/handler.Handler"
  );
  assert.equal(
    providerDisplayLabel({
      function_name: "handler.Provide.func1",
      output_type: "*handler.Handler",
      output_pkg: "github.com/pubgo/dix/example/http/domain/inventory/handler",
    }),
    "inventory/handler.Handler"
  );
  assert.equal(
    providerDisplayLabel({
      function_name: "service.Provide.func1",
      output_type: "*service.Service",
      output_pkg: "github.com/pubgo/dix/example/http/domain/billing/service",
    }),
    "billing/service.Service"
  );
});

test("labelLodVisibleIds keeps hubs when zoomed out", async () => {
  const { labelLodVisibleIds } = await import("./graph_state.mjs");
  const nodes = [{ id: "hub" }, { id: "a" }, { id: "b" }, { id: "c" }];
  const edges = [
    { from: "hub", to: "a" },
    { from: "hub", to: "b" },
    { from: "hub", to: "c" },
  ];
  const zoomedOut = labelLodVisibleIds(nodes, edges, 0.4, { hubLimit: 2 });
  assert.equal(zoomedOut.has("hub"), true);
  assert.equal(zoomedOut.size <= 2, true);
  const zoomedIn = labelLodVisibleIds(nodes, edges, 1.0, { hubLimit: 2 });
  assert.equal(zoomedIn.size, 4);
});

test("module maps prefer fit camera; dense graphs prefer focus", async () => {
  const { resolveCameraStrategy } = await import("./graph_state.mjs");
  assert.equal(resolveCameraStrategy("modules", 11), "fit");
  assert.equal(resolveCameraStrategy("ego", 40), "fit");
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
  ], { hierarchy: false });
  assert.equal(nodes.length, 2);
  assert.equal(edges.length, 1);
  assert.equal(nodes[0].data.type, "module");
  assert.match(nodes[0].label, /2p\/5o/);
});

test("buildModuleMapGraph expands package hierarchy and disperses fat packages", async () => {
  const { buildModuleMapGraph } = await import("./graph_state.mjs");
  const modules = [
    {
      name: "github.com/pubgo/dix/example/http",
      provider_count: 30,
      object_count: 30,
      depends_on: ["github.com/pubgo/dix/example/http/domain/analytics"],
    },
    {
      name: "github.com/pubgo/dix/example/http/domain/analytics",
      provider_count: 7,
      object_count: 8,
      depends_on: [],
    },
  ];
  const providers = [
    { output_type: "*main.Application", output_pkg: "github.com/pubgo/dix/example/http" },
    { output_type: "*main.PluginPlatform", output_pkg: "github.com/pubgo/dix/example/http" },
    ...Array.from({ length: 20 }, (_, i) => ({
      output_type: `*main.Worker[github.com/pubgo/dix/example/http.Role${i}]`,
      output_pkg: "github.com/pubgo/dix/example/http",
    })),
    ...Array.from({ length: 8 }, (_, i) => ({
      output_type: `*main.Plugin[github.com/pubgo/dix/example/http.Role${i}]`,
      output_pkg: "github.com/pubgo/dix/example/http",
    })),
    {
      output_type: "*github.com/pubgo/dix/example/http/domain/analytics.Handler",
      output_pkg: "github.com/pubgo/dix/example/http/domain/analytics",
    },
  ];
  const g = buildModuleMapGraph(modules, { providers, minDepth: 5, fatThreshold: 12 });
  const maxLevel = Math.max(...g.nodes.map((n) => n.level || 1));
  assert.ok(maxLevel >= 3, `maxLevel=${maxLevel}`);
  // Must stay readable: not hundreds of micro-leaves / thousand depends edges.
  assert.ok(g.nodes.length < 40, `nodes=${g.nodes.length}`);
  assert.ok(g.edges.length < 80, `edges=${g.edges.length}`);
  assert.equal(g.edges.every((e) => e.data && e.data.kind === "containment"), true);
  assert.equal(g.nodes.every((n) => n.data.type === "module"), true);
  // Fat main should be split into coarse buckets, not one blob.
  const buckets = g.nodes.filter((n) => n.data.leaf && /\/(app|plugins|workers|diag|other)$/.test(n.id));
  assert.ok(buckets.length >= 2, `bucket leaves=${buckets.map((n) => n.id)}`);
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

test("declutterEdges keeps cross-package and hub edges first", async () => {
  const { declutterEdges } = await import("./graph_state.mjs");
  const nodes = [
    { id: "a.X", data: { packagePath: "pkg/a" } },
    { id: "b.Y", data: { packagePath: "pkg/b" } },
    { id: "a.Z", data: { packagePath: "pkg/a" } },
    { id: "a.W", data: { packagePath: "pkg/a" } },
  ];
  const edges = [
    { from: "a.X", to: "b.Y" }, // cross
    { from: "a.X", to: "a.Z" },
    { from: "a.Z", to: "a.W" },
    { from: "a.W", to: "a.X" },
    { from: "a.Z", to: "a.X" },
  ];
  const result = declutterEdges(nodes, edges, { maxEdges: 2, hubLimit: 2 });
  assert.equal(result.decluttered, true);
  assert.equal(result.edges.some((e) => e.from === "a.X" && e.to === "b.Y"), true);
  // May keep > maxEdges to avoid orphaning nodes that were previously connected.
  assert.equal(result.edges.length >= 2, true);
  for (const id of ["a.X", "b.Y", "a.Z", "a.W"]) {
    assert.equal(result.edges.some((e) => e.from === id || e.to === id), true);
  }
});

test("buildPackageSummary lists providers for OutputPkg", async () => {
  const { buildPackageSummary, typeNodeIdentity } = await import("./graph_state.mjs");
  const summary = buildPackageSummary({
    providers: [
      { id: "p1", function_name: "analytics.New", output_type: "*analytics.Client", output_pkg: "domain/analytics" },
      { id: "p2", function_name: "billing.New", output_type: "*billing.Client", output_pkg: "domain/billing" },
    ],
  }, "domain/analytics");
  assert.equal(summary.providers.length, 1);
  assert.equal(summary.types[0].id, typeNodeIdentity("*analytics.Client", "domain/analytics"));
  assert.equal(summary.types[0].fullType, "*analytics.Client");
});

test("buildProviderInventory lists all providers when unscoped", async () => {
  const { buildProviderInventory } = await import("./graph_state.mjs");
  const inv = buildProviderInventory({
    providers: [
      { id: "p1", function_name: "a.New", output_type: "*a.T", output_pkg: "pkg/a" },
      { id: "p2", function_name: "b.New", output_type: "*b.T", output_pkg: "pkg/b" },
    ],
  });
  assert.equal(inv.providers.length, 2);
  assert.ok(inv.types.length >= 2);
});

test("buildProviderDependencyGraph is provider-only with depends-on edges", async () => {
  const { buildProviderDependencyGraph } = await import("./graph_state.mjs");
  const g = buildProviderDependencyGraph([
    { id: "leaf", function_name: "leaf.New", output_type: "*leaf.T", output_pkg: "pkg/leaf", input_types: [] },
    {
      id: "mid", function_name: "mid.New", output_type: "*mid.T", output_pkg: "pkg/mid",
      input_types: ["*leaf.T"],
    },
    {
      id: "app", function_name: "app.New", output_type: "*app.T", output_pkg: "pkg/app",
      input_types: ["*mid.T"],
    },
  ]);
  assert.deepEqual(g.nodes.map((n) => n.id).sort(), ["app", "leaf", "mid"]);
  assert.equal(g.nodes.every((n) => n.data.type === "provider"), true);
  assert.equal(g.edges.some((e) => e.from === "app" && e.to === "mid"), true);
  assert.equal(g.edges.some((e) => e.from === "mid" && e.to === "leaf"), true);
  assert.equal(g.edges.some((e) => e.from === "leaf"), false);
});

test("assignProviderPyramidLevels puts unconsumed entries at level 1", async () => {
  const { buildProviderDependencyGraph, assignProviderPyramidLevels } = await import("./graph_state.mjs");
  const g = buildProviderDependencyGraph([
    { id: "leaf", output_type: "*leaf.T", input_types: [] },
    { id: "mid", output_type: "*mid.T", input_types: ["*leaf.T"] },
    { id: "app", output_type: "*app.T", input_types: ["*mid.T"] },
  ]);
  const levels = assignProviderPyramidLevels(g.nodes, g.edges);
  assert.equal(levels.get("app"), 1);
  assert.equal(levels.get("mid"), 2);
  assert.equal(levels.get("leaf"), 3);
});

test("truncateProviderPyramid keeps only levels up to depth", async () => {
  const { buildProviderDependencyGraph, assignProviderPyramidLevels, truncateProviderPyramid } = await import("./graph_state.mjs");
  const g = buildProviderDependencyGraph([
    { id: "leaf", output_type: "*leaf.T", input_types: [] },
    { id: "mid", output_type: "*mid.T", input_types: ["*leaf.T"] },
    { id: "app", output_type: "*app.T", input_types: ["*mid.T"] },
  ]);
  const levels = assignProviderPyramidLevels(g.nodes, g.edges);
  const cut = truncateProviderPyramid(g.nodes, g.edges, levels, 2);
  assert.deepEqual(cut.nodes.map((n) => n.id).sort(), ["app", "mid"]);
  assert.equal(cut.edges.length, 1);
  assert.equal(cut.edges[0].from, "app");
});

test("providerRelatedSubgraph keeps upstream and downstream of seeds", async () => {
  const { buildProviderDependencyGraph, providerRelatedSubgraph } = await import("./graph_state.mjs");
  const g = buildProviderDependencyGraph([
    { id: "leaf", output_type: "*leaf.T", output_pkg: "pkg/leaf", input_types: [] },
    { id: "mid", output_type: "*mid.T", output_pkg: "pkg/mid", input_types: ["*leaf.T"] },
    { id: "app", output_type: "*app.T", output_pkg: "pkg/app", input_types: ["*mid.T"] },
    { id: "other", output_type: "*other.T", output_pkg: "pkg/other", input_types: [] },
  ]);
  const sub = providerRelatedSubgraph(g.nodes, g.edges, ["mid"]);
  assert.deepEqual(sub.nodes.map((n) => n.id).sort(), ["app", "leaf", "mid"]);
});

test("buildProvidersPyramidView scopes package and applies depth", async () => {
  const { buildProvidersPyramidView } = await import("./graph_state.mjs");
  const providers = [
    { id: "leaf", function_name: "leaf.New", output_type: "*leaf.T", output_pkg: "pkg/leaf", input_types: [] },
    {
      id: "mid", function_name: "mid.New", output_type: "*mid.T", output_pkg: "pkg/mid",
      input_types: ["*leaf.T"],
    },
    {
      id: "app", function_name: "app.New", output_type: "*app.T", output_pkg: "pkg/app",
      input_types: ["*mid.T"],
    },
    {
      id: "orphan", function_name: "orphan.New", output_type: "*orphan.T", output_pkg: "pkg/other",
      input_types: [],
    },
  ];
  const full = buildProvidersPyramidView(providers, { depth: 2 });
  assert.equal(full.nodes.some((n) => n.id === "leaf"), false);
  assert.equal(full.nodes.some((n) => n.id === "app"), true);
  assert.equal(full.entries.length >= 1, true);
  assert.equal(full.nodes.find((n) => n.id === "app").level, 1);

  const scoped = buildProvidersPyramidView(providers, { depth: 0, packageName: "pkg/mid" });
  assert.deepEqual(scoped.nodes.map((n) => n.id).sort(), ["app", "leaf", "mid"]);
});

test("buildProviderDependencyGraph matches by input_pkgs when type names collide", async () => {
  const { buildProviderDependencyGraph } = await import("./graph_state.mjs");
  const g = buildProviderDependencyGraph([
    {
      id: "bh", output_type: "*handler.Handler", output_pkg: "domain/billing/handler",
      input_types: ["*service.Service"], input_pkgs: ["domain/billing/service"],
    },
    {
      id: "ih", output_type: "*handler.Handler", output_pkg: "domain/inventory/handler",
      input_types: ["*service.Service"], input_pkgs: ["domain/inventory/service"],
    },
    { id: "bs", output_type: "*service.Service", output_pkg: "domain/billing/service", input_types: [] },
    { id: "is", output_type: "*service.Service", output_pkg: "domain/inventory/service", input_types: [] },
  ]);
  assert.equal(g.edges.some((e) => e.from === "bh" && e.to === "bs"), true);
  assert.equal(g.edges.some((e) => e.from === "ih" && e.to === "is"), true);
  assert.equal(g.edges.some((e) => e.from === "bh" && e.to === "is"), false);
  assert.equal(g.edges.some((e) => e.from === "ih" && e.to === "bs"), false);
});

test("declutterEdges does not orphan previously connected nodes", async () => {
  const { declutterEdges } = await import("./graph_state.mjs");
  const nodes = [
    { id: "hub", data: { packagePath: "pkg/hub" } },
    { id: "a", data: { packagePath: "pkg/a" } },
    { id: "b", data: { packagePath: "pkg/b" } },
    { id: "w1", data: { packagePath: "pkg/plugins" } },
    { id: "w2", data: { packagePath: "pkg/plugins" } },
  ];
  const edges = [
    { from: "hub", to: "a" },
    { from: "hub", to: "b" },
    { from: "hub", to: "w1" },
    { from: "hub", to: "w2" },
    { from: "a", to: "b" },
  ];
  const cleaned = declutterEdges(nodes, edges, { maxEdges: 2, hubLimit: 2 });
  const degree = (id) => cleaned.edges.filter((e) => e.from === id || e.to === id).length;
  assert.equal(degree("w1") > 0 || degree("w2") > 0 || cleaned.edges.length >= 2, true);
  for (const id of ["hub", "a", "b", "w1", "w2"]) {
    const had = edges.some((e) => e.from === id || e.to === id);
    if (had) assert.equal(degree(id) > 0, true, `${id} should not be orphaned`);
  }
});

test("applyHiddenNodeSeeds removes seed and downstream only", async () => {
  const { applyHiddenNodeSeeds } = await import("./graph_state.mjs");
  const nodes = [{ id: "app" }, { id: "platform" }, { id: "worker" }, { id: "plugin" }, { id: "handler" }];
  const edges = [
    { from: "app", to: "platform" },
    { from: "app", to: "handler" },
    { from: "platform", to: "worker" },
    { from: "worker", to: "plugin" },
  ];
  const cut = applyHiddenNodeSeeds(nodes, edges, ["platform"]);
  assert.deepEqual(cut.nodes.map((n) => n.id).sort(), ["app", "handler"]);
  assert.equal(cut.edges.some((e) => e.from === "app" && e.to === "handler"), true);
  assert.equal(cut.edges.some((e) => e.to === "platform" || e.from === "platform"), false);
  assert.equal(cut.hiddenIds.has("plugin"), true);
});

test("applyHiddenNodeSeeds supports packagePrefix seeds", async () => {
  const { applyHiddenNodeSeeds, expandHiddenSeedsToNodeIds } = await import("./graph_state.mjs");
  const nodes = [
    { id: "p1", data: { packagePath: "example/http/plugins/auth" } },
    { id: "p2", data: { packagePath: "example/http/plugins/cache" } },
    { id: "h1", data: { packagePath: "example/http/domain/billing/handler" } },
    { id: "app", data: { packagePath: "example/http/app" } },
  ];
  const edges = [
    { from: "app", to: "p1" },
    { from: "app", to: "h1" },
    { from: "p1", to: "p2" },
  ];
  const ids = expandHiddenSeedsToNodeIds(nodes, [{ packagePrefix: "example/http/plugins" }]);
  assert.deepEqual(ids.sort(), ["p1", "p2"]);
  const cut = applyHiddenNodeSeeds(nodes, edges, [{ packagePrefix: "example/http/plugins", id: "pkg:example/http/plugins" }]);
  assert.deepEqual(cut.nodes.map((n) => n.id).sort(), ["app", "h1"]);
});

test("encode/decode hidden seeds for URL round-trip", async () => {
  const { encodeHiddenSeedsForUrl, decodeHiddenSeedsFromUrl } = await import("./graph_state.mjs");
  const seeds = [
    { id: "prov-1", label: "App" },
    { id: "pkg:example/http/plugins", label: "…/plugins", packagePrefix: "example/http/plugins" },
  ];
  const encoded = encodeHiddenSeedsForUrl(seeds);
  assert.equal(encoded, "prov-1|pkg:example/http/plugins");
  const decoded = decodeHiddenSeedsFromUrl(encoded);
  assert.equal(decoded.length, 2);
  assert.equal(decoded[0].id, "prov-1");
  assert.equal(decoded[1].packagePrefix, "example/http/plugins");
  assert.equal(decoded[1].id, "pkg:example/http/plugins");
  assert.deepEqual(decodeHiddenSeedsFromUrl(""), []);
  assert.deepEqual(decodeHiddenSeedsFromUrl(null), []);
});

test("providerDisplayLabel uses plugin impl path for shared Worker type", async () => {
  const { providerDisplayLabel } = await import("./graph_state.mjs");
  assert.equal(
    providerDisplayLabel({
      output_type: "plugins.Worker",
      output_pkg: "github.com/pubgo/dix/example/http/plugins",
      function_pkg: "github.com/pubgo/dix/example/http/plugins/auth.Provide",
    }),
    "auth.Worker"
  );
});

test("buildTypesPyramidView is type-only with entry types at level 1", async () => {
  const { buildTypesPyramidView, typeNodeIdentity } = await import("./graph_state.mjs");
  const appId = typeNodeIdentity("*app.T", "pkg/app");
  const midId = typeNodeIdentity("*mid.T", "pkg/mid");
  const view = buildTypesPyramidView({
    providers: [
      { output_type: "*leaf.T", output_pkg: "pkg/leaf", input_types: [] },
      {
        output_type: "*mid.T",
        output_pkg: "pkg/mid",
        input_types: ["*leaf.T"],
        input_pkgs: ["pkg/leaf"],
      },
      {
        output_type: "*app.T",
        output_pkg: "pkg/app",
        input_types: ["*mid.T"],
        input_pkgs: ["pkg/mid"],
      },
    ],
  }, { depth: 2 });
  assert.equal(view.nodes.every((n) => n.data.type === "type"), true);
  assert.equal(view.nodes.find((n) => n.id === appId).level, 1);
  assert.equal(view.nodes.find((n) => n.id === midId).level, 2);
  assert.equal(view.nodes.some((n) => n.data.fullType === "*leaf.T"), false);
  assert.equal(view.edges.some((e) => e.from === appId && e.to === midId), true);
});

test("buildTypeDependencyGraph keeps same short type from different packages", async () => {
  const { buildTypeDependencyGraph } = await import("./graph_state.mjs");
  const { nodes, edges } = buildTypeDependencyGraph({
    providers: [
      {
        output_type: "*handler.Handler",
        output_pkg: "example/http/domain/billing/handler",
        input_types: ["*service.Service"],
        input_pkgs: ["example/http/domain/billing/service"],
      },
      {
        output_type: "*service.Service",
        output_pkg: "example/http/domain/billing/service",
        input_types: [],
        input_pkgs: [],
      },
      {
        output_type: "*handler.Handler",
        output_pkg: "example/http/domain/inventory/handler",
        input_types: ["*service.Service"],
        input_pkgs: ["example/http/domain/inventory/service"],
      },
      {
        output_type: "*service.Service",
        output_pkg: "example/http/domain/inventory/service",
        input_types: [],
        input_pkgs: [],
      },
      {
        output_type: "*app.Application",
        output_pkg: "example/http/app",
        input_types: ["*handler.Handler", "*handler.Handler"],
        input_pkgs: [
          "example/http/domain/billing/handler",
          "example/http/domain/inventory/handler",
        ],
      },
    ],
  });
  const handlers = nodes.filter((n) => n.data.fullType === "*handler.Handler");
  assert.equal(handlers.length, 2);
  assert.equal(handlers.some((n) => n.label.includes("billing")), true);
  assert.equal(handlers.some((n) => n.label.includes("inventory")), true);
  const services = nodes.filter((n) => n.data.fullType === "*service.Service");
  assert.equal(services.length, 2);
  assert.ok(nodes.length >= 5);
  assert.ok(edges.length >= 4);
});
