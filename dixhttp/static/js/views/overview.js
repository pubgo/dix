// 概览:全局统计 + 解析热度 + 慢/错误 provider + 最近注入错误
window.DIX = window.DIX || {};
DIX.views = DIX.views || {};
DIX.views.overview = {
  async render(el) {
    try {
      const [stats, issues] = await Promise.all([
        DIX.get("/api/stats"),
        DIX.get("/api/issues", { limit: 12 }),
      ]);
      el.innerHTML = `
        <div class="grid cards">
          <div class="card stat"><div class="num">${stats.provider_count}</div><div class="cap">Providers</div></div>
          <div class="card stat"><div class="num">${stats.object_count}</div><div class="cap">已创建对象</div></div>
          <div class="card stat"><div class="num">${stats.modules}</div><div class="cap">模块</div></div>
          <div class="card stat"><div class="num">${stats.package_count}</div><div class="cap">包</div></div>
          <div class="card stat"><div class="num">${stats.edge_count}</div><div class="cap">依赖边</div></div>
        </div>
        <div class="grid cols-2">
          <div class="card"><h3>解析热度 TopN</h3><div id="ov-top"></div></div>
          <div class="card"><h3>问题 Provider</h3><div id="ov-problem"></div></div>
          <div class="card"><h3>调试问题</h3><div id="ov-issues"></div></div>
          <div class="card"><h3>下一步</h3>
            <p class="muted">用顶部搜索或「检索」视图定位类型;在「依赖图」查看模块级关系并下钻;</p>
            <p class="muted">「调用链」还原一次注入的完整解析过程;「诊断」排查启动耗时与错误。</p>
          </div>
        </div>`;

      const top = stats.top_resolved || [];
      document.getElementById("ov-top").innerHTML = top.length
        ? `<table class="tbl"><tr><th>类型</th><th class="num">解析次数</th></tr>` +
          top.map(r => `<tr><td class="mono">${DIX.esc(r.type)}</td><td class="num">${r.count}</td></tr>`).join("") +
          `</table>`
        : '<p class="muted">暂无解析记录(尚未注入)。</p>';

      const slow = stats.slow_providers || [], errored = stats.error_providers || [];
      document.getElementById("ov-problem").innerHTML =
        (slow.length ? `<h3 style="margin-top:2px">慢 Provider</h3>` + slow.map(n => `<span class="chip mono">${DIX.esc(n)}</span>`).join("") : "") +
        (errored.length ? `<h3 style="margin-top:10px">错误 Provider</h3>` + errored.map(n => `<span class="chip mono err-text">${DIX.esc(n)}</span>`).join("") : "") +
        (!slow.length && !errored.length ? '<p class="muted">一切正常。</p>' : "");

      const list = issues || [];
      const issueBox = document.getElementById("ov-issues");
      issueBox.innerHTML = list.length
        ? `<table class="tbl"><tr><th>级别</th><th>问题</th><th>定位</th><th></th></tr>` +
          list.map((issue, index) =>
            `<tr><td><span class="badge ${issue.severity === "error" ? "err" : "warn"}">${DIX.esc(issue.severity)}</span></td>` +
            `<td><b>${DIX.esc(issue.title)}</b><div class="mono muted">${DIX.esc(issue.root_cause || issue.output_type || issue.provider || "")}</div></td>` +
            `<td class="mono muted">${DIX.esc(issue.output_type || issue.module || issue.provider || "")}</td>` +
            `<td><button class="btn ghost" data-issue-graph="${index}">图</button> <button class="btn ghost" data-issue-trace="${index}">Trace</button></td></tr>`).join("") +
          `</table>`
        : '<p class="muted">没有当前问题。</p>';
      issueBox.querySelectorAll("[data-issue-graph]").forEach(button => {
        button.onclick = () => { location.hash = window.DIXGraphState.issueGraphHash(list[Number(button.dataset.issueGraph)]); };
      });
      issueBox.querySelectorAll("[data-issue-trace]").forEach(button => {
        button.onclick = () => { location.hash = window.DIXGraphState.issueTraceHash(list[Number(button.dataset.issueTrace)]); };
      });
    } catch (err) {
      DIX.renderError(el, err);
    }
  },
};
