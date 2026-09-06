// Shared helpers for legacy architecture visualization (labels, budget, groups, export).

export const GROUP_RULES_STORAGE_KEY = "dix.groupRules.v1";

export const ERROR_TYPE_GUIDE = [
  {
    code: "provider_registration_invalid",
    summary: "Provider 注册参数非法（如 nil、非函数、返回值不合法）。",
    action: "检查 Provide/TryProvide 的入参签名与返回值数量。",
  },
  {
    code: "inject_dependency_missing",
    summary: "注入阶段缺失依赖，当前类型没有可用 provider。",
    action: "确认依赖已注册、类型精确匹配，并检查导入顺序。",
  },
  {
    code: "provider_input_unresolved",
    summary: "Provider 的输入参数无法解析（上游依赖缺失或不匹配）。",
    action: "根据 input_type/input_types 逐级回溯缺失节点。",
  },
  {
    code: "provider_return_error",
    summary: "Provider 正常执行但返回了业务错误。",
    action: "优先排查外部资源（DB/缓存/HTTP）与配置有效性。",
  },
  {
    code: "provider_panic",
    summary: "Provider 执行中发生 panic。",
    action: "建议 provider 内捕获并返回 error，必要时开启 debug 栈。",
  },
  {
    code: "provider_timeout",
    summary: "Provider 执行超时。",
    action: "优化初始化路径，或临时调大 WithProviderTimeout。",
  },
  {
    code: "inject_callback_error",
    summary: "Inject/TryInject 的回调函数主动返回了 error。",
    action: "检查回调里的参数校验和业务逻辑分支。",
  },
  {
    code: "dependency_cycle",
    summary: "检测到循环依赖。",
    action: "拆分互相引用组件，改为接口或延迟注入解环。",
  },
  {
    code: "inject_failed",
    summary: "注入失败兜底分类（需要结合 stage/message 细看）。",
    action: "结合 root_cause + hint + provider_function 做链路定位。",
  },
];

export function normalizeTypeForMatch(typeName) {
  let t = String(typeName || "").trim();
  t = t.replace(/^\*/, "").replace(/^\[\]/, "");
  if (t.startsWith("map[")) {
    const idx = t.indexOf("]");
    if (idx > -1 && idx < t.length - 1) {
      t = t.slice(idx + 1).replace(/^\*/, "");
    }
  }
  return t;
}

export function extractPackagePath(typeName) {
  const t = String(typeName || "").trim();
  if (!t) return "";
  const lastSlash = t.lastIndexOf("/");
  const lastDot = t.lastIndexOf(".");
  if (lastDot > -1 && lastDot > lastSlash) return t.slice(0, lastDot);
  return "";
}

export function isPathLikePrefix(prefix) {
  return (
    prefix.includes("/") ||
    prefix.startsWith("github.com/") ||
    prefix.startsWith("gitee.com/") ||
    prefix.startsWith("gitlab.com/")
  );
}

export function nodeKind(node) {
  const d = (node && node.data) || {};
  return d.kind || d.type || "";
}

export function nodePackagePath(node) {
  const d = (node && node.data) || {};
  return d.packagePath || d.pkg || "";
}

export function matchGroup(typeName, pkgPathOverride, groupRules) {
  if (!typeName) return null;
  const normalized = normalizeTypeForMatch(typeName);
  const pkgPath = pkgPathOverride || extractPackagePath(normalized);
  for (const grp of groupRules || []) {
    for (const prefix of grp.prefixes || []) {
      const p = String(prefix || "").trim();
      if (!p) continue;
      const target = pkgPath || normalized;
      if (isPathLikePrefix(p)) {
        if (target.startsWith(p) || target.includes(p)) return grp.name;
        continue;
      }
      if (target.includes(p)) return grp.name;
    }
  }
  return null;
}

