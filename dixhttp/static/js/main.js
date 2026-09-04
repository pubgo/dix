// hash 路由:#/overview | #/graph | #/search | #/trace | #/diag(可带 ?a=b 查询)
window.DIX = window.DIX || {};
(function () {
  function parseHash() {
    const h = location.hash.replace(/^#\/?/, "");
    const [view, qs] = h.split("?");
    return { view: view || "overview", query: new URLSearchParams(qs || "") };
  }

  function activateTab(name) {
    document.querySelectorAll("#nav-tabs a").forEach(a => {
      a.classList.toggle("active", a.dataset.view === name);
    });
  }

  function render() {
    const { view, query } = parseHash();
    const el = document.getElementById("view");
    const v = DIX.views[view];
    activateTab(view);
    if (!v || typeof v.render !== "function") {
      el.innerHTML = '<div class="card"><p class="muted">未知视图:' + DIX.esc(view) + "</p></div>";
      return;
    }
    el.innerHTML = '<div class="loading">加载中…</div>';
    v.render(el, query);
  }

  window.addEventListener("hashchange", render);
  document.addEventListener("DOMContentLoaded", render);
})();
