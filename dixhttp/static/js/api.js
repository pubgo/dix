// API 客户端:所有请求经 DIX_BASE 前缀(basePath 部署兼容)。
window.DIX = window.DIX || {};
(function () {
  const base = window.DIX_BASE || "";

  DIX.get = async function (path, params) {
    let url = base + path;
    if (params) {
      const qs = new URLSearchParams();
      for (const [k, v] of Object.entries(params)) {
        if (v !== undefined && v !== null && v !== "") qs.set(k, v);
      }
      const q = qs.toString();
      if (q) url += "?" + q;
    }
    const resp = await fetch(url);
    if (!resp.ok) {
      const text = await resp.text();
      throw new Error(resp.status + " " + text.slice(0, 200));
    }
    return resp.json();
  };

  DIX.renderError = function (el, err) {
    el.innerHTML = '<div class="card"><h3>加载失败</h3><p class="err-text mono"></p></div>';
    el.querySelector(".err-text").textContent = String(err);
  };

  DIX.esc = function (s) {
    return String(s ?? "").replace(/[&<>"]/g, c => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]));
  };

  // 全局搜索:Enter 跳检索视图
  document.addEventListener("DOMContentLoaded", () => {
    const form = document.getElementById("global-search");
    if (!form) return;
    form.addEventListener("submit", ev => {
      ev.preventDefault();
      const q = document.getElementById("gsearch-input").value.trim();
      location.hash = "#/search" + (q ? "?q=" + encodeURIComponent(q) : "");
    });
  });
})();
