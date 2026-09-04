// 依赖图:四种模式——邻域子图 / 全局·Providers / 全局·类型依赖 / 模块图;
// 任意节点点击 → 右侧详情抽屉(信息 + 运行时状态 + 快捷操作)。
window.DIX = window.DIX || {};
DIX.views = DIX.views || {};

(function () {
  const state = {
    mode: "ego", layout: "hierarchical", depth: 0, prefix: "",
    allData: null, runtimeStats: null, focus: null, network: null,
  };

  const pkgOf = s => (s || "").split(".").slice(0, -1).join(".") || "(anonymous)";
  const shortType = t => (t || "").replace(/^\*/, "").split(".").pop();

  function toolbarHTML(query) {
    return `
      <form class="filters" id="g-form">
        <select id="g-mode">
          <option value="ego">邻域子图</option>
          <option value="providers">全局 · Providers</option>
          <option value="types">全局 · 类型依赖</option>
          <option value="modules">模块图</option>
        </select>
        <span id="g-ego-controls" style="display:inline-flex;gap:8px">
          <input type="text" id="g-center" placeholder="中心类型" value="${DIX.esc(query.get("center") || "")}" style="min-width:260px">
          <select id="g-depth-ego"><option value="1">1 跳</option><option value="2" selected>2 跳</option><option value="3">3 跳</option><option value="4">4 跳</option><option value="5">5 跳</option></select>
          <select id="g-dir"><option value="both">双向</option><option value="deps">依赖</option><option value="dependents">被依赖</option></select>
        </span>
        <select id="g-layout"><option value="hierarchical">层级布局</option><option value="physics">物理布局</option></select>
        <select id="g-depth-all"><option value="0">深度:全部</option><option value="1">1 跳</option><option value="2">2 跳</option><option value="3">3 跳</option><option value="5">5 跳</option></select>
        <input type="text" id="g-prefix" placeholder="前缀过滤(包/类型)" value="${DIX.esc(query.get("prefix") || "")}" style="min-width:180px">
        <button class="btn" type="submit">绘制</button>
        <button class="btn ghost" type="button" id="g-reset">重置</button>
      </form>
      <div class="grid" style="grid-template-columns: 1fr 320px; align-items:start">
        <div id="graph-canvas"></div>
        <div class="card" id="g-detail"><h3>节点详情</h3><div class="muted">点击图中节点查看详情。</div></div>
      </div>
      <p class="muted" style="margin-top:10px">全局图:绿色实线=产物,黄色虚线=依赖;box=provider,椭圆=类型。单击节点看详情,双击类型节点聚焦。</p>`;
  }

  async function loadData() {
    if (!state.allData) state.allData = await DIX.get("/api/dependencies");
    if (!state.runtimeStats) {
      try { state.runtimeStats = await DIX.get("/api/runtime-stats", { limit: 500 }); }
      catch { state.runtimeStats = []; }
    }
  }

  function statFor(fnName) {
    return (state.runtimeStats || []).find(s => s.function_name === fnName) || null;
  }

  // ---------- 全局图构建(移植自旧版 renderGraph) ----------
  function buildGlobal() {
    if (!state.allData) return { nodes: [], edges: [] };
    const providers = state.allData.providers || [];
    const edges = state.allData.edges || [];
    const nodes = [], gedges = [];
    const nodeMap = new Map();
    const addType = label => {
      if (nodeMap.has(label)) return;
      nodeMap.set(label, true);
      nodes.push({
        id: label, label: shortType(label), shape: "ellipse", font: { size: 13 },
        color: { background: "#bfdbfe", border: "#3b82f6" },
        data: { kind: "type", label, pkg: pkgOf(label) },
      });
    };
    if (state.mode === "providers") {
      providers.forEach(p => {
        if (!nodeMap.has(p.id)) {
          nodeMap.set(p.id, true);
          const st = statFor(p.function_name);
          const isErr = !!(st && st.last_error);
          nodes.push({
            id: p.id, label: shortType(p.output_type || p.function_name),
            shape: "box", font: { size: 13 },
            color: { background: isErr ? "#fdeaea" : "#eef0ff", border: isErr ? "#dc2626" : "#4f5ce5" },
            data: { kind: "provider", label: p.id, provider: p, pkg: p.output_pkg || pkgOf(p.output_type) },
          });
        }
        (p.output_types && p.output_types.length ? p.output_types : [p.output_type]).forEach(ot => {
          if (!ot) return;
          addType(ot);
          gedges.push({ from: p.id, to: ot, color: { color: "#22c55e" }, arrows: "to" });
        });
        (p.input_types || []).forEach(it => {
          if (!it) return;
          addType(it);
          gedges.push({ from: it, to: p.id, dashes: true, color: { color: "#f59e0b" }, arrows: "to" });
        });
      });
    } else {
      edges.forEach(e => {
        if (e.type !== "provider") return;
        addType(e.from);
        addType(e.to);
        gedges.push({ from: e.from, to: e.to, color: { color: "#9ca3af" }, arrows: "to" });
      });
    }
    return { nodes, edges: gedges };
  }

  function buildModules() {
    return DIX.get("/api/modules").then(modules => {
      const nodes = [], edges = [];
      modules.forEach(m => {
        nodes.push({
          id: m.name, label: shortType(m.name) + "\n(" + m.provider_count + "p/" + m.object_count + "o)",
          shape: "box", font: { size: 12 }, color: { background: "#eef0ff", border: "#4f5ce5" },
          data: { kind: "module", label: m.name, module: m },
        });
        (m.depends_on || []).forEach(dep => edges.push({ from: m.name, to: dep, arrows: "to", color: { color: "#9ca3af" } }));
      });
      return { nodes, edges };
    });
  }

  // ---------- 深度裁剪 + 前缀过滤 ----------
  function prune(nodes, edges) {
    let ns = nodes, es = edges;
    const prefix = state.prefix.trim();
    if (prefix) {
      const keep = new Set(ns.filter(n => ((n.data && n.data.pkg) || n.id || "").includes(prefix)).map(n => n.id));
      es = es.filter(e => keep.has(e.from) && keep.has(e.to));
      ns = ns.filter(n => keep.has(n.id));
    }
    if (state.depth > 0) {
      const focus = state.focus || highestDegreeNode(ns, es);
      if (focus) {
        const adj = new Map();
        es.forEach(e => {
          if (!adj.has(e.from)) adj.set(e.from, []);
          if (!adj.has(e.to)) adj.set(e.to, []);
          adj.get(e.from).push(e.to);
          adj.get(e.to).push(e.from);
        });
        const keep = new Set([focus]);
        let frontier = [focus];
        for (let d = 0; d < state.depth; d++) {
          const next = [];
          frontier.forEach(id => (adj.get(id) || []).forEach(t => { if (!keep.has(t)) { keep.add(t); next.push(t); } }));
          frontier = next;
        }
        ns = ns.filter(n => keep.has(n.id));
        es = es.filter(e => keep.has(e.from) && keep.has(e.to));
      }
    }
    return { nodes: ns, edges: es };
  }

  function highestDegreeNode(nodes, edges) {
    const deg = new Map();
    edges.forEach(e => {
      deg.set(e.from, (deg.get(e.from) || 0) + 1);
      deg.set(e.to, (deg.get(e.to) || 0) + 1);
    });
    let best = null, bestDeg = -1;
    nodes.forEach(n => {
      const d = deg.get(n.id) || 0;
      if (d > bestDeg) { bestDeg = d; best = n.id; }
    });
    return best;
  }

  function renderNetwork(container, nodes, edges) {
    if (state.network) state.network.destroy();
    const hierarchical = state.layout === "hierarchical";
    state.network = new vis.Network(container, {
      nodes: new vis.DataSet(nodes), edges: new vis.DataSet(edges),
    }, {
      layout: hierarchical
        ? { hierarchical: { direction: "LR", sortMethod: "hubsize" }, improvedLayout: true }
        : { improvedLayout: true },
      edges: { smooth: hierarchical ? { type: "cubicBezier", roundness: 0.4 } : false },
      physics: { enabled: !hierarchical, stabilization: { iterations: 150 } },
      interaction: { hover: true },
    });
    state.network.once("afterDrawing", () => {
      state.network.fit({ padding: 30 });
      if (state.network.getScale() < 0.9) state.network.moveTo({ scale: 0.9, offset: { x: 0, y: 0 } });
    });
    state.network.on("click", params => {
      const id = params.nodes && params.nodes[0];
      const node = id !== undefined ? nodes.find(n => n.id === id) : null;
      showDetail(node || null);
    });
    state.network.on("doubleClick", params => {
      const id = params.nodes && params.nodes[0];
      const node = id !== undefined ? nodes.find(n => n.id === id) : null;
      if (node && node.data && node.data.kind === "type") {
        state.focus = node.data.label;
        state.mode = "types";
        document.getElementById("g-mode").value = "types";
        redraw();
      }
    });
  }

  // ---------- 详情抽屉 ----------
  function showDetail(node) {
    const box = document.getElementById("g-detail");
    if (!box) return;
    if (!node) { box.innerHTML = '<h3>节点详情</h3><div class="muted">点击图中节点查看详情。</div>'; return; }
    const d = node.data || {};
    if (d.kind === "provider") return providerDetail(box, d);
    if (d.kind === "type") return typeDetail(box, d);
    if (d.kind === "module") return moduleDetail(box, d);
    box.innerHTML = '<h3>' + DIX.esc(node.label || "") + '</h3>';
  }

  function providerDetail(box, d) {
    const p = d.provider || {};
    const st = statFor(p.function_name);
    box.innerHTML = `
      <h3>Provider</h3>
      <p class="mono">${DIX.esc(p.function_name || d.label)}</p>
      <p class="muted mono">${DIX.esc(p.function_pkg || "")}${p.function_file ? " @ " + DIX.esc(p.function_file.split("/").pop()) + ":" + (p.function_line || "") : ""}</p>
      <p><b>输出:</b> ${(p.output_types && p.output_types.length ? p.output_types : [p.output_type]).filter(Boolean).map(t => '<span class="chip mono">' + DIX.esc(t) + '</span>').join("")}</p>
      <p><b>输入:</b> ${(p.input_types || []).map(t => '<span class="chip mono">' + DIX.esc(t) + '</span>').join("") || '<span class="muted">无</span>'}</p>
      ${st ? '<p><b>执行:</b> ' + st.call_count + ' 次,平均 ' + (st.average_duration / 1e6).toFixed(2) + ' ms' + (st.last_error ? ' <span class="err-text">最近错误: ' + DIX.esc(st.last_error) + '</span>' : '') + '</p>' : '<p class="muted">尚未执行。</p>'}
      <p><button class="btn ghost" id="d-locate">在全局图中定位</button></p>`;
    document.getElementById("d-locate").onclick = () => {
      state.mode = "providers";
      document.getElementById("g-mode").value = "providers";
      state.focus = p.output_type || d.label;
      redraw();
    };
  }

  function typeDetail(box, d) {
    const label = d.label;
    const producers = (state.allData.providers || []).filter(p =>
      (p.output_types || []).includes(label) || p.output_type === label);
    const objects = (state.allData.objects || []).filter(o => o.type === label);
    box.innerHTML = `
      <h3>类型</h3>
      <p class="mono">${DIX.esc(label)}</p>
      <p class="muted mono">${DIX.esc(d.pkg || "")}</p>
      <p><b>状态:</b> ${objects.length ? '<span class="badge ok">已实例化</span>' : '<span class="badge">未实例化</span>'}</p>
      <p><b>生产者:</b></p>
      <div>${producers.map(p => '<div class="chip mono click" data-fn="' + DIX.esc(p.function_name) + '">' + DIX.esc(shortType(p.function_name)) + '</div>').join("") || '<span class="muted">无(外部类型或缺失 provider)</span>'}</div>
      <p><b>对象:</b> ${objects.length ? objects.map(o => '<span class="chip mono">' + DIX.esc(o.group) + (o.is_initialized ? " ✓" : "") + '</span>').join("") : '<span class="muted">无</span>'}</p>
      <p>
        <button class="btn ghost" id="d-ego">以此为中心(邻域)</button>
        <button class="btn ghost" id="d-focus">聚焦全图(类型依赖)</button>
      </p>`;

    box.querySelectorAll(".chip.click").forEach(chip => {
      chip.addEventListener("click", () => {
        const st2 = statFor(chip.dataset.fn);
        if (st2) {
          box.insertAdjacentHTML("beforeend",
            '<p class="mono">' + DIX.esc(st2.function_name) + '</p><p class="muted">执行 ' + st2.call_count +
            ' 次,平均 ' + (st2.average_duration / 1e6).toFixed(2) + ' ms' +
            (st2.last_error ? ',最近错误: ' + DIX.esc(st2.last_error) : '') + '</p>');
        }
      });
    });
    document.getElementById("d-ego").onclick = () => {
      state.mode = "ego";
      document.getElementById("g-mode").value = "ego";
      document.getElementById("g-center").value = label;
      redraw();
    };
    document.getElementById("d-focus").onclick = () => {
      state.mode = "types";
      document.getElementById("g-mode").value = "types";
      state.focus = label;
      redraw();
    };
  }

  function moduleDetail(box, d) {
    const m = d.module || {};
    box.innerHTML = `
      <h3>模块</h3>
      <p class="mono">${DIX.esc(m.name)}</p>
      <p>类型 ${m.type_count} · provider ${m.provider_count} · 对象 ${m.object_count}</p>
      <p><b>依赖模块:</b> ${(m.depends_on || []).map(x => '<span class="chip mono">' + DIX.esc(shortType(x)) + '</span>').join("") || '<span class="muted">无</span>'}</p>
      <p><button class="btn ghost" id="d-prefix">在全局图中按此模块过滤</button></p>`;
    document.getElementById("d-prefix").onclick = () => {
      state.mode = "providers";
      document.getElementById("g-mode").value = "providers";
      state.prefix = m.name;
      document.getElementById("g-prefix").value = m.name;
      redraw();
    };
  }

  // ---------- 主流程 ----------
  async function redraw() {
    const canvas = document.getElementById("graph-canvas");
    try {
      await loadData(); // 详情抽屉也依赖 allData,所有模式都预载
      let graph;
      if (state.mode === "modules") {
        graph = await buildModules();
      } else if (state.mode === "ego") {
        const center = document.getElementById("g-center").value.trim();
        const depth = document.getElementById("g-depth-ego").value;
        const dir = document.getElementById("g-dir").value;
        if (!center) {
          canvas.innerHTML = '<p class="muted" style="padding:20px">输入中心类型,或切换到全局/模块图。</p>';
          return;
        }
        const view = await DIX.get("/api/ego", { center, depth, direction: dir });
        graph = {
          nodes: view.nodes.map(n => ({
            id: n.label, label: shortType(n.label), shape: "ellipse", font: { size: 11 },
            color: { background: "#eef0ff", border: "#4f5ce5" },
            data: { kind: "type", label: n.label, pkg: n.pkg },
          })),
          edges: view.edges.map(e => ({ from: e.from, to: e.to, arrows: "to", color: { color: "#9ca3af" } })),
        };
      } else {
        const built = buildGlobal();
        graph = prune(built.nodes, built.edges);
      }
      renderNetwork(canvas, graph.nodes, graph.edges);
    } catch (err) {
      DIX.renderError(canvas, err);
    }
  }

  DIX.views.graph = {
    async render(el, query) {
      el.innerHTML = toolbarHTML(query);

      state.mode = query.get("mode") || "ego";
      document.getElementById("g-mode").value = state.mode;
      state.layout = query.get("layout") || "hierarchical";
      document.getElementById("g-layout").value = state.layout;
      state.prefix = query.get("prefix") || "";
      document.getElementById("g-prefix").value = state.prefix;
      state.depth = Number(query.get("depth")) || 0;
      document.getElementById("g-depth-all").value = String(state.depth);
      if (query.get("dir")) document.getElementById("g-dir").value = query.get("dir");

      document.getElementById("g-mode").addEventListener("change", () => {
        state.mode = document.getElementById("g-mode").value;
        document.getElementById("g-ego-controls").style.display = state.mode === "ego" ? "inline-flex" : "none";
        redraw();
      });
      document.getElementById("g-layout").addEventListener("change", () => {
        state.layout = document.getElementById("g-layout").value;
        redraw();
      });
      document.getElementById("g-depth-all").addEventListener("change", () => {
        state.depth = Number(document.getElementById("g-depth-all").value);
        redraw();
      });
      document.getElementById("g-prefix").addEventListener("change", () => {
        state.prefix = document.getElementById("g-prefix").value;
        redraw();
      });
      document.getElementById("g-reset").addEventListener("click", () => {
        state.focus = null; state.prefix = ""; state.depth = 0;
        document.getElementById("g-prefix").value = "";
        document.getElementById("g-depth-all").value = "0";
        redraw();
      });
      document.getElementById("g-form").addEventListener("submit", ev => {
        ev.preventDefault();
        state.mode = document.getElementById("g-mode").value;
        redraw();
      });

      document.getElementById("g-ego-controls").style.display = state.mode === "ego" ? "inline-flex" : "none";
      await redraw();
    },
  };
})();
