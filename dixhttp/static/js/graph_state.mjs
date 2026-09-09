export function resolveGraphMode(query) {
  const mode = (query.get("mode") || "").trim();
  return ["modules", "module", "ego", "providers", "types"].includes(mode) ? mode : "modules";
}

/** Soft cap for on-canvas readability; denser data stays in hubs/table warnings. */
export const READABLE_NODE_CAP = 40;

/** Soft budget when user explicitly chooses hierarchical/force. */
export const HIERARCHICAL_NODE_BUDGET = 150;
export const HIERARCHICAL_EDGE_BUDGET = 400;

/**
 * Shorten package paths and type names for readable node labels.
 * Keep a package/type qualifier (e.g. analytics.Service) so many "Service"
 * nodes do not become indistinguishable when zoomed out.
 * Generic types like Worker[pkg.RoleX] keep the type args readable.
 */
export function shortGraphLabel(name) {
  const s = String(name || "").replace(/^\*/, "");
  if (!s) return "";
  const generic = s.match(/^([^\[\]]+)\[(.+)\]$/);
  if (generic) {
    const base = shortGraphLabel(generic[1]);
    const inner = shortGraphLabel(generic[2]);
    const baseLeaf = base.includes(".") ? base.split(".").pop() : base;
    return `${baseLeaf}[${inner}]`;
  }
  if (s.includes("/")) {
    const parts = s.split("/").filter(Boolean);
    const last = parts[parts.length - 1] || s;
    // Import path ending in pkg.Type → keep type qualifier, drop long module path.
    if (last.includes(".")) {
      return shortGraphLabel(last);
    }
    return parts.slice(-2).join("/") || s;
  }
  const dotted = s.split(".").filter(Boolean);
  if (dotted.length >= 2) {
    return dotted.slice(-2).join(".");
  }
  return dotted[dotted.length - 1] || s;
}

/** Shared layer leaf names that collide across domains (models/handler/…). */
const LAYER_PKG_LEAVES = new Set([
  "models", "infra", "logic", "service", "handler", "router", "plugins",
]);

/**
 * Qualify a type label with output_pkg parent when many packages share
 * the same Go package name (e.g. domain/billing/handler → handler.Handler).
 */
export function labelTypeWithPackage(typeName, pkgPath = "") {
  const short = shortGraphLabel(typeName);
  const parts = String(pkgPath || "").split("/").filter(Boolean);
  if (!short || parts.length < 2) return short;
  const leaf = parts[parts.length - 1];
  const parent = parts[parts.length - 2];
  const typeLeaf = short.includes(".") ? short.split(".").pop() : short;
  const typePkg = short.includes(".") ? short.slice(0, short.lastIndexOf(".")) : "";
  if (typePkg === leaf || LAYER_PKG_LEAVES.has(leaf)) {
    return `${parent}/${leaf}.${typeLeaf}`;
  }
  return short;
}

/**
 * Provider node label for the Providers canvas.
 * Prefer the Provide symbol (vault.Provide); arrow suffix only disambiguates
 * which Dix map/interface namespace that provider registers into — not a Go type
 * living under the plugin package.
 */
export function providerDisplayLabel(provider = {}) {
  const outs = provider.output_types && provider.output_types.length
    ? provider.output_types.filter(Boolean)
    : (provider.output_type ? [provider.output_type] : []);
  const pkg = provider.output_pkg || provider.function_pkg || "";
  const symbol = providerSymbolLabel(provider);
  const nsLeaf = outs.length === 1 ? dixNamespaceLeaf(outs[0]) : "";

  // Plugin package providers: always show as providers, not fake types.
  if (symbol && /\/plugins\/[^/]+/.test(String(provider.function_pkg || provider.function_name || ""))) {
    return nsLeaf ? `${symbol} → ${nsLeaf}` : symbol;
  }

  if (outs.length === 1) {
    return labelTypeWithPackage(outs[0], pkg);
  }
  if (outs.length > 1) {
    return outs.map((t) => labelTypeWithPackage(t, pkg)).join(", ");
  }
  return symbol || shortGraphLabel(provider.function_name || provider.id || "");
}

/** `vault.Provide.func2` → `vault.Provide` */
export function providerSymbolLabel(provider = {}) {
  const fn = String(provider.function_name || "").trim();
  if (!fn) return "";
  const leaf = fn.includes("/") ? fn.slice(fn.lastIndexOf("/") + 1) : fn;
  const cleaned = leaf.replace(/\.func\d+$/i, "");
  if (cleaned && cleaned.includes(".")) return cleaned;
  const pkg = String(provider.function_pkg || "").trim();
  if (pkg) {
    const parts = pkg.split("/").filter(Boolean);
    const last = parts[parts.length - 1] || "";
    if (last && cleaned) return `${last}.${cleaned}`;
    if (last) return last;
  }
  return cleaned || leaf;
}

/** Map/interface namespace leaf: map[string]…plugins.Worker → Worker */
export function dixNamespaceLeaf(typeName = "") {
  const raw = String(typeName || "").replace(/^\*/, "");
  const mapMatch = raw.match(/^map\[([^\]]+)\](.+)$/);
  const inner = mapMatch ? mapMatch[2] : raw;
  const short = shortGraphLabel(inner);
  return short.includes(".") ? short.split(".").pop() : short;
}

/** Pick which node ids keep visible labels at a given zoom scale. */
export function labelLodVisibleIds(nodes = [], edges = [], scale = 1, opts = {}) {
  const hubLimit = opts.hubLimit ?? 12;
  const hideBelow = opts.hideBelow ?? 0.62;
  const ids = nodes.map((n) => n.id);
  if (scale >= hideBelow || ids.length <= hubLimit) {
    return new Set(ids);
  }
  const hubs = rankHubNodes(nodes, edges, hubLimit).map((h) => h.id);
  return new Set(hubs);
}

/** Full or scoped provider inventory for the completeness channel (sidebar). */
export function buildProviderInventory(allData = {}, pkgName = "") {
  const pkg = String(pkgName || "").trim();
  if (pkg) return buildPackageSummary(allData, pkg);
  const list = allData.providers || [];
  const types = [];
  const seen = new Set();
  for (const p of list) {
    const pkgPath = p.output_pkg || p.function_pkg || "";
    for (const t of (p.output_types && p.output_types.length ? p.output_types : [p.output_type])) {
      if (!t) continue;
      const id = typeNodeIdentity(t, pkgPath);
      if (!id || seen.has(id)) continue;
      seen.add(id);
      types.push({ id, fullType: t, packagePath: pkgPath, label: labelTypeWithPackage(t, pkgPath) });
    }
  }
  return {
    providers: list.map((p) => ({
      id: p.id,
      function_name: p.function_name,
      output_type: p.output_type,
      output_pkg: p.output_pkg || p.function_pkg || "",
      label: providerDisplayLabel(p),
    })),
    types,
  };
}

