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