export function filterByPrefix(nodes, edges, prefix, protectedIds = new Set()) {
  const p = String(prefix || "").trim();
  if (!p) return { nodes, edges };
  const matches = (node) => {
    if (!node || !node.data) return false;
    const pkg = String(nodePackagePath(node) || "");
    const id = String(node.id || "");
    const fullType = String(node.data.fullType || node.data.label || "");
    const fn = String((node.data.provider && node.data.provider.function_name) || node.data.function_name || "");
    return pkg.includes(p) || id.includes(p) || fullType.includes(p) || fn.includes(p);
  };
  const keep = new Set();
  nodes.forEach((n) => {
    if (protectedIds.has(n.id) || matches(n)) keep.add(n.id);
  });
  return {
    nodes: nodes.filter((n) => keep.has(n.id)),
    edges: edges.filter((e) => keep.has(e.from) && keep.has(e.to)),
  };
}

export function aggregateByGroups(nodes, edges, options = {}) {
  const {
    enabled = false,
    groupRules = [],
    expandedGroups = new Set(),
    protectedIds = new Set(),
    debug = false,
  } = options;

  if (!enabled || !groupRules.length) {
    return { nodes, edges, groupMembers: {}, debugStats: null };
  }

  const mapping = new Map();
  const groupNodes = new Map();
  const debugStats = new Map();
  const members = {};

  const makeGroup = (name) => ({
    id: `group:${name}`,
    label: name,
    shape: "box",
    font: { size: 13 },
    color: { background: "#e5e7eb", border: "#6b7280" },
    data: { kind: "group", type: "group", group: name },
  });

  nodes.forEach((n) => {
    if (protectedIds.has(n.id)) return;
    const kind = nodeKind(n);
    if (kind !== "type" && kind !== "provider") return;
    const pkg = nodePackagePath(n);
    const groupName = matchGroup(n.id, pkg, groupRules);
    if (!groupName) return;
    if (expandedGroups.has(groupName)) return;
    mapping.set(n.id, `group:${groupName}`);
    if (!groupNodes.has(groupName)) groupNodes.set(groupName, makeGroup(groupName));
    if (!members[groupName]) members[groupName] = [];
    if (members[groupName].length < 200) {
      members[groupName].push({ id: n.id, packagePath: pkg, nodeType: kind });
    }
    if (debug) {
      if (!debugStats.has(groupName)) debugStats.set(groupName, []);
      if (debugStats.get(groupName).length < 50) {
        debugStats.get(groupName).push({ id: n.id, packagePath: pkg });
      }
    }
  });

  if (mapping.size === 0) {
    return { nodes, edges, groupMembers: members, debugStats: debug ? debugStats : null };
  }

  const mappedNodes = nodes.filter((n) => !mapping.has(n.id));
  groupNodes.forEach((n) => mappedNodes.push(n));

  const edgeMap = new Map();
  edges.forEach((e) => {
    const from = mapping.get(e.from) || e.from;
    const to = mapping.get(e.to) || e.to;
    if (from === to) return;
    const key = `${from}=>${to}`;
    if (edgeMap.has(key)) return;
    edgeMap.set(key, { ...e, from, to });
  });

  return {
    nodes: mappedNodes,
    edges: Array.from(edgeMap.values()),
    groupMembers: members,
    debugStats: debug ? debugStats : null,
  };
}

export function loadGroupRulesState(storage = globalThis.localStorage) {
  const empty = { aggregateGroups: false, groupRules: [] };
  try {
    const raw = storage && storage.getItem(GROUP_RULES_STORAGE_KEY);
    if (!raw) return empty;
    const data = JSON.parse(raw);
    return {
      aggregateGroups: typeof data.aggregateGroups === "boolean" ? data.aggregateGroups : false,
      groupRules: Array.isArray(data.groupRules)
        ? data.groupRules.map((g) => ({
            name: g.name,
            prefixes: Array.isArray(g.prefixes) ? g.prefixes : [],
          }))
        : [],
    };
  } catch {
    return empty;
  }
}

