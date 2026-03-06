'use strict';

// ── WASM bootstrap ────────────────────────────────────────────────────────────

const go = new Go();
let wasmReady = false;

WebAssembly.instantiateStreaming(fetch('knobman.wasm'), go.importObject)
  .then(result => {
    go.run(result.instance);
    wasmReady = true;
    onWasmReady();
  })
  .catch(err => {
    document.getElementById('loading').textContent = 'Failed to load WASM: ' + err;
    console.error(err);
  });

// ── App state ─────────────────────────────────────────────────────────────────

let canvasW = 64;
let canvasH = 64;
let zoomFactor = 8;
let currentFrame = 0;
let dirty = true;
let rafPending = false;
let pixelBuf = null;
let imageData = null;
let selectedLayer = 0;

// ── Initialise after WASM load ────────────────────────────────────────────────

function onWasmReady() {
  document.getElementById('loading').style.display = 'none';
  document.getElementById('app').style.display = 'flex';

  wireControls();

  window.knobman_init(64, 64, zoomFactor);
  refreshFromDoc();
  scheduleRender();

  setStatus('Ready');
}

function refreshFromDoc() {
  syncPrefsFromGo();
  syncCanvasSize();
  refreshLayerList();

  const frames = parseInt(document.getElementById('prefFrames').value, 10) || 1;
  if (currentFrame >= frames) currentFrame = frames - 1;
  if (currentFrame < 0) currentFrame = 0;
  document.getElementById('frameSlider').max = Math.max(0, frames - 1);
  document.getElementById('frameSlider').value = currentFrame;
  document.getElementById('frameValue').textContent = String(currentFrame);
  window.knobman_setPreviewFrame(currentFrame);
  markDirty();
}

// ── Canvas ────────────────────────────────────────────────────────────────────

function syncCanvasSize() {
  const dims = window.knobman_getDimensions();
  if (!dims) return;
  if (canvasW === dims.width && canvasH === dims.height && pixelBuf && imageData) return;

  canvasW = dims.width;
  canvasH = dims.height;

  const canvas = document.getElementById('knobCanvas');
  canvas.width = canvasW;
  canvas.height = canvasH;
  canvas.style.width = canvasW + 'px';
  canvas.style.height = canvasH + 'px';

  imageData = new ImageData(canvasW, canvasH);
  pixelBuf = new Uint8Array(canvasW * canvasH * 4);
}

function scheduleRender() {
  if (!rafPending) {
    rafPending = true;
    requestAnimationFrame(renderFrame);
  }
}

function renderFrame() {
  rafPending = false;
  if (!wasmReady || !dirty) return;
  dirty = false;

  syncCanvasSize();
  window.knobman_render(pixelBuf);
  imageData.data.set(pixelBuf);

  const canvas = document.getElementById('knobCanvas');
  canvas.getContext('2d').putImageData(imageData, 0, 0);
}

function markDirty() {
  dirty = true;
  scheduleRender();
}

// ── Controls ──────────────────────────────────────────────────────────────────

function wireControls() {
  // Frame scrubber
  const slider = document.getElementById('frameSlider');
  slider.addEventListener('input', () => {
    currentFrame = parseInt(slider.value, 10) || 0;
    document.getElementById('frameValue').textContent = String(currentFrame);
    window.knobman_setPreviewFrame(currentFrame);
    markDirty();
  });

  // Prefs bar
  document.getElementById('prefWidth').addEventListener('change', onPrefsChange);
  document.getElementById('prefHeight').addEventListener('change', onPrefsChange);
  document.getElementById('prefFrames').addEventListener('change', onPrefsChange);
  document.getElementById('prefBgColor').addEventListener('input', onPrefsChange);
  document.getElementById('prefOversample').addEventListener('change', onPrefsChange);
  document.getElementById('prefExport').addEventListener('change', onPrefsChange);

  // Toolbar
  document.getElementById('btnNew').addEventListener('click', onNew);
  document.getElementById('btnOpen').addEventListener('click', () => document.getElementById('fileInput').click());
  document.getElementById('fileInput').addEventListener('change', onFileOpen);
  document.getElementById('btnSave').addEventListener('click', onSave);
  document.getElementById('btnExport').addEventListener('click', onExport);
  document.getElementById('btnUndo').addEventListener('click', onUndo);
  document.getElementById('btnRedo').addEventListener('click', onRedo);

  // Layer controls
  document.getElementById('btnAddLayer').addEventListener('click', onAddLayer);
  document.getElementById('btnDeleteLayer').addEventListener('click', onDeleteLayer);
  document.getElementById('btnMoveUp').addEventListener('click', onMoveUp);
  document.getElementById('btnMoveDown').addEventListener('click', onMoveDown);
  document.getElementById('btnDuplicate').addEventListener('click', onDuplicate);

  document.addEventListener('keydown', onKeyDown);
}

function onPrefsChange() {
  const prefs = {
    width: parseInt(document.getElementById('prefWidth').value, 10) || 64,
    height: parseInt(document.getElementById('prefHeight').value, 10) || 64,
    frames: parseInt(document.getElementById('prefFrames').value, 10) || 1,
    oversampling: parseInt(document.getElementById('prefOversample').value, 10) || 0,
    exportOption: parseInt(document.getElementById('prefExport').value, 10) || 0,
    bgColor: document.getElementById('prefBgColor').value
  };
  window.knobman_setPrefs(prefs);
  refreshFromDoc();
}

