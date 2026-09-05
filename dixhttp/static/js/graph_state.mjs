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
  if (issue.trace_id) params.set("trace_id", issue.trace_id);
  if (issue.provider) params.set("provider", issue.provider);
  if (issue.output_type) params.set("output_type", issue.output_type);
  params.set("status", issue.severity === "slow" ? "slow" : "error");
  return "#/trace?" + params.toString();
}

export function matchTraceRecord(rec = {}, filter = {}) {
  if (filter.trace_id && rec.trace_id !== filter.trace_id) return false;
  const provider = rec.provider_function || rec.provider || "";
  if (filter.provider && provider !== filter.provider) return false;
  if (filter.output_type && rec.output_type !== filter.output_type) return false;
  if (filter.status === "error" && rec.status !== "error") return false;
  if (filter.status === "slow") {
    return true;
  }
  return true;
}

export function filterTraceRecords(records = [], filter = {}) {
  return records.filter((rec) => matchTraceRecord(rec, filter));
}

if (typeof window !== "undefined") {
  window.DIXGraphState = {
    resolveGraphMode, applyGraphBudget, createLoadGuard, rankHubNodes,
    issueGraphHash, issueTraceHash, matchTraceRecord, filterTraceRecords,
  };
}
