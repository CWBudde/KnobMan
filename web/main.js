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

// ── Initialise after WASM load ────────────────────────────────────────────────

function onWasmReady() {
  document.getElementById('loading').style.display = 'none';
  document.getElementById('app').style.display = 'flex';

  const dims = window.knobman_getDimensions();
  canvasW = dims.width;
  canvasH = dims.height;

  setupCanvas();
  scheduleRender();
  wireControls();

  setStatus('Ready — Phase 0 skeleton');
}

// ── Canvas ────────────────────────────────────────────────────────────────────

let pixelBuf = null;
let imageData = null;

function setupCanvas() {
  const canvas = document.getElementById('knobCanvas');
  canvas.width  = canvasW;
  canvas.height = canvasH;
  canvas.style.width  = canvasW + 'px';
  canvas.style.height = canvasH + 'px';

  imageData = new ImageData(canvasW, canvasH);
  pixelBuf  = new Uint8Array(canvasW * canvasH * 4);
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
  const slider  = document.getElementById('frameSlider');
  const frameVal = document.getElementById('frameValue');
  const frames  = parseInt(document.getElementById('prefFrames').value, 10);
  slider.max = frames - 1;

  slider.addEventListener('input', () => {
    currentFrame = parseInt(slider.value, 10);
    frameVal.textContent = currentFrame;
    markDirty();
  });

  // Prefs bar
  document.getElementById('prefWidth').addEventListener('change', onPrefsChange);
  document.getElementById('prefHeight').addEventListener('change', onPrefsChange);
  document.getElementById('prefFrames').addEventListener('change', () => {
    const f = parseInt(document.getElementById('prefFrames').value, 10);
    slider.max = f - 1;
    if (currentFrame >= f) { currentFrame = f - 1; slider.value = currentFrame; frameVal.textContent = currentFrame; }
    onPrefsChange();
  });
  document.getElementById('prefBgColor').addEventListener('input', onPrefsChange);
  document.getElementById('prefOversample').addEventListener('change', onPrefsChange);
  document.getElementById('prefExport').addEventListener('change', () => {});

  // Toolbar buttons (stubs for now)
  document.getElementById('btnNew').addEventListener('click', onNew);
  document.getElementById('btnOpen').addEventListener('click', () => document.getElementById('fileInput').click());
  document.getElementById('fileInput').addEventListener('change', onFileOpen);
  document.getElementById('btnSave').addEventListener('click', onSave);
  document.getElementById('btnExport').addEventListener('click', onExport);
  document.getElementById('btnUndo').addEventListener('click', onUndo);
  document.getElementById('btnRedo').addEventListener('click', onRedo);

  // Layer controls (stubs)
  document.getElementById('btnAddLayer').addEventListener('click', onAddLayer);
  document.getElementById('btnDeleteLayer').addEventListener('click', onDeleteLayer);
  document.getElementById('btnMoveUp').addEventListener('click', onMoveUp);
  document.getElementById('btnMoveDown').addEventListener('click', onMoveDown);
  document.getElementById('btnDuplicate').addEventListener('click', onDuplicate);

  // Keyboard shortcuts
  document.addEventListener('keydown', onKeyDown);
}

function onPrefsChange() {
  markDirty();
}

// ── Stub handlers (implemented in later phases) ───────────────────────────────

function onNew()          { setStatus('New document (Phase 1)'); markDirty(); }
function onSave()         { setStatus('Save (Phase 1)'); }
function onExport()       { setStatus('Export (Phase 8)'); }
function onUndo()         { setStatus('Undo (Phase 9)'); markDirty(); }
function onRedo()         { setStatus('Redo (Phase 9)'); markDirty(); }
function onAddLayer()     { setStatus('Add layer (Phase 5)'); }
function onDeleteLayer()  { setStatus('Delete layer (Phase 5)'); }
function onMoveUp()       { setStatus('Move up (Phase 5)'); }
function onMoveDown()     { setStatus('Move down (Phase 5)'); }
function onDuplicate()    { setStatus('Duplicate (Phase 5)'); }

function onFileOpen(e) {
  const file = e.target.files[0];
  if (!file) return;
  setStatus('Loading ' + file.name + ' (Phase 1)...');
  // Phase 1 will wire: reader.readAsArrayBuffer → knobman_loadFile(bytes)
  e.target.value = '';
}

function onKeyDown(e) {
  const mod = e.ctrlKey || e.metaKey;
  if (mod && e.key === 'z') { e.preventDefault(); onUndo(); }
  if (mod && (e.key === 'y' || (e.shiftKey && e.key === 'z'))) { e.preventDefault(); onRedo(); }
  if (mod && e.key === 's') { e.preventDefault(); onSave(); }
  if (mod && e.key === 'o') { e.preventDefault(); document.getElementById('fileInput').click(); }
}

// ── Status bar ────────────────────────────────────────────────────────────────

function setStatus(msg) {
  document.getElementById('statusBar').textContent = msg;
}
