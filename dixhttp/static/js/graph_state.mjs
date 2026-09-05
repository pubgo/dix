export function resolveGraphMode(query) {
  const mode = (query.get("mode") || "").trim();
  return ["modules", "module", "ego", "providers", "types"].includes(mode) ? mode : "modules";
}

/** Soft cap for on-canvas readability; denser data stays in hubs/table warnings. */
export const READABLE_NODE_CAP = 40;

/** Shorten package paths and type names for readable node labels. */
export function shortGraphLabel(name) {
  const s = String(name || "").replace(/^\*/, "");
  if (!s) return "";
  if (s.includes("/")) {
    const parts = s.split("/").filter(Boolean);
    return parts.slice(-2).join("/") || s;
  }
  const dotted = s.split(".");
  return dotted[dotted.length - 1] || s;
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

export function resolveEffectiveLayout(preferred, nodeCount) {
  const choice = (preferred || "auto").trim() || "auto";
  if (choice === "physics") return "physics";
  if (choice === "hierarchical") {
    return nodeCount > READABLE_NODE_CAP ? "physics" : "hierarchical";
  }
  // auto (default): small graphs get hierarchy, denser graphs get physics
  return nodeCount > READABLE_NODE_CAP ? "physics" : "hierarchical";
}

/** Module maps and tiny graphs should fit the canvas; denser views focus a hub. */
export function resolveCameraStrategy(mode, nodeCount) {
  if (mode === "modules" || nodeCount <= 12) return "fit";
  return "focus";
}

/**
 * Place a hub at origin and remaining nodes on a circle — readable for module maps.
 * Returns { [id]: { x, y } }.
 */
export function layoutStarPositions(nodes = [], edges = [], opts = {}) {
  const radius = opts.radius ?? Math.max(260, 40 * Math.max(nodes.length - 1, 1));
  const hubId = pickFocusNodeId(nodes, edges);
  const others = nodes.filter((n) => n.id !== hubId);
  const positions = {};
  if (hubId) positions[hubId] = { x: 0, y: 0 };
  const n = Math.max(others.length, 1);
  others.forEach((node, i) => {
    const angle = (2 * Math.PI * i) / n - Math.PI / 2;
    positions[node.id] = {
      x: Math.cos(angle) * radius,
      y: Math.sin(angle) * radius,
    };
  });
  return positions;
}

/**
 * Judge whether node positions are a readable 2D spread (not a hairline).
 * positions: { [id]: { x, y } }
 */
export function assessLayoutMetrics(positions, opts = {}) {
  const maxAspect = opts.maxAspect ?? 6;
  const minSpan = opts.minSpan ?? 80;
  const ids = Object.keys(positions || {});
  if (ids.length === 0) {
    return { ok: false, reason: "empty", nodeCount: 0, aspect: 0, spanX: 0, spanY: 0, thin: false };
  }
  let minX = Infinity, maxX = -Infinity, minY = Infinity, maxY = -Infinity;
  for (const id of ids) {
    const p = positions[id];
    if (!p) continue;
    minX = Math.min(minX, p.x);
    maxX = Math.max(maxX, p.x);
    minY = Math.min(minY, p.y);
    maxY = Math.max(maxY, p.y);
  }
  const spanX = Math.max(0, maxX - minX);
  const spanY = Math.max(0, maxY - minY);
  const major = Math.max(spanX, spanY, 1);
  const minor = Math.max(Math.min(spanX, spanY), 1);
  const aspect = major / minor;
  const thin = ids.length >= 5 && aspect > maxAspect;
  const tooSmall = ids.length >= 5 && Math.min(spanX, spanY) < minSpan && aspect > 3;
  const ok = !thin && !tooSmall;
  return {
    ok,
    reason: ok ? "" : (thin ? "thin-strip" : "too-small-spread"),
    nodeCount: ids.length,
    aspect,
    spanX,
    spanY,
    thin,
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

export function pickFocusNodeId(nodes = [], edges = [], preferredId = "") {
  if (preferredId && nodes.some(node => node.id === preferredId)) return preferredId;
  const hubs = rankHubNodes(nodes, edges, 1);
  if (hubs.length) return hubs[0].id;
  return nodes[0] ? nodes[0].id : null;
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
    resolveGraphMode, applyGraphBudget, READABLE_NODE_CAP, shortGraphLabel,
    resolveEffectiveLayout, resolveCameraStrategy, layoutStarPositions, assessLayoutMetrics, pickFocusNodeId,
    createLoadGuard, rankHubNodes,
    issueGraphHash, issueTraceHash, matchTraceRecord, filterTraceRecords,
  };
}
