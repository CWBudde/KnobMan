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

const PRIM_TYPES = [
  { value: 0, label: 'None' },
  { value: 1, label: 'Image' },
  { value: 2, label: 'Circle' },
  { value: 3, label: 'CircleFill' },
  { value: 4, label: 'MetalCircle' },
  { value: 5, label: 'WaveCircle' },
  { value: 6, label: 'Sphere' },
  { value: 7, label: 'Rect' },
  { value: 8, label: 'RectFill' },
  { value: 9, label: 'Triangle' },
  { value: 10, label: 'Line' },
  { value: 11, label: 'RadiateLine' },
  { value: 12, label: 'H-Lines' },
  { value: 13, label: 'V-Lines' },
  { value: 14, label: 'Text' },
  { value: 15, label: 'Shape' }
];

const PARAM_DEFS = {
  name:        { label: 'Layer Name', type: 'text' },
  primType:    { label: 'Primitive', type: 'select', numeric: 'int', options: PRIM_TYPES },
  color:       { label: 'Color', type: 'color' },
  text:        { label: 'Text', type: 'text' },
  shape:       { label: 'Shape', type: 'textarea' },
  fill:        { label: 'Fill', type: 'checkbox' },
  width:       { label: 'Width', type: 'number', numeric: 'float', min: 0, max: 200, step: 0.1 },
  length:      { label: 'Length', type: 'number', numeric: 'float', min: 0, max: 200, step: 0.1 },
  aspect:      { label: 'Aspect', type: 'number', numeric: 'float', min: -200, max: 200, step: 0.1 },
  round:       { label: 'Round', type: 'number', numeric: 'float', min: -100, max: 100, step: 0.1 },
  step:        { label: 'Step', type: 'number', numeric: 'float', min: -360, max: 360, step: 0.1 },
  angleStep:   { label: 'Angle Step', type: 'number', numeric: 'float', min: -360, max: 360, step: 0.1 },
  lightDir:    { label: 'Light Dir', type: 'number', numeric: 'float', min: -360, max: 360, step: 0.1 },
  diffuse:     { label: 'Diffuse', type: 'number', numeric: 'float', min: -100, max: 200, step: 0.1 },
  fontSize:    { label: 'Font Size', type: 'number', numeric: 'float', min: 1, max: 300, step: 0.1 },
  textAlign:   { label: 'Text Align', type: 'select', numeric: 'int', options: [
    { value: 0, label: 'Left' },
    { value: 1, label: 'Center' },
    { value: 2, label: 'Right' }
  ] },
  frameAlign:  { label: 'Frame Align', type: 'select', numeric: 'int', options: [
    { value: 0, label: 'Vertical Strip' },
    { value: 1, label: 'Horizontal Strip' },
    { value: 2, label: 'Files' }
  ] },
  numFrame:    { label: 'Frames', type: 'number', numeric: 'int', min: 1, max: 256, step: 1 },
  autoFit:     { label: 'Auto Fit', type: 'checkbox' },
  transparent: { label: 'Transparent', type: 'select', numeric: 'int', options: [
    { value: 0, label: 'Off' },
    { value: 1, label: 'On' }
  ] },
  intelliAlpha:{ label: 'IntelliAlpha', type: 'select', numeric: 'int', options: [
    { value: 0, label: 'Off' },
    { value: 1, label: 'On' }
  ] }
};

// Phase 5.4 scope: primitive-type-aware panel using the currently exposed WASM params.
const PARAMS_BY_PRIM_TYPE = {
  0: [],
  1: ['autoFit', 'intelliAlpha', 'numFrame', 'frameAlign', 'transparent'],
  2: ['color', 'width', 'round', 'diffuse', 'lightDir'],
  3: ['color', 'aspect', 'diffuse', 'lightDir'],
  4: ['color', 'aspect', 'diffuse', 'lightDir'],
  5: ['color', 'width', 'step', 'length', 'diffuse'],
  6: ['color', 'aspect', 'diffuse', 'step', 'angleStep'],
  7: ['color', 'width', 'round', 'length', 'aspect', 'diffuse'],
  8: ['color', 'round', 'aspect', 'diffuse'],
  9: ['color', 'width', 'round', 'length', 'fill', 'diffuse'],
  10: ['color', 'width', 'length', 'lightDir'],
  11: ['color', 'width', 'length', 'angleStep', 'step'],
  12: ['color', 'width', 'step'],
  13: ['color', 'width', 'step'],
  14: ['color', 'text', 'fontSize', 'textAlign', 'frameAlign'],
  15: ['color', 'shape', 'fill', 'round', 'diffuse']
};

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
  refreshParamPanel();

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
  if (selectedLayer >= layers.length) selectedLayer = Math.max(0, layers.length - 1);

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
      refreshParamPanel();
      markDirty();
    });

    layerList.appendChild(li);
  });
}