function syncPrefsFromGo() {
  const p = window.knobman_getPrefs();
  if (!p) return;
  if (p.width != null) document.getElementById('prefWidth').value = p.width;
  if (p.height != null) document.getElementById('prefHeight').value = p.height;
  if (p.frames != null) document.getElementById('prefFrames').value = p.frames;
  if (p.oversampling != null) document.getElementById('prefOversample').value = p.oversampling;
  if (p.exportOption != null) document.getElementById('prefExport').value = p.exportOption;
  if (p.bgColor) document.getElementById('prefBgColor').value = p.bgColor;
}

// ── Layers ────────────────────────────────────────────────────────────────────

function refreshLayerList() {
  const layerList = document.getElementById('layerList');
  layerList.innerHTML = '';

  const layers = window.knobman_getLayerList() || [];
  layers.forEach(layer => {
    const li = document.createElement('li');
    if (layer.selected) {
      li.classList.add('active');
      selectedLayer = layer.index;
    }

    const vis = document.createElement('span');
    vis.className = 'layer-vis' + (layer.visible ? ' on' : '');
    vis.textContent = 'V';
    vis.title = 'Toggle visibility';
    vis.addEventListener('click', (e) => {
      e.stopPropagation();
      window.knobman_setLayerVisible(layer.index, !layer.visible);
      refreshLayerList();
      markDirty();
    });

    const solo = document.createElement('span');
    solo.className = 'layer-solo' + (layer.solo ? ' on' : '');
    solo.textContent = 'S';
    solo.title = 'Toggle solo';
    solo.addEventListener('click', (e) => {
      e.stopPropagation();
      window.knobman_setLayerSolo(layer.index, !layer.solo);
      refreshLayerList();
      markDirty();
    });

    const name = document.createElement('span');
    name.className = 'layer-name';
    name.textContent = layer.name || `Layer ${layer.index + 1}`;

    li.appendChild(vis);
    li.appendChild(solo);
    li.appendChild(name);

    li.addEventListener('click', () => {
      selectedLayer = window.knobman_selectLayer(layer.index);
      refreshLayerList();
      markDirty();
    });

    layerList.appendChild(li);
  });
}

// ── Toolbar handlers ──────────────────────────────────────────────────────────

function onNew() {
  window.knobman_newDocument();
  currentFrame = 0;
  refreshFromDoc();
  setStatus('New document');
}

function onSave() {
  const data = window.knobman_saveFile();
  if (!data || data.length === 0) {
    setStatus('Save failed');
    return;
  }
  const blob = new Blob([data], { type: 'application/octet-stream' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = 'project.knob';
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
  setStatus('Saved project.knob');
}

function onExport() { setStatus('Export (Phase 8)'); }
function onUndo()   { setStatus('Undo (Phase 9)'); }
function onRedo()   { setStatus('Redo (Phase 9)'); }

function onAddLayer() {
  selectedLayer = window.knobman_addLayer();
  refreshLayerList();
  markDirty();
}

function onDeleteLayer() {
  window.knobman_deleteLayer(selectedLayer);
  refreshLayerList();
  markDirty();
}

function onMoveUp() {
  selectedLayer = window.knobman_moveLayer(-1);
  refreshLayerList();
  markDirty();
}

function onMoveDown() {
  selectedLayer = window.knobman_moveLayer(1);
  refreshLayerList();
  markDirty();
}

function onDuplicate() {
  selectedLayer = window.knobman_duplicateLayer();
  refreshLayerList();
  markDirty();
}

function onFileOpen(e) {
  const file = e.target.files[0];
  if (!file) return;

  const reader = new FileReader();
  reader.onload = () => {
    const ok = window.knobman_loadFile(new Uint8Array(reader.result));
    if (!ok) {
      setStatus('Failed to load ' + file.name);
      return;
    }
    currentFrame = 0;
    refreshFromDoc();
    setStatus('Loaded ' + file.name);
  };
  reader.readAsArrayBuffer(file);
  e.target.value = '';
}

function onKeyDown(e) {
  const mod = e.ctrlKey || e.metaKey;
  if (mod && e.key === 'z') { e.preventDefault(); onUndo(); }
  if (mod && (e.key === 'y' || (e.shiftKey && e.key === 'z'))) { e.preventDefault(); onRedo(); }
  if (mod && e.key === 's') { e.preventDefault(); onSave(); }
  if (mod && e.key === 'o') { e.preventDefault(); document.getElementById('fileInput').click(); }
  if (mod && e.key === 'd') { e.preventDefault(); onDuplicate(); }
  if (e.key === 'Delete') { e.preventDefault(); onDeleteLayer(); }
  if (e.key === 'ArrowUp') { e.preventDefault(); onMoveUp(); }
  if (e.key === 'ArrowDown') { e.preventDefault(); onMoveDown(); }
}

// ── Status bar ────────────────────────────────────────────────────────────────

function setStatus(msg) {
  document.getElementById('statusBar').textContent = msg;
}
