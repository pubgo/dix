// 检索:服务端过滤(关键字/类别/模块/状态),结果可跳转依赖图
window.DIX = window.DIX || {};
DIX.views = DIX.views || {};
DIX.views.search = {
  async render(el, query) {
    el.innerHTML = `
      <form class="filters" id="s-form">
        <input type="search" id="s-q" placeholder="关键字(类型名 / 函数名)" value="${DIX.esc(query.get("q") || "")}" style="min-width:280px">
        <select id="s-kind">
          <option value="">全部类别</option><option value="type">类型</option>
          <option value="provider">Provider</option><option value="object">对象</option>
        </select>
        <select id="s-state">
          <option value="">全部状态</option><option value="instantiated">已实例化</option>
          <option value="error">有错误</option><option value="slow">慢</option>
        </select>
        <input type="text" id="s-module" placeholder="模块前缀(可选)" value="${DIX.esc(query.get("module") || "")}">
        <button class="btn" type="submit">检索</button>
      </form>
      <div class="card"><div id="s-result" class="muted">输入条件后检索。</div></div>`;

    const run = async () => {
      const box = document.getElementById("s-result");
      box.textContent = "检索中…";
      try {
        const hits = await DIX.get("/api/search", {
          q: document.getElementById("s-q").value,
          kind: document.getElementById("s-kind").value,
          state: document.getElementById("s-state").value,
          module: document.getElementById("s-module").value,
          limit: 100,
        });
        if (!hits.length) {
          box.innerHTML = '<p class="muted">没有命中。</p>';
          return;
        }
        box.innerHTML = `
          <table class="tbl"><tr><th>类别</th><th>名称</th><th>模块</th><th>状态</th><th></th></tr>` +
          hits.map((h, i) => {
            const stateBadge = h.state ? `<span class="badge ${h.state === "error" ? "err" : h.state === "slow" ? "warn" : "ok"}">${DIX.esc(h.state)}</span>` : "";
            return `<tr><td><span class="badge">${DIX.esc(h.kind)}</span></td>` +
              `<td class="mono">${DIX.esc(h.label)}</td>` +
              `<td class="mono muted">${DIX.esc(h.pkg || "")}</td>` +
              `<td>${stateBadge}</td>` +
              `<td><button class="btn ghost" data-i="${i}">图中查看</button></td></tr>`;
          }).join("") + `</table>`;
        box.querySelectorAll("button[data-i]").forEach(btn => {
          btn.addEventListener("click", () => {
            const h = hits[Number(btn.dataset.i)];
            // SearchHit 的 label 即该依赖的类型名(provider 节点为其输出类型),可直接作邻域中心
            location.hash = "#/graph?center=" + encodeURIComponent(h.label);
          });
        });
      } catch (err) {
        DIX.renderError(box.parentElement, err);
      }
    };

    document.getElementById("s-form").addEventListener("submit", ev => {
      ev.preventDefault();
      run();
    });
    if (query.get("q") || query.get("module") || query.get("state")) run();
  },
};