// ── Primitive parameter panel ─────────────────────────────────────────────────

function fieldsForPrimType(primType) {
  return PARAMS_BY_PRIM_TYPE[primType] || [];
}

function coerceParamValue(def, input) {
  if (def.type === 'checkbox') {
    return input.checked;
  }
  if (def.type === 'color' || def.type === 'text' || def.type === 'textarea') {
    return input.value;
  }
  if (def.numeric === 'int') {
    const v = parseInt(input.value, 10);
    return Number.isFinite(v) ? v : 0;
  }
  if (def.numeric === 'float') {
    const v = parseFloat(input.value);
    return Number.isFinite(v) ? v : 0;
  }
  return input.value;
}

function buildParamRow(key, value) {
  const def = PARAM_DEFS[key];
  if (!def) return null;

  const row = document.createElement('label');
  row.className = 'param-row';
  if (def.type === 'checkbox') row.classList.add('checkbox');

  const caption = document.createElement('span');
  caption.textContent = def.label;

  let input;
  if (def.type === 'select') {
    input = document.createElement('select');
    (def.options || []).forEach(opt => {
      const el = document.createElement('option');
      el.value = String(opt.value);
      el.textContent = opt.label;
      input.appendChild(el);
    });
    input.value = String(value ?? 0);
  } else if (def.type === 'textarea') {
    input = document.createElement('textarea');
    input.rows = 4;
    input.value = String(value ?? '');
  } else {
    input = document.createElement('input');
    input.type = def.type;
    if (def.min != null) input.min = String(def.min);
    if (def.max != null) input.max = String(def.max);
    if (def.step != null) input.step = String(def.step);
    if (def.type === 'checkbox') {
      input.checked = Boolean(value);
    } else if (def.type === 'number') {
      input.value = String(value ?? 0);
    } else if (def.type === 'color') {
      input.value = String(value || '#000000');
    } else {
      input.value = String(value ?? '');
    }
  }

  const eventName = (def.type === 'select' || def.type === 'checkbox') ? 'change' : 'input';
  input.addEventListener(eventName, () => {
    const v = coerceParamValue(def, input);
    const ok = window.knobman_setParam(selectedLayer, key, v);
    if (!ok) return;

    if (key === 'name') {
      refreshLayerList();
    }
    if (key === 'primType') {
      refreshLayerList();
      refreshParamPanel();
    }
    markDirty();
  });

  if (def.type === 'checkbox') {
    row.appendChild(caption);
    row.appendChild(input);
  } else {
    row.appendChild(caption);
    row.appendChild(input);
  }
  return row;
}

function refreshParamPanel() {
  const content = document.getElementById('paramContent');
  content.innerHTML = '';

  const layers = window.knobman_getLayerList() || [];
  if (layers.length === 0) {
    const p = document.createElement('p');
    p.className = 'placeholder';
    p.textContent = 'No layer selected.';
    content.appendChild(p);
    return;
  }
  selectedLayer = Math.max(0, Math.min(selectedLayer, layers.length - 1));

  const primType = window.knobman_getParam(selectedLayer, 'primType') ?? 0;
  const fields = ['name', 'primType', ...fieldsForPrimType(primType)];

  fields.forEach(key => {
    const value = window.knobman_getParam(selectedLayer, key);
    const row = buildParamRow(key, value);
    if (row) content.appendChild(row);
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
  refreshParamPanel();
  markDirty();
}

function onDeleteLayer() {
  window.knobman_deleteLayer(selectedLayer);
  refreshLayerList();
  refreshParamPanel();
  markDirty();
}

function onMoveUp() {
  selectedLayer = window.knobman_moveLayer(-1);
  refreshLayerList();
  refreshParamPanel();
  markDirty();
}

function onMoveDown() {
  selectedLayer = window.knobman_moveLayer(1);
  refreshLayerList();
  refreshParamPanel();
  markDirty();
}

function onDuplicate() {
  selectedLayer = window.knobman_duplicateLayer();
  refreshLayerList();
  refreshParamPanel();
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
