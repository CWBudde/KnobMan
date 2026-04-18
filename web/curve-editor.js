import { CURVE_POINT_LIMIT } from "./state.js";
import {
  clampFloat,
  clampInt,
  syncCanvasBackingToDisplaySize,
} from "./utils.js";

export function createCurveEditor({ state, el, markDirty, setStatus }) {
  function curveCanvasMetrics() {
    const w = state.curveCanvas ? state.curveCanvas.width : 320;
    const h = state.curveCanvas ? state.curveCanvas.height : 200;
    const pad = { left: 28, right: 12, top: 12, bottom: 22 };
    return {
      w,
      h,
      pad,
      plotW: Math.max(1, w - pad.left - pad.right),
      plotH: Math.max(1, h - pad.top - pad.bottom),
    };
  }

  function syncCurveCanvasSize() {
    syncCanvasBackingToDisplaySize(state.curveCanvas);
  }

  function curveToCanvasPoint(tm, lv, m) {
    return {
      x: m.pad.left + (tm / 100) * m.plotW,
      y: m.h - m.pad.bottom - (lv / 100) * m.plotH,
    };
  }

  function canvasToCurvePoint(x, y, m) {
    const tx = (x - m.pad.left) / m.plotW;
    const ty = (m.h - m.pad.bottom - y) / m.plotH;
    return {
      t: clampInt(tx * 100, 0, 100),
      l: clampInt(ty * 100, 0, 100),
    };
  }

  function normalizeCurvePoints(points) {
    const out = [];
    (points || []).forEach((p) => {
      const t = clampInt(Number(p.t), 0, 100);
      const l = clampInt(Number(p.l), 0, 100);
      if (!Number.isFinite(t) || !Number.isFinite(l)) return;
      out.push({ t, l });
    });
    out.sort((a, b) => a.t - b.t);

    const unique = [];
    out.forEach((p) => {
      if (unique.length === 0 || unique[unique.length - 1].t !== p.t) {
        unique.push(p);
      }
    });

    if (unique.length === 0) {
      unique.push({ t: 0, l: 0 }, { t: 100, l: 100 });
    }
    if (unique[0].t !== 0) unique.unshift({ t: 0, l: unique[0].l });
    if (unique[unique.length - 1].t !== 100) {
      unique.push({ t: 100, l: unique[unique.length - 1].l });
    }

    unique[0].t = 0;
    unique[unique.length - 1].t = 100;

    while (unique.length > CURVE_POINT_LIMIT) {
      unique.splice(unique.length - 2, 1);
    }

    for (let i = 1; i < unique.length - 1; i++) {
      const minT = unique[i - 1].t + 1;
      const maxT = unique[i + 1].t - 1;
      unique[i].t = clampInt(unique[i].t, minT, maxT);
    }
    return unique;
  }

  function readSelectedCurve() {
    const raw = window.knobman_getCurve
      ? window.knobman_getCurve(state.selectedCurve)
      : null;
    const rawTm = raw && Array.isArray(raw.tm) ? raw.tm : [];
    const rawLv = raw && Array.isArray(raw.lv) ? raw.lv : [];
    const points = [];
    for (let i = 0; i < CURVE_POINT_LIMIT; i++) {
      const t = Number(rawTm[i]);
      const l = Number(rawLv[i]);
      if (!Number.isFinite(t) || !Number.isFinite(l)) continue;
      if (t < 0 || l < 0) continue;
      points.push({ t, l });
    }
    return {
      points: normalizeCurvePoints(points),
      stepReso: clampInt(
        Number(raw && raw.stepReso != null ? raw.stepReso : 0),
        0,
        64,
      ),
    };
  }

  function writeSelectedCurve(points, stepReso) {
    if (!window.knobman_setCurve) return false;
    const clean = normalizeCurvePoints(points);
    const tm = new Array(CURVE_POINT_LIMIT).fill(-1);
    const lv = new Array(CURVE_POINT_LIMIT).fill(-1);
    tm[0] = 0;
    lv[0] = clean[0].l;
    tm[CURVE_POINT_LIMIT - 1] = 100;
    lv[CURVE_POINT_LIMIT - 1] = clean[clean.length - 1].l;
    const interior = clean.slice(1, -1).slice(0, CURVE_POINT_LIMIT - 2);
    interior.forEach((p, i) => {
      tm[i + 1] = p.t;
      lv[i + 1] = p.l;
    });

    const ok = window.knobman_setCurve(state.selectedCurve, {
      tm,
      lv,
      stepReso: clampInt(Number(stepReso), 0, 64),
    });
    if (ok) markDirty();
    return ok;
  }

  function frameRatioForCurve() {
    const renderFrames = parseInt(el("prefFrames").value, 10) || 1;
    if (renderFrames <= 1) return 0;
    return clampFloat(state.currentFrame / (renderFrames - 1), 0, 1);
  }

  function syncCurveTabState() {
    const tabs = el("curveTabs");
    if (!tabs) return;
    tabs.querySelectorAll("button").forEach((btn) => {
      const idx = parseInt(btn.dataset.curve, 10) || 1;
      btn.classList.toggle("active", idx === state.selectedCurve);
    });
  }

  function drawCurveEditor() {
    if (!state.curveCanvas || !state.curveCtx) return;
    syncCurveCanvasSize();
    const m = curveCanvasMetrics();
    const curveState = readSelectedCurve();

    const stepInput = el("curveStepReso");
    if (stepInput && document.activeElement !== stepInput) {
      stepInput.value = String(curveState.stepReso);
    }

    state.curveCtx.clearRect(0, 0, m.w, m.h);
    state.curveCtx.fillStyle = "#141414";
    state.curveCtx.fillRect(0, 0, m.w, m.h);

    state.curveCtx.strokeStyle = "#2f2f2f";
    state.curveCtx.lineWidth = 1;
    for (let i = 0; i <= 4; i++) {
      const x = m.pad.left + (i / 4) * m.plotW;
      const y = m.pad.top + (i / 4) * m.plotH;
      state.curveCtx.beginPath();
      state.curveCtx.moveTo(x, m.pad.top);
      state.curveCtx.lineTo(x, m.h - m.pad.bottom);
      state.curveCtx.stroke();
      state.curveCtx.beginPath();
      state.curveCtx.moveTo(m.pad.left, y);
      state.curveCtx.lineTo(m.w - m.pad.right, y);
      state.curveCtx.stroke();
    }

    state.curveCtx.strokeStyle = "#4b4b4b";
    state.curveCtx.strokeRect(m.pad.left, m.pad.top, m.plotW, m.plotH);

    const ratio = frameRatioForCurve();
    const evalLv = clampFloat(
      Number(
        window.knobman_evalCurve
          ? window.knobman_evalCurve(state.selectedCurve, ratio)
          : 0,
      ),
      0,
      100,
    );
    const frameX = curveToCanvasPoint(ratio * 100, 0, m).x;
    const evalY = curveToCanvasPoint(0, evalLv, m).y;

    state.curveCtx.save();
    state.curveCtx.setLineDash([4, 4]);
    state.curveCtx.strokeStyle = "#f3b24a";
    state.curveCtx.beginPath();
    state.curveCtx.moveTo(frameX, m.pad.top);
    state.curveCtx.lineTo(frameX, m.h - m.pad.bottom);
    state.curveCtx.stroke();

    state.curveCtx.strokeStyle = "#5db2ff";
    state.curveCtx.beginPath();
    state.curveCtx.moveTo(m.pad.left, evalY);
    state.curveCtx.lineTo(m.w - m.pad.right, evalY);
    state.curveCtx.stroke();
    state.curveCtx.restore();

    state.curveCtx.strokeStyle = "#7ec7ff";
    state.curveCtx.lineWidth = 2;
    state.curveCtx.beginPath();
    curveState.points.forEach((p, i) => {
      const c = curveToCanvasPoint(p.t, p.l, m);
      if (i === 0) state.curveCtx.moveTo(c.x, c.y);
      else state.curveCtx.lineTo(c.x, c.y);
    });
    state.curveCtx.stroke();

    curveState.points.forEach((p, i) => {
      const c = curveToCanvasPoint(p.t, p.l, m);
      const isEndpoint = i === 0 || i === curveState.points.length - 1;
      state.curveCtx.beginPath();
      state.curveCtx.arc(
        c.x,
        c.y,
        i === state.curveSelectedPoint ? 5 : 4,
        0,
        Math.PI * 2,
      );
      state.curveCtx.fillStyle = isEndpoint ? "#e4e4e4" : "#f3b24a";
      state.curveCtx.fill();
      state.curveCtx.strokeStyle = "#000";
      state.curveCtx.lineWidth = 1;
      state.curveCtx.stroke();
    });

    state.curveCtx.fillStyle = "#a0a0a0";
    state.curveCtx.font = "10px system-ui, sans-serif";
    state.curveCtx.fillText("0", m.pad.left - 6, m.h - 6);
    state.curveCtx.fillText("100", m.w - m.pad.right - 18, m.h - 6);
    state.curveCtx.fillText("100", 3, m.pad.top + 3);
    state.curveCtx.fillStyle = "#7d7d7d";
    state.curveCtx.fillText(`Frame ${state.currentFrame}`, m.pad.left + 6, m.h - 6);
    state.curveCtx.fillText(
      `Value ${evalLv.toFixed(1)}`,
      m.pad.left + 78,
      m.h - 6,
    );
  }

  function refreshCurveEditor() {
    syncCurveTabState();
    drawCurveEditor();
  }

  function focusCurve(curveIdx) {
    state.selectedCurve = clampInt(Number(curveIdx), 1, 8);
    state.curveSelectedPoint = -1;
    refreshCurveEditor();
  }

  function curveHitPoint(points, x, y) {
    const m = curveCanvasMetrics();
    let best = -1;
    let bestDist = 1e9;
    points.forEach((p, i) => {
      const c = curveToCanvasPoint(p.t, p.l, m);
      const dx = c.x - x;
      const dy = c.y - y;
      const d2 = dx * dx + dy * dy;
      if (d2 < bestDist) {
        best = i;
        bestDist = d2;
      }
    });
    return bestDist <= 64 ? best : -1;
  }

  function curveEventToCanvasXY(e) {
    if (!state.curveCanvas) return { x: 0, y: 0 };
    const rect = state.curveCanvas.getBoundingClientRect();
    const scaleX = rect.width > 0 ? state.curveCanvas.width / rect.width : 1;
    const scaleY = rect.height > 0 ? state.curveCanvas.height / rect.height : 1;
    return {
      x: (e.clientX - rect.left) * scaleX,
      y: (e.clientY - rect.top) * scaleY,
    };
  }

  function onCurvePointerDown(e) {
    if (e.button !== 0 || !state.curveCanvas) return;
    const m = curveCanvasMetrics();
    const curveState = readSelectedCurve();
    const pos = curveEventToCanvasXY(e);
    const hit = curveHitPoint(curveState.points, pos.x, pos.y);

    if (hit >= 0) {
      state.curveDragPoint = hit;
      state.curveSelectedPoint = hit;
      state.curveCanvas.setPointerCapture(e.pointerId);
      refreshCurveEditor();
      return;
    }

    if (curveState.points.length >= CURVE_POINT_LIMIT) {
      setStatus("Curve has reached the 12-point limit");
      return;
    }

    const p = canvasToCurvePoint(pos.x, pos.y, m);
    if (p.t <= 0 || p.t >= 100) return;
    let insertAt = curveState.points.findIndex((pt) => pt.t > p.t);
    if (insertAt < 0) insertAt = curveState.points.length - 1;
    curveState.points.splice(insertAt, 0, p);
    state.curveSelectedPoint = insertAt;
    writeSelectedCurve(curveState.points, curveState.stepReso);
    refreshCurveEditor();
  }

  function onCurvePointerMove(e) {
    if (state.curveDragPoint < 0 || !state.curveCanvas) return;
    const m = curveCanvasMetrics();
    const curveState = readSelectedCurve();
    if (state.curveDragPoint >= curveState.points.length) {
      state.curveDragPoint = -1;
      return;
    }

    const pos = curveEventToCanvasXY(e);
    const p = canvasToCurvePoint(pos.x, pos.y, m);
    const i = state.curveDragPoint;
    if (i === 0) {
      p.t = 0;
    } else if (i === curveState.points.length - 1) {
      p.t = 100;
    } else {
      p.t = clampInt(
        p.t,
        curveState.points[i - 1].t + 1,
        curveState.points[i + 1].t - 1,
      );
    }
    curveState.points[i] = p;
    writeSelectedCurve(curveState.points, curveState.stepReso);
    refreshCurveEditor();
  }

  function onCurvePointerUp(e) {
    if (!state.curveCanvas) return;
    if (state.curveDragPoint >= 0) {
      state.curveCanvas.releasePointerCapture(e.pointerId);
    }
    state.curveDragPoint = -1;
  }

  function onCurveContextMenu(e) {
    e.preventDefault();
    const curveState = readSelectedCurve();
    const pos = curveEventToCanvasXY(e);
    const hit = curveHitPoint(curveState.points, pos.x, pos.y);
    if (hit <= 0 || hit >= curveState.points.length - 1) return;
    curveState.points.splice(hit, 1);
    state.curveSelectedPoint = -1;
    writeSelectedCurve(curveState.points, curveState.stepReso);
    refreshCurveEditor();
  }

  function deleteSelectedCurvePoint() {
    const curveState = readSelectedCurve();
    if (
      state.curveSelectedPoint <= 0 ||
      state.curveSelectedPoint >= curveState.points.length - 1
    ) {
      return;
    }
    curveState.points.splice(state.curveSelectedPoint, 1);
    state.curveSelectedPoint = -1;
    writeSelectedCurve(curveState.points, curveState.stepReso);
    refreshCurveEditor();
  }

  function initCurveEditor() {
    const tabs = el("curveTabs");
    state.curveCanvas = el("curveCanvas");
    if (!tabs || !state.curveCanvas) return;
    state.curveCtx = state.curveCanvas.getContext("2d");

    const details = el("curveEditorDetails");
    if (details) {
      details.addEventListener("toggle", () => {
        if (details.open) refreshCurveEditor();
      });
    }

    tabs.innerHTML = "";
    for (let i = 1; i <= 8; i++) {
      const btn = document.createElement("button");
      btn.type = "button";
      btn.dataset.curve = String(i);
      btn.textContent = `Curve ${i}`;
      btn.addEventListener("click", () => focusCurve(i));
      tabs.appendChild(btn);
    }

    const stepInput = el("curveStepReso");
    if (stepInput) {
      stepInput.addEventListener("change", () => {
        const curveState = readSelectedCurve();
        const step = clampInt(parseInt(stepInput.value, 10) || 0, 0, 64);
        writeSelectedCurve(curveState.points, step);
        refreshCurveEditor();
      });
    }

    const deleteBtn = el("curveDeletePoint");
    if (deleteBtn) deleteBtn.addEventListener("click", deleteSelectedCurvePoint);

    state.curveCanvas.addEventListener("pointerdown", onCurvePointerDown);
    state.curveCanvas.addEventListener("pointermove", onCurvePointerMove);
    state.curveCanvas.addEventListener("pointerup", onCurvePointerUp);
    state.curveCanvas.addEventListener("pointercancel", onCurvePointerUp);
    state.curveCanvas.addEventListener("contextmenu", onCurveContextMenu);

    syncCurveTabState();
    refreshCurveEditor();
  }

  return {
    initCurveEditor,
    refreshCurveEditor,
    focusCurve,
  };
}
