import { createCurveEditor } from "./curve-editor.js";
import { createDomCache } from "./dom.js";
import { createProjects } from "./projects.js";
import { createShapeEditor } from "./shape-editor.js";
import {
  createAppState,
  EFFECT_DEFS,
  EFFECT_SECTIONS,
  LAYER_PREVIEW_SIZE,
  PARAM_DEFS,
  PARAMS_BY_PRIM_TYPE,
} from "./state.js";
import {
  canvasToBlobSync,
  clampInt,
  getColorEffectRows,
  getDropShadowEffectRows,
  getEmbossEffectRows,
  getInnerShadowEffectRows,
  getLayerControlLabel,
  getLayerToggleLabel,
  getSpecularHighlightRows,
  getTransformEffectRows,
  hasBoundedRangeControl,
  isCurveSelectorField,
  syncCanvasElementSize,
} from "./utils.js";

const state = createAppState();
const el = createDomCache(document);

const curveEditor = createCurveEditor({
  state,
  el,
  markDirty,
  setStatus,
});
const shapeEditor = createShapeEditor({
  state,
  el,
  markDirty,
  setStatus,
});
const projects = createProjects({
  state,
  el,
  invalidateLayerPreviews,
  markDirty,
  refreshFromDoc,
  refreshParamPanel,
  setStatus,
  syncAspectRatioFromInputs,
});

function isDetachedPreviewOpen() {
  try {
    return Boolean(
      state.detachedPreviewWindow && !state.detachedPreviewWindow.closed,
    );
  } catch (_) {
    return false;
  }
}

function detachedPreviewRenderFrames() {
  return Math.max(1, parseInt(el("prefFrames").value, 10) || 1);
}

function detachedPreviewDelayMs() {
  const duration = Math.max(1, parseInt(el("prefDuration").value, 10) || 100);
  const frames = detachedPreviewRenderFrames();
  return Math.max(16, Math.round(duration / Math.max(1, frames)));
}

function cleanupDetachedPreviewWindow() {
  if (state.detachedPreviewTimer) {
    clearInterval(state.detachedPreviewTimer);
    state.detachedPreviewTimer = null;
  }
  state.detachedPreviewPlaying = false;
  state.detachedPreviewWindow = null;
  const btn = el("btnPreviewWin");
  if (btn) btn.textContent = "Preview";
}

function closeDetachedPreviewWindow() {
  if (isDetachedPreviewOpen()) {
    state.detachedPreviewWindow.close();
  }
  cleanupDetachedPreviewWindow();
}

function renderDetachedPreviewFrame(frame) {
  if (!isDetachedPreviewOpen() || !window.knobman_renderFrameRaw) return;
  const raw = window.knobman_renderFrameRaw(frame);
  if (!raw || !raw.data) return;
  const doc = state.detachedPreviewWindow.document;
  const canvas = doc.getElementById("detachedPreviewCanvas");
  const info = doc.getElementById("detachedPreviewInfo");
  if (!canvas) return;

  const width = Number(raw.width) || 1;
  const height = Number(raw.height) || 1;
  syncCanvasElementSize(canvas, width, height);

  const ctx = canvas.getContext("2d");
  if (!ctx) return;
  ctx.imageSmoothingEnabled = false;
  const px = new Uint8ClampedArray(raw.data.length);
  px.set(raw.data);
  ctx.putImageData(new ImageData(px, width, height), 0, 0);

  if (info) {
    const shown = (Number(raw.frame) || 0) + 1;
    const total = detachedPreviewRenderFrames();
    info.textContent = `Frame ${shown}/${total} · ${width}x${height}`;
  }
}

function advanceDetachedPreviewFrame() {
  const total = detachedPreviewRenderFrames();
  if (total <= 1) {
    state.detachedPreviewFrame = 0;
    return;
  }
  if (el("prefBiDir").checked) {
    state.detachedPreviewFrame += state.detachedPreviewDir;
    if (state.detachedPreviewFrame >= total - 1) {
      state.detachedPreviewFrame = total - 1;
      state.detachedPreviewDir = -1;
    } else if (state.detachedPreviewFrame <= 0) {
      state.detachedPreviewFrame = 0;
      state.detachedPreviewDir = 1;
    }
    return;
  }
  state.detachedPreviewFrame = (state.detachedPreviewFrame + 1) % total;
}

function detachedPreviewTick() {
  if (!isDetachedPreviewOpen()) {
    cleanupDetachedPreviewWindow();
    return;
  }
  renderDetachedPreviewFrame(state.detachedPreviewFrame);
  advanceDetachedPreviewFrame();
}

function setDetachedPreviewPlaying(playing) {
  if (!isDetachedPreviewOpen()) return;
  state.detachedPreviewPlaying = Boolean(playing);
  if (state.detachedPreviewTimer) {
    clearInterval(state.detachedPreviewTimer);
    state.detachedPreviewTimer = null;
  }
  if (state.detachedPreviewPlaying) {
    state.detachedPreviewTimer = setInterval(
      detachedPreviewTick,
      detachedPreviewDelayMs(),
    );
  }
  const toggle =
    state.detachedPreviewWindow.document.getElementById(
      "detachedPreviewToggle",
    );
  if (toggle) {
    toggle.textContent = state.detachedPreviewPlaying ? "Pause" : "Play";
  }
}

function refreshDetachedPreviewNow() {
  if (!isDetachedPreviewOpen()) return;
  if (state.detachedPreviewPlaying) {
    setDetachedPreviewPlaying(true);
    return;
  }
  state.detachedPreviewFrame = clampInt(
    state.currentFrame,
    0,
    detachedPreviewRenderFrames() - 1,
  );
  renderDetachedPreviewFrame(state.detachedPreviewFrame);
}

