package main

const defaultPageTemplate = `{{.Generated}}
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{if .Page.IsHome}}{{.Site.Title}}{{else}}{{.Page.Title}} - {{.Site.Title}}{{end}}</title>
  <link rel="stylesheet" href="{{.CSSURL}}">
  <script>window.ONYX_ROOT = {{.RootScript}}; window.ONYX_PAGE = {{.PageID}};</script>
  {{if or .Search .Graph}}<script defer src="{{.JSURL}}"></script>{{end}}{{if .Page.HasMath}}<script async src="https://cdn.jsdelivr.net/npm/mathjax@3/es5/tex-chtml.js"></script>{{end}}
</head>
<body>
  <a class="skip-link" href="#content">Skip to content</a>
  <div class="onyx-shell">
    <aside class="onyx-sidebar">
      <a class="onyx-brand" href="{{.HomeURL}}">{{.Site.Title}}</a>
      {{if .Search}}
      <div class="onyx-search">
        <input id="onyx-search" type="search" autocomplete="off" placeholder="Search notes">
        <div id="onyx-search-results" class="onyx-search-results" hidden></div>
      </div>
      {{end}}
      {{if .Graph}}
      <button type="button" id="onyx-graph-open" class="onyx-graph-btn" aria-haspopup="dialog">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" aria-hidden="true">
          <circle cx="5" cy="6" r="2.2"></circle>
          <circle cx="18" cy="7" r="2.2"></circle>
          <circle cx="12" cy="17" r="2.2"></circle>
          <line x1="6.8" y1="7.2" x2="10.4" y2="15.4"></line>
          <line x1="16.4" y1="8.6" x2="13.3" y2="15.5"></line>
          <line x1="7" y1="6.5" x2="16" y2="6.8"></line>
        </svg>
        Graph view
      </button>
      {{end}}
      <nav aria-label="Notes">
        {{.Nav}}
      </nav>
    </aside>
    <main id="content" class="onyx-main">
      <article class="onyx-note">
        <header class="onyx-note-header">
          {{if .Page.SourceRel}}<p class="onyx-path">{{.Page.SourceRel}}</p>{{end}}
          <h1>{{.Page.Title}}</h1>
          {{if .ShowSource}}<a class="onyx-source" href="{{.Page.SourceURL}}">Markdown</a>{{end}}
        </header>
        <div class="onyx-content">
          {{.Page.HTML}}
        </div>
        {{if and .Backlinks .Site.Title}}
        <section class="onyx-backlinks" aria-labelledby="backlinks-title">
          <h2 id="backlinks-title">Linked From</h2>
          <ul>
            {{range .Backlinks}}<li><a href="{{.URL}}">{{.Title}}</a> <span>{{.Path}}</span></li>{{end}}
          </ul>
        </section>
        {{end}}
      </article>
    </main>
  </div>
  {{if .Graph}}
  <div id="onyx-graph-modal" class="onyx-graph-modal" hidden role="dialog" aria-modal="true" aria-label="Knowledge graph">
    <div class="onyx-graph-backdrop" data-graph-close></div>
    <div class="onyx-graph-panel">
      <div class="onyx-graph-toolbar">
        <span class="onyx-graph-title">Graph</span>
        <span class="onyx-graph-hint">drag to pan &middot; scroll to zoom &middot; click a node to open</span>
        <button type="button" class="onyx-graph-close" data-graph-close aria-label="Close graph">&times;</button>
      </div>
      <canvas id="onyx-graph-canvas"></canvas>
    </div>
  </div>
  {{end}}
</body>
</html>
`

