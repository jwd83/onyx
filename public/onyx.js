(function () {
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
  document.addEventListener("onyx-theme-change", function () {
    if (!isOpen()) return;
    readColors();
    dirty = true;
    draw();
  });
})();
