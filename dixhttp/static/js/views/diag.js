// 诊断:最近注入错误(含结构化根因/提示)+ provider 启动耗时
window.DIX = window.DIX || {};
DIX.views = DIX.views || {};
DIX.views.diag = {
  async render(el) {
    el.innerHTML = `<div class="loading">加载诊断数据…</div>`;
    try {
      const [errors, stats] = await Promise.all([
        DIX.get("/api/errors", { limit: 30 }),
        DIX.get("/api/runtime-stats", { limit: 40 }),
      ]);

      const errRows = (errors || []).map(e =>
        `<tr><td><span class="badge err">${DIX.esc(e.error_type || "error")}</span></td>` +
        `<td class="mono">${DIX.esc(e.root_cause || e.message || "")}</td>` +
        `<td class="mono muted">${DIX.esc((e.component || "").slice(0, 50))}</td>` +
        `<td class="muted">${DIX.esc(e.hint || "")}</td></tr>`).join("");

      const statRows = (stats || []).map(s =>
        `<tr><td class="mono">${DIX.esc(s.function_name)}</td>` +
        `<td class="mono muted">${DIX.esc(s.output_type || "")}</td>` +
        `<td class="num">${s.call_count}</td>` +
        `<td class="num">${((s.average_duration || 0) / 1e6).toFixed(2)} ms</td>` +
        `<td class="err-text mono">${DIX.esc(s.last_error || "")}</td></tr>`).join("");

      el.innerHTML = `
        <div class="card" style="margin-bottom:16px"><h3>最近注入错误</h3>
          ${errRows ? `<table class="tbl"><tr><th>错误类型</th><th>根因</th><th>组件</th><th>提示</th></tr>${errRows}</table>` : '<p class="muted">没有错误记录。</p>'}
        </div>
        <div class="card"><h3>Provider 启动耗时</h3>
          ${statRows ? `<table class="tbl"><tr><th>函数</th><th>输出类型</th><th class="num">执行次数</th><th class="num">平均耗时</th><th>最近错误</th></tr>${statRows}</table>` : '<p class="muted">暂无执行记录。</p>'}
        </div>`;
    } catch (err) {
      DIX.renderError(el, err);
    }
  },
};
