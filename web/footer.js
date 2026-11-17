/*
 * 站点页脚公共链接 —— 【自动生成，请勿直接修改本文件】
 *
 * 数据源：/Users/51pwn/MyWork/site-links/links.json
 * 模板　：/Users/51pwn/MyWork/site-links/footer.template.js
 * 重新生成：python3 /Users/51pwn/MyWork/site-links/sync.py
 *
 * 生成时间：2026-11-17 17:13:11　版本：20261117a　站点：startAllSs-admin
 */
(function () {
  var CFG = {
    "site": "startAllSs-admin",
    "title": "startAllSs 管理台",
    "linksApi": false,
    "footerClass": "site-footer",
    "headerSel": "#top-chrome",
    "links": [
      {
        "label": "选股策略",
        "href": "https://stock.51pwn.com/",
        "external": true
      },
      {
        "label": "高匿聊天",
        "href": "https://chat.51pwn.com/",
        "external": true
      },
      {
        "label": "智见股市",
        "href": "https://gp.51pwn.com/",
        "external": true
      },
      {
        "label": "AI Tags",
        "href": "https://aitag.51pwn.com",
        "external": true
      },
      {
        "label": "见到你",
        "href": "https://vi.51pwn.com/",
        "external": true
      },
      {
        "label": "远程管理",
        "href": "http://127.0.0.1:8088/",
        "external": true
      },
      {
        "label": "域名管理",
        "href": "http://127.0.0.1:8787/",
        "external": true
      }
    ],
    "extras": [],
    "about": []
  };

  var esc = function (s) {
    return String(s == null ? "" : s).replace(/[&<>"']/g, function (c) {
      return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c];
    });
  };

  // 页面没有提供 GP_openAbout（例如登录页）时，用配置里的简述兜底
  function aboutFallback() {
    if (!CFG.about || !CFG.about.length) return;
    var mask = document.createElement("div");
    mask.className = "mask";
    var body = CFG.about.map(function (p) {
      return "<p>" + esc(p) + "</p>";
    }).join("");
    mask.innerHTML =
      '<div class="detail-panel users-panel">' +
        '<div class="detail-head"><span>关于 · ' + esc(CFG.title) + '</span>' +
          '<button class="fb-close">×</button></div>' +
        '<div class="users-body">' + body + "</div>" +
      "</div>";
    var close = function () { mask.remove(); };
    mask.querySelector(".fb-close").addEventListener("click", close);
    mask.addEventListener("click", function (e) { if (e.target === mask) close(); });
    document.body.appendChild(mask);
  }

  function headerEl() {
    return document.querySelector(CFG.headerSel || "header.topbar");
  }

  function footerEl() {
    return document.querySelector("footer");
  }

  // 顶栏/底栏是 fixed 浮层，body 用 CSS 变量给它们留位置。
  // 高度会随窗口宽度（导航折行、链接折行）变化，所以实测后写回变量，而不是写死数值。
  function syncChrome() {
    var root = document.documentElement;
    var hd = headerEl();
    var ft = footerEl();
    var h = hd ? Math.round(hd.getBoundingClientRect().height) : 0;
    var f = ft ? Math.round(ft.getBoundingClientRect().height) : 0;
    root.style.setProperty("--header-h", (h > 0 ? h : 0) + "px");
    root.style.setProperty("--footer-h", (f > 0 ? f : 0) + "px");
  }

  function watchChrome() {
    syncChrome();
    if (typeof ResizeObserver === "function") {
      var ro = new ResizeObserver(syncChrome);
      var hd = headerEl();
      var ft = footerEl();
      if (hd) ro.observe(hd);
      if (ft) ro.observe(ft);
    }
    window.addEventListener("resize", syncChrome);
    window.addEventListener("orientationchange", function () { setTimeout(syncChrome, 200); });
    if (document.fonts && document.fonts.ready) document.fonts.ready.then(syncChrome);
  }

  function paint(box, items) {
    box.innerHTML = items.map(function (it) {
      if (it.action === "about") {
        return '<a href="#" data-about="1">' + esc(it.label) + "</a>";
      }
      var ext = it.external === false ? "" : ' target="_blank" rel="noopener noreferrer"';
      return '<a href="' + esc(it.href || "#") + '"' + ext + ">" + esc(it.label) + "</a>";
    }).join("");
    Array.prototype.forEach.call(box.querySelectorAll("[data-about]"), function (a) {
      a.addEventListener("click", function (e) {
        e.preventDefault();
        if (typeof window.GP_openAbout === "function") window.GP_openAbout();
        else aboutFallback();
      });
    });
  }

  async function render() {
    var items = [];
    // 后端能下发时以 /api/links 为准（可热改，不用重新部署前端）
    if (CFG.linksApi) {
      try {
        var r = await fetch("/api/links", { cache: "no-store" });
        var d = await r.json();
        if (d && Array.isArray(d.items) && d.items.length) items = d.items;
      } catch (_) { /* 接口不可用时用内置默认值 */ }
    }
    if (!items.length) items = CFG.links || [];

    // HTML 里没有 <footer> 时整块由脚本生成（页脚结构本身也归本模块统一管理）
    var foot = document.querySelector("footer");
    if (!foot) {
      foot = document.createElement("footer");
      foot.className = CFG.footerClass || "site-footer";
      document.body.appendChild(foot);
    }
    var box = document.getElementById("foot-links");
    if (!box) {
      box = document.createElement("div");
      box.id = "foot-links";
      box.className = "foot-links";
      foot.insertBefore(box, foot.firstChild);
    }
    paint(box, items);

    // 右侧附加项（GitHub 等站点自有链接）：只加一次，避免重复渲染
    if (CFG.extras && CFG.extras.length && !foot.querySelector(".foot-extras")) {
      var right = document.createElement("div");
      right.className = "foot-extras";
      right.innerHTML = CFG.extras.map(function (it) {
        var ext = it.external === false ? "" : ' target="_blank" rel="noopener noreferrer"';
        return '<a href="' + esc(it.href || "#") + '"' + ext + ">" + esc(it.label) + "</a>";
      }).join("");
      foot.appendChild(right);
    }
  }

  function boot() {
    render().finally(watchChrome);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot);
  } else {
    boot();
  }
})();
