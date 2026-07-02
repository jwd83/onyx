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
  var countEl = document.getElementById("onyx-graph-count");
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

  var scale = 1, fitScale = 1, offsetX = 0, offsetY = 0;
  var dpr = 1;
  var hoverNode = null, dragNode = null, panning = false;
  var pressX = 0, pressY = 0, lastX = 0, lastY = 0, moved = false;
  var colors = {};

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
      var title = n.title || n.id;
      var node = {
        id: n.id,
        title: title,
        label: title.length > 34 ? title.slice(0, 33) + "…" : title,
        url: n.url || "",
        degree: n.degree || 0,
        x: w / 2 + Math.cos(ang) * rad,
        y: h / 2 + Math.sin(ang) * rad,
        vx: 0, vy: 0,
        neighbors: new Set()
      };
      node.r = 3.5 + Math.sqrt(node.degree) * 2;
      nodeById.set(n.id, node);
      return node;
    });
    links = (data.links || []).map(function (l) {
      return { source: nodeById.get(l.source), target: nodeById.get(l.target) };
    }).filter(function (l) { return l.source && l.target; });
    links.forEach(function (l) { l.source.neighbors.add(l.target); l.target.neighbors.add(l.source); });
    if (countEl) {
      var plural = function (count, word) { return count + " " + word + (count === 1 ? "" : "s"); };
      countEl.textContent = nodes.length ? plural(nodes.length, "note") + " · " + plural(links.length, "link") : "";
    }
  }

  function tick() {
    alpha += (0 - alpha) * 0.0228;
    var i, j, a, b, dx, dy, d2, dist, f, w, minD;
    var charge = -120;
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
        // Overlapping circles shove apart harder than plain charge so labels
        // and dots keep breathing room around dense hubs.
        minD = a.r + b.r + 10;
        if (d2 < minD * minD) {
          dist = Math.sqrt(d2);
          w -= (minD - dist) / dist * alpha * 0.7;
        }
        a.vx += dx * w; a.vy += dy * w;
        b.vx -= dx * w; b.vy -= dy * w;
      }
    }
    var linkDist = 64, linkStr = 0.4;
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
    fitScale = scale;
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
    colors.halo = v("--graph-halo", v("--bg", "#f7f5ef"));
    colors.grid = v("--graph-grid", "rgba(32,32,29,0.06)");
  }

  // A faint dot grid in world coordinates: it pans and zooms with the nodes,
  // giving the eye a spatial anchor. Spacing doubles/halves with zoom so the
  // on-screen density stays roughly constant at any scale.
  function drawGrid(w, h) {
    var step = 56;
    while (step * scale < 34) step *= 2;
    while (step * scale > 68) step /= 2;
    var x1 = (w - offsetX) / scale, y1 = (h - offsetY) / scale;
    var gx0 = Math.floor((-offsetX / scale) / step) * step;
    var gy0 = Math.floor((-offsetY / scale) / step) * step;
    var r = 1.1 / scale;
    ctx.fillStyle = colors.grid;
    ctx.beginPath();
    for (var gx = gx0; gx <= x1; gx += step) {
      for (var gy = gy0; gy <= y1; gy += step) {
        ctx.moveTo(gx + r, gy);
        ctx.arc(gx, gy, r, 0, Math.PI * 2);
      }
    }
    ctx.fill();
  }

  function draw() {
    var w = canvas.clientWidth, h = canvas.clientHeight;
    ctx.setTransform(1, 0, 0, 1, 0, 0);
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    ctx.setTransform(dpr * scale, 0, 0, dpr * scale, dpr * offsetX, dpr * offsetY);

    drawGrid(w, h);

    if (loaded && !nodes.length) {
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      ctx.font = "13px ui-sans-serif, system-ui, -apple-system, sans-serif";
      ctx.textAlign = "center";
      ctx.textBaseline = "middle";
      ctx.fillStyle = colors.label;
      ctx.globalAlpha = 0.7;
      ctx.fillText("No linked notes yet.", w / 2, h / 2);
      ctx.globalAlpha = 1;
      return;
    }

    var focus = hoverNode;
    var i, n, s, t;

    // Links go down in two batched passes: the resting set in one path, then
    // (while hovering) the focused node's edges bright on top.
    ctx.lineCap = "round";
    ctx.lineWidth = 1 / scale;
    ctx.strokeStyle = colors.link;
    ctx.globalAlpha = focus ? 0.1 : 1;
    ctx.beginPath();
    for (i = 0; i < links.length; i++) {
      s = links[i].source; t = links[i].target;
      if (focus && (s === focus || t === focus)) continue;
      ctx.moveTo(s.x, s.y);
      ctx.lineTo(t.x, t.y);
    }
    ctx.stroke();
    if (focus) {
      ctx.lineWidth = 1.6 / scale;
      ctx.strokeStyle = colors.current;
      ctx.globalAlpha = 0.85;
      ctx.beginPath();
      for (i = 0; i < links.length; i++) {
        s = links[i].source; t = links[i].target;
        if (s !== focus && t !== focus) continue;
        ctx.moveTo(s.x, s.y);
        ctx.lineTo(t.x, t.y);
      }
      ctx.stroke();
    }

    ctx.lineWidth = 1.4 / scale;
    ctx.strokeStyle = colors.halo;
    for (i = 0; i < nodes.length; i++) {
      n = nodes[i];
      var isCurrent = n.id === currentId;
      var related = focus && (n === focus || focus.neighbors.has(n));
      var r = n.r;
      if (n === focus) r += 1.5 / scale + 1.5;

      // Soft beacon behind the reader's current note, and a matching glow
      // behind whichever node is hovered.
      if (isCurrent || n === focus) {
        ctx.globalAlpha = focus && !related ? 0.12 : (isCurrent ? 0.28 : 0.18);
        ctx.fillStyle = isCurrent ? colors.current : colors.focus;
        ctx.beginPath();
        ctx.arc(n.x, n.y, r + 6.5 / scale, 0, Math.PI * 2);
        ctx.fill();
      }

      ctx.globalAlpha = focus ? (related ? 1 : 0.22) : 1;
      ctx.fillStyle = isCurrent ? colors.current : (related ? colors.focus : colors.node);
      ctx.beginPath();
      ctx.arc(n.x, n.y, r, 0, Math.PI * 2);
      ctx.fill();
      // Rim in the backdrop tone cuts each dot out from the lines beneath it.
      ctx.stroke();
    }
    ctx.globalAlpha = 1;

    var fontPx = 11 / scale;
    ctx.font = "500 " + fontPx + "px ui-sans-serif, system-ui, -apple-system, sans-serif";
    ctx.textAlign = "center";
    ctx.textBaseline = "top";
    ctx.fillStyle = colors.label;
    ctx.strokeStyle = colors.halo;
    ctx.lineJoin = "round";
    ctx.lineWidth = 3 / scale;
    // Small graphs and hubs stay labeled at rest; rank-and-file labels fade in
    // as you zoom past ~1x. Everything melts away when zoomed well below the
    // fitted overview (labels would just pile into a smudge), measured against
    // fitScale so the behavior holds for any vault size.
    var smallGraph = nodes.length <= 40;
    var zoomT = Math.max(0, Math.min(1, (scale - 0.8) / 0.6));
    var farT = Math.max(0, Math.min(1, (scale / fitScale - 0.4) / 0.3));
    for (i = 0; i < nodes.length; i++) {
      n = nodes[i];
      var rel = focus && (n === focus || focus.neighbors.has(n));
      var la;
      if (focus) la = rel ? 1 : 0.12;
      else if (n.id === currentId) la = farT;
      else if (smallGraph || n.degree >= 7) la = 0.85 * farT;
      else la = 0.85 * zoomT;
      if (la < 0.03) continue;
      ctx.globalAlpha = la;
      var ly = n.y + n.r + 4.5 / scale;
      ctx.strokeText(n.label, n.x, ly);
      ctx.fillText(n.label, n.x, ly);
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
      var r = n.r + 4 / scale;
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