export function saveGroupRulesState(state, storage = globalThis.localStorage) {
  try {
    const payload = {
      aggregateGroups: !!state.aggregateGroups,
      groupRules: (state.groupRules || []).map((g) => ({
        name: g.name,
        prefixes: g.prefixes || [],
      })),
    };
    storage.setItem(GROUP_RULES_STORAGE_KEY, JSON.stringify(payload));
  } catch {
    /* ignore quota / private mode */
  }
}

export function mergeServerGroupRules(localRules, serverRules) {
  if (localRules && localRules.length) return localRules;
  if (!Array.isArray(serverRules) || !serverRules.length) return localRules || [];
  return serverRules.map((g) => ({
    name: g.name,
    prefixes: Array.isArray(g.prefixes) ? g.prefixes : [],
  }));
}

export function filterPackages(packages, query) {
  const q = String(query || "").toLowerCase().trim();
  const list = Array.isArray(packages) ? [...packages] : [];
  list.sort((a, b) => (b.provider_count || 0) - (a.provider_count || 0));
  if (!q) return list;
  return list.filter((p) => String(p.name || "").toLowerCase().includes(q));
}

export function escapeMermaidLabel(label) {
  return String(label || "")
    .replace(/"/g, "'")
    .replace(/\n/g, " ")
    .replace(/\r/g, " ");
}

export function buildMermaidSource(nodes, edges) {
  if (!nodes || !nodes.length) return "flowchart TD\n  A[No data]";
  const idMap = new Map();
  const lines = ["flowchart TD"];
  nodes.forEach((n, idx) => {
    const safeId = `N${idx}`;
    idMap.set(n.id, safeId);
    lines.push(`  ${safeId}["${escapeMermaidLabel(n.label || n.id)}"]`);
  });
  (edges || []).forEach((e) => {
    const from = idMap.get(e.from);
    const to = idMap.get(e.to);
    if (!from || !to) return;
    lines.push(`  ${from} --> ${to}`);
  });
  return lines.join("\n");
}

function escapeXml(s) {
  return String(s || "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

export function buildSvgFromNetwork(network) {
  if (!network || !network.body) return "";
  const nodesData = network.body.data.nodes.get();
  const edgesData = network.body.data.edges.get();
  if (!nodesData || !nodesData.length) return "";

  const positions = network.getPositions(nodesData.map((n) => n.id));
  let minX = Infinity;
  let minY = Infinity;
  let maxX = -Infinity;
  let maxY = -Infinity;

  nodesData.forEach((n) => {
    let box;
    try {
      box = network.getBoundingBox(n.id);
    } catch {
      box = null;
    }
    const pos = positions[n.id] || { x: 0, y: 0 };
    const left = box ? box.left : pos.x - 50;
    const right = box ? box.right : pos.x + 50;
    const top = box ? box.top : pos.y - 25;
    const bottom = box ? box.bottom : pos.y + 25;
    minX = Math.min(minX, left);
    minY = Math.min(minY, top);
    maxX = Math.max(maxX, right);
    maxY = Math.max(maxY, bottom);
  });

  if (!isFinite(minX) || !isFinite(minY) || !isFinite(maxX) || !isFinite(maxY)) return "";

  const padding = 20;
  const width = Math.max(1, Math.round(maxX - minX + padding * 2));
  const height = Math.max(1, Math.round(maxY - minY + padding * 2));
  const mapX = (x) => x - minX + padding;
  const mapY = (y) => y - minY + padding;
  const parts = [];
  parts.push(`<?xml version="1.0" encoding="UTF-8"?>`);
  parts.push(`<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}">`);
  parts.push(`<defs>`);
  parts.push(`<marker id="arrow" markerWidth="10" markerHeight="10" refX="8" refY="3" orient="auto" markerUnits="strokeWidth">`);
  parts.push(`<path d="M0,0 L0,6 L9,3 z" fill="#9ca3af" />`);
  parts.push(`</marker></defs>`);

  edgesData.forEach((e) => {
    const from = positions[e.from];
    const to = positions[e.to];
    if (!from || !to) return;
    const color = (e.color && e.color.color) || "#9ca3af";
    const dash = e.dashes ? ' stroke-dasharray="6 4"' : "";
    parts.push(
      `<line x1="${mapX(from.x)}" y1="${mapY(from.y)}" x2="${mapX(to.x)}" y2="${mapY(to.y)}" stroke="${color}" stroke-width="1.5" marker-end="url(#arrow)"${dash} />`,
    );
  });

  nodesData.forEach((n) => {
    const pos = positions[n.id] || { x: 0, y: 0 };
    let box;
    try {
      box = network.getBoundingBox(n.id);
    } catch {
      box = null;
    }
    const left = box ? box.left : pos.x - 50;
    const right = box ? box.right : pos.x + 50;
    const top = box ? box.top : pos.y - 25;
    const bottom = box ? box.bottom : pos.y + 25;
    const w = Math.max(10, right - left);
    const h = Math.max(10, bottom - top);
    const x = mapX(left);
    const y = mapY(top);
    const cx = mapX(pos.x);
    const cy = mapY(pos.y);
    const bg = (n.color && n.color.background) || "#bfdbfe";
    const border = (n.color && n.color.border) || "#3b82f6";
    const fontSize = (n.font && n.font.size) || 12;
    const fontColor = (n.font && n.font.color) || "#111827";
    const kind = nodeKind(n);
    const isBox = n.shape === "box" || kind === "provider" || kind === "group";
    if (isBox) {
      const rx = kind === "group" ? 8 : 4;
      parts.push(`<rect x="${x}" y="${y}" width="${w}" height="${h}" rx="${rx}" fill="${bg}" stroke="${border}" stroke-width="1.5" />`);
    } else {
      parts.push(`<ellipse cx="${cx}" cy="${cy}" rx="${w / 2}" ry="${h / 2}" fill="${bg}" stroke="${border}" stroke-width="1.5" />`);
    }
    const label = escapeXml(String(n.label || n.id || "").split("\n")[0]);
    parts.push(`<text x="${cx}" y="${cy + fontSize / 3}" text-anchor="middle" font-size="${fontSize}" fill="${fontColor}" font-family="sans-serif">${label}</text>`);
  });

  parts.push(`</svg>`);
  return parts.join("");
}

export function downloadTextFile(filename, content, mime = "text/plain;charset=utf-8") {
  const blob = new Blob([content], { type: mime });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

export function filterErrorGuide(items, query) {
  const q = String(query || "").toLowerCase().trim();
  if (!q) return items || [];
  return (items || []).filter((item) => {
    const hay = `${item.code} ${item.summary} ${item.action}`.toLowerCase();
    return hay.includes(q);
  });
}

export function filterRuntimeStats(stats, options = {}) {
  const q = String(options.query || "").toLowerCase().trim();
  const onlyExecuted = !!options.onlyExecuted;
  return (stats || []).filter((s) => {
    if (onlyExecuted && !(s.call_count > 0)) return false;
    if (!q) return true;
    const hay = `${s.function_name || ""} ${s.output_type || ""} ${s.last_error || ""}`.toLowerCase();
    return hay.includes(q);
  });
}

if (typeof window !== "undefined") {
  window.DIXGraphWorkbench = {
    GROUP_RULES_STORAGE_KEY,
    ERROR_TYPE_GUIDE,
    normalizeTypeForMatch,
    extractPackagePath,
    isPathLikePrefix,
    nodeKind,
    nodePackagePath,
    matchGroup,
    filterByPrefix,
    aggregateByGroups,
    loadGroupRulesState,
    saveGroupRulesState,
    mergeServerGroupRules,
    filterPackages,
    escapeMermaidLabel,
    buildMermaidSource,
    buildSvgFromNetwork,
    downloadTextFile,
    filterErrorGuide,
    filterRuntimeStats,
  };
}
