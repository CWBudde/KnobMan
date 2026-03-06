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
  file:        { label: 'Image Name', type: 'text' },
  embeddedImage: { label: 'Image File', type: 'file', accept: 'image/*' },
  text:        { label: 'Text', type: 'text' },
  shape:       { label: 'Shape', type: 'textarea' },
  fill:        { label: 'Fill', type: 'checkbox' },
  bold:        { label: 'Bold', type: 'checkbox' },
  italic:      { label: 'Italic', type: 'checkbox' },
  font:        { label: 'Font', type: 'number', numeric: 'int', min: 0, max: 64, step: 1 },
  textureFile: { label: 'Texture Slot', type: 'number', numeric: 'int', min: 0, max: 64, step: 1 },
  textureName: { label: 'Texture Name', type: 'text' },
  width:       { label: 'Width', type: 'number', numeric: 'float', min: 0, max: 200, step: 0.1 },
  length:      { label: 'Length', type: 'number', numeric: 'float', min: 0, max: 200, step: 0.1 },
  aspect:      { label: 'Aspect', type: 'number', numeric: 'float', min: -200, max: 200, step: 0.1 },
  round:       { label: 'Round', type: 'number', numeric: 'float', min: -100, max: 100, step: 0.1 },
  step:        { label: 'Step', type: 'number', numeric: 'float', min: -360, max: 360, step: 0.1 },
  angleStep:   { label: 'Angle Step', type: 'number', numeric: 'float', min: -360, max: 360, step: 0.1 },
  emboss:      { label: 'Emboss', type: 'number', numeric: 'float', min: -100, max: 100, step: 0.1 },
  embossDiffuse: { label: 'Emboss Diffuse', type: 'number', numeric: 'float', min: -100, max: 200, step: 0.1 },
  ambient:     { label: 'Ambient', type: 'number', numeric: 'float', min: 0, max: 100, step: 0.1 },
  lightDir:    { label: 'Light Dir', type: 'number', numeric: 'float', min: -360, max: 360, step: 0.1 },
  specular:    { label: 'Specular', type: 'number', numeric: 'float', min: 0, max: 200, step: 0.1 },
  specularWidth: { label: 'Spec Width', type: 'number', numeric: 'float', min: 0, max: 200, step: 0.1 },
  textureDepth:{ label: 'Texture Depth', type: 'number', numeric: 'float', min: -100, max: 200, step: 0.1 },
  textureZoom: { label: 'Texture Zoom', type: 'number', numeric: 'float', min: 1, max: 400, step: 0.1 },
  diffuse:     { label: 'Diffuse', type: 'number', numeric: 'float', min: -100, max: 200, step: 0.1 },
  fontSize:    { label: 'Font Size', type: 'number', numeric: 'float', min: 1, max: 300, step: 0.1 },
  textAlign:   { label: 'Text Align', type: 'select', numeric: 'int', options: [
    { value: 0, label: 'Center' },
    { value: 1, label: 'Left' },
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
    { value: 0, label: 'File Alpha' },
    { value: 1, label: 'Force Opaque' },
    { value: 2, label: 'Color Key' }
  ] },
  intelliAlpha:{ label: 'IntelliAlpha', type: 'number', numeric: 'int', min: 0, max: 100, step: 1 }
};