function providerOutputTypesOf(p = {}) {
  if (p.output_types && p.output_types.length) return p.output_types.filter(Boolean);
  return p.output_type ? [p.output_type] : [];
}

function providerMatchesPkg(p = {}, pkgName = "") {
  const pkg = String(pkgName || "").trim();
  if (!pkg) return true;
  const out = String(p.output_pkg || "").trim();
  const fn = String(p.function_pkg || "").trim();
  if (out === pkg || fn === pkg) return true;
  if (!pkg.includes("/") && (out.endsWith("/" + pkg) || fn.endsWith("/" + pkg))) return true;
  return false;
}

/**
 * Provider-only dependency graph.
 * Edge A→B means A depends on B (A consumes an output type produced by B).
 * Entries (pyramid tip) = providers with indegree 0 (not consumed by anyone).
 *
 * When many packages share a Go package name (handler.Handler in billing vs
 * inventory), OutputType strings collide. Prefer matching producers whose
 * output_pkg equals the consumer's input_pkgs[i] when that is present.
 */
export function buildProviderDependencyGraph(providers = []) {
  const list = Array.isArray(providers) ? providers : [];
  const nodes = list.map((p) => {
    const id = p.id || p.provider_id || p.function_name || p.output_type;
    const label = providerDisplayLabel(p);
    return {
      id,
      label,
      title: (p.output_type || label)
        + (p.function_name ? "\n" + p.function_name : ""),
      shape: "box",
      font: { size: 11 },
      data: {
        ...p,
        type: "provider",
        packagePath: p.output_pkg || p.function_pkg || "",
        displayLabel: label,
      },
    };
  });
  const byId = new Map(nodes.map((n) => [n.id, n]));
  const typeProducers = new Map();
  for (const p of list) {
    const id = p.id || p.provider_id || p.function_name || p.output_type;
    if (!byId.has(id)) continue;
    for (const t of providerOutputTypesOf(p)) {
      if (!typeProducers.has(t)) typeProducers.set(t, new Set());
      typeProducers.get(t).add(id);
    }
  }
  const edgeKeys = new Set();
  const edges = [];
  for (const p of list) {
    const from = p.id || p.provider_id || p.function_name || p.output_type;
    if (!byId.has(from)) continue;
    const inputs = p.input_types || [];
    const inputPkgs = p.input_pkgs || [];
    for (let i = 0; i < inputs.length; i++) {
      const input = inputs[i];
      const wantPkg = String(inputPkgs[i] || "").trim();
      const producers = typeProducers.get(input);
      if (!producers) continue;
      const producerList = [...producers];
      for (const to of producerList) {
        if (!to || to === from) continue;
        if (wantPkg) {
          const prod = byId.get(to);
          const prodPkg = String(
            (prod && prod.data && (prod.data.output_pkg || prod.data.packagePath)) || ""
          ).trim();
          if (prodPkg && prodPkg !== wantPkg) continue;
        }
        const key = from + "->" + to;
        if (edgeKeys.has(key)) continue;
        edgeKeys.add(key);
        edges.push({
          from,
          to,
          arrows: "to",
          color: { color: "#9ca3af" },
        });
      }
    }
  }
  return { nodes, edges };
}

/** Level 1 = unconsumed entries; deeper levels follow depends-on edges. */
export function assignProviderPyramidLevels(nodes = [], edges = []) {
  const ids = nodes.map((n) => n.id);
  const idSet = new Set(ids);
  const outgoing = new Map(ids.map((id) => [id, []]));
  const indeg = new Map(ids.map((id) => [id, 0]));
  for (const e of edges) {
    if (!idSet.has(e.from) || !idSet.has(e.to)) continue;
    outgoing.get(e.from).push(e.to);
    indeg.set(e.to, (indeg.get(e.to) || 0) + 1);
  }
  const levels = new Map();
  const queue = [];
  for (const id of ids) {
    if ((indeg.get(id) || 0) === 0) {
      levels.set(id, 1);
      queue.push(id);
    }
  }
  // Isolated / cycle leftovers: treat remaining as entries if never reached.
  while (queue.length) {
    const id = queue.shift();
    const level = levels.get(id);
    for (const to of outgoing.get(id) || []) {
      if (levels.has(to)) continue;
      levels.set(to, level + 1);
      queue.push(to);
    }
  }
  for (const id of ids) {
    if (!levels.has(id)) levels.set(id, 1);
  }
  return levels;
}

export function truncateProviderPyramid(nodes = [], edges = [], levels = new Map(), maxLevel = 0) {
  const max = Number(maxLevel) || 0;
  if (max <= 0) {
    return {
      nodes: nodes.map((n) => ({ ...n, level: levels.get(n.id) || n.level || 1 })),
      edges: [...edges],
    };
  }
  const keep = new Set(
    nodes.filter((n) => (levels.get(n.id) || 1) <= max).map((n) => n.id)
  );
  return {
    nodes: nodes
      .filter((n) => keep.has(n.id))
      .map((n) => ({ ...n, level: levels.get(n.id) || 1 })),
    edges: edges.filter((e) => keep.has(e.from) && keep.has(e.to)),
  };
}

/**
 * Collect seed node ids plus all descendants following edge from→to
 * (provider/types: consumer→dependency; module map: parent→child containment).
 */
export function collectDownstreamNodeIds(nodes = [], edges = [], seedIds = []) {
  const seeds = [...new Set((seedIds || []).map(String).filter(Boolean))];
  if (!seeds.length) return new Set();
  const idSet = new Set(nodes.map((n) => n.id));
  const down = new Map([...idSet].map((id) => [id, []]));
  for (const e of edges) {
    if (!idSet.has(e.from) || !idSet.has(e.to)) continue;
    down.get(e.from).push(e.to);
  }
  const hide = new Set();
  const queue = [];
  for (const id of seeds) {
    hide.add(id);
    if (idSet.has(id)) queue.push(id);
  }
  while (queue.length) {
    const id = queue.shift();
    for (const to of down.get(id) || []) {
      if (hide.has(to)) continue;
      hide.add(to);
      queue.push(to);
    }
  }
  return hide;
}

/**
 * Expand hide seeds (node ids and/or packagePrefix) to concrete node ids
 * present on the current canvas. Skips `exact` seeds (those hide one id only).
 */
