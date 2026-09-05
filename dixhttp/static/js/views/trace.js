// 调用链:按 trace_id 分组的列表 + 服务端调用树还原
window.DIX = window.DIX || {};
DIX.views = DIX.views || {};
DIX.views.trace = {
  async render(el, query = new URLSearchParams()) {
    const filter = {
      provider: query.get("provider") || "",
      output_type: query.get("output_type") || "",
      status: query.get("status") || "",
      trace_id: query.get("trace_id") || "",
    };
    const hasFilter = !!(filter.provider || filter.output_type || filter.status || filter.trace_id);
    el.innerHTML =
      (hasFilter
        ? `<div class="card muted" id="t-filter">已按问题过滤调用链。 <button class="btn ghost" type="button" id="t-clear">清除过滤</button></div>`
        : "") +
      `<div class="card"><div id="t-list" class="muted">加载 trace…</div></div><div id="t-tree"></div>`;
    const list = document.getElementById("t-list");
    const clear = document.getElementById("t-clear");
    if (clear) clear.onclick = () => { location.hash = "#/trace"; };

    try {
      const res = await DIX.get("/api/trace", { limit: 300 });
      const records = window.DIXGraphState.filterTraceRecords(res.records || [], filter);
      const groups = new Map();
      for (const rec of records) {
        if (!rec.trace_id) continue; // DI 点事件无 trace_id,不属于任何调用链
        let g = groups.get(rec.trace_id);
        if (!g) {
          g = { id: rec.trace_id, events: 0, errors: 0, lastTs: 0, ops: new Set() };
          groups.set(rec.trace_id, g);
        }
        g.events++;
        if (rec.status === "error") g.errors++;
        if (rec.occurred_at_unix_nano > g.lastTs) g.lastTs = rec.occurred_at_unix_nano;
      }
      const rows = [...groups.values()].sort((a, b) =>
        (b.errors > 0) - (a.errors > 0) || b.lastTs - a.lastTs
      );
      if (!rows.length) {
        list.innerHTML = hasFilter
          ? '<p class="muted">没有匹配过滤条件的 trace。</p>'
          : '<p class="muted">还没有 trace 事件(注入一次即产生)。</p>';
        return;
      }

      list.innerHTML = `<table class="tbl"><tr><th>Trace</th><th class="num">事件</th><th class="num">错误</th><th></th></tr>` +
        rows.map((g, i) => {
          const short = g.id.length > 18 ? g.id.slice(0, 8) + "…" + g.id.slice(-6) : g.id;
          const badge = g.errors > 0 ? `<span class="badge err">${g.errors} 错误</span>` : '<span class="badge ok">ok</span>';
          return `<tr><td class="mono">${DIX.esc(short)}</td><td class="num">${g.events}</td>` +
            `<td class="num">${badge}</td><td><button class="btn ghost" data-i="${i}">查看调用树</button></td></tr>`;
        }).join("") + `</table>`;

      const openTree = async (g) => {
        const box = document.getElementById("t-tree");
        box.innerHTML = '<div class="card"><div class="loading">还原调用树…</div></div>';
        try {
          const tree = await DIX.get("/api/trace-tree", { trace_id: g.id });
          const container = box.querySelector(".card") || box;
          container.innerHTML = `<h3 style="margin:0 0 10px" class="mono">trace ${DIX.esc(g.id)}</h3>` +
            '<ul class="tree" id="t-root"></ul>';
          const rootUl = document.getElementById("t-root");
          for (const node of tree.roots || []) {
            rootUl.appendChild(DIX.views.trace.renderNode(node));
          }
        } catch (err) {
          DIX.renderError(box, err);
        }
      };

      list.querySelectorAll("button[data-i]").forEach(btn => {
        btn.addEventListener("click", () => openTree(rows[Number(btn.dataset.i)]));
      });

      if (filter.trace_id) {
        const hit = rows.find(g => g.id === filter.trace_id);
        if (hit) await openTree(hit);
      }
    } catch (err) {
      DIX.renderError(list, err);
    }
  },

  renderNode(node) {
    const li = document.createElement("li");
    const line = document.createElement("div");
    line.className = "node-line";
    const status = node.end ? node.end.status : (node.event.status || "start");
    const badge = status === "error" ? '<span class="badge err">error</span>' : status === "ok" ? '<span class="badge ok">ok</span>' : "";
    const dur = node.end ? `<span class="dur">${node.end.duration_ns / 1e6} ms</span>` : "";
    line.innerHTML =
      `<span class="mono">${DIX.esc(node.event.operation)}</span>` + badge + dur;
    li.appendChild(line);

    if (node.children && node.children.length) {
      const ul = document.createElement("ul");
      for (const c of node.children) ul.appendChild(DIX.views.trace.renderNode(c));
      li.appendChild(ul);
    }
    return li;
  },
};