const PARAMS_BY_PRIM_TYPE = {
  0: [],
  1: ['embeddedImage', 'file', 'autoFit', 'intelliAlpha', 'numFrame', 'frameAlign', 'transparent'],
  2: ['color', 'width', 'round', 'diffuse', 'emboss', 'embossDiffuse', 'ambient', 'lightDir', 'specular', 'specularWidth', 'textureDepth', 'textureZoom', 'textureFile'],
  3: ['color', 'aspect', 'diffuse', 'ambient', 'lightDir', 'specular', 'specularWidth', 'textureDepth', 'textureZoom', 'textureFile'],
  4: ['color', 'aspect', 'diffuse', 'ambient', 'lightDir', 'specular', 'specularWidth', 'textureDepth', 'textureZoom', 'textureFile'],
  5: ['color', 'width', 'step', 'length', 'diffuse', 'emboss', 'embossDiffuse', 'ambient', 'lightDir', 'specular', 'specularWidth'],
  6: ['color', 'aspect', 'diffuse', 'step', 'angleStep'],
  7: ['color', 'width', 'round', 'length', 'aspect', 'diffuse', 'emboss', 'embossDiffuse', 'ambient', 'lightDir', 'specular', 'specularWidth'],
  8: ['color', 'round', 'aspect', 'diffuse', 'emboss', 'embossDiffuse', 'ambient', 'lightDir', 'specular', 'specularWidth', 'textureDepth', 'textureZoom', 'textureFile'],
  9: ['color', 'width', 'round', 'length', 'fill', 'diffuse'],
  10: ['color', 'width', 'length', 'lightDir'],
  11: ['color', 'width', 'length', 'angleStep', 'step'],
  12: ['color', 'width', 'step'],
  13: ['color', 'width', 'step'],
  14: ['color', 'text', 'font', 'fontSize', 'textAlign', 'frameAlign', 'bold', 'italic'],
  15: ['color', 'shape', 'fill', 'round', 'diffuse', 'width']
};

const CURVE_OPTIONS = [
  { value: 0, label: 'Linear' },
  { value: 1, label: 'Curve 1' },
  { value: 2, label: 'Curve 2' },
  { value: 3, label: 'Curve 3' },
  { value: 4, label: 'Curve 4' },
  { value: 5, label: 'Curve 5' },
  { value: 6, label: 'Curve 6' },
  { value: 7, label: 'Curve 7' },
  { value: 8, label: 'Curve 8' }
];