export function expandHiddenSeedsToNodeIds(nodes = [], seeds = []) {
  const ids = [];
  for (const s of seeds || []) {
    if (s == null) continue;
    if (typeof s === "string" || typeof s === "number") {
      ids.push(String(s));
      continue;
    }
    if (s.exact) continue;
    const prefix = String(s.packagePrefix || "").trim();
    if (prefix) {
      for (const n of nodes) {
        const pkg = String(
          (n.data && (n.data.packagePath || n.data.output_pkg || n.data.function_pkg)) || ""
        );
        const nid = String(n.id || "");
        if (
          pkg === prefix
          || pkg.startsWith(prefix + "/")
          || nid === prefix
          || nid.startsWith(prefix + "/")
        ) {
          ids.push(nid);
        }
      }
      continue;
    }
    if (s.id) ids.push(String(s.id));
  }
  return [...new Set(ids)];
}

/** Compact hide seeds for URL `hide=` (pipe-separated; pkg: / exact: prefixes). */
export function encodeHiddenSeedsForUrl(seeds = []) {
  return (seeds || [])
    .map((s) => {
      if (s == null) return "";
      if (typeof s === "string" || typeof s === "number") return String(s);
      const prefix = String(s.packagePrefix || "").trim();
      if (prefix) return "pkg:" + prefix;
      if (s.exact && s.id) return "exact:" + String(s.id);
      return String(s.id || "").trim();
    })
    .filter(Boolean)
    .join("|");
}

/** Parse URL `hide=` back into seed objects (labels are best-effort). */
export function decodeHiddenSeedsFromUrl(param) {
  if (param == null) return [];
  const raw = String(param).trim();
  if (!raw) return [];
  return raw.split("|").map((part) => {
    const token = String(part || "").trim();
    if (!token) return null;
    if (token.startsWith("pkg:")) {
      const pref = token.slice(4).trim();
      if (!pref) return null;
      return {
        id: "pkg:" + pref,
        label: pref.split("/").filter(Boolean).slice(-2).join("/") || pref,
        packagePrefix: pref,
      };
    }
    if (token.startsWith("exact:")) {
      const id = token.slice(6).trim();
      if (!id) return null;
      const short = id.includes("\x00") ? id.split("\x00").pop() : id;
      return { id, label: short, exact: true };
    }
    return { id: token, label: token };
  }).filter(Boolean);
}

/** Remove hidden seeds (ids or package prefixes) and their downstream.
 * Seeds marked `exact: true` hide only that node (invert / keep-neighborhood).
 */
export function applyHiddenNodeSeeds(nodes = [], edges = [], seedIds = []) {
  const raw = seedIds || [];
  if (!raw.length) {
    return { nodes: [...nodes], edges: [...edges], hiddenIds: new Set() };
  }
  const exactIds = [];
  const soft = [];
  for (const s of raw) {
    if (s != null && typeof s === "object" && s.exact && s.id) {
      exactIds.push(String(s.id));
    } else {
      soft.push(s);
    }
  }
  const expanded = soft.length && typeof soft[0] === "object"
    ? expandHiddenSeedsToNodeIds(nodes, soft)
    : soft.map(String).filter(Boolean);
  const hiddenIds = new Set(exactIds);
  if (expanded.length) {
    for (const id of collectDownstreamNodeIds(nodes, edges, expanded)) {
      hiddenIds.add(id);
    }
  }
  if (!hiddenIds.size) {
    return { nodes: [...nodes], edges: [...edges], hiddenIds };
  }
  return {
    nodes: nodes.filter((n) => !hiddenIds.has(n.id)),
    edges: edges.filter((e) => !hiddenIds.has(e.from) && !hiddenIds.has(e.to)),
    hiddenIds,
  };
}

/** Seeds + all ancestors (upstream consumers) + descendants (downstream deps). */
export function providerRelatedSubgraph(nodes = [], edges = [], seedIds = []) {
  const seeds = new Set((seedIds || []).filter(Boolean).map(String));
  if (!seeds.size) return { nodes: [...nodes], edges: [...edges] };
  const idSet = new Set(nodes.map((n) => n.id));
  const down = new Map(nodes.map((n) => [n.id, []]));
  const up = new Map(nodes.map((n) => [n.id, []]));
  for (const e of edges) {
    if (!idSet.has(e.from) || !idSet.has(e.to)) continue;
    down.get(e.from).push(e.to);
    up.get(e.to).push(e.from);
  }
  const keep = new Set();
  const walk = (startIds, adj) => {
    const q = [...startIds];
    for (const id of q) keep.add(id);
    while (q.length) {
      const id = q.shift();
      for (const n of adj.get(id) || []) {
        if (keep.has(n)) continue;
        keep.add(n);
        q.push(n);
      }
    }
  };
  const presentSeeds = [...seeds].filter((id) => idSet.has(id));
  walk(presentSeeds, down);
  walk(presentSeeds, up);
  return {
    nodes: nodes.filter((n) => keep.has(n.id)),
    edges: edges.filter((e) => keep.has(e.from) && keep.has(e.to)),
  };
}

/**
 * Role of each node relative to focus seeds.
 * Edge direction is depends-on (from consumer → dependency):
 * - dependency = downstream (what the seed depends on)
 * - dependent = upstream (who depends on the seed)
 */
export function classifyRelatedRoles(nodes = [], edges = [], seedIds = []) {
  const seeds = new Set((seedIds || []).filter(Boolean).map(String));
  const roles = new Map();
  if (!seeds.size) return roles;
  const idSet = new Set(nodes.map((n) => n.id));
  const down = new Map(nodes.map((n) => [n.id, []]));
  const up = new Map(nodes.map((n) => [n.id, []]));
  for (const e of edges) {
    if (!idSet.has(e.from) || !idSet.has(e.to)) continue;
    down.get(e.from).push(e.to);
    up.get(e.to).push(e.from);
  }
  const walkReachable = (adj) => {
    const reached = new Set();
    const q = [...seeds].filter((id) => idSet.has(id));
    while (q.length) {
      const id = q.shift();
      for (const n of adj.get(id) || []) {
        if (seeds.has(n) || reached.has(n)) continue;
        reached.add(n);
        q.push(n);
      }
    }
    return reached;
  };
  const dependencies = walkReachable(down);
  const dependents = walkReachable(up);
  for (const id of seeds) {
    if (idSet.has(id)) roles.set(id, "seed");
  }
  for (const id of dependencies) {
    roles.set(id, dependents.has(id) ? "both" : "dependency");
  }
  for (const id of dependents) {
    if (roles.has(id) && roles.get(id) !== "seed") {
      if (roles.get(id) === "dependency") roles.set(id, "both");
    } else if (!roles.has(id)) {
      roles.set(id, "dependent");
    }
  }
  return roles;
}

