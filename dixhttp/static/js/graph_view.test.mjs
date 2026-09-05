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