const defaultCSS = `:root {
  color-scheme: light;
  --bg: #f7f5ef;
  --panel: #efece3;
  --text: #20201d;
  --muted: #6d6a60;
  --line: #ddd7ca;
  --accent: #2f6f73;
  --accent-2: #8a5a35;
  --code: #ece5d8;
  --warn: #9a3412;
  --hover: #fdf6e7;
  --active: #f4ead3;
  --ring: rgb(47 111 115 / 22%);
  --radius: 8px;
  --graph-node: #9a948733;
  --graph-node-solid: #8c867a;
  --graph-link: rgb(109 106 96 / 28%);
  --graph-current: var(--accent);
  --graph-focus: var(--accent-2);
  --graph-label: #4a4740;
  --graph-backdrop: rgb(24 23 20 / 55%);
  font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
}

:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

* { box-sizing: border-box; }
html { min-height: 100%; }
body {
  min-height: 100%;
  margin: 0;
  background: var(--bg);
  color: var(--text);
  line-height: 1.65;
}

a { color: var(--accent); text-decoration-thickness: .08em; text-underline-offset: .18em; }
a:hover { color: var(--accent-2); }

.skip-link {
  position: absolute;
  top: .75rem;
  left: .75rem;
  transform: translateY(-200%);
  background: var(--text);
  color: var(--bg);
  padding: .45rem .7rem;
  border-radius: 6px;
  z-index: 10;
}
.skip-link:focus { transform: translateY(0); }

.onyx-shell {
  display: grid;
  grid-template-columns: minmax(17rem, 22rem) minmax(0, 1fr);
  min-height: 100vh;
}

.onyx-sidebar {
  position: sticky;
  top: 0;
  height: 100vh;
  overflow: auto;
  border-right: 1px solid var(--line);
  background: var(--panel);
  padding: 1.1rem;
}

.onyx-brand {
  display: block;
  color: var(--text);
  font-size: 1.05rem;
  font-weight: 750;
  line-height: 1.2;
  margin-bottom: 1rem;
  text-decoration: none;
}

.onyx-search { position: relative; margin-bottom: 1rem; }
.onyx-search input {
  width: 100%;
  border: 1px solid var(--line);
  border-radius: 7px;
  background: #fffdfa;
  color: var(--text);
  font: inherit;
  line-height: 1.2;
  padding: .55rem .65rem;
  transition: border-color .12s ease, box-shadow .12s ease;
}
.onyx-search input:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--ring);
}
.onyx-search-results {
  position: absolute;
  z-index: 5;
  top: calc(100% + .35rem);
  left: 0;
  right: 0;
  max-height: 60vh;
  overflow: auto;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: #fffdfa;
  box-shadow: 0 14px 28px rgb(32 32 29 / 15%);
}
.onyx-search-results a {
  display: block;
  padding: .65rem .75rem;
  color: var(--text);
  text-decoration: none;
  border-bottom: 1px solid var(--line);
}
.onyx-search-results a:last-child { border-bottom: 0; }
.onyx-search-results strong { display: block; line-height: 1.25; }
.onyx-search-results span {
  display: block;
  color: var(--muted);
  font-size: .82rem;
  line-height: 1.35;
  margin-top: .15rem;
}

.nav-tree, .nav-tree ul {
  list-style: none;
  margin: 0;
  padding-left: 0;
}
.nav-tree ul {
  margin-left: .55rem;
  padding-left: .55rem;
  border-left: 1px solid var(--line);
}
.nav-tree li { margin: .05rem 0; }
.nav-tree a, .nav-tree span, .nav-tree summary {
  display: flex;
  align-items: center;
  gap: .5rem;
  color: var(--muted);
  font-size: .92rem;
  line-height: 1.3;
  padding: .28rem .45rem;
  text-decoration: none;
  border-radius: 7px;
  transition: background .12s ease, color .12s ease;
}
.nav-tree a:hover, .nav-tree summary:hover { background: var(--hover); color: var(--text); }
.nav-tree a[aria-current="page"] {
  background: var(--active);
  color: var(--accent);
  font-weight: 700;
}

/* Folder rows: a caret that rotates open, with a folder-toned label. */
.nav-tree details { margin: .05rem 0; }
.nav-tree summary {
  cursor: pointer;
  list-style: none;
  font-weight: 600;
  color: var(--text);
}
.nav-tree summary::-webkit-details-marker { display: none; }
.nav-tree summary::before {
  content: "";
  flex: 0 0 auto;
  width: 0;
  height: 0;
  border-left: 5px solid currentColor;
  border-top: 4px solid transparent;
  border-bottom: 4px solid transparent;
  opacity: .5;
  transition: transform .15s ease;
}
.nav-tree details[open] > summary::before { transform: rotate(90deg); }
.nav-tree summary a, .nav-tree summary span {
  display: inline;
  gap: 0;
  padding: 0;
  color: inherit;
  font: inherit;
}
.nav-tree summary a:hover { background: none; color: inherit; }

/* File rows: a small square dot, filled when it's the current page. */
.nav-tree li > a::before {
  content: "";
  flex: 0 0 auto;
  width: .4rem;
  height: .4rem;
  margin: 0 .05rem;
  border-radius: 2px;
  background: currentColor;
  opacity: .35;
}
.nav-tree li > a:hover::before { opacity: .6; }
.nav-tree li > a[aria-current="page"]::before { opacity: 1; }

.onyx-main {
  width: min(100%, 78rem);
  padding: clamp(1.2rem, 4vw, 4rem);
}
.onyx-note {
  max-width: 52rem;
}
.onyx-note-header {
  border-bottom: 1px solid var(--line);
  margin-bottom: 1.35rem;
  padding-bottom: 1rem;
}
.onyx-path {
  color: var(--muted);
  font-size: .85rem;
  line-height: 1.2;
  margin: 0 0 .35rem;
}
.onyx-note h1 {
  font-size: clamp(2rem, 5vw, 3.2rem);
  letter-spacing: 0;
  line-height: 1.05;
  margin: 0;
}
.onyx-source {
  display: inline-block;
  font-size: .86rem;
  margin-top: .65rem;
}

.onyx-content h2, .onyx-content h3, .onyx-content h4 {
  line-height: 1.15;
  margin-top: 2rem;
}
.onyx-content p, .onyx-content ul, .onyx-content ol, .onyx-content blockquote, .onyx-content table, .onyx-content pre {
  margin-bottom: 1rem;
}
.onyx-content img {
  max-width: 100%;
  height: auto;
  border-radius: 8px;
}
.onyx-content code {
  background: var(--code);
  border-radius: 5px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: .92em;
  padding: .08rem .25rem;
}
.onyx-content pre {
  background: #24231f;
  color: #f7f5ef;
  overflow: auto;
  padding: 1rem;
  border-radius: 8px;
}
.onyx-content pre code {
  background: transparent;
  color: inherit;
  padding: 0;
}
.onyx-content blockquote {
  border-left: 3px solid var(--line);
  color: var(--muted);
  margin-left: 0;
  padding-left: 1rem;
}
.onyx-content table {
  border-collapse: collapse;
  display: block;
  max-width: 100%;
  overflow-x: auto;
}
.onyx-content th, .onyx-content td {
  border: 1px solid var(--line);
  padding: .45rem .6rem;
  vertical-align: top;
}
.onyx-content th { background: var(--panel); text-align: left; }
.onyx-content .onyx-math {
  margin: 1.1rem 0;
  overflow-x: auto;
  overflow-y: hidden;
}

.callout {
  border: 1px solid var(--line);
  border-left: 4px solid var(--accent);
  border-radius: 8px;
  margin: 1rem 0;
  padding: .8rem 1rem;
  background: #fffdfa;
}
.callout-title {
  color: var(--accent);
  font-weight: 800;
  margin: 0 0 .35rem;
}
.callout-warning, .callout-danger, .callout-caution { border-left-color: var(--warn); }
.callout-warning .callout-title, .callout-danger .callout-title, .callout-caution .callout-title { color: var(--warn); }

.broken-link {
  color: var(--warn);
  font-weight: 700;
}
.embed-note {
  border: 1px solid var(--line);
  border-radius: 6px;
  display: inline-block;
  padding: .05rem .35rem;
  text-decoration: none;
}

.onyx-backlinks {
  border-top: 1px solid var(--line);
  margin-top: 2rem;
  padding-top: 1rem;
}
.onyx-backlinks h2 {
  font-size: 1rem;
  letter-spacing: 0;
  margin: 0 0 .5rem;
}
.onyx-backlinks ul { padding-left: 1.1rem; }
.onyx-backlinks span {
  color: var(--muted);
  font-size: .85rem;
}

/* Graph view: a sidebar trigger that opens a full-screen force-directed map. */
.onyx-graph-btn {
  display: flex;
  align-items: center;
  gap: .5rem;
  width: 100%;
  margin-bottom: 1rem;
  padding: .5rem .65rem;
  border: 1px solid var(--line);
  border-radius: 7px;
  background: #fffdfa;
  color: var(--text);
  font: inherit;
  line-height: 1.2;
  cursor: pointer;
  transition: background .12s ease, border-color .12s ease, color .12s ease;
}
.onyx-graph-btn:hover { background: var(--hover); border-color: var(--accent); color: var(--accent); }
.onyx-graph-btn svg { width: 1.1em; height: 1.1em; flex: 0 0 auto; opacity: .85; }

.onyx-graph-modal { position: fixed; inset: 0; z-index: 50; display: flex; }
.onyx-graph-modal[hidden] { display: none; }
.onyx-graph-backdrop {
  position: absolute;
  inset: 0;
  background: var(--graph-backdrop);
  -webkit-backdrop-filter: blur(2px);
  backdrop-filter: blur(2px);
}
.onyx-graph-panel {
  position: relative;
  margin: auto;
  width: min(94vw, 1100px);
  height: min(88vh, 780px);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--bg);
  border: 1px solid var(--line);
  border-radius: 14px;
  box-shadow: 0 24px 60px rgb(24 23 20 / 38%);
  animation: onyx-graph-pop .16s ease;
}
@keyframes onyx-graph-pop {
  from { opacity: 0; transform: translateY(8px) scale(.99); }
  to { opacity: 1; transform: none; }
}
.onyx-graph-toolbar {
  display: flex;
  align-items: center;
  gap: .75rem;
  padding: .55rem .9rem;
  border-bottom: 1px solid var(--line);
  background: var(--panel);
}
.onyx-graph-title { font-weight: 750; }
.onyx-graph-hint { color: var(--muted); font-size: .8rem; margin-right: auto; }
.onyx-graph-close {
  border: 0;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  font-size: 1.5rem;
  line-height: 1;
  padding: 0 .25rem;
  border-radius: 6px;
}
.onyx-graph-close:hover { color: var(--text); background: var(--hover); }
#onyx-graph-canvas {
  flex: 1 1 auto;
  width: 100%;
  height: 100%;
  display: block;
  background: var(--bg);
  cursor: grab;
  touch-action: none;
}
#onyx-graph-canvas.is-grabbing { cursor: grabbing; }
#onyx-graph-canvas.is-pointing { cursor: pointer; }
body.onyx-graph-lock { overflow: hidden; }

@media (max-width: 820px) {
  .onyx-shell { display: block; }
  .onyx-sidebar {
    position: static;
    height: auto;
    max-height: 45vh;
    border-right: 0;
    border-bottom: 1px solid var(--line);
  }
  .onyx-main { padding: 1.2rem; }
  .onyx-note h1 { font-size: 2rem; }
  .onyx-graph-panel { width: 96vw; height: 90vh; }
  .onyx-graph-hint { display: none; }
}
`