/**
 * Providers canvas pipeline: provider-only pyramid, optional package/provider seeds,
 * then depth truncation by pyramid level (depth<=0 keeps all levels).
 */
export function buildProvidersPyramidView(providers = [], opts = {}) {
  const depth = Number(opts.depth) || 0;
  const packageName = String(opts.packageName || "").trim();
  const seedProviderIds = (opts.seedProviderIds || []).map(String).filter(Boolean);
  let { nodes, edges } = buildProviderDependencyGraph(providers);

  const seeds = new Set(seedProviderIds);
  if (packageName) {
    for (const p of providers) {
      if (providerMatchesPkg(p, packageName)) {
        seeds.add(String(p.id || p.function_name || p.output_type));
      }
    }
  }
  if (seeds.size) {
    const sub = providerRelatedSubgraph(nodes, edges, [...seeds]);
    nodes = sub.nodes;
    edges = sub.edges;
  }

  const levels = assignProviderPyramidLevels(nodes, edges);
  const cut = truncateProviderPyramid(nodes, edges, levels, depth);
  const entries = cut.nodes.filter((n) => (n.level || levels.get(n.id) || 1) === 1).map((n) => n.id);
  return {
    nodes: cut.nodes,
    edges: cut.edges,
    levels,
    entries,
    depth,
  };
}

function typeMatchesPkg(typeName = "", pkgName = "") {
  const pkg = String(pkgName || "").trim();
  if (!pkg) return true;
  const t = String(typeName || "");
  if (!t) return false;
  if (t.includes(pkg)) return true;
  const short = pkg.includes("/") ? pkg.split("/").pop() : pkg;
  return !!(short && (t.includes("/" + short + ".") || t.includes("." + short + ".") || t.endsWith("." + short)));
}

/**
 * Stable type-node identity: Go typ.String() collides across packages
 * (*handler.Handler in billing vs inventory). Prefer pkg\x00type.
 */
export function typeNodeIdentity(typeName = "", pkgPath = "") {
  const t = String(typeName || "").trim();
  if (!t) return "";
  if (t.includes("\x00")) return t;
  const pkg = String(pkgPath || "").trim();
  return pkg ? `${pkg}\x00${t}` : t;
}

/**
 * Type-only dependency graph.
 * Edge A→B means type A depends on type B (A's provider consumes B).
 * Nodes are keyed by package + type so same short names stay distinct.
 */
export function buildTypeDependencyGraph(allData = {}) {
  const providers = allData.providers || [];
  const nodeMap = new Map();
  const ensure = (typeName, pkgHint = "") => {
    const id = typeNodeIdentity(typeName, pkgHint);
    if (!id) return "";
    if (nodeMap.has(id)) {
      const existing = nodeMap.get(id);
      if (pkgHint && !(existing.data && existing.data.packagePath)) {
        existing.data.packagePath = pkgHint;
        existing.label = labelTypeWithPackage(typeName, pkgHint);
        existing.data.displayLabel = existing.label;
        existing.title = pkgHint + "\n" + typeName;
      }
      return id;
    }
    const label = labelTypeWithPackage(typeName, pkgHint);
    nodeMap.set(id, {
      id,
      label,
      title: (pkgHint ? pkgHint + "\n" : "") + typeName,
      shape: "ellipse",
      font: { size: 11 },
      color: { background: "#bfdbfe", border: "#3b82f6" },
      data: {
        type: "type",
        fullType: typeName,
        packagePath: pkgHint || "",
        displayLabel: label,
      },
    });
    return id;
  };
  const edgeKeys = new Set();
  const edges = [];
  const addEdge = (from, to) => {
    if (!from || !to || from === to) return;
    const key = from + "->" + to;
    if (edgeKeys.has(key)) return;
    edgeKeys.add(key);
    edges.push({ from, to, arrows: "to", color: { color: "#9ca3af" } });
  };

  // First pass: register all produced types (authoritative package paths).
  const producersByType = new Map();
  for (const p of providers) {
    const outs = providerOutputTypesOf(p);
    const pkg = String(p.output_pkg || p.function_pkg || "").trim();
    for (const out of outs) {
      const id = ensure(out, pkg);
      if (!id) continue;
      if (!producersByType.has(out)) producersByType.set(out, []);
      producersByType.get(out).push({ id, pkg });
    }
  }

  const resolveInputId = (typeName, hintPkg) => {
    const hint = String(hintPkg || "").trim();
    if (hint) return ensure(typeName, hint);
    const producers = producersByType.get(typeName) || [];
    if (producers.length === 1) return ensure(typeName, producers[0].pkg);
    // Ambiguous or external: keep type-only id (last resort).
    return ensure(typeName, "");
  };

  for (const p of providers) {
    const outs = providerOutputTypesOf(p);
    const pkg = String(p.output_pkg || p.function_pkg || "").trim();
    const outIds = outs.map((out) => ensure(out, pkg)).filter(Boolean);
    const inputs = p.input_types || [];
    const inputPkgs = p.input_pkgs || [];
    for (let i = 0; i < inputs.length; i++) {
      const input = inputs[i];
      const inId = resolveInputId(input, inputPkgs[i]);
      for (const outId of outIds) addEdge(outId, inId);
    }
  }

  return { nodes: [...nodeMap.values()], edges };
}

/**
 * Types canvas pyramid: same depth/package/seed semantics as providers.
 */
