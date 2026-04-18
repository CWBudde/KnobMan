import { clampFloat } from "./utils.js";

export function createShapeEditor({ state, el, markDirty, setStatus }) {
  function shapeCommandArity(cmd) {
    switch (cmd) {
      case "M":
      case "L":
        return 2;
      case "Q":
        return 4;
      case "C":
        return 6;
      case "Z":
        return 0;
      default:
        return -1;
    }
  }

  function tokenizeShapePath(path) {
    const tokens = [];
    const s = String(path || "");
    let i = 0;
    while (i < s.length) {
      const ch = s[i];
      if ((ch >= "A" && ch <= "Z") || (ch >= "a" && ch <= "z")) {
        tokens.push(ch);
        i++;
        continue;
      }
      if (ch === "," || ch === " " || ch === "\t" || ch === "\n" || ch === "\r") {
        i++;
        continue;
      }
      let j = i;
      if (s[j] === "+" || s[j] === "-") j++;
      let sawDigit = false;
      let sawDot = false;
      while (j < s.length) {
        const c = s[j];
        if (c >= "0" && c <= "9") {
          sawDigit = true;
          j++;
          continue;
        }
        if (c === "." && !sawDot) {
          sawDot = true;
          j++;
          continue;
        }
        break;
      }
      if (!sawDigit) {
        i++;
        continue;
      }
      tokens.push(s.slice(i, j));
      i = j;
    }
    return tokens;
  }

  function parseSvgShapeCommands(path) {
    const out = [];
    const tokens = tokenizeShapePath(path);
    if (tokens.length === 0) return out;

    let cmd = "";
    let i = 0;
    while (i < tokens.length) {
      const tk = tokens[i];
      if (tk.length === 1 && /[A-Za-z]/.test(tk)) {
        cmd = tk.toUpperCase();
        i++;
        if (cmd === "Z") {
          out.push({ cmd: "Z", values: [] });
        }
        continue;
      }
      if (!cmd) {
        i++;
        continue;
      }
      const arity = shapeCommandArity(cmd);
      if (arity <= 0) {
        i++;
        continue;
      }
      const vals = [];
      while (i < tokens.length && vals.length < arity) {
        const n = Number(tokens[i]);
        if (!Number.isFinite(n)) break;
        vals.push(n);
        i++;
      }
      if (vals.length === arity) {
        out.push({ cmd, values: vals });
        if (cmd === "M") cmd = "L";
        continue;
      }
      break;
    }
    return out;
  }

  function parseKnobShapeToSvgCommands(path) {
    const text = String(path || "").trim();
    if (!text.includes("/") || !text.includes(":")) return [];

    const chunks = text
      .split("/")
      .map((p) => p.trim())
      .filter(Boolean);
    if (chunks.length === 0) return [];
    const knotText = chunks[0]
      .split(":")
      .map((c) => c.trim())
      .filter(Boolean);
    const knots = [];
    knotText.forEach((ch) => {
      const vals = ch.split(",").map((v) => Number(v.trim()));
      if (vals.length !== 6 || vals.some((v) => !Number.isFinite(v))) return;
      knots.push({
        inX: vals[0],
        inY: vals[1],
        pX: vals[2],
        pY: vals[3],
        outX: vals[4],
        outY: vals[5],
      });
    });
    if (knots.length < 2) return [];

    const toPct = (v) => ((v - 128) / 256) * 100;
    const out = [
      {
        cmd: "M",
        values: [toPct(knots[0].pX), toPct(knots[0].pY)],
      },
    ];

    for (let i = 1; i < knots.length; i++) {
      const prev = knots[i - 1];
      const cur = knots[i];
      out.push({
        cmd: "C",
        values: [
          toPct(prev.outX),
          toPct(prev.outY),
          toPct(cur.inX),
          toPct(cur.inY),
          toPct(cur.pX),
          toPct(cur.pY),
        ],
      });
    }

    const first = knots[0];
    const last = knots[knots.length - 1];
    out.push({
      cmd: "C",
      values: [
        toPct(last.outX),
        toPct(last.outY),
        toPct(first.inX),
        toPct(first.inY),
        toPct(first.pX),
        toPct(first.pY),
      ],
    });
    out.push({ cmd: "Z", values: [] });
    return out;
  }

  function normalizeShapeCommands(commands) {
    const out = [];
    let hasClose = false;
    (commands || []).forEach((raw) => {
      const cmd = String(raw && raw.cmd ? raw.cmd : "").toUpperCase();
      const arity = shapeCommandArity(cmd);
      if (arity < 0) return;
      if (cmd === "Z") {
        hasClose = true;
        return;
      }
      const vals = Array.isArray(raw.values) ? raw.values.slice(0, arity) : [];
      if (vals.length !== arity) return;
      out.push({ cmd, values: vals.map((v) => clampFloat(Number(v), 0, 100)) });
    });

    if (out.length === 0 || out[0].cmd !== "M") {
      out.unshift({ cmd: "M", values: [50, 50] });
    }
    const cleaned = [out[0]];
    for (let i = 1; i < out.length; i++) {
      if (out[i].cmd === "M") continue;
      cleaned.push(out[i]);
    }
    if (hasClose) cleaned.push({ cmd: "Z", values: [] });
    return cleaned;
  }

  function formatShapeNumber(v) {
    const n = clampFloat(Number(v), 0, 100);
    if (!Number.isFinite(n)) return "0";
    if (Math.abs(Math.round(n) - n) < 1e-6) return String(Math.round(n));
    return n.toFixed(2).replace(/\.?0+$/, "");
  }

  function serializeShapeCommands(commands) {
    const parts = [];
    (commands || []).forEach((c) => {
      if (!c || !c.cmd) return;
      if (c.cmd === "Z") {
        parts.push("Z");
        return;
      }
      parts.push(c.cmd);
      for (let i = 0; i < c.values.length; i += 2) {
        parts.push(
          formatShapeNumber(c.values[i]),
          formatShapeNumber(c.values[i + 1]),
        );
      }
    });
    return parts.join(" ");
  }

  function isShapeLayerSelected() {
    return Number(window.knobman_getParam(state.selectedLayer, "primType") || 0) === 15;
  }

  function shapeEditorEnabled() {
    return isShapeLayerSelected() && state.shapeCanvas && state.shapeCtx;
  }

  function setShapeOverlayVisible(visible) {
    if (!state.shapeCanvas) return;
    state.shapeCanvas.style.display = visible ? "block" : "none";
    state.shapeCanvas.style.pointerEvents = visible ? "auto" : "none";
  }

  function shapeMetrics() {
    const w = state.shapeCanvas ? state.shapeCanvas.width : state.canvasW;
    const h = state.shapeCanvas ? state.shapeCanvas.height : state.canvasH;
    const pad = 10;
    return {
      w,
      h,
      pad,
      plotW: Math.max(1, w - pad * 2),
      plotH: Math.max(1, h - pad * 2),
    };
  }

  function shapeToCanvas(x, y, m) {
    return {
      x: m.pad + (clampFloat(x, 0, 100) / 100) * m.plotW,
      y: m.h - m.pad - (clampFloat(y, 0, 100) / 100) * m.plotH,
    };
  }

  function canvasToShape(x, y, m) {
    return {
      x: clampFloat(((x - m.pad) / m.plotW) * 100, 0, 100),
      y: clampFloat(((m.h - m.pad - y) / m.plotH) * 100, 0, 100),
    };
  }

  function shapeCurrentPoint(commands) {
    let cur = null;
    let start = null;
    (commands || []).forEach((c) => {
      switch (c.cmd) {
        case "M":
          cur = { x: c.values[0], y: c.values[1] };
          start = { x: cur.x, y: cur.y };
          break;
        case "L":
          cur = { x: c.values[0], y: c.values[1] };
          break;
        case "Q":
          cur = { x: c.values[2], y: c.values[3] };
          break;
        case "C":
          cur = { x: c.values[4], y: c.values[5] };
          break;
        case "Z":
          if (start) cur = { x: start.x, y: start.y };
          break;
        default:
          break;
      }
    });
    return cur || { x: 50, y: 50 };
  }

  function parseShapeFromLayer() {
    const raw = String(window.knobman_getParam(state.selectedLayer, "shape") || "");
    let commands = parseSvgShapeCommands(raw);
    if (commands.length === 0 && raw.includes("/")) {
      commands = parseKnobShapeToSvgCommands(raw);
    }
    return normalizeShapeCommands(commands);
  }

  function syncShapeTextarea(path) {
    const ta = document.querySelector("#paramContent textarea");
    if (!ta || ta === document.activeElement) return;
    ta.value = path;
  }

  function writeShapeCommands(commands) {
    const normalized = normalizeShapeCommands(commands);
    const path = serializeShapeCommands(normalized);
    const ok = window.knobman_setParam(state.selectedLayer, "shape", path);
    if (!ok) return false;
    state.shapeCommands = normalized;
    syncShapeTextarea(path);
    markDirty();
    return true;
  }

  function shapeHandles(commands) {
    const handles = [];
    (commands || []).forEach((c, idx) => {
      if (c.cmd === "M" || c.cmd === "L") {
        handles.push({
          cmdIndex: idx,
          valueIndex: 0,
          role: "anchor",
          x: c.values[0],
          y: c.values[1],
        });
      } else if (c.cmd === "Q") {
        handles.push({
          cmdIndex: idx,
          valueIndex: 0,
          role: "control",
          x: c.values[0],
          y: c.values[1],
        });
        handles.push({
          cmdIndex: idx,
          valueIndex: 2,
          role: "anchor",
          x: c.values[2],
          y: c.values[3],
        });
      } else if (c.cmd === "C") {
        handles.push({
          cmdIndex: idx,
          valueIndex: 0,
          role: "control",
          x: c.values[0],
          y: c.values[1],
        });
        handles.push({
          cmdIndex: idx,
          valueIndex: 2,
          role: "control",
          x: c.values[2],
          y: c.values[3],
        });
        handles.push({
          cmdIndex: idx,
          valueIndex: 4,
          role: "anchor",
          x: c.values[4],
          y: c.values[5],
        });
      }
    });
    return handles;
  }

  function shapeHitHandle(handles, x, y, m) {
    let best = null;
    let bestDist = 1e9;
    handles.forEach((h) => {
      const p = shapeToCanvas(h.x, h.y, m);
      const dx = p.x - x;
      const dy = p.y - y;
      const d2 = dx * dx + dy * dy;
      if (d2 < bestDist) {
        best = h;
        bestDist = d2;
      }
    });
    return bestDist <= 80 ? best : null;
  }

  function shapeEventToCanvasXY(e) {
    if (!state.shapeCanvas) return { x: 0, y: 0 };
    const rect = state.shapeCanvas.getBoundingClientRect();
    const scaleX = rect.width > 0 ? state.shapeCanvas.width / rect.width : 1;
    const scaleY = rect.height > 0 ? state.shapeCanvas.height / rect.height : 1;
    return {
      x: (e.clientX - rect.left) * scaleX,
      y: (e.clientY - rect.top) * scaleY,
    };
  }

  function applyShapeModeButtons() {
    const toolbar = el("shapeToolbar");
    if (!toolbar) return;
    toolbar.querySelectorAll("button[data-shape-mode]").forEach((btn) => {
      btn.classList.toggle("active", btn.dataset.shapeMode === state.shapeMode);
    });
  }

  function addShapeCommandAt(mode, x, y) {
    const cmds = normalizeShapeCommands(state.shapeCommands);
    const p = { x: clampFloat(x, 0, 100), y: clampFloat(y, 0, 100) };
    if (cmds.length === 0) {
      cmds.push({ cmd: "M", values: [p.x, p.y] });
      return cmds;
    }

    if (mode === "M") {
      cmds[0] = { cmd: "M", values: [p.x, p.y] };
      return cmds;
    }

    const cur = shapeCurrentPoint(cmds);
    if (mode === "L") {
      cmds.push({ cmd: "L", values: [p.x, p.y] });
      return cmds;
    }
    if (mode === "Q") {
      const cx = (cur.x + p.x) * 0.5;
      const cy = (cur.y + p.y) * 0.5;
      cmds.push({ cmd: "Q", values: [cx, cy, p.x, p.y] });
      return cmds;
    }
    if (mode === "C") {
      const c1x = cur.x + (p.x - cur.x) / 3;
      const c1y = cur.y + (p.y - cur.y) / 3;
      const c2x = cur.x + (p.x - cur.x) * (2 / 3);
      const c2y = cur.y + (p.y - cur.y) * (2 / 3);
      cmds.push({ cmd: "C", values: [c1x, c1y, c2x, c2y, p.x, p.y] });
      return cmds;
    }
    return cmds;
  }

  function closeShapePath() {
    if (!shapeEditorEnabled()) return;
    const cmds = normalizeShapeCommands(state.shapeCommands);
    if (cmds.length === 0 || cmds[cmds.length - 1].cmd === "Z") return;
    cmds.push({ cmd: "Z", values: [] });
    writeShapeCommands(cmds);
    drawShapeOverlay();
  }

  function deleteShapeSelection() {
    if (!shapeEditorEnabled() || !state.shapeSelectedHandle) return;
    const cmds = normalizeShapeCommands(state.shapeCommands);
    const idx = state.shapeSelectedHandle.cmdIndex;
    if (idx <= 0 || idx >= cmds.length) {
      setStatus("Initial move point cannot be deleted");
      return;
    }
    cmds.splice(idx, 1);
    state.shapeSelectedHandle = null;
    state.shapeDragHandle = null;
    writeShapeCommands(cmds);
    drawShapeOverlay();
  }

  function drawShapeOverlay() {
    if (!state.shapeCtx || !state.shapeCanvas) return;
    state.shapeCtx.clearRect(0, 0, state.shapeCanvas.width, state.shapeCanvas.height);
    if (!shapeEditorEnabled()) return;

    const m = shapeMetrics();
    const fillEnabled = Boolean(window.knobman_getParam(state.selectedLayer, "fill"));
    state.shapeCtx.save();
    state.shapeCtx.strokeStyle = "rgba(220,220,220,0.25)";
    state.shapeCtx.lineWidth = 1;
    for (let i = 0; i <= 4; i++) {
      const x = m.pad + (i / 4) * m.plotW;
      const y = m.pad + (i / 4) * m.plotH;
      state.shapeCtx.beginPath();
      state.shapeCtx.moveTo(x, m.pad);
      state.shapeCtx.lineTo(x, m.h - m.pad);
      state.shapeCtx.stroke();
      state.shapeCtx.beginPath();
      state.shapeCtx.moveTo(m.pad, y);
      state.shapeCtx.lineTo(m.w - m.pad, y);
      state.shapeCtx.stroke();
    }

    state.shapeCtx.beginPath();
    let cur = null;
    let start = null;
    state.shapeCommands.forEach((c) => {
      if (c.cmd === "M") {
        const p = shapeToCanvas(c.values[0], c.values[1], m);
        state.shapeCtx.moveTo(p.x, p.y);
        cur = { x: c.values[0], y: c.values[1] };
        start = { x: cur.x, y: cur.y };
      } else if (c.cmd === "L" && cur) {
        const p = shapeToCanvas(c.values[0], c.values[1], m);
        state.shapeCtx.lineTo(p.x, p.y);
        cur = { x: c.values[0], y: c.values[1] };
      } else if (c.cmd === "Q" && cur) {
        const c1 = shapeToCanvas(c.values[0], c.values[1], m);
        const p = shapeToCanvas(c.values[2], c.values[3], m);
        state.shapeCtx.quadraticCurveTo(c1.x, c1.y, p.x, p.y);
        cur = { x: c.values[2], y: c.values[3] };
      } else if (c.cmd === "C" && cur) {
        const c1 = shapeToCanvas(c.values[0], c.values[1], m);
        const c2 = shapeToCanvas(c.values[2], c.values[3], m);
        const p = shapeToCanvas(c.values[4], c.values[5], m);
        state.shapeCtx.bezierCurveTo(c1.x, c1.y, c2.x, c2.y, p.x, p.y);
        cur = { x: c.values[4], y: c.values[5] };
      } else if (c.cmd === "Z" && start) {
        const p = shapeToCanvas(start.x, start.y, m);
        state.shapeCtx.lineTo(p.x, p.y);
        cur = { x: start.x, y: start.y };
      }
    });
    if (fillEnabled) {
      state.shapeCtx.fillStyle = "rgba(14,99,156,0.15)";
      state.shapeCtx.fill();
    }
    state.shapeCtx.strokeStyle = "#f6d06f";
    state.shapeCtx.lineWidth = 2;
    state.shapeCtx.stroke();

    let prev = null;
    let subStart = null;
    state.shapeCommands.forEach((c) => {
      if (c.cmd === "M") {
        prev = { x: c.values[0], y: c.values[1] };
        subStart = { x: prev.x, y: prev.y };
        return;
      }
      if (c.cmd === "Q" && prev) {
        const a = shapeToCanvas(prev.x, prev.y, m);
        const ctrl = shapeToCanvas(c.values[0], c.values[1], m);
        const end = shapeToCanvas(c.values[2], c.values[3], m);
        state.shapeCtx.setLineDash([4, 4]);
        state.shapeCtx.strokeStyle = "rgba(98,180,255,0.85)";
        state.shapeCtx.beginPath();
        state.shapeCtx.moveTo(a.x, a.y);
        state.shapeCtx.lineTo(ctrl.x, ctrl.y);
        state.shapeCtx.lineTo(end.x, end.y);
        state.shapeCtx.stroke();
        state.shapeCtx.setLineDash([]);
        prev = { x: c.values[2], y: c.values[3] };
        return;
      }
      if (c.cmd === "C" && prev) {
        const a = shapeToCanvas(prev.x, prev.y, m);
        const c1 = shapeToCanvas(c.values[0], c.values[1], m);
        const c2 = shapeToCanvas(c.values[2], c.values[3], m);
        const end = shapeToCanvas(c.values[4], c.values[5], m);
        state.shapeCtx.setLineDash([4, 4]);
        state.shapeCtx.strokeStyle = "rgba(98,180,255,0.85)";
        state.shapeCtx.beginPath();
        state.shapeCtx.moveTo(a.x, a.y);
        state.shapeCtx.lineTo(c1.x, c1.y);
        state.shapeCtx.moveTo(end.x, end.y);
        state.shapeCtx.lineTo(c2.x, c2.y);
        state.shapeCtx.stroke();
        state.shapeCtx.setLineDash([]);
        prev = { x: c.values[4], y: c.values[5] };
        return;
      }
      if (c.cmd === "L") {
        prev = { x: c.values[0], y: c.values[1] };
        return;
      }
      if (c.cmd === "Z" && subStart) {
        prev = { x: subStart.x, y: subStart.y };
      }
    });

    const handles = shapeHandles(state.shapeCommands);
    handles.forEach((h) => {
      const p = shapeToCanvas(h.x, h.y, m);
      const selected = Boolean(
        state.shapeSelectedHandle &&
          state.shapeSelectedHandle.cmdIndex === h.cmdIndex &&
          state.shapeSelectedHandle.valueIndex === h.valueIndex,
      );
      state.shapeCtx.beginPath();
      state.shapeCtx.arc(p.x, p.y, selected ? 5 : 4, 0, Math.PI * 2);
      state.shapeCtx.fillStyle = h.role === "anchor" ? "#ffe28a" : "#78c9ff";
      state.shapeCtx.fill();
      state.shapeCtx.strokeStyle = "#101010";
      state.shapeCtx.lineWidth = 1;
      state.shapeCtx.stroke();
    });

    state.shapeCtx.fillStyle = "#fff";
    state.shapeCtx.font = "11px system-ui, sans-serif";
    state.shapeCtx.fillText("Shape Edit Overlay", m.pad + 6, m.pad + 14);
    state.shapeCtx.restore();
  }

  function refreshShapeEditor() {
    const panel = el("shapeEditorPanel");
    if (!panel) return;
    const active = isShapeLayerSelected();
    panel.classList.toggle("disabled", !active);
    setShapeOverlayVisible(active);

    const hint = el("shapeHelp");
    if (!active) {
      state.shapeCommands = [];
      state.shapeDragHandle = null;
      state.shapeSelectedHandle = null;
      if (hint) {
        hint.textContent =
          "Select a layer with Primitive = Shape to edit its path.";
      }
      if (state.shapeCtx && state.shapeCanvas) {
        state.shapeCtx.clearRect(0, 0, state.shapeCanvas.width, state.shapeCanvas.height);
      }
      return;
    }
    if (hint) {
      hint.textContent =
        "Click to add points. Drag handles to edit. Right click or Delete removes selected segment.";
    }
    state.shapeCommands = parseShapeFromLayer();
    drawShapeOverlay();
  }

  function onShapePointerDown(e) {
    if (!shapeEditorEnabled() || e.button !== 0) return;
    const m = shapeMetrics();
    const pos = shapeEventToCanvasXY(e);
    const handles = shapeHandles(state.shapeCommands);
    const hit = shapeHitHandle(handles, pos.x, pos.y, m);
    if (hit) {
      state.shapeSelectedHandle = {
        cmdIndex: hit.cmdIndex,
        valueIndex: hit.valueIndex,
      };
      state.shapeDragHandle = {
        cmdIndex: hit.cmdIndex,
        valueIndex: hit.valueIndex,
      };
      state.shapeCanvas.setPointerCapture(e.pointerId);
      drawShapeOverlay();
      return;
    }
    const p = canvasToShape(pos.x, pos.y, m);
    const next = addShapeCommandAt(state.shapeMode, p.x, p.y);
    writeShapeCommands(next);
    drawShapeOverlay();
  }

  function onShapePointerMove(e) {
    if (!shapeEditorEnabled() || !state.shapeDragHandle) return;
    const m = shapeMetrics();
    const pos = shapeEventToCanvasXY(e);
    const p = canvasToShape(pos.x, pos.y, m);
    const cmds = normalizeShapeCommands(state.shapeCommands);
    const cmd = cmds[state.shapeDragHandle.cmdIndex];
    if (!cmd || cmd.cmd === "Z") return;
    const i = state.shapeDragHandle.valueIndex;
    if (i < 0 || i + 1 >= cmd.values.length) return;
    cmd.values[i] = p.x;
    cmd.values[i + 1] = p.y;
    writeShapeCommands(cmds);
    drawShapeOverlay();
  }

  function onShapePointerUp(e) {
    if (!state.shapeCanvas) return;
    if (state.shapeDragHandle) {
      state.shapeCanvas.releasePointerCapture(e.pointerId);
    }
    state.shapeDragHandle = null;
  }

  function onShapeContextMenu(e) {
    if (!shapeEditorEnabled()) return;
    e.preventDefault();
    const m = shapeMetrics();
    const pos = shapeEventToCanvasXY(e);
    const hit = shapeHitHandle(shapeHandles(state.shapeCommands), pos.x, pos.y, m);
    if (!hit) return;
    state.shapeSelectedHandle = {
      cmdIndex: hit.cmdIndex,
      valueIndex: hit.valueIndex,
    };
    deleteShapeSelection();
  }

  function initShapeEditor() {
    state.shapeCanvas = el("shapeOverlay");
    if (!state.shapeCanvas) return;
    state.shapeCtx = state.shapeCanvas.getContext("2d");
    setShapeOverlayVisible(false);

    state.shapeCanvas.addEventListener("pointerdown", onShapePointerDown);
    state.shapeCanvas.addEventListener("pointermove", onShapePointerMove);
    state.shapeCanvas.addEventListener("pointerup", onShapePointerUp);
    state.shapeCanvas.addEventListener("pointercancel", onShapePointerUp);
    state.shapeCanvas.addEventListener("contextmenu", onShapeContextMenu);

    const toolbar = el("shapeToolbar");
    if (toolbar) {
      toolbar.querySelectorAll("button[data-shape-mode]").forEach((btn) => {
        btn.addEventListener("click", () => {
          state.shapeMode = btn.dataset.shapeMode || "L";
          applyShapeModeButtons();
        });
      });
    }
    applyShapeModeButtons();

    const closeBtn = el("shapeClosePath");
    if (closeBtn) closeBtn.addEventListener("click", closeShapePath);
    const delBtn = el("shapeDeleteSeg");
    if (delBtn) delBtn.addEventListener("click", deleteShapeSelection);
  }

  return {
    deleteShapeSelection,
    drawShapeOverlay,
    initShapeEditor,
    isShapeLayerSelected,
    refreshShapeEditor,
  };
}