const defaultJS = `(function () {
  const input = document.getElementById("onyx-search");
  const results = document.getElementById("onyx-search-results");
  if (!input || !results) return;

  const root = window.ONYX_ROOT || "";
  let index = [];

  fetch(root + "public/search-index.json")
    .then((response) => response.ok ? response.json() : [])
    .then((items) => { index = Array.isArray(items) ? items : []; })
    .catch(() => { index = []; });

  function escapeHTML(value) {
    return String(value).replace(/[&<>"']/g, (ch) => ({
      "&": "&amp;",
      "<": "&lt;",
      ">": "&gt;",
      '"': "&quot;",
      "'": "&#39;"
    })[ch]);
  }

  function score(item, terms) {
    const haystack = [
      item.title || "",
      item.path || "",
      item.excerpt || "",
      (item.headings || []).join(" "),
      (item.tags || []).join(" ")
    ].join(" ").toLowerCase();
    let total = 0;
    for (const term of terms) {
      if (!haystack.includes(term)) return 0;
      total += (item.title || "").toLowerCase().includes(term) ? 4 : 1;
      total += (item.path || "").toLowerCase().includes(term) ? 2 : 0;
    }
    return total;
  }

  function withRoot(url) {
    return root + (url || "./");
  }

  function render() {
    const terms = input.value.trim().toLowerCase().split(/\s+/).filter(Boolean);
    if (!terms.length) {
      results.hidden = true;
      results.innerHTML = "";
      return;
    }
    const matches = index
      .map((item) => ({ item, value: score(item, terms) }))
      .filter((entry) => entry.value > 0)
      .sort((a, b) => b.value - a.value || String(a.item.title).localeCompare(String(b.item.title)))
      .slice(0, 12);

    results.innerHTML = matches.length
      ? matches.map(({ item }) => '<a href="' + escapeHTML(withRoot(item.url)) + '"><strong>' + escapeHTML(item.title) + '</strong><span>' + escapeHTML(item.excerpt || item.path || "") + '</span></a>').join("")
      : '<a href="#"><strong>No matches</strong><span>Try a different phrase.</span></a>';
    results.hidden = false;
  }

  input.addEventListener("input", render);
  input.addEventListener("focus", render);
  document.addEventListener("click", (event) => {
    if (!results.contains(event.target) && event.target !== input) results.hidden = true;
  });
  document.addEventListener("keydown", (event) => {
    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
      event.preventDefault();
      input.focus();
      input.select();
    }
    if (event.key === "Escape") {
      results.hidden = true;
      input.blur();
    }
  });
})();

(function () {
  var modal = document.getElementById("onyx-graph-modal");
  var canvas = document.getElementById("onyx-graph-canvas");
  var openBtn = document.getElementById("onyx-graph-open");
  if (!modal || !canvas) return;

  var root = window.ONYX_ROOT || "";
  var currentId = window.ONYX_PAGE || "";
  var ctx = canvas.getContext("2d");

  var nodes = [];
  var links = [];
  var nodeById = new Map();
  var loaded = false;
  var loading = false;

  var timer = null;
  var dirty = true;
  var alpha = 0;
  var alphaMin = 0.0015;

  var scale = 1, offsetX = 0, offsetY = 0;
  var dpr = 1;
  var hoverNode = null, dragNode = null, panning = false;
  var pressX = 0, pressY = 0, lastX = 0, lastY = 0, moved = false;
  var colors = {};

  function radiusOf(n) { return 3 + Math.sqrt(n.degree) * 1.7; }

  function load() {
    if (loaded || loading) return;
    loading = true;
    fetch(root + "public/graph.json")
      .then(function (r) { return r.ok ? r.json() : null; })
      .then(function (data) {
        build(data || { nodes: [], links: [] });
        loaded = true;
        loading = false;
        layout();
      })
      .catch(function () { loading = false; });
  }

  function build(data) {
    nodeById = new Map();
    var w = canvas.clientWidth || 800;
    var h = canvas.clientHeight || 600;
    nodes = (data.nodes || []).map(function (n, i) {
      var ang = i * 2.3999632;
      var rad = 16 + Math.sqrt(i + 1) * 16;
      var node = {
        id: n.id,
        title: n.title || n.id,
        url: n.url || "",
        degree: n.degree || 0,
        x: w / 2 + Math.cos(ang) * rad,
        y: h / 2 + Math.sin(ang) * rad,
        vx: 0, vy: 0,
        neighbors: new Set()
      };
      nodeById.set(n.id, node);
      return node;
    });
    links = (data.links || []).map(function (l) {
      return { source: nodeById.get(l.source), target: nodeById.get(l.target) };
    }).filter(function (l) { return l.source && l.target; });
    links.forEach(function (l) { l.source.neighbors.add(l.target); l.target.neighbors.add(l.source); });
  }

  function tick() {
    alpha += (0 - alpha) * 0.0228;
    var i, j, a, b, dx, dy, d2, dist, f, w;
    var charge = -60;
    for (i = 0; i < nodes.length; i++) {
      a = nodes[i];
      for (j = i + 1; j < nodes.length; j++) {
        b = nodes[j];
        dx = b.x - a.x; dy = b.y - a.y;
        d2 = dx * dx + dy * dy;
        // Floor the distance so two near-coincident nodes never produce a
        // runaway impulse (without this the layout diverges to infinity).
        if (d2 < 1) { dx = Math.random() - 0.5; dy = Math.random() - 0.5; d2 = dx * dx + dy * dy + 1; }
        w = charge * alpha / d2;
        a.vx += dx * w; a.vy += dy * w;
        b.vx -= dx * w; b.vy -= dy * w;
      }
    }
    var linkDist = 46, linkStr = 0.4;
    for (i = 0; i < links.length; i++) {
      a = links[i].source; b = links[i].target;
      dx = b.x - a.x; dy = b.y - a.y;
      dist = Math.sqrt(dx * dx + dy * dy) || 1;
      f = (dist - linkDist) / dist * alpha * linkStr;
      dx *= f; dy *= f;
      b.vx -= dx; b.vy -= dy;
      a.vx += dx; a.vy += dy;
    }
    var cx = canvas.clientWidth / 2, cy = canvas.clientHeight / 2, g = 0.05 * alpha;
    var maxV = 50;
    for (i = 0; i < nodes.length; i++) {
      a = nodes[i];
      a.vx += (cx - a.x) * g;
      a.vy += (cy - a.y) * g;
      if (a === dragNode) { a.vx = 0; a.vy = 0; continue; }
      a.vx *= 0.6; a.vy *= 0.6;
      var sp = a.vx * a.vx + a.vy * a.vy;
      if (sp > maxV * maxV) { var s = maxV / Math.sqrt(sp); a.vx *= s; a.vy *= s; }
      a.x += a.vx; a.y += a.vy;
    }
  }

  function layout() {
    alpha = 1;
    for (var k = 0; k < 140; k++) tick();
    fit();
    dirty = true;
  }

  function fit() {
    if (!nodes.length) return;
    var minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
    for (var i = 0; i < nodes.length; i++) {
      var n = nodes[i];
      if (n.x < minX) minX = n.x;
      if (n.x > maxX) maxX = n.x;
      if (n.y < minY) minY = n.y;
      if (n.y > maxY) maxY = n.y;
    }
    var w = canvas.clientWidth || 800, h = canvas.clientHeight || 600;
    var gw = (maxX - minX) || 1, gh = (maxY - minY) || 1;
    var pad = 80;
    scale = Math.min((w - pad) / gw, (h - pad) / gh);
    if (!isFinite(scale) || scale <= 0) scale = 1;
    scale = Math.max(0.15, Math.min(scale, 2.2));
    offsetX = w / 2 - ((minX + maxX) / 2) * scale;
    offsetY = h / 2 - ((minY + maxY) / 2) * scale;
  }

  function readColors() {
    var s = getComputedStyle(document.documentElement);
    function v(name, fallback) { var c = s.getPropertyValue(name).trim(); return c || fallback; }
    colors.link = v("--graph-link", "rgba(120,118,108,0.3)");
    colors.current = v("--graph-current", "#2f6f73");
    colors.focus = v("--graph-focus", "#8a5a35");
    colors.node = v("--graph-node-solid", "#8c867a");
    colors.label = v("--graph-label", "#4a4740");
  }

  function draw() {
    ctx.setTransform(1, 0, 0, 1, 0, 0);
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    ctx.setTransform(dpr * scale, 0, 0, dpr * scale, dpr * offsetX, dpr * offsetY);

    var focus = hoverNode;
    var i, n;

    ctx.lineWidth = 1 / scale;
    for (i = 0; i < links.length; i++) {
      var s = links[i].source, t = links[i].target;
      var active = focus && (s === focus || t === focus);
      ctx.strokeStyle = active ? colors.current : colors.link;
      ctx.globalAlpha = focus ? (active ? 0.9 : 0.12) : 1;
      ctx.beginPath();
      ctx.moveTo(s.x, s.y);
      ctx.lineTo(t.x, t.y);
      ctx.stroke();
    }

    for (i = 0; i < nodes.length; i++) {
      n = nodes[i];
      var isCurrent = n.id === currentId;
      var related = focus && (n === focus || focus.neighbors.has(n));
      var r = radiusOf(n);
      if (n === focus) r += 1.5 / scale + 1.5;
      ctx.globalAlpha = focus ? (related ? 1 : 0.22) : 1;
      ctx.fillStyle = isCurrent ? colors.current : (related ? colors.focus : colors.node);
      ctx.beginPath();
      ctx.arc(n.x, n.y, r, 0, Math.PI * 2);
      ctx.fill();
      if (isCurrent) {
        ctx.globalAlpha = focus ? (related ? 1 : 0.4) : 1;
        ctx.lineWidth = 2 / scale;
        ctx.strokeStyle = colors.current;
        ctx.beginPath();
        ctx.arc(n.x, n.y, r + 4 / scale, 0, Math.PI * 2);
        ctx.stroke();
      }
    }
    ctx.globalAlpha = 1;

    var fontPx = 11 / scale;
    ctx.font = fontPx + "px ui-sans-serif, system-ui, -apple-system, sans-serif";
    ctx.textAlign = "center";
    ctx.textBaseline = "top";
    ctx.fillStyle = colors.label;
    var showAll = scale > 1.25;
    for (i = 0; i < nodes.length; i++) {
      n = nodes[i];
      var rel = focus && (n === focus || focus.neighbors.has(n));
      var hub = n.degree >= 7;
      var show = rel || n.id === currentId || (!focus && (hub || showAll));
      if (!show) continue;
      ctx.globalAlpha = focus ? (rel ? 1 : 0.18) : (n.id === currentId ? 1 : 0.8);
      ctx.fillText(n.title, n.x, n.y + radiusOf(n) + 3 / scale);
    }
    ctx.globalAlpha = 1;
  }

  // A timer drives the loop rather than requestAnimationFrame so the first
  // paint never depends on the tab being actively composited. While the layout
  // is settling we tick + redraw; once settled we only redraw when something
  // changed (hover, pan, zoom, drag), keeping an idle graph near-zero cost.
  function step() {
    if (alpha > alphaMin) { tick(); draw(); }
    else if (dirty) { draw(); dirty = false; }
  }
  function start() { if (!timer) timer = setInterval(step, 1000 / 60); }
  function stop() { if (timer) { clearInterval(timer); timer = null; } }
  function reheat(a) { if (a > alpha) alpha = a; dirty = true; }

  function resize() {
    dpr = Math.max(1, window.devicePixelRatio || 1);
    var w = canvas.clientWidth, h = canvas.clientHeight;
    canvas.width = Math.round(w * dpr);
    canvas.height = Math.round(h * dpr);
  }

  function isOpen() { return !modal.hidden; }

  function open() {
    modal.hidden = false;
    document.body.classList.add("onyx-graph-lock");
    readColors();
    resize();
    load();
    if (loaded) { fit(); reheat(0.3); }
    start();
    dirty = true;
    draw();
  }

  function close() {
    modal.hidden = true;
    document.body.classList.remove("onyx-graph-lock");
    stop();
    hoverNode = null;
    dragNode = null;
    panning = false;
    canvas.classList.remove("is-grabbing", "is-pointing");
  }

  function toWorld(clientX, clientY) {
    var rect = canvas.getBoundingClientRect();
    return { x: (clientX - rect.left - offsetX) / scale, y: (clientY - rect.top - offsetY) / scale };
  }

  function pick(wx, wy) {
    var best = null, bestD = Infinity;
    for (var i = 0; i < nodes.length; i++) {
      var n = nodes[i];
      var r = radiusOf(n) + 4 / scale;
      var dx = n.x - wx, dy = n.y - wy, d = dx * dx + dy * dy;
      if (d <= r * r && d < bestD) { bestD = d; best = n; }
    }
    return best;
  }

  function navigate(n) {
    if (!n) return;
    window.location.href = root + (n.url || "") || "./";
  }

  canvas.addEventListener("pointerdown", function (e) {
    try { canvas.setPointerCapture(e.pointerId); } catch (err) {}
    pressX = e.clientX; pressY = e.clientY;
    lastX = e.clientX; lastY = e.clientY;
    moved = false;
    var p = toWorld(e.clientX, e.clientY);
    var n = pick(p.x, p.y);
    if (n) { dragNode = n; reheat(0.4); } else { panning = true; }
    canvas.classList.add("is-grabbing");
    dirty = true;
  });

  canvas.addEventListener("pointermove", function (e) {
    if (Math.abs(e.clientX - pressX) + Math.abs(e.clientY - pressY) > 3) moved = true;
    var mvx = e.movementX !== undefined ? e.movementX : e.clientX - lastX;
    var mvy = e.movementY !== undefined ? e.movementY : e.clientY - lastY;
    lastX = e.clientX; lastY = e.clientY;
    if (dragNode) {
      var p = toWorld(e.clientX, e.clientY);
      dragNode.x = p.x; dragNode.y = p.y; dragNode.vx = 0; dragNode.vy = 0;
      reheat(0.3);
    } else if (panning) {
      offsetX += mvx; offsetY += mvy;
    } else {
      var q = toWorld(e.clientX, e.clientY);
      var hit = pick(q.x, q.y);
      hoverNode = hit;
      canvas.classList.toggle("is-pointing", !!hit);
    }
    dirty = true;
  });

  function endPointer(e) {
    if (dragNode && !moved) navigate(dragNode);
    dragNode = null;
    panning = false;
    canvas.classList.remove("is-grabbing");
    try { canvas.releasePointerCapture(e.pointerId); } catch (err) {}
  }
  canvas.addEventListener("pointerup", endPointer);
  canvas.addEventListener("pointercancel", endPointer);

  canvas.addEventListener("wheel", function (e) {
    e.preventDefault();
    var rect = canvas.getBoundingClientRect();
    var sx = e.clientX - rect.left, sy = e.clientY - rect.top;
    var wx = (sx - offsetX) / scale, wy = (sy - offsetY) / scale;
    var factor = Math.exp(-e.deltaY * 0.0012);
    scale = Math.max(0.1, Math.min(scale * factor, 6));
    offsetX = sx - wx * scale;
    offsetY = sy - wy * scale;
    dirty = true;
  }, { passive: false });

  if (openBtn) openBtn.addEventListener("click", open);
  modal.addEventListener("click", function (e) {
    if (e.target && e.target.hasAttribute && e.target.hasAttribute("data-graph-close")) close();
  });
  window.addEventListener("resize", function () { if (isOpen()) { resize(); dirty = true; } });
  document.addEventListener("keydown", function (e) {
    var el = document.activeElement;
    var tag = (el && el.tagName) || "";
    var typing = tag === "INPUT" || tag === "TEXTAREA" || (el && el.isContentEditable);
    if (!isOpen() && !typing && (e.key === "g" || e.key === "G")) { e.preventDefault(); open(); }
    else if (isOpen() && e.key === "Escape") { e.preventDefault(); close(); }
  });
})();`