export function buildTypesPyramidView(allData = {}, opts = {}) {
  const depth = Number(opts.depth) || 0;
  const packageName = String(opts.packageName || "").trim();
  const seedTypeIds = (opts.seedTypeIds || []).map(String).filter(Boolean);
  let { nodes, edges } = buildTypeDependencyGraph(allData);

  const seeds = new Set(seedTypeIds);
  // Allow unqualified type strings to match all package-qualified nodes.
  if (seeds.size) {
    for (const seed of [...seeds]) {
      if (String(seed).includes("\x00")) continue;
      for (const n of nodes) {
        if (n.id === seed || (n.data && n.data.fullType === seed)) {
          seeds.add(n.id);
        }
      }
    }
  }
  if (packageName) {
    for (const n of nodes) {
      const pkgPath = (n.data && n.data.packagePath) || "";
      if (providerMatchesPkg({ output_pkg: pkgPath }, packageName) || typeMatchesPkg(n.id, packageName)) {
        seeds.add(n.id);
      }
    }
  }
  if (seeds.size) {
    const sub = providerRelatedSubgraph(nodes, edges, [...seeds]);
    nodes = sub.nodes;
    edges = sub.edges;
  }

  const levels = assignProviderPyramidLevels(nodes, edges);
  const cut = truncateProviderPyramid(nodes, edges, levels, depth);
  const entries = cut.nodes.filter((n) => (n.level || levels.get(n.id) || 1) === 1).map((n) => n.id);
  return {
    nodes: cut.nodes,
    edges: cut.edges,
    levels,
    entries,
    depth,
  };
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

export function resolveEffectiveLayout(preferred, nodeCount, opts = {}) {
  const choice = (preferred || "auto").trim() || "auto";
  // Explicit user choice always wins — crowding is handled by scope/budget, not layout hijack.
  if (choice === "physics") return "physics";
  if (choice === "hierarchical") return "hierarchical";
  // auto only: small graphs get hierarchy, denser graphs get physics
  void opts;
  return nodeCount > READABLE_NODE_CAP ? "physics" : "hierarchical";
}

/** Minimum vis scale after fit before we focus a hub instead of showing a microscopic strip. */
export const MIN_READABLE_FIT_SCALE = 0.55;

/** After fit(), decide whether to keep overview or focus a hub for readable labels. */
export function resolvePostFitCamera(scale, opts = {}) {
  const floor = opts.floor ?? MIN_READABLE_FIT_SCALE;
  return Number(scale) < floor ? "focus" : "fit";
}

/**
 * Estimate whether hierarchical UD will sprawl into a thin horizontal strip.
 * Used for density tips (shrink scope), never to silently swap the user's layout.
 */
export function estimateHierarchicalStripRisk(nodes = [], edges = [], opts = {}) {
  const list = Array.isArray(nodes) ? nodes : [];
  const edgeList = Array.isArray(edges) ? edges : [];
  const minRoots = opts.minRoots ?? 6;
  if (list.length < 8) {
    return { wide: false, roots: list.length, components: list.length };
  }
  const ids = new Set(list.map((n) => n.id));
  const indeg = new Map([...ids].map((id) => [id, 0]));
  const parent = new Map([...ids].map((id) => [id, id]));
  const find = (x) => {
    let r = x;
    while (parent.get(r) !== r) r = parent.get(r);
    return r;
  };
  const unite = (a, b) => {
    const ra = find(a);
    const rb = find(b);
    if (ra !== rb) parent.set(ra, rb);
  };
  for (const e of edgeList) {
    if (!ids.has(e.from) || !ids.has(e.to)) continue;
    indeg.set(e.to, (indeg.get(e.to) || 0) + 1);
    unite(e.from, e.to);
  }
  let roots = 0;
  for (const id of ids) {
    if ((indeg.get(id) || 0) === 0) roots += 1;
  }
  const comps = new Set([...ids].map((id) => find(id))).size;
  const wide = roots >= minRoots
    || comps >= minRoots
    || (list.length >= 12 && roots >= Math.ceil(list.length * 0.35));
  return { wide, roots, components: comps };
}

/** Architecture views prefer fit so the whole slice is visible; label LOD keeps hubs readable when zoomed out. */
export function resolveCameraStrategy(mode, nodeCount) {
  if (mode === "modules") return "fit";
  if (nodeCount <= 20) return "fit";
  return "fit";
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

/** Longest common path-prefix length among module names (by segment). */
function commonPathPrefixLen(paths = []) {
  if (!paths.length) return 0;
  const split = paths.map((p) => String(p).split("/").filter(Boolean));
  const first = split[0] || [];
  let n = first.length;
  for (const parts of split.slice(1)) {
    let i = 0;
    while (i < n && i < parts.length && parts[i] === first[i]) i += 1;
    n = i;
  }
  // Keep at least one shared root segment visible when possible, but never eat the whole path.
  return Math.max(0, Math.min(n - 1, n));
}

/** Coarse bucket under a fat Go package — keep module map readable. */
export function providerPackageBucket(provider = {}) {
  const out = String(provider.output_type || "");
  if (/Worker\[/.test(out)) return "workers";
  if (/Plugin\[/.test(out)) return "plugins";
  if (/\.(Application|PluginPlatform)\b/.test(out) || /Application|PluginPlatform/.test(out)) return "app";
  if (/Timeout|SlowRemote|Startup|Logger|ConsoleLogger/.test(out)) return "diag";
  return "other";
}

/** Type-based path suffix used only when explicitly requesting fine split. */
export function providerPackageSuffix(provider = {}) {
  return providerPackageBucket(provider);
}

/**
 * Spread fat leaf modules into a few coarse child buckets (app/plugins/workers/…).
 * Avoids exploding the module map into hundreds of per-type micro-leaves.
 */
export function disperseModulesByProviders(modules = [], providers = [], opts = {}) {
  const fatThreshold = opts.fatThreshold ?? 12;
  const list = Array.isArray(modules) ? modules : [];
  if (!providers.length) return list;

  const byPkg = new Map();
  for (const p of providers) {
    const pkg = String(p.output_pkg || p.function_pkg || "").trim();
    if (!pkg) continue;
    if (!byPkg.has(pkg)) byPkg.set(pkg, []);
    byPkg.get(pkg).push(p);
  }

  const out = [];
  for (const m of list) {
    const name = String(m.name || "");
    const bucket = byPkg.get(name) || [];
    const shouldSplit = bucket.length >= fatThreshold;
    if (!shouldSplit) {
      out.push({ ...m });
      continue;
    }
    const groups = new Map();
    for (const p of bucket) {
      const suffix = providerPackageBucket(p);
      const child = `${name}/${suffix}`;
      if (!groups.has(child)) {
        groups.set(child, {
          name: child,
          provider_count: 0,
          object_count: 0,
          type_count: 0,
          depends_on: [],
          parent: name,
          scopePkg: name,
          virtual: true,
        });
      }
      const g = groups.get(child);
      g.provider_count += 1;
      g.type_count += 1;
    }
    // Keep cross-package depends once on the package root via a synthetic rollup
    // node edge elsewhere — do NOT copy depends_on onto every bucket (edge bomb).
    for (const g of groups.values()) out.push(g);
  }
  return out;
}

/**
 * Expand module paths into a parent/child package tree and assign pyramid levels.
 * Cross-package depends_on edges are kept between leaves; containment edges link parents→children.
 */
export function expandPackageHierarchy(modules = [], opts = {}) {
  const minDepth = opts.minDepth ?? 5;
  const list = Array.isArray(modules) ? modules.map((m) => ({ ...m })) : [];
  const names = list.map((m) => m.name).filter(Boolean);
  const strip = opts.stripSegments ?? commonPathPrefixLen(names);

  const norm = (name) => {
    const parts = String(name).split("/").filter(Boolean);
    const sliced = parts.slice(strip);
    return sliced.join("/") || String(name);
  };

  const nodes = new Map();
  const ensure = (path, seed = {}) => {
    if (!path) return null;
    if (!nodes.has(path)) {
      const level = path.split("/").filter(Boolean).length;
      nodes.set(path, {
        name: path,
        provider_count: 0,
        object_count: 0,
        type_count: 0,
        depends_on: [],
        leaf: false,
        level,
        ...seed,
      });
    }
    return nodes.get(path);
  };

  const leafMap = new Map(); // original/full name → normalized leaf path
  for (const m of list) {
    const path = norm(m.name);
    if (!path) continue;
    leafMap.set(m.name, path);
    const segs = path.split("/").filter(Boolean);
    for (let i = 1; i <= segs.length; i++) {
      ensure(segs.slice(0, i).join("/"));
    }
    const leaf = ensure(path);
    leaf.leaf = true;
    leaf.provider_count += m.provider_count || 0;
    leaf.object_count += m.object_count || 0;
    leaf.type_count += m.type_count || 0;
    leaf.fullName = m.name;
    leaf.scopePkg = m.scopePkg || m.parent || m.name;
    leaf.virtual = !!m.virtual;
  }

  // Roll up counts once from each leaf to its ancestors (ancestors are never leaves).
  for (const leaf of [...nodes.values()].filter((n) => n.leaf)) {
    const segs = leaf.name.split("/").filter(Boolean);
    for (let i = segs.length - 1; i >= 1; i--) {
      const parent = segs.slice(0, i).join("/");
      const p = nodes.get(parent);
      if (!p || p.leaf) continue;
      p.provider_count += leaf.provider_count;
      p.object_count += leaf.object_count;
      p.type_count += leaf.type_count;
    }
  }

  // Containment edges parent → child
  const containment = [];
  for (const path of nodes.keys()) {
    const segs = path.split("/").filter(Boolean);
    if (segs.length < 2) continue;
    const parent = segs.slice(0, -1).join("/");
    containment.push({ from: parent, to: path });
  }

  // Map depends_on onto normalized leaf paths
  const depEdges = [];
  for (const m of list) {
    const from = leafMap.get(m.name);
    for (const dep of m.depends_on || []) {
      const to = leafMap.get(dep) || norm(dep);
      if (!from || !to || from === to) continue;
      ensure(to);
      depEdges.push({ from, to, kind: "depends" });
    }
  }

  void minDepth;
  return {
    modules: [...nodes.values()].sort((a, b) => a.name.localeCompare(b.name)),
    containment,
    depEdges,
    strip,
  };
}

/** Build package/module architecture graph; object counts are label text only. */
export function buildModuleMapGraph(modules = [], opts = {}) {
  const hierarchy = opts.hierarchy !== false;
  if (!hierarchy) {
    const nodes = modules.map((m) => ({
      id: m.name,
      label: `${shortGraphLabel(m.name)}\n(${m.provider_count || 0}p/${m.object_count || 0}o)`,
      title: m.name,
      shape: "box",
      font: { size: 12 },
      color: { background: "#eef0ff", border: "#4f5ce5" },
      data: { type: "module", module: m, packagePath: m.name },
    }));
    const edges = [];
    for (const m of modules) {
      for (const dep of m.depends_on || []) {
        edges.push({ from: m.name, to: dep, arrows: "to", color: { color: "#9ca3af" } });
      }
    }
    return { nodes, edges };
  }

  const dispersed = disperseModulesByProviders(modules, opts.providers || [], {
    fatThreshold: opts.fatThreshold ?? 12,
  });
  const expanded = expandPackageHierarchy(dispersed, {
    minDepth: opts.minDepth ?? 5,
    stripSegments: opts.stripSegments,
  });

  const nodes = expanded.modules.map((m) => {
    const labelName = m.name.split("/").filter(Boolean).slice(-2).join("/") || m.name;
    return {
      id: m.name,
      label: `${labelName}\n(${m.provider_count || 0}p/${m.object_count || 0}o)`,
      title: (m.fullName || m.name) + (m.virtual ? " (bucket)" : ""),
      shape: "box",
      font: { size: m.leaf ? 12 : 11 },
      level: m.level || m.name.split("/").filter(Boolean).length,
      color: m.leaf
        ? { background: "#eef0ff", border: "#4f5ce5" }
        : { background: "#f8fafc", border: "#94a3b8" },
      data: {
        type: "module",
        module: { ...m, name: m.scopePkg || m.fullName || m.name },
        packagePath: m.scopePkg || m.fullName || m.name,
        leaf: !!m.leaf,
        provider_count: m.provider_count || 0,
        displayLabel: labelName,
      },
    };
  });

  // Module map layout must stay a package TREE. Only containment edges —
  // cross-package depends_on fan-out turns the canvas into an unreadable clump.
  const edges = [];
  const seen = new Set();
  for (const e of expanded.containment) {
    const key = e.from + "=>" + e.to;
    if (seen.has(key)) continue;
    seen.add(key);
    edges.push({
      from: e.from,
      to: e.to,
      arrows: "to",
      color: { color: "#94a3b8" },
      dashes: true,
      data: { kind: "containment" },
    });
  }
  return { nodes, edges };
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

/** Package path used for cross-boundary edge scoring. */
export function nodePackageKey(node = {}) {
  const d = node.data || {};
  const raw = d.packagePath || d.pkg || d.group || "";
  if (raw) return String(raw);
  const id = String(node.id || "");
  if (id.includes("/")) {
    const lastDot = id.lastIndexOf(".");
    const lastSlash = id.lastIndexOf("/");
    if (lastDot > lastSlash) return id.slice(0, lastDot).replace(/^\*/, "");
  }
  const short = shortGraphLabel(id);
  const dot = short.lastIndexOf(".");
  return dot > 0 ? short.slice(0, dot) : "";
}

/** Coarse architecture bucket from a Go package path (domain/plugins/infra/…). */
export function architecturePathBucket(pkgPath = "") {
  const parts = String(pkgPath || "").split("/").filter(Boolean);
  for (const key of ["domain", "plugins", "infra", "app", "bootstrap", "router"]) {
    if (parts.includes(key)) return key;
  }
  return "other";
}

/** Domain name under …/domain/<name>/… ; empty if not a domain path. */
export function architectureDomainName(pkgPath = "") {
  const parts = String(pkgPath || "").split("/").filter(Boolean);
  const i = parts.indexOf("domain");
  if (i < 0 || i + 1 >= parts.length) return "";
  return parts[i + 1];
}

export const FINDINGS_HUB_DEGREE = 12;
export const FINDINGS_ENTRY_FANOUT = 8;
export const FINDINGS_FAT_PACKAGE = 12;
export const FINDINGS_MAX = 30;

const SEVERITY_RANK = { error: 0, warn: 1, info: 2 };

/**
 * Static architecture findings from provider graph (no runtime /api/issues).
 * Returns actionable items with view/package/focus hints for the UI.
 */
export function buildArchitectureFindings(allData = {}, opts = {}) {
  const providers = (allData && allData.providers) || [];
  if (!providers.length) return [];

  const hubDegree = opts.hubDegree ?? FINDINGS_HUB_DEGREE;
  const entryFanout = opts.entryFanout ?? FINDINGS_ENTRY_FANOUT;
  const fatPackage = opts.fatPackage ?? FINDINGS_FAT_PACKAGE;
  const maxFindings = opts.max ?? FINDINGS_MAX;

  const { nodes, edges } = buildProviderDependencyGraph(providers);
  const byId = new Map(nodes.map((n) => [n.id, n]));
  const findings = [];

  // --- cross_bucket (aggregate by smell key) ---
  const crossGroups = new Map();
  for (const e of edges) {
    const from = byId.get(e.from);
    const to = byId.get(e.to);
    if (!from || !to) continue;
    const fromPkg = nodePackageKey(from);
    const toPkg = nodePackageKey(to);
    const fb = architecturePathBucket(fromPkg);
    const tb = architecturePathBucket(toPkg);

    let smellKey = "";
    let title = "";
    let summary = "";
    if ((fb === "plugins" || fb === "infra") && tb === "domain") {
      smellKey = `cross:${fb}->domain`;
      title = `${fb} 依赖 domain 实现`;
      summary = `存在 ${fb} → domain 的依赖边，插件/基础设施可能耦合了业务实现。`;
    } else {
      const fd = architectureDomainName(fromPkg);
      const td = architectureDomainName(toPkg);
      if (fb === "domain" && tb === "domain" && fd && td && fd !== td) {
        smellKey = `cross:domain:${fd}->${td}`;
        title = `跨域依赖 ${fd} → ${td}`;
        summary = `domain/${fd} 直接依赖 domain/${td}，边界可能需要通过接口或共享内核解耦。`;
      }
    }
    if (!smellKey) continue;

    let g = crossGroups.get(smellKey);
    if (!g) {
      g = {
        id: smellKey,
        kind: "cross_bucket",
        severity: "warn",
        title,
        summary,
        packages: new Set(),
        providerIds: [],
        edgeCount: 0,
        seedId: e.from,
        seedPkg: fromPkg,
      };
      crossGroups.set(smellKey, g);
    }
    g.edgeCount += 1;
    if (fromPkg) g.packages.add(fromPkg);
    if (toPkg) g.packages.add(toPkg);
    if (e.from && !g.providerIds.includes(e.from)) g.providerIds.push(e.from);
    if (e.to && !g.providerIds.includes(e.to)) g.providerIds.push(e.to);
  }
                for (const g of crossGroups.values()) {
    const pkgs = [...g.packages];
    findings.push({
      id: g.id,
      kind: g.kind,
      severity: g.severity,
      title: g.title,
      summary: g.summary + `（${g.edgeCount} 边）`,
      evidence: {
        packages: pkgs,
        providerIds: g.providerIds,
        typeIds: [],
        edgeCount: g.edgeCount,
      },
      action: {
        view: "providers",
        package: "",
        focusProviderId: g.seedId,
        keepNeighborhood: true,
      },
    });
  }

  // --- super_hub ---
  for (const h of rankHubNodes(nodes, edges, 25)) {
    if (h.degree < hubDegree) continue;
    const node = byId.get(h.id);
    if (!node) continue;
    const pkg = nodePackageKey(node);
    const label = (node.data && node.data.displayLabel) || node.label || h.id;
    findings.push({
      id: "hub:" + h.id,
      kind: "super_hub",
      severity: "warn",
      title: `超级枢纽 · ${label}`,
      summary: `该节点度数 ${h.degree}（阈值 ${hubDegree}），上下游过密，建议拆分或引入门面。`,
      evidence: {
        packages: pkg ? [pkg] : [],
        providerIds: [h.id],
        typeIds: [],
        edgeCount: h.degree,
      },
      action: {
        view: "providers",
        package: "",
        focusProviderId: h.id,
        keepNeighborhood: true,
      },
    });
  }

  // --- entry_fanout ---
  const idSet = new Set(nodes.map((n) => n.id));
  const indeg = new Map([...idSet].map((id) => [id, 0]));
  for (const e of edges) {
    if (!idSet.has(e.from) || !idSet.has(e.to)) continue;
    indeg.set(e.to, (indeg.get(e.to) || 0) + 1);
  }
  const entries = nodes.filter((n) => (indeg.get(n.id) || 0) === 0);
  if (entries.length >= entryFanout) {
    findings.push({
      id: "entry_fanout",
      kind: "entry_fanout",
      severity: "info",
      title: `入口过多 · ${entries.length} 个`,
      summary: `未被子依赖消费的入口 provider 有 ${entries.length} 个（阈值 ${entryFanout}）。可收敛为业务入口或改用模块地图。`,
      evidence: {
        packages: [],
        providerIds: entries.slice(0, 20).map((n) => n.id),
        typeIds: [],
        edgeCount: 0,
      },
      action: {
        view: "modules",
        keepNeighborhood: false,
      },
    });
  }

  // --- fat_package ---
  const pkgCounts = new Map();
  for (const p of providers) {
    const pkg = String(p.output_pkg || p.function_pkg || "").trim();
    if (!pkg) continue;
    if (!pkgCounts.has(pkg)) pkgCounts.set(pkg, []);
    const id = p.id || p.provider_id || p.function_name || p.output_type;
    pkgCounts.get(pkg).push(id);
  }
  for (const [pkg, ids] of pkgCounts.entries()) {
    if (ids.length < fatPackage) continue;
    const leaf = pkg.split("/").filter(Boolean).slice(-2).join("/") || pkg;
    findings.push({
      id: "fat:" + pkg,
      kind: "fat_package",
      severity: "info",
      title: `包过肥 · ${leaf}`,
      summary: `${pkg} 含 ${ids.length} 个 provider（阈值 ${fatPackage}），建议按职责拆分子包。`,
      evidence: {
        packages: [pkg],
        providerIds: ids.slice(0, 30),
        typeIds: [],
        edgeCount: 0,
      },
      action: {
        view: "providers",
        package: pkg,
        keepNeighborhood: false,
      },
    });
  }

  findings.sort((a, b) => {
    const sa = SEVERITY_RANK[a.severity] ?? 9;
    const sb = SEVERITY_RANK[b.severity] ?? 9;
    if (sa !== sb) return sa - sb;
    const ea = (a.evidence && a.evidence.edgeCount) || (a.evidence && a.evidence.providerIds && a.evidence.providerIds.length) || 0;
    const eb = (b.evidence && b.evidence.edgeCount) || (b.evidence && b.evidence.providerIds && b.evidence.providerIds.length) || 0;
    if (eb !== ea) return eb - ea;
    return String(a.id).localeCompare(String(b.id));
  });

  return findings.slice(0, maxFindings);
}

/**
 * Prefer cross-package and hub-touching edges when the canvas is too dense.
 * Same-package peripheral edges are dropped first.
 * Never leave a previously-connected node with degree 0 (no orphan isolates).
 */
export function declutterEdges(nodes = [], edges = [], opts = {}) {
  const maxEdges = opts.maxEdges ?? 80;
  if (!edges.length || edges.length <= maxEdges) {
    return { edges, decluttered: false, dropped: 0 };
  }
  const byId = new Map(nodes.map((n) => [n.id, n]));
  const hubs = new Set(rankHubNodes(nodes, edges, opts.hubLimit ?? 12).map((h) => h.id));
  const scored = edges.map((e) => {
    const from = byId.get(e.from);
    const to = byId.get(e.to);
    const fp = nodePackageKey(from);
    const tp = nodePackageKey(to);
    const cross = !!(fp && tp && fp !== tp);
    const hubTouch = hubs.has(e.from) || hubs.has(e.to);
    let score = 0;
    if (cross) score += 4;
    if (hubTouch) score += 3;
    if (e.dashes) score += 1;
    return { e, score };
  });
  scored.sort((a, b) => b.score - a.score || String(a.e.from).localeCompare(String(b.e.from)));
  const kept = [];
  const keptKeys = new Set();
  const degree = new Map(nodes.map((n) => [n.id, 0]));
  const pushEdge = (e) => {
    const key = e.from + "->" + e.to;
    if (keptKeys.has(key)) return;
    keptKeys.add(key);
    kept.push(e);
    degree.set(e.from, (degree.get(e.from) || 0) + 1);
    degree.set(e.to, (degree.get(e.to) || 0) + 1);
  };
  for (const s of scored) {
    if (kept.length >= maxEdges) break;
    pushEdge(s.e);
  }
  // Restore one edge for any node that had connectivity but became isolated.
  const originallyConnected = new Set();
  for (const e of edges) {
    originallyConnected.add(e.from);
    originallyConnected.add(e.to);
  }
  for (const n of nodes) {
    if (!originallyConnected.has(n.id)) continue;
    if ((degree.get(n.id) || 0) > 0) continue;
    const candidate = scored.find((s) => s.e.from === n.id || s.e.to === n.id);
    if (candidate) pushEdge(candidate.e);
  }
  return {
    edges: kept,
    decluttered: true,
    dropped: Math.max(0, edges.length - kept.length),
  };
}

/** Build a package-scoped inventory for the summary card (not canvas nodes). */
export function buildPackageSummary(allData = {}, pkgName = "") {
  const pkg = String(pkgName || "").trim();
  if (!pkg || !allData) return { providers: [], types: [] };
  const list = (allData.providers || []).filter((p) => {
    const out = p.output_pkg || "";
    const fn = p.function_pkg || "";
    if (out === pkg || fn === pkg) return true;
    if (!pkg.includes("/") && (out.endsWith("/" + pkg) || fn.endsWith("/" + pkg))) return true;
    return false;
  });
  const types = [];
  const seen = new Set();
  for (const p of list) {
    const pkgPath = p.output_pkg || p.function_pkg || "";
    for (const t of (p.output_types && p.output_types.length ? p.output_types : [p.output_type])) {
      if (!t) continue;
      const id = typeNodeIdentity(t, pkgPath);
      if (!id || seen.has(id)) continue;
      seen.add(id);
      types.push({ id, fullType: t, packagePath: pkgPath, label: labelTypeWithPackage(t, pkgPath) });
    }
  }
  return {
    providers: list.map((p) => ({
      id: p.id,
      function_name: p.function_name,
      output_type: p.output_type,
      output_pkg: p.output_pkg || p.function_pkg || "",
      label: providerDisplayLabel(p),
    })),
    types,
  };
}

if (typeof window !== "undefined") {
  window.DIXGraphState = {
    resolveGraphMode, applyGraphBudget, READABLE_NODE_CAP, shortGraphLabel, labelTypeWithPackage, providerDisplayLabel, labelLodVisibleIds,
    providerSymbolLabel, dixNamespaceLeaf,
    HIERARCHICAL_NODE_BUDGET, HIERARCHICAL_EDGE_BUDGET,
    resolveEffectiveLayout, resolveCameraStrategy, layoutStarPositions, assessLayoutMetrics, pickFocusNodeId,
    estimateHierarchicalStripRisk, resolvePostFitCamera, MIN_READABLE_FIT_SCALE,
    buildProviderInventory, buildProviderDependencyGraph, assignProviderPyramidLevels,
    truncateProviderPyramid, providerRelatedSubgraph, classifyRelatedRoles, buildProvidersPyramidView,
    buildTypeDependencyGraph, buildTypesPyramidView,
    collectDownstreamNodeIds, expandHiddenSeedsToNodeIds, applyHiddenNodeSeeds,
    encodeHiddenSeedsForUrl, decodeHiddenSeedsFromUrl,
    typeNodeIdentity, createLoadGuard, rankHubNodes, buildModuleMapGraph, disperseModulesByProviders,
    expandPackageHierarchy, providerPackageSuffix, providerPackageBucket, nodePackageKey, declutterEdges, buildPackageSummary,
    architecturePathBucket, architectureDomainName, buildArchitectureFindings,
    FINDINGS_HUB_DEGREE, FINDINGS_ENTRY_FANOUT, FINDINGS_FAT_PACKAGE, FINDINGS_MAX,
    issueGraphHash, issueTraceHash, matchTraceRecord, filterTraceRecords,
  };
}
