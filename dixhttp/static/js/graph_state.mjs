export function resolveGraphMode(query) {
  const mode = (query.get("mode") || "").trim();
  return ["modules", "module", "ego", "providers", "types"].includes(mode) ? mode : "modules";
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

  const ranked = [...nodes].sort((a, b) =>
    (degree.get(b.id) || 0) - (degree.get(a.id) || 0) ||
    String(a.id).localeCompare(String(b.id))
  ).slice(0, budget.nodes);
  const keptNodes = new Set(ranked.map(node => node.id));
  const keptEdges = edges.filter(edge =>
    keptNodes.has(edge.from) && keptNodes.has(edge.to)
  ).slice(0, budget.edges);

  return {
    nodes: nodes.filter(node => keptNodes.has(node.id)),
    edges: keptEdges,
    degraded: true,
  };
}


export function issueGraphHash(issue = {}) {
  const params = new URLSearchParams();
  if (issue.output_type) {
    params.set("mode", "ego");
    params.set("center", issue.output_type);
  } else if (issue.module) {
    params.set("mode", "module");
    params.set("module", issue.module);
  } else if (issue.provider) {
    params.set("mode", "providers");
    params.set("prefix", issue.provider);
  } else {
    params.set("mode", "modules");
  }
  return "#/graph?" + params.toString();
}

export function issueTraceHash(issue = {}) {
  const params = new URLSearchParams();
  if (issue.provider) params.set("provider", issue.provider);
  if (issue.output_type) params.set("output_type", issue.output_type);
  params.set("status", issue.severity === "slow" ? "slow" : "error");
  return "#/trace?" + params.toString();
}

if (typeof window !== "undefined") {
  window.DIXGraphState = { resolveGraphMode, applyGraphBudget, issueGraphHash, issueTraceHash };
}