function openDetachedPreviewWindow() {
  if (isDetachedPreviewOpen()) return;
  const win = window.open(
    "",
    "knobmanDetachedPreview",
    "popup=yes,width=520,height=420",
  );
  if (!win) {
    setStatus("Popup blocked: allow popups to open detached preview");
    return;
  }
  win.document.open();
  win.document.write(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>KnobMan Preview</title>
  <style>
    :root { color-scheme: dark; }
    body { margin: 0; background: #1e1e1e; color: #d0d0d0; font: 12px/1.4 system-ui, sans-serif; display: flex; flex-direction: column; height: 100vh; }
    header { display: flex; gap: 8px; align-items: center; padding: 8px; border-bottom: 1px solid #3a3a3a; background: #252526; }
    button { background: #2d2d30; color: #ddd; border: 1px solid #4a4a4a; border-radius: 3px; padding: 3px 10px; cursor: pointer; }
    button:hover { background: #0e639c; border-color: #0e639c; color: #fff; }
    #detachedPreviewInfo { color: #9a9a9a; margin-left: auto; }
    main { flex: 1; display: flex; align-items: center; justify-content: center; overflow: auto; padding: 12px; }
    canvas { image-rendering: pixelated; border: 1px solid #3a3a3a; background: #111; }
  </style>
</head>
<body>
  <header>
    <button id="detachedPreviewToggle" type="button">Pause</button>
    <button id="detachedPreviewSync" type="button">Sync Frame</button>
    <button id="detachedPreviewClose" type="button">Close</button>
    <span id="detachedPreviewInfo"></span>
  </header>
  <main><canvas id="detachedPreviewCanvas" width="64" height="64"></canvas></main>
</body>
</html>`);
  win.document.close();

  state.detachedPreviewWindow = win;
  state.detachedPreviewFrame = clampInt(
    state.currentFrame,
    0,
    detachedPreviewRenderFrames() - 1,
  );
  state.detachedPreviewDir = 1;

  const toggle = win.document.getElementById("detachedPreviewToggle");
  const syncBtn = win.document.getElementById("detachedPreviewSync");
  const closeBtn = win.document.getElementById("detachedPreviewClose");
  if (toggle) {
    toggle.addEventListener("click", () => {
      setDetachedPreviewPlaying(!state.detachedPreviewPlaying);
    });
  }
  if (syncBtn) {
    syncBtn.addEventListener("click", () => {
      state.detachedPreviewFrame = clampInt(
        state.currentFrame,
        0,
        detachedPreviewRenderFrames() - 1,
      );
      state.detachedPreviewDir = 1;
      renderDetachedPreviewFrame(state.detachedPreviewFrame);
    });
  }
  if (closeBtn) {
    closeBtn.addEventListener("click", closeDetachedPreviewWindow);
  }

  win.addEventListener("beforeunload", cleanupDetachedPreviewWindow);

  setDetachedPreviewPlaying(true);
  renderDetachedPreviewFrame(state.detachedPreviewFrame);
  const btn = el("btnPreviewWin");
  if (btn) btn.textContent = "Close Preview";
}

function toggleDetachedPreviewWindow() {
  if (isDetachedPreviewOpen()) {
    closeDetachedPreviewWindow();
    return;
  }
  openDetachedPreviewWindow();
}

function releaseLayerPreviewCache() {
  state.layerPreviewCache.forEach((entry) => {
    if (entry && entry.url) URL.revokeObjectURL(entry.url);
  });
  state.layerPreviewCache.clear();
}

function invalidateLayerPreviews() {
  state.layerPreviewRevision += 1;
  releaseLayerPreviewCache();
}

function layerPreviewCacheKey(layerIndex, frame) {
  return `${state.layerPreviewRevision}:${layerIndex}:${frame}`;
}

function buildLayerPreviewFromRaw(raw) {
  if (!raw || !raw.data || !raw.width || !raw.height) return null;
  const width = Number(raw.width) || 0;
  const height = Number(raw.height) || 0;
  if (width <= 0 || height <= 0) return null;
  const src = raw.data;
  const pixels = new Uint8ClampedArray(src.length);
  pixels.set(src);
  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;
  const ctx = canvas.getContext("2d");
  ctx.putImageData(new ImageData(pixels, width, height), 0, 0);
  const blob = canvasToBlobSync(canvas);
  if (!blob) return null;
  return {
    width,
    height,
    url: URL.createObjectURL(blob),
  };
}

function getLayerPreviewCached(layerIndex, frame) {
  const key = layerPreviewCacheKey(layerIndex, frame);
  const cached = state.layerPreviewCache.get(key);
  if (cached) return cached;
  if (!window.knobman_getLayerPreview) return null;
  const raw = window.knobman_getLayerPreview(
    layerIndex,
    frame,
    LAYER_PREVIEW_SIZE,
  );
  const built = buildLayerPreviewFromRaw(raw);
  if (!built) return null;
  state.layerPreviewCache.set(key, built);
  return built;
}

function renderLayerPreviewsAsync(jobs) {
  const token = ++state.layerPreviewToken;
  (async () => {
    for (let i = 0; i < jobs.length; i++) {
      if (token !== state.layerPreviewToken) return;
      if (i > 0) await new Promise((resolve) => setTimeout(resolve, 0));
      const job = jobs[i];
      if (!job || !job.img || !job.img.isConnected) continue;
      const preview = getLayerPreviewCached(job.layerIndex, job.frame);
      if (!preview) continue;
      if (token !== state.layerPreviewToken || !job.img.isConnected) return;
      job.img.src = preview.url;
      job.img.classList.add("ready");
    }
  })();
}

function onWasmReady() {
  el("loading").style.display = "none";
  el("app").style.display = "flex";

  wireControls();
  projects.wireWelcomeOverlay();
  projects.renderSampleList();

  window.knobman_init(64, 64, state.zoomFactor);
  el("zoomSelect").value = String(state.zoomFactor);
  curveEditor.initCurveEditor();
  shapeEditor.initShapeEditor();

  const restored = projects.restoreSession();
  refreshFromDoc();
  projects.ensureBuiltinTextures().then(() => {
    refreshParamPanel();
    markDirty();
  });
  scheduleRender();

  if (!restored && projects.shouldShowWelcome()) {
    projects.openWelcomeOverlay();
    setStatus("Ready");
  } else {
    setStatus(restored ? "Session restored" : "Ready");
  }
}

function refreshFromDoc() {
  syncPrefsFromGo();
  syncCanvasSize();
  refreshLayerList();
  refreshParamPanel();
  shapeEditor.refreshShapeEditor();

  const renderFrames = parseInt(el("prefFrames").value, 10) || 1;
  const visibleFrames = Math.max(1, renderFrames);
  const maxPreviewFrame = Math.max(0, visibleFrames - 1);
  if (state.currentFrame > maxPreviewFrame) state.currentFrame = maxPreviewFrame;
  if (state.currentFrame < 0) state.currentFrame = 0;
  el("frameSlider").max = Math.max(0, visibleFrames - 1);
  el("frameSlider").value = state.currentFrame;
  el("frameValue").textContent = String(state.currentFrame);
  window.knobman_setPreviewFrame(state.currentFrame);
  curveEditor.refreshCurveEditor();
  refreshDetachedPreviewNow();
  markDirty();
}

function syncCanvasSize() {
  const dims = window.knobman_getDimensions();
  if (!dims) return;
  const targetW = Math.max(1, Math.round(Number(dims.width) || 0));
  const targetH = Math.max(1, Math.round(Number(dims.height) || 0));
  const docSizeChanged =
    state.canvasW !== targetW || state.canvasH !== targetH;

  syncCanvasElementSize(el("knobCanvas"), targetW, targetH);

  const overlay = el("shapeOverlay");
  if (overlay) {
    syncCanvasElementSize(overlay, targetW, targetH);
  }

  state.canvasW = targetW;
  state.canvasH = targetH;
  if (
    !docSizeChanged &&
    state.pixelBuf &&
    state.imageData &&
    state.imageData.width === state.canvasW &&
    state.imageData.height === state.canvasH
  ) {
    return;
  }

  state.imageData = new ImageData(state.canvasW, state.canvasH);
  state.pixelBuf = new Uint8Array(state.canvasW * state.canvasH * 4);
}

function scheduleRender() {
  if (!state.rafPending) {
    state.rafPending = true;
    requestAnimationFrame(renderFrame);
  }
}

function renderFrame() {
  state.rafPending = false;
  if (!state.wasmReady || !state.dirty) return;
  state.dirty = false;

  try {
    syncCanvasSize();
    if (!window.knobman_render || !state.pixelBuf || !state.imageData) return;
    const t0 = performance.now();
    window.knobman_render(state.pixelBuf);
    state.lastRenderMs = Math.round(performance.now() - t0);
    state.imageData.data.set(state.pixelBuf);

    const canvas = el("knobCanvas");
    const ctx = canvas.getContext("2d");
    if (!ctx) return;
    ctx.imageSmoothingEnabled = false;
    ctx.putImageData(state.imageData, 0, 0);

    const layers = window.knobman_getLayerList
      ? window.knobman_getLayerList() || []
      : [];
    const active = layers.find((layer) => layer.selected);
    updateStatusMetrics(
      active ? active.name || `Layer ${active.index + 1}` : "",
    );
    projects.saveSession();
  } catch (err) {
    state.dirty = true;
    console.error("renderFrame failed:", err);
    setStatus("Render failed");
  }
}

function markDirty() {
  state.dirty = true;
  state.modifiedSinceSave = true;
  scheduleRender();
  if (isDetachedPreviewOpen() && !state.detachedPreviewPlaying) {
    refreshDetachedPreviewNow();
  }
  updateUndoRedoButtons();
}

function wireControls() {
  const slider = el("frameSlider");
  slider.addEventListener("input", () => {
    state.currentFrame = parseInt(slider.value, 10) || 0;
    const renderFrames = parseInt(el("prefFrames").value, 10) || 1;
    if (state.currentFrame >= renderFrames) state.currentFrame = renderFrames - 1;
    if (state.currentFrame < 0) state.currentFrame = 0;
    slider.value = String(state.currentFrame);
    el("frameValue").textContent = String(state.currentFrame);
    window.knobman_setPreviewFrame(state.currentFrame);
    curveEditor.refreshCurveEditor();
    refreshDetachedPreviewNow();
    markDirty();
  });

  const zoomSelect = el("zoomSelect");
  zoomSelect.addEventListener("change", () => {
    const z = parseInt(zoomSelect.value, 10) || 8;
    state.zoomFactor = z;
    if (window.knobman_setZoom) window.knobman_setZoom(z);
    syncCanvasSize();
    markDirty();
  });

  el("prefWidth").addEventListener("change", onPrefWidthChange);
  el("prefHeight").addEventListener("change", onPrefHeightChange);
  el("prefLockAspect").addEventListener("change", onPrefLockAspectChange);
  el("prefFrames").addEventListener("change", onPrefsChange);
  el("prefPreviewFrames").addEventListener("change", onPrefsChange);
  el("prefBgColor").addEventListener("input", onPrefsChange);
  el("prefBgAlpha").addEventListener("input", onPrefsChange);
  el("prefOversample").addEventListener("change", onPrefsChange);
  el("prefExport").addEventListener("change", onPrefsChange);
  el("prefAlign").addEventListener("change", onPrefsChange);
  el("prefDuration").addEventListener("change", onPrefsChange);
  el("prefLoop").addEventListener("change", onPrefsChange);
  el("prefBiDir").addEventListener("change", onPrefsChange);

  el("btnNew").addEventListener("click", projects.onNew);
  el("btnOpen").addEventListener("click", () => {
    void projects.openProjectWithPicker();
  });
  el("btnRecent").addEventListener("click", projects.openRecentOverlay);
  el("btnSamples").addEventListener("click", projects.openSamplesOverlay);
  el("fileInput").addEventListener("change", projects.onFileOpen);
  el("btnSave").addEventListener("click", projects.onSave);
  el("btnExport").addEventListener("click", projects.openExportOverlay);
  el("btnPreviewWin").addEventListener("click", toggleDetachedPreviewWindow);
  el("btnUndo").addEventListener("click", onUndo);
  el("btnRedo").addEventListener("click", onRedo);

  el("btnAddLayer").addEventListener("click", onAddLayer);
  el("btnDeleteLayer").addEventListener("click", onDeleteLayer);
  el("btnMoveUp").addEventListener("click", onMoveUp);
  el("btnMoveDown").addEventListener("click", onMoveDown);
  el("btnDuplicate").addEventListener("click", onDuplicate);
  el("btnAddLayer").title = getLayerControlLabel("add");
  el("btnAddLayer").setAttribute("aria-label", getLayerControlLabel("add"));
  el("btnDeleteLayer").title = getLayerControlLabel("delete");
  el("btnDeleteLayer").setAttribute(
    "aria-label",
    getLayerControlLabel("delete"),
  );
  el("btnMoveUp").title = getLayerControlLabel("up");
  el("btnMoveUp").setAttribute("aria-label", getLayerControlLabel("up"));
  el("btnMoveDown").title = getLayerControlLabel("down");
  el("btnMoveDown").setAttribute("aria-label", getLayerControlLabel("down"));
  el("btnDuplicate").title = getLayerControlLabel("duplicate");
  el("btnDuplicate").setAttribute(
    "aria-label",
    getLayerControlLabel("duplicate"),
  );

  const samplesOverlay = el("samplesOverlay");
  const sampleSearch = el("sampleSearch");
  const closeSamples = el("btnCloseSamples");
  if (samplesOverlay) {
    samplesOverlay.addEventListener("click", (e) => {
      if (e.target === samplesOverlay) projects.closeSamplesOverlay();
    });
  }
  if (sampleSearch) {
    sampleSearch.addEventListener("input", projects.renderSampleList);
  }
  if (closeSamples) {
    closeSamples.addEventListener("click", projects.closeSamplesOverlay);
  }

  const exportOverlay = el("exportOverlay");
  const closeExport = el("btnCloseExport");
  const cancelExport = el("btnExportCancel");
  const confirmExport = el("btnExportConfirm");
  if (exportOverlay) {
    exportOverlay.addEventListener("click", (e) => {
      if (e.target === exportOverlay) projects.closeExportOverlay();
    });
  }
  if (closeExport) {
    closeExport.addEventListener("click", projects.closeExportOverlay);
  }
  if (cancelExport) {
    cancelExport.addEventListener("click", projects.closeExportOverlay);
  }
  if (confirmExport) {
    confirmExport.addEventListener("click", () => {
      void projects.onExport();
    });
  }

  const recentOverlay = el("recentOverlay");
  const closeRecent = el("btnCloseRecent");
  if (recentOverlay) {
    recentOverlay.addEventListener("click", (e) => {
      if (e.target === recentOverlay) projects.closeRecentOverlay();
    });
  }
  if (closeRecent) {
    closeRecent.addEventListener("click", projects.closeRecentOverlay);
  }

  document.addEventListener("keydown", onKeyDown);
  window.addEventListener("beforeunload", (e) => {
    if (state.modifiedSinceSave) e.preventDefault();
  });
  window.addEventListener("resize", () => {
    syncCanvasSize();
    curveEditor.refreshCurveEditor();
    shapeEditor.drawShapeOverlay();
    markDirty();
  });
}

function syncAspectRatioFromInputs() {
  const w = parseInt(el("prefWidth").value, 10) || 64;
  const h = parseInt(el("prefHeight").value, 10) || 64;
  state.prefAspectRatio = Math.max(0.01, w / Math.max(1, h));
}

function onPrefLockAspectChange() {
  state.prefAspectLock = el("prefLockAspect").checked;
  if (state.prefAspectLock) {
    syncAspectRatioFromInputs();
  }
}

function onPrefWidthChange() {
  if (state.prefAspectLock) {
    const w = parseInt(el("prefWidth").value, 10) || 64;
    const h = Math.max(1, Math.round(w / Math.max(0.01, state.prefAspectRatio)));
    el("prefHeight").value = h;
  }
  syncAspectRatioFromInputs();
  onPrefsChange();
}

function onPrefHeightChange() {
  if (state.prefAspectLock) {
    const h = parseInt(el("prefHeight").value, 10) || 64;
    const w = Math.max(1, Math.round(h * Math.max(0.01, state.prefAspectRatio)));
    el("prefWidth").value = w;
  }
  syncAspectRatioFromInputs();
  onPrefsChange();
}

function onPrefsChange() {
  const renderFrames = parseInt(el("prefFrames").value, 10) || 1;
  const previewFrames =
    parseInt(el("prefPreviewFrames").value, 10) || renderFrames;
  const prefs = {
    width: parseInt(el("prefWidth").value, 10) || 64,
    height: parseInt(el("prefHeight").value, 10) || 64,
    frames: renderFrames,
    renderFrames,
    previewFrames,
    oversampling: parseInt(el("prefOversample").value, 10) || 0,
    alignHorizontal: parseInt(el("prefAlign").value, 10) || 0,
    exportOption: parseInt(el("prefExport").value, 10) || 0,
    duration: parseInt(el("prefDuration").value, 10) || 100,
    loop: parseInt(el("prefLoop").value, 10) || 0,
    biDir: el("prefBiDir").checked,
    bgAlpha: parseInt(el("prefBgAlpha").value, 10) || 0,
    bgColor: el("prefBgColor").value,
  };
  window.knobman_setPrefs(prefs);
  refreshFromDoc();
  refreshDetachedPreviewNow();
}

function syncPrefsFromGo() {
  const prefs = window.knobman_getPrefs();
  if (!prefs) return;
  if (prefs.width != null) el("prefWidth").value = prefs.width;
  if (prefs.height != null) el("prefHeight").value = prefs.height;
  if (prefs.renderFrames != null) el("prefFrames").value = prefs.renderFrames;
  else if (prefs.frames != null) el("prefFrames").value = prefs.frames;
  if (prefs.previewFrames != null) {
    el("prefPreviewFrames").value = prefs.previewFrames;
  }
  if (prefs.oversampling != null) {
    el("prefOversample").value = prefs.oversampling;
  }
  if (prefs.alignHorizontal != null) {
    el("prefAlign").value = prefs.alignHorizontal;
  }
  if (prefs.exportOption != null) {
    el("prefExport").value = prefs.exportOption;
  }
  if (prefs.duration != null) el("prefDuration").value = prefs.duration;
  if (prefs.loop != null) el("prefLoop").value = prefs.loop;
  if (prefs.biDir != null) el("prefBiDir").checked = Boolean(prefs.biDir);
  if (prefs.bgAlpha != null) el("prefBgAlpha").value = prefs.bgAlpha;
  if (prefs.bgColor) el("prefBgColor").value = prefs.bgColor;
  state.prefAspectLock = el("prefLockAspect").checked;
  syncAspectRatioFromInputs();
}

function refreshLayerList() {
  const layerList = el("layerList");
  layerList.innerHTML = "";

  const layers = window.knobman_getLayerList() || [];
  if (state.selectedLayer >= layers.length) {
    state.selectedLayer = Math.max(0, layers.length - 1);
  }
  const previewJobs = [];
  state.layerPreviewToken += 1;

  layers.forEach((layer) => {
    const li = document.createElement("li");
    li.dataset.layerIndex = String(layer.index);
    if (layer.selected) {
      li.classList.add("active");
      state.selectedLayer = layer.index;
    }

    const vis = document.createElement("button");
    vis.type = "button";
    vis.className = "layer-vis" + (layer.visible ? " on" : "");
    vis.textContent = "V";
    vis.title = getLayerToggleLabel("visibility", layer.visible);
    vis.setAttribute("aria-label", vis.title);
    vis.addEventListener("click", (e) => {
      e.stopPropagation();
      window.knobman_setLayerVisible(layer.index, !layer.visible);
      refreshLayerList();
      markDirty();
    });

    const solo = document.createElement("button");
    solo.type = "button";
    solo.className = "layer-solo" + (layer.solo ? " on" : "");
    solo.textContent = "S";
    solo.title = getLayerToggleLabel("solo", layer.solo);
    solo.setAttribute("aria-label", solo.title);
    solo.addEventListener("click", (e) => {
      e.stopPropagation();
      window.knobman_setLayerSolo(layer.index, !layer.solo);
      refreshLayerList();
      markDirty();
    });

    const name = document.createElement("span");
    name.className = "layer-name";
    name.textContent = layer.name || `Layer ${layer.index + 1}`;

    const previews = document.createElement("div");
    previews.className = "layer-previews";
    [0, 1].forEach((frame) => {
      const slot = document.createElement("div");
      slot.className = "layer-preview-slot";
      const img = document.createElement("img");
      img.className = "layer-preview";
      img.alt = `L${layer.index + 1} F${frame}`;
      img.width = LAYER_PREVIEW_SIZE;
      img.height = LAYER_PREVIEW_SIZE;
      const label = document.createElement("span");
      label.className = "layer-preview-label";
      label.textContent = `F${frame}`;
      slot.appendChild(img);
      slot.appendChild(label);
      previews.appendChild(slot);
      previewJobs.push({ img, layerIndex: layer.index, frame });
    });

    li.appendChild(vis);
    li.appendChild(solo);
    li.appendChild(name);
    li.appendChild(previews);

    li.addEventListener("click", () => {
      state.selectedLayer = window.knobman_selectLayer(layer.index);
      refreshLayerList();
      refreshParamPanel();
      markDirty();
    });

    layerList.appendChild(li);
  });

  renderLayerPreviewsAsync(previewJobs);
}

function fieldsForPrimType(primType) {
  return PARAMS_BY_PRIM_TYPE[primType] || [];
}

function coerceParamValue(def, input) {
  if (def.type === "checkbox") {
    return input.checked;
  }
  if (
    def.type === "color" ||
    def.type === "text" ||
    def.type === "textarea" ||
    def.type === "file"
  ) {
    return input.value;
  }
  if (def.numeric === "int") {
    const value = parseInt(input.value, 10);
    return Number.isFinite(value) ? value : 0;
  }
  if (def.numeric === "float") {
    const value = parseFloat(input.value);
    return Number.isFinite(value) ? value : 0;
  }
  return input.value;
}

function parseNumericValue(def, rawValue) {
  if (def.numeric === "int") {
    const value = parseInt(rawValue, 10);
    return Number.isFinite(value) ? value : 0;
  }
  if (def.numeric === "float") {
    const value = parseFloat(rawValue);
    return Number.isFinite(value) ? value : 0;
  }
  return rawValue;
}

function buildBoundedNumericControl(def, value, onChange, options = {}) {
  const wrap = document.createElement("div");
  wrap.className = "param-numeric";

  const slider = document.createElement("input");
  slider.type = "range";
  slider.min = String(def.min);
  slider.max = String(def.max);
  slider.step = String(def.step ?? 1);
  slider.value = String(value ?? 0);

  const number = document.createElement("input");
  number.type = "number";
  number.min = String(def.min);
  number.max = String(def.max);
  number.step = String(def.step ?? 1);
  number.value = String(value ?? 0);
  if (options.disabled) {
    slider.disabled = true;
    number.disabled = true;
  }

  function syncAndApply(rawValue, source) {
    const nextValue = parseNumericValue(def, rawValue);
    const normalized = String(nextValue);
    if (source !== slider) slider.value = normalized;
    if (source !== number) number.value = normalized;
    onChange(nextValue);
  }

  slider.addEventListener("input", () => {
    syncAndApply(slider.value, slider);
  });
  number.addEventListener("input", () => {
    syncAndApply(number.value, number);
  });

  wrap.appendChild(slider);
  wrap.appendChild(number);
  return wrap;
}

function applyParamChange(key, value) {
  const ok = window.knobman_setParam(state.selectedLayer, key, value);
  if (!ok) return false;
  if (key !== "name" && key !== "primType") {
    invalidateLayerPreviews();
  }
  if (key === "name") {
    refreshLayerList();
  }
  if (key === "primType") {
    invalidateLayerPreviews();
    refreshLayerList();
    refreshParamPanel();
  }
  if (key === "shape") {
    shapeEditor.refreshShapeEditor();
  }
  markDirty();
  return true;
}

function applyEffectParamChange(key, value) {
  const ok = window.knobman_setEffectParam(state.selectedLayer, key, value);
  if (!ok) return false;
  markDirty();
  return true;
}

function buildParamRow(key, value) {
  const def = PARAM_DEFS[key];
  if (!def) return null;

  const row = document.createElement("div");
  row.className = "param-row";
  if (def.type === "checkbox") row.classList.add("checkbox");

  const caption = document.createElement("span");
  caption.textContent = def.label;

  if (def.type === "color") {
    row.appendChild(caption);
    const wrap = document.createElement("div");
    wrap.className = "param-color";

    const colorInput = document.createElement("input");
    colorInput.type = "color";
    colorInput.value = String(value || "#000000");

    const alphaInput = document.createElement("input");
    alphaInput.type = "range";
    alphaInput.min = "0";
    alphaInput.max = "255";
    alphaInput.step = "1";
    const alpha = window.knobman_getParam(state.selectedLayer, "colorAlpha");
    alphaInput.value = String(alpha == null ? 255 : alpha);

    const alphaOut = document.createElement("output");
    alphaOut.textContent = alphaInput.value;

    colorInput.addEventListener("input", () => {
      applyParamChange("color", colorInput.value);
    });
    alphaInput.addEventListener("input", () => {
      alphaOut.textContent = alphaInput.value;
      applyParamChange("colorAlpha", parseInt(alphaInput.value, 10) || 0);
    });

    wrap.appendChild(colorInput);
    wrap.appendChild(alphaInput);
    wrap.appendChild(alphaOut);
    row.appendChild(wrap);
    return row;
  }

  if (def.type === "file") {
    row.appendChild(caption);

    const wrap = document.createElement("div");
    wrap.className = "param-file";

    const input = document.createElement("input");
    input.type = "file";
    if (def.accept) input.accept = def.accept;

    const clear = document.createElement("button");
    clear.type = "button";
    clear.textContent = "Clear";
    clear.addEventListener("click", () => {
      if (!applyParamChange("clearEmbeddedImage", 1)) return;
      applyParamChange("file", "");
      refreshParamPanel();
    });

    input.addEventListener("change", () => {
      const file = input.files && input.files[0];
      if (!file) return;
      const reader = new FileReader();
      reader.onload = () => {
        const data = new Uint8Array(reader.result);
        const okImg = applyParamChange("embeddedImage", data);
        const okName = applyParamChange("file", file.name);
        if (okImg && okName) {
          refreshParamPanel();
          setStatus("Loaded image " + file.name);
        }
      };
      reader.readAsArrayBuffer(file);
      input.value = "";
    });

    const hint = document.createElement("small");
    const has = Boolean(
      window.knobman_getParam(state.selectedLayer, "hasEmbeddedImage"),
    );
    const name = String(window.knobman_getParam(state.selectedLayer, "file") || "");
    hint.textContent = has
      ? "Embedded: " + (name || "(unnamed)")
      : "No embedded image";

    wrap.appendChild(input);
    wrap.appendChild(clear);
    row.appendChild(wrap);
    row.appendChild(hint);
    return row;
  }

  let input;
  if (def.type === "select") {
    input = document.createElement("select");
    (def.options || []).forEach((opt) => {
      const option = document.createElement("option");
      option.value = String(opt.value);
      option.textContent = opt.label;
      input.appendChild(option);
    });
    input.value = String(value ?? 0);
  } else if (def.type === "textarea") {
    input = document.createElement("textarea");
    input.rows = 4;
    input.value = String(value ?? "");
  } else {
    input = document.createElement("input");
    input.type = def.type;
    if (def.min != null) input.min = String(def.min);
    if (def.max != null) input.max = String(def.max);
    if (def.step != null) input.step = String(def.step);
    if (def.type === "checkbox") {
      input.checked = Boolean(value);
    } else if (def.type === "number") {
      input.value = String(value ?? 0);
    } else if (def.type === "color") {
      input.value = String(value || "#000000");
    } else {
      input.value = String(value ?? "");
    }
  }

  const eventName =
    def.type === "select" || def.type === "checkbox" ? "change" : "input";

  row.appendChild(caption);
  if (hasBoundedRangeControl(def)) {
    row.appendChild(
      buildBoundedNumericControl(def, value, (nextValue) => {
        applyParamChange(key, nextValue);
      }),
    );
  } else {
    input.addEventListener(eventName, () => {
      const nextValue = coerceParamValue(def, input);
      applyParamChange(key, nextValue);
    });
    row.appendChild(input);
  }
  return row;
}

function buildEffectRow(key, value, options = {}) {
  const def = EFFECT_DEFS[key];
  if (!def) return null;

  const row = document.createElement("div");
  row.className = "param-row";
  if (def.type === "checkbox") row.classList.add("checkbox");
  if (options.disabled) row.classList.add("disabled");

  const caption = document.createElement("span");
  caption.textContent = options.label || def.label;

  let input;
  if (def.type === "select") {
    input = document.createElement("select");
    (def.options || []).forEach((opt) => {
      const option = document.createElement("option");
      option.value = String(opt.value);
      option.textContent = opt.label;
      input.appendChild(option);
    });
    input.value = String(value ?? 0);
  } else if (def.type === "textarea") {
    input = document.createElement("textarea");
    input.rows = 3;
    input.value = String(value ?? "");
  } else {
    input = document.createElement("input");
    input.type = def.type;
    if (def.min != null) input.min = String(def.min);
    if (def.max != null) input.max = String(def.max);
    if (def.step != null) input.step = String(def.step);
    if (def.type === "checkbox") {
      input.checked = Boolean(value);
    } else {
      input.value = String(value ?? "");
    }
  }
  if (options.disabled) {
    input.disabled = true;
  }

  const eventName =
    def.type === "select" || def.type === "checkbox" ? "change" : "input";

  row.appendChild(caption);
  if (hasBoundedRangeControl(def)) {
    row.appendChild(
      buildBoundedNumericControl(def, value, (nextValue) => {
        applyEffectParamChange(key, nextValue);
        if (isCurveSelectorField(key) && Number(nextValue) > 0) {
          curveEditor.focusCurve(Number(nextValue));
        }
      }, options),
    );
  } else {
    input.addEventListener(eventName, () => {
      const nextValue = coerceParamValue(def, input);
      applyEffectParamChange(key, nextValue);
      if (isCurveSelectorField(key) && Number(nextValue) > 0) {
        curveEditor.focusCurve(Number(nextValue));
      }
    });
    row.appendChild(input);
  }
  return row;
}

function appendEffectSections(content) {
  const title = document.createElement("div");
  title.className = "param-group-title";
  title.textContent = "Effects";
  content.appendChild(title);

  EFFECT_SECTIONS.forEach((section) => {
    const details = document.createElement("details");
    details.className = "effect-section";
    details.open = Boolean(section.open);

    const summary = document.createElement("summary");
    summary.textContent = section.title;
    details.appendChild(summary);

    const body = document.createElement("div");
    body.className = "effect-section-body";
    const rows =
      section.title === "Transform"
        ? getTransformEffectRows(
            Object.fromEntries(
              section.fields.map((key) => [
                key,
                window.knobman_getEffectParam(state.selectedLayer, key),
              ]),
            ),
          )
        : section.title === "Color"
          ? getColorEffectRows(
              Object.fromEntries(
                section.fields.map((key) => [
                  key,
                  window.knobman_getEffectParam(state.selectedLayer, key),
                ]),
              ),
            )
        : section.title === "Specular Highlight"
          ? getSpecularHighlightRows(
              Object.fromEntries(
                section.fields.map((key) => [
                  key,
                  window.knobman_getEffectParam(state.selectedLayer, key),
                ]),
              ),
            )
        : section.title === "Drop Shadow"
          ? getDropShadowEffectRows(
              Object.fromEntries(
                section.fields.map((key) => [
                  key,
                  window.knobman_getEffectParam(state.selectedLayer, key),
                ]),
              ),
            )
        : section.title === "Inner Shadow"
          ? getInnerShadowEffectRows(
              Object.fromEntries(
                section.fields.map((key) => [
                  key,
                  window.knobman_getEffectParam(state.selectedLayer, key),
                ]),
              ),
            )
        : section.title === "Emboss"
          ? getEmbossEffectRows(
              Object.fromEntries(
                section.fields.map((key) => [
                  key,
                  window.knobman_getEffectParam(state.selectedLayer, key),
                ]),
              ),
            )
        : section.fields.map((key) => ({
            key,
            label: EFFECT_DEFS[key]?.label || key,
            disabled: false,
          }));
    rows.forEach((rowInfo) => {
      const value = window.knobman_getEffectParam(state.selectedLayer, rowInfo.key);
      const row = buildEffectRow(rowInfo.key, value, rowInfo);
      if (row) body.appendChild(row);
    });
    details.appendChild(body);
    content.appendChild(details);
  });
}

function primitiveSupportsTexture(primType) {
  return fieldsForPrimType(primType).includes("textureFile");
}

function appendTexturePanel(content, primType) {
  if (!primitiveSupportsTexture(primType)) return;

  const title = document.createElement("div");
  title.className = "param-group-title";
  title.textContent = "Textures";
  content.appendChild(title);

  const textures = window.knobman_getTextureList
    ? window.knobman_getTextureList() || []
    : [];
  const selectedTexture =
    window.knobman_getParam(state.selectedLayer, "textureFile") ?? 0;

  const selectRow = document.createElement("div");
  selectRow.className = "param-row";
  const selectLabel = document.createElement("span");
  selectLabel.textContent = "Texture Slot";
  const select = document.createElement("select");
  const noneOption = document.createElement("option");
  noneOption.value = "0";
  noneOption.textContent = "None";
  select.appendChild(noneOption);
  textures.forEach((tex) => {
    const option = document.createElement("option");
    option.value = String(tex.index);
    option.textContent = `${tex.index}: ${tex.name} (${tex.width}x${tex.height})`;
    select.appendChild(option);
  });
  select.value = String(selectedTexture || 0);
  select.addEventListener("change", () => {
    const idx = parseInt(select.value, 10) || 0;
    applyParamChange("textureFile", idx);
    refreshParamPanel();
  });
  selectRow.appendChild(selectLabel);
  selectRow.appendChild(select);
  content.appendChild(selectRow);

  const uploadRow = document.createElement("div");
  uploadRow.className = "param-row";
  const uploadLabel = document.createElement("span");
  uploadLabel.textContent = "Add Texture";
  const uploadWrap = document.createElement("div");
  uploadWrap.className = "param-file";
  const uploadInput = document.createElement("input");
  uploadInput.type = "file";
  uploadInput.accept = "image/*";
  uploadInput.addEventListener("change", () => {
    const file = uploadInput.files && uploadInput.files[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      const data = new Uint8Array(reader.result);
      const idx = window.knobman_addTexture(file.name, data);
      if (!idx) {
        setStatus("Failed to add texture " + file.name);
        return;
      }
      applyParamChange("textureFile", idx);
      refreshParamPanel();
      markDirty();
      setStatus("Added texture " + file.name);
    };
    reader.readAsArrayBuffer(file);
    uploadInput.value = "";
  });
  uploadWrap.appendChild(uploadInput);
  uploadRow.appendChild(uploadLabel);
  uploadRow.appendChild(uploadWrap);
  content.appendChild(uploadRow);

  if (textures.length === 0) {
    const empty = document.createElement("p");
    empty.className = "placeholder";
    empty.textContent = "No textures loaded.";
    content.appendChild(empty);
    return;
  }

  const gallery = document.createElement("div");
  gallery.className = "texture-gallery";
  textures.forEach((tex) => {
    const card = document.createElement("button");
    card.type = "button";
    card.className =
      "texture-card" + (tex.index === selectedTexture ? " active" : "");
    card.title = tex.name;
    card.addEventListener("click", () => {
      applyParamChange("textureFile", tex.index);
      refreshParamPanel();
    });

    const thumb = document.createElement("div");
    thumb.className = "texture-thumb";
    const arr = window.knobman_getTextureData
      ? window.knobman_getTextureData(tex.index)
      : null;
    if (arr && arr.length) {
      const img = document.createElement("img");
      const blob = new Blob([arr]);
      const url = URL.createObjectURL(blob);
      img.src = url;
      img.alt = tex.name;
      img.loading = "lazy";
      img.addEventListener(
        "load",
        () => URL.revokeObjectURL(url),
        { once: true },
      );
      thumb.appendChild(img);
    } else {
      const fallback = document.createElement("span");
      fallback.textContent = "No preview";
      thumb.appendChild(fallback);
    }

    const meta = document.createElement("div");
    meta.className = "texture-meta";
    meta.textContent = `${tex.index}: ${tex.name}`;

    card.appendChild(thumb);
    card.appendChild(meta);
    gallery.appendChild(card);
  });
  content.appendChild(gallery);
}

function refreshParamPanel() {
  const content = el("paramContent");
  content.innerHTML = "";

  const layers = window.knobman_getLayerList() || [];
  if (layers.length === 0) {
    const p = document.createElement("p");
    p.className = "placeholder";
    p.textContent = "No layer selected.";
    content.appendChild(p);
    return;
  }
  state.selectedLayer = Math.max(
    0,
    Math.min(state.selectedLayer, layers.length - 1),
  );

  const primType = window.knobman_getParam(state.selectedLayer, "primType") ?? 0;
  const fields = ["name", "primType", ...fieldsForPrimType(primType)];

  fields.forEach((key) => {
    const value = window.knobman_getParam(state.selectedLayer, key);
    const row = buildParamRow(key, value);
    if (row) content.appendChild(row);
  });

  appendTexturePanel(content, primType);
  appendEffectSections(content);
  shapeEditor.refreshShapeEditor();
}

function onUndo() {
  if (!window.knobman_undo || !window.knobman_undo()) return;
  invalidateLayerPreviews();
  refreshFromDoc();
  setStatus("Undo");
  updateUndoRedoButtons();
}

function onRedo() {
  if (!window.knobman_redo || !window.knobman_redo()) return;
  invalidateLayerPreviews();
  refreshFromDoc();
  setStatus("Redo");
  updateUndoRedoButtons();
}

function updateUndoRedoButtons() {
  const btnUndo = el("btnUndo");
  const btnRedo = el("btnRedo");
  if (btnUndo) {
    btnUndo.disabled = !(window.knobman_canUndo && window.knobman_canUndo());
  }
  if (btnRedo) {
    btnRedo.disabled = !(window.knobman_canRedo && window.knobman_canRedo());
  }
}

function onAddLayer() {
  state.selectedLayer = window.knobman_addLayer();
  invalidateLayerPreviews();
  refreshLayerList();
  refreshParamPanel();
  markDirty();
}

function onDeleteLayer() {
  window.knobman_deleteLayer(state.selectedLayer);
  invalidateLayerPreviews();
  refreshLayerList();
  refreshParamPanel();
  markDirty();
}

function onMoveUp() {
  state.selectedLayer = window.knobman_moveLayer(-1);
  invalidateLayerPreviews();
  refreshLayerList();
  refreshParamPanel();
  markDirty();
}

function onMoveDown() {
  state.selectedLayer = window.knobman_moveLayer(1);
  invalidateLayerPreviews();
  refreshLayerList();
  refreshParamPanel();
  markDirty();
}

function onDuplicate() {
  state.selectedLayer = window.knobman_duplicateLayer();
  invalidateLayerPreviews();
  refreshLayerList();
  refreshParamPanel();
  markDirty();
}

function keyName(e) {
  return String(e && e.key ? e.key : "").toLowerCase();
}

function isEditableTarget(target) {
  if (!target) return false;
  if (target.isContentEditable) return true;
  const tag = String(target.tagName || "").toLowerCase();
  return tag === "input" || tag === "textarea" || tag === "select";
}

function onKeyDown(e) {
  const key = keyName(e);
  if (key === "escape") {
    const welcomeOverlay = el("welcomeOverlay");
    if (welcomeOverlay && !welcomeOverlay.hidden) {
      e.preventDefault();
      projects.closeWelcomeOverlay();
      return;
    }
  }
  if (key === "escape" && projects.isSamplesOverlayOpen()) {
    e.preventDefault();
    projects.closeSamplesOverlay();
    return;
  }
  if (key === "escape" && projects.isRecentOverlayOpen()) {
    e.preventDefault();
    projects.closeRecentOverlay();
    return;
  }
  if (key === "escape" && projects.isExportOverlayOpen()) {
    e.preventDefault();
    projects.closeExportOverlay();
    return;
  }
  const mod = e.ctrlKey || e.metaKey;
  const editing = isEditableTarget(e.target);
  if (mod && key === "z") {
    e.preventDefault();
    onUndo();
    return;
  }
  if (mod && (key === "y" || (e.shiftKey && key === "z"))) {
    e.preventDefault();
    onRedo();
    return;
  }
  if (mod && key === "s") {
    e.preventDefault();
    void projects.onSave();
    return;
  }
  if (mod && key === "o") {
    e.preventDefault();
    void projects.openProjectWithPicker();
    return;
  }
  if (mod && key === "e") {
    e.preventDefault();
    projects.openExportOverlay();
    return;
  }
  if (mod && key === "d") {
    e.preventDefault();
    onDuplicate();
    return;
  }
  if (editing) return;
  if (key === "delete" || key === "backspace") {
    e.preventDefault();
    if (
      shapeEditor.isShapeLayerSelected() &&
      state.shapeSelectedHandle
    ) {
      shapeEditor.deleteShapeSelection();
    } else {
      onDeleteLayer();
    }
    return;
  }
  if (key === "arrowup") {
    e.preventDefault();
    onMoveUp();
    return;
  }
  if (key === "arrowdown") {
    e.preventDefault();
    onMoveDown();
  }
}

function setStatus(msg) {
  el("statusMsg").textContent = msg;
}

function updateStatusMetrics(layerName) {
  const prefs = window.knobman_getPrefs ? window.knobman_getPrefs() : null;
  const w = prefs ? prefs.width : 0;
  const h = prefs ? prefs.height : 0;
  const frames = prefs ? prefs.frames : 0;
  const parts = [];
  if (w && h) parts.push(`${w}×${h}`);
  if (frames) parts.push(`${frames} fr`);
  parts.push(`F${state.currentFrame}`);
  if (layerName) parts.push(`L: ${layerName}`);
  if (state.lastRenderMs > 0) parts.push(`${state.lastRenderMs}ms`);
  el("statusMetrics").textContent = parts.join(" | ");
}

const go = new Go();

WebAssembly.instantiateStreaming(
  fetch("knobman.wasm", { cache: "no-store" }),
  go.importObject,
)
  .then((result) => {
    go.run(result.instance);
    state.wasmReady = true;
    onWasmReady();
  })
  .catch((err) => {
    el("loading").textContent = "Failed to load WASM: " + err;
    console.error(err);
  });