const EFFECT_DEFS = {
  antiAlias:   { label: 'AntiAlias', type: 'checkbox' },
  unfold:      { label: 'Unfold', type: 'checkbox' },
  animStep:    { label: 'AnimStep', type: 'number', numeric: 'int', min: 0, max: 1024, step: 1 },
  zoomXYSepa:  { label: 'Zoom XY Sepa', type: 'checkbox' },
  zoomXF:      { label: 'Zoom X From', type: 'number', numeric: 'float', min: -500, max: 500, step: 0.1 },
  zoomXT:      { label: 'Zoom X To', type: 'number', numeric: 'float', min: -500, max: 500, step: 0.1 },
  zoomXAnim:   { label: 'Zoom X Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },
  zoomYF:      { label: 'Zoom Y From', type: 'number', numeric: 'float', min: -500, max: 500, step: 0.1 },
  zoomYT:      { label: 'Zoom Y To', type: 'number', numeric: 'float', min: -500, max: 500, step: 0.1 },
  zoomYAnim:   { label: 'Zoom Y Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },
  offXF:       { label: 'Offset X From', type: 'number', numeric: 'float', min: -500, max: 500, step: 0.1 },
  offXT:       { label: 'Offset X To', type: 'number', numeric: 'float', min: -500, max: 500, step: 0.1 },
  offXAnim:    { label: 'Offset X Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },
  offYF:       { label: 'Offset Y From', type: 'number', numeric: 'float', min: -500, max: 500, step: 0.1 },
  offYT:       { label: 'Offset Y To', type: 'number', numeric: 'float', min: -500, max: 500, step: 0.1 },
  offYAnim:    { label: 'Offset Y Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },
  keepDir:     { label: 'Keep Dir', type: 'checkbox' },
  centerX:     { label: 'Center X', type: 'number', numeric: 'float', min: -500, max: 500, step: 0.1 },
  centerY:     { label: 'Center Y', type: 'number', numeric: 'float', min: -500, max: 500, step: 0.1 },
  angleF:      { label: 'Angle From', type: 'number', numeric: 'float', min: -3600, max: 3600, step: 0.1 },
  angleT:      { label: 'Angle To', type: 'number', numeric: 'float', min: -3600, max: 3600, step: 0.1 },
  angleAnim:   { label: 'Angle Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },

  alphaF:      { label: 'Alpha From', type: 'number', numeric: 'float', min: 0, max: 100, step: 0.1 },
  alphaT:      { label: 'Alpha To', type: 'number', numeric: 'float', min: 0, max: 100, step: 0.1 },
  alphaAnim:   { label: 'Alpha Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },
  brightF:     { label: 'Brightness From', type: 'number', numeric: 'float', min: -100, max: 100, step: 0.1 },
  brightT:     { label: 'Brightness To', type: 'number', numeric: 'float', min: -100, max: 100, step: 0.1 },
  brightAnim:  { label: 'Brightness Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },
  contrastF:   { label: 'Contrast From', type: 'number', numeric: 'float', min: -100, max: 100, step: 0.1 },
  contrastT:   { label: 'Contrast To', type: 'number', numeric: 'float', min: -100, max: 100, step: 0.1 },
  contrastAnim:{ label: 'Contrast Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },
  saturationF: { label: 'Saturation From', type: 'number', numeric: 'float', min: -100, max: 100, step: 0.1 },
  saturationT: { label: 'Saturation To', type: 'number', numeric: 'float', min: -100, max: 100, step: 0.1 },
  saturationAnim: { label: 'Saturation Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },
  hueF:        { label: 'Hue From', type: 'number', numeric: 'float', min: -360, max: 360, step: 0.1 },
  hueT:        { label: 'Hue To', type: 'number', numeric: 'float', min: -360, max: 360, step: 0.1 },
  hueAnim:     { label: 'Hue Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },

  mask1Ena:    { label: 'Enable', type: 'checkbox' },
  mask1Type:   { label: 'Type', type: 'select', numeric: 'int', options: [
    { value: 0, label: 'Rotation' }, { value: 1, label: 'Radial' }, { value: 2, label: 'Horizontal' }, { value: 3, label: 'Vertical' }
  ] },
  mask1Grad:   { label: 'Gradation', type: 'number', numeric: 'float', min: 0, max: 100, step: 0.1 },
  mask1GradDir:{ label: 'Grad Dir', type: 'number', numeric: 'int', min: 0, max: 8, step: 1 },
  mask1StartF: { label: 'Start From', type: 'number', numeric: 'float', min: -360, max: 360, step: 0.1 },
  mask1StartT: { label: 'Start To', type: 'number', numeric: 'float', min: -360, max: 360, step: 0.1 },
  mask1StartAnim: { label: 'Start Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },
  mask1StopF:  { label: 'Stop From', type: 'number', numeric: 'float', min: -360, max: 360, step: 0.1 },
  mask1StopT:  { label: 'Stop To', type: 'number', numeric: 'float', min: -360, max: 360, step: 0.1 },
  mask1StopAnim: { label: 'Stop Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },

  mask2Ena:    { label: 'Enable', type: 'checkbox' },
  mask2Op:     { label: 'Operation', type: 'select', numeric: 'int', options: [
    { value: 0, label: 'AND' }, { value: 1, label: 'OR' }
  ] },
  mask2Type:   { label: 'Type', type: 'select', numeric: 'int', options: [
    { value: 0, label: 'Rotation' }, { value: 1, label: 'Radial' }, { value: 2, label: 'Horizontal' }, { value: 3, label: 'Vertical' }
  ] },
  mask2Grad:   { label: 'Gradation', type: 'number', numeric: 'float', min: 0, max: 100, step: 0.1 },
  mask2GradDir:{ label: 'Grad Dir', type: 'number', numeric: 'int', min: 0, max: 8, step: 1 },
  mask2StartF: { label: 'Start From', type: 'number', numeric: 'float', min: -360, max: 360, step: 0.1 },
  mask2StartT: { label: 'Start To', type: 'number', numeric: 'float', min: -360, max: 360, step: 0.1 },
  mask2StartAnim: { label: 'Start Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },
  mask2StopF:  { label: 'Stop From', type: 'number', numeric: 'float', min: -360, max: 360, step: 0.1 },
  mask2StopT:  { label: 'Stop To', type: 'number', numeric: 'float', min: -360, max: 360, step: 0.1 },
  mask2StopAnim: { label: 'Stop Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },

  fMaskEna:    { label: 'Mode', type: 'select', numeric: 'int', options: [
    { value: 0, label: 'Off' }, { value: 1, label: 'Range' }, { value: 2, label: 'Bitmask' }
  ] },
  fMaskStart:  { label: 'Range Start', type: 'number', numeric: 'float', min: 0, max: 100, step: 0.1 },
  fMaskStop:   { label: 'Range Stop', type: 'number', numeric: 'float', min: 0, max: 100, step: 0.1 },
  fMaskBits:   { label: 'Bitmask', type: 'text' },

  sLightDirF:  { label: 'LightDir From', type: 'number', numeric: 'float', min: -360, max: 360, step: 0.1 },
  sLightDirT:  { label: 'LightDir To', type: 'number', numeric: 'float', min: -360, max: 360, step: 0.1 },
  sLightDirAnim: { label: 'LightDir Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },
  sDensityF:   { label: 'Density From', type: 'number', numeric: 'float', min: 0, max: 100, step: 0.1 },
  sDensityT:   { label: 'Density To', type: 'number', numeric: 'float', min: 0, max: 100, step: 0.1 },
  sDensityAnim:{ label: 'Density Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },

  dLightDirEna:{ label: 'Enable', type: 'checkbox' },
  dLightDirF:  { label: 'LightDir From', type: 'number', numeric: 'float', min: -360, max: 360, step: 0.1 },
  dLightDirT:  { label: 'LightDir To', type: 'number', numeric: 'float', min: -360, max: 360, step: 0.1 },
  dLightDirAnim: { label: 'LightDir Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },
  dOffsetF:    { label: 'Offset From', type: 'number', numeric: 'float', min: -500, max: 500, step: 0.1 },
  dOffsetT:    { label: 'Offset To', type: 'number', numeric: 'float', min: -500, max: 500, step: 0.1 },
  dOffsetAnim: { label: 'Offset Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },
  dDensityF:   { label: 'Density From', type: 'number', numeric: 'float', min: 0, max: 100, step: 0.1 },
  dDensityT:   { label: 'Density To', type: 'number', numeric: 'float', min: 0, max: 100, step: 0.1 },
  dDensityAnim:{ label: 'Density Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },
  dDiffuseF:   { label: 'Diffuse From', type: 'number', numeric: 'float', min: 0, max: 100, step: 0.1 },
  dDiffuseT:   { label: 'Diffuse To', type: 'number', numeric: 'float', min: 0, max: 100, step: 0.1 },
  dDiffuseAnim:{ label: 'Diffuse Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },
  dsType:      { label: 'Shadow Type', type: 'select', numeric: 'int', options: [
    { value: 0, label: 'Soft' }, { value: 1, label: 'Hard' }
  ] },
  dsGrad:      { label: 'Gradient', type: 'number', numeric: 'float', min: 0, max: 100, step: 0.1 },

  iLightDirEna:{ label: 'Enable', type: 'checkbox' },
  iLightDirF:  { label: 'LightDir From', type: 'number', numeric: 'float', min: -360, max: 360, step: 0.1 },
  iLightDirT:  { label: 'LightDir To', type: 'number', numeric: 'float', min: -360, max: 360, step: 0.1 },
  iLightDirAnim: { label: 'LightDir Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },
  iOffsetF:    { label: 'Offset From', type: 'number', numeric: 'float', min: -500, max: 500, step: 0.1 },
  iOffsetT:    { label: 'Offset To', type: 'number', numeric: 'float', min: -500, max: 500, step: 0.1 },
  iOffsetAnim: { label: 'Offset Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },
  iDensityF:   { label: 'Density From', type: 'number', numeric: 'float', min: 0, max: 100, step: 0.1 },
  iDensityT:   { label: 'Density To', type: 'number', numeric: 'float', min: 0, max: 100, step: 0.1 },
  iDensityAnim:{ label: 'Density Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },
  iDiffuseF:   { label: 'Diffuse From', type: 'number', numeric: 'float', min: 0, max: 100, step: 0.1 },
  iDiffuseT:   { label: 'Diffuse To', type: 'number', numeric: 'float', min: 0, max: 100, step: 0.1 },
  iDiffuseAnim:{ label: 'Diffuse Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },

  eLightDirEna:{ label: 'Enable', type: 'checkbox' },
  eLightDirF:  { label: 'LightDir From', type: 'number', numeric: 'float', min: -360, max: 360, step: 0.1 },
  eLightDirT:  { label: 'LightDir To', type: 'number', numeric: 'float', min: -360, max: 360, step: 0.1 },
  eLightDirAnim: { label: 'LightDir Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },
  eOffsetF:    { label: 'Offset From', type: 'number', numeric: 'float', min: -500, max: 500, step: 0.1 },
  eOffsetT:    { label: 'Offset To', type: 'number', numeric: 'float', min: -500, max: 500, step: 0.1 },
  eOffsetAnim: { label: 'Offset Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS },
  eDensityF:   { label: 'Density From', type: 'number', numeric: 'float', min: 0, max: 100, step: 0.1 },
  eDensityT:   { label: 'Density To', type: 'number', numeric: 'float', min: 0, max: 100, step: 0.1 },
  eDensityAnim:{ label: 'Density Curve', type: 'select', numeric: 'int', options: CURVE_OPTIONS }
};

const EFFECT_SECTIONS = [
  { title: 'Transform', open: true, fields: ['antiAlias', 'unfold', 'animStep', 'zoomXYSepa', 'zoomXF', 'zoomXT', 'zoomXAnim', 'zoomYF', 'zoomYT', 'zoomYAnim', 'offXF', 'offXT', 'offXAnim', 'offYF', 'offYT', 'offYAnim', 'keepDir', 'centerX', 'centerY', 'angleF', 'angleT', 'angleAnim'] },
  { title: 'Color', open: true, fields: ['alphaF', 'alphaT', 'alphaAnim', 'brightF', 'brightT', 'brightAnim', 'contrastF', 'contrastT', 'contrastAnim', 'saturationF', 'saturationT', 'saturationAnim', 'hueF', 'hueT', 'hueAnim'] },
  { title: 'Mask 1', fields: ['mask1Ena', 'mask1Type', 'mask1Grad', 'mask1GradDir', 'mask1StartF', 'mask1StartT', 'mask1StartAnim', 'mask1StopF', 'mask1StopT', 'mask1StopAnim'] },
  { title: 'Mask 2', fields: ['mask2Ena', 'mask2Op', 'mask2Type', 'mask2Grad', 'mask2GradDir', 'mask2StartF', 'mask2StartT', 'mask2StartAnim', 'mask2StopF', 'mask2StopT', 'mask2StopAnim'] },
  { title: 'Frame Mask', fields: ['fMaskEna', 'fMaskStart', 'fMaskStop', 'fMaskBits'] },
  { title: 'Specular Highlight', fields: ['sLightDirF', 'sLightDirT', 'sLightDirAnim', 'sDensityF', 'sDensityT', 'sDensityAnim'] },
  { title: 'Drop Shadow', fields: ['dLightDirEna', 'dLightDirF', 'dLightDirT', 'dLightDirAnim', 'dOffsetF', 'dOffsetT', 'dOffsetAnim', 'dDensityF', 'dDensityT', 'dDensityAnim', 'dDiffuseF', 'dDiffuseT', 'dDiffuseAnim', 'dsType', 'dsGrad'] },
  { title: 'Inner Shadow', fields: ['iLightDirEna', 'iLightDirF', 'iLightDirT', 'iLightDirAnim', 'iOffsetF', 'iOffsetT', 'iOffsetAnim', 'iDensityF', 'iDensityT', 'iDensityAnim', 'iDiffuseF', 'iDiffuseT', 'iDiffuseAnim'] },
  { title: 'Emboss', fields: ['eLightDirEna', 'eLightDirF', 'eLightDirT', 'eLightDirAnim', 'eOffsetF', 'eOffsetT', 'eOffsetAnim', 'eDensityF', 'eDensityT', 'eDensityAnim'] }
];

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
  if (def.type === 'color' || def.type === 'text' || def.type === 'textarea' || def.type === 'file') {
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

function applyParamChange(key, value) {
  const ok = window.knobman_setParam(selectedLayer, key, value);
  if (!ok) return false;
  if (key === 'name') {
    refreshLayerList();
  }
  if (key === 'primType') {
    refreshLayerList();
    refreshParamPanel();
  }
  markDirty();
  return true;
}

function applyEffectParamChange(key, value) {
  const ok = window.knobman_setEffectParam(selectedLayer, key, value);
  if (!ok) return false;
  markDirty();
  return true;
}

function buildParamRow(key, value) {
  const def = PARAM_DEFS[key];
  if (!def) return null;

  const row = document.createElement('div');
  row.className = 'param-row';
  if (def.type === 'checkbox') row.classList.add('checkbox');

  const caption = document.createElement('span');
  caption.textContent = def.label;

  if (def.type === 'file') {
    row.appendChild(caption);

    const wrap = document.createElement('div');
    wrap.className = 'param-file';

    const input = document.createElement('input');
    input.type = 'file';
    if (def.accept) input.accept = def.accept;

    const clear = document.createElement('button');
    clear.type = 'button';
    clear.textContent = 'Clear';
    clear.addEventListener('click', () => {
      if (!applyParamChange('clearEmbeddedImage', 1)) return;
      applyParamChange('file', '');
      refreshParamPanel();
    });

    input.addEventListener('change', () => {
      const file = input.files && input.files[0];
      if (!file) return;
      const reader = new FileReader();
      reader.onload = () => {
        const data = new Uint8Array(reader.result);
        const okImg = applyParamChange('embeddedImage', data);
        const okName = applyParamChange('file', file.name);
        if (okImg && okName) {
          refreshParamPanel();
          setStatus('Loaded image ' + file.name);
        }
      };
      reader.readAsArrayBuffer(file);
      input.value = '';
    });

    const hint = document.createElement('small');
    const has = Boolean(window.knobman_getParam(selectedLayer, 'hasEmbeddedImage'));
    const name = String(window.knobman_getParam(selectedLayer, 'file') || '');
    hint.textContent = has ? ('Embedded: ' + (name || '(unnamed)')) : 'No embedded image';

    wrap.appendChild(input);
    wrap.appendChild(clear);
    row.appendChild(wrap);
    row.appendChild(hint);
    return row;
  }

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
    applyParamChange(key, v);
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

function buildEffectRow(key, value) {
  const def = EFFECT_DEFS[key];
  if (!def) return null;

  const row = document.createElement('div');
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
    input.rows = 3;
    input.value = String(value ?? '');
  } else {
    input = document.createElement('input');
    input.type = def.type;
    if (def.min != null) input.min = String(def.min);
    if (def.max != null) input.max = String(def.max);
    if (def.step != null) input.step = String(def.step);
    if (def.type === 'checkbox') {
      input.checked = Boolean(value);
    } else {
      input.value = String(value ?? '');
    }
  }

  const eventName = (def.type === 'select' || def.type === 'checkbox') ? 'change' : 'input';
  input.addEventListener(eventName, () => {
    const v = coerceParamValue(def, input);
    applyEffectParamChange(key, v);
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

function appendEffectSections(content) {
  const title = document.createElement('div');
  title.className = 'param-group-title';
  title.textContent = 'Effects';
  content.appendChild(title);

  EFFECT_SECTIONS.forEach(section => {
    const details = document.createElement('details');
    details.className = 'effect-section';
    details.open = Boolean(section.open);

    const summary = document.createElement('summary');
    summary.textContent = section.title;
    details.appendChild(summary);

    const body = document.createElement('div');
    body.className = 'effect-section-body';
    section.fields.forEach(key => {
      const value = window.knobman_getEffectParam(selectedLayer, key);
      const row = buildEffectRow(key, value);
      if (row) body.appendChild(row);
    });
    details.appendChild(body);
    content.appendChild(details);
  });
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

  appendEffectSections(content);
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
