// 依赖图:模块级列表 + 以任意类型为中心的邻域子图(vis-network,本地 vendored)
window.DIX = window.DIX || {};
DIX.views = DIX.views || {};
DIX.views.graph = {
  async render(el, query) {
    el.innerHTML = `
      <form class="filters" id="g-form">
        <input type="text" id="g-center" placeholder="中心类型(如 *example/http.Application)" value="${DIX.esc(query.get("center") || "")}" style="min-width:320px">
        <!-- depth/direction 由下方脚本按 URL 参数回填 -->
        <select id="g-depth"><option value="1">1 跳</option><option value="2" selected>2 跳</option><option value="3">3 跳</option><option value="4">4 跳</option><option value="5">5 跳</option></select>
        <select id="g-dir"><option value="both">双向</option><option value="deps">依赖(上游)</option><option value="dependents">被依赖(下游)</option></select>
        <button class="btn" type="submit">绘制邻域</button>
        <span class="muted">点击模块查看其类型;点击类型以它为中心重绘。</span>
      </form>
      <div class="grid" style="grid-template-columns: 280px 1fr; align-items:start">
        <div class="card"><h3>模块</h3><div id="g-modules" class="muted">加载中…</div></div>
        <div>
          <div id="graph-canvas"></div>
          <div class="card" style="margin-top:14px"><h3>选中节点</h3><div id="g-detail" class="muted">点击图中节点查看详情。</div></div>
        </div>
      </div>`;

    let network = null;
    const draw = view => {
      const nodes = view.nodes.map(n => ({
        id: n.label, label: n.label,
        color: n.State === "error" ? "#fdeaea" : n.State === "slow" ? "#fdf3e2" : "#eef0ff",
        border: n.State === "error" ? "#dc2626" : n.State === "slow" ? "#d97706" : "#4f5ce5",
        font: { size: 11 },
      }));
      const edges = view.edges.map(e => ({ from: e.from, to: e.to, arrows: "to", color: "#aab2d5" }));
      if (network) network.destroy();
      network = new vis.Network(document.getElementById("graph-canvas"), {
        nodes: new vis.DataSet(nodes),
        edges: new vis.DataSet(edges),
      }, {
        layout: { hierarchical: { direction: "LR", sortMethod: "hubsize" }, improvedLayout: true },
        edges: { smooth: false },
        interaction: { hover: true },
      });
      network.on("click", params => {
        const label = params.nodes && params.nodes[0];
        if (!label) return;
        document.getElementById("g-detail").innerHTML =
          `<p class="mono">${DIX.esc(label)}</p>` +
          `<button class="btn ghost" id="g-recenter">以它为中心重绘</button>`;
        document.getElementById("g-recenter").onclick = () => {
          document.getElementById("g-center").value = label;
          document.getElementById("g-form").requestSubmit();
        };
      });
    };

    const submit = ev => {
      if (ev) ev.preventDefault();
      const center = document.getElementById("g-center").value.trim();
      if (!center) return;
      document.getElementById("g-detail").textContent = "绘制中…";
      DIX.get("/api/ego", {
        center,
        depth: document.getElementById("g-depth").value,
        direction: document.getElementById("g-dir").value,
      }).then(view => {
        if (!view.nodes.length) {
          document.getElementById("graph-canvas").innerHTML = "";
          document.getElementById("g-detail").textContent = "没有命中的声明依赖(检查中心类型拼写)。";
          return;
        }
        draw(view);
        document.getElementById("g-detail").textContent =
          `节点 ${view.nodes.length},声明边 ${view.edges.length}。点击节点查看详情。`;
      }).catch(err => DIX.renderError(document.getElementById("g-detail").parentElement, err));
    };
    // 占位:上方 “draw loading;” 会在下方被真实实现替换(见 plan Task 2 Step 4)
    if (query.get("depth")) document.getElementById("g-depth").value = query.get("depth");
    if (query.get("direction")) document.getElementById("g-dir").value = query.get("direction");
    document.getElementById("g-form").addEventListener("submit", submit);

    // 模块列表:点击模块 → 该模块类型列表;点击类型 → 以它为中心重绘
    try {
      const modules = await DIX.get("/api/modules");
      const box = document.getElementById("g-modules");
      box.innerHTML = "";
      for (const m of modules) {
        const head = document.createElement("div");
        head.className = "click";
        head.style.marginBottom = "8px";
        head.innerHTML = `<b class="mono">${DIX.esc(m.name)}</b><br>
          <span class="muted">类型 ${m.type_count} · provider ${m.provider_count} · 对象 ${m.object_count}` +
          (m.depends_on && m.depends_on.length ? ` · 依赖 ${m.depends_on.length} 模块` : "") + `</span>`;
        head.addEventListener("click", async () => {
          const types = await DIX.get("/api/search", { module: m.name, kind: "type", limit: 100 });
          const list = document.createElement("div");
          list.style.margin = "4px 0 12px";
          for (const t of types) {
            const item = document.createElement("div");
            item.className = "chip click mono";
            item.textContent = t.Label || t.label || "";
            const label = t.label || t.Label;
            item.addEventListener("click", () => {
              document.getElementById("g-center").value = label;
              document.getElementById("g-form").requestSubmit();
            });
            list.appendChild(item);
          }
          head.after(list);
        });
        box.appendChild(head);
      }
    } catch (err) {
      document.getElementById("g-modules").textContent = "模块加载失败:" + err;
    }

    if (query.get("center")) submit();
  },
};
