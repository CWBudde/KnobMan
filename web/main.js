"use strict";

// ── WASM bootstrap ────────────────────────────────────────────────────────────

const go = new Go();
let wasmReady = false;

WebAssembly.instantiateStreaming(
  fetch("knobman.wasm", { cache: "no-store" }),
  go.importObject,
)
  .then((result) => {
    go.run(result.instance);
    wasmReady = true;
    onWasmReady();
  })
  .catch((err) => {
    document.getElementById("loading").textContent =
      "Failed to load WASM: " + err;
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
let prefAspectLock = false;
let prefAspectRatio = 1;
let builtinTextureLoadPromise = null;
let projectBaseName = "project";
let detachedPreviewWindow = null;
let detachedPreviewTimer = null;
let detachedPreviewPlaying = false;
let detachedPreviewFrame = 0;
let detachedPreviewDir = 1;
let layerPreviewToken = 0;
let layerPreviewRevision = 1;
const layerPreviewCache = new Map();
const LAYER_PREVIEW_SIZE = 36;
let selectedCurve = 1;
let curveCanvas = null;
let curveCtx = null;
let curveDragPoint = -1;
let curveSelectedPoint = -1;
let shapeMode = "L";
let shapeCanvas = null;
let shapeCtx = null;
let shapeCommands = [];
let shapeDragHandle = null;
let shapeSelectedHandle = null;

const CURVE_POINT_LIMIT = 12;

const BUILTIN_TEXTURE_FILES = [
  "Aura.jpg",
  "Checkers.bmp",
  "Circle.jpg",
  "Coal.jpg",
  "Fabric.bmp",
  "Hairline.bmp",
  "HairlineV.bmp",
  "Hexagon.png",
  "Magma.bmp",
  "Mosaic.bmp",
  "Plant.png",
  "Plasma.bmp",
  "PunchingMetal.png",
  "Radiation.jpg",
  "Sand.bmp",
  "Scratch.jpg",
  "Stripe.bmp",
  "StripeV.bmp",
];

const SAMPLE_PROJECT_FILES = [
  "2Color_Pointed.knob",
  "3p_wedge.knob",
  "Aqua.knob",
  "Black_Gear.knob",
  "BlueDot_V.knob",
  "Blue_HSW.knob",
  "Blue_HSW2.knob",
  "CheckBox.knob",
  "ColorRing.knob",
  "CorkBoard.knob",
  "FabricStar.knob",
  "Granite.knob",
  "Gray_Ring2.knob",
  "GreenAb.knob",
  "Green_Radar.knob",
  "Green_VSW.knob",
  "LineShadow.knob",
  "Monotone_Simple.knob",
  "Number.knob",
  "Number_HSwitch.knob",
  "NumberedTick.knob",
  "Orange_Lever.knob",
  "Orange_Round.knob",
  "Pop_Meter.knob",
  "Red_Gear.knob",
  "Shape_sample.knob",
  "Small_Gaged.knob",
  "Ticked_HSlider.knob",
  "Waveform.knob",
  "White_Dip.knob",
  "White_Pan.knob",
  "White_Vol.knob",
  "White_Wave.knob",
  "Wood_Gear.knob",
  "face.knob",
  "g200kglogo.knob",
  "led.knob",
  "vu3.knob",
];

const PRIM_TYPES = [
  { value: 0, label: "None" },
  { value: 1, label: "Image" },
  { value: 2, label: "Circle" },
  { value: 3, label: "CircleFill" },
  { value: 4, label: "MetalCircle" },
  { value: 5, label: "WaveCircle" },
  { value: 6, label: "Sphere" },
  { value: 7, label: "Rect" },
  { value: 8, label: "RectFill" },
  { value: 9, label: "Triangle" },
  { value: 10, label: "Line" },
  { value: 11, label: "RadiateLine" },
  { value: 12, label: "H-Lines" },
  { value: 13, label: "V-Lines" },
  { value: 14, label: "Text" },
  { value: 15, label: "Shape" },
];

const PARAM_DEFS = {
  name: { label: "Layer Name", type: "text" },
  primType: {
    label: "Primitive",
    type: "select",
    numeric: "int",
    options: PRIM_TYPES,
  },
  color: { label: "Color", type: "color" },
  file: { label: "Image Name", type: "text" },
  embeddedImage: { label: "Image File", type: "file", accept: "image/*" },
  text: { label: "Text", type: "text" },
  shape: { label: "Shape", type: "textarea" },
  fill: { label: "Fill", type: "checkbox" },
  bold: { label: "Bold", type: "checkbox" },
  italic: { label: "Italic", type: "checkbox" },
  font: {
    label: "Font",
    type: "number",
    numeric: "int",
    min: 0,
    max: 64,
    step: 1,
  },
  textureFile: {
    label: "Texture Slot",
    type: "number",
    numeric: "int",
    min: 0,
    max: 64,
    step: 1,
  },
  textureName: { label: "Texture Name", type: "text" },
  width: {
    label: "Width",
    type: "number",
    numeric: "float",
    min: 0,
    max: 200,
    step: 0.1,
  },
  length: {
    label: "Length",
    type: "number",
    numeric: "float",
    min: 0,
    max: 200,
    step: 0.1,
  },
  aspect: {
    label: "Aspect",
    type: "number",
    numeric: "float",
    min: -200,
    max: 200,
    step: 0.1,
  },
  round: {
    label: "Round",
    type: "number",
    numeric: "float",
    min: -100,
    max: 100,
    step: 0.1,
  },
  step: {
    label: "Step",
    type: "number",
    numeric: "float",
    min: -360,
    max: 360,
    step: 0.1,
  },
  angleStep: {
    label: "Angle Step",
    type: "number",
    numeric: "float",
    min: -360,
    max: 360,
    step: 0.1,
  },
  emboss: {
    label: "Emboss",
    type: "number",
    numeric: "float",
    min: -100,
    max: 100,
    step: 0.1,
  },
  embossDiffuse: {
    label: "Emboss Diffuse",
    type: "number",
    numeric: "float",
    min: -100,
    max: 200,
    step: 0.1,
  },
  ambient: {
    label: "Ambient",
    type: "number",
    numeric: "float",
    min: 0,
    max: 100,
    step: 0.1,
  },
  lightDir: {
    label: "Light Dir",
    type: "number",
    numeric: "float",
    min: -360,
    max: 360,
    step: 0.1,
  },
  specular: {
    label: "Specular",
    type: "number",
    numeric: "float",
    min: 0,
    max: 200,
    step: 0.1,
  },
  specularWidth: {
    label: "Spec Width",
    type: "number",
    numeric: "float",
    min: 0,
    max: 200,
    step: 0.1,
  },
  textureDepth: {
    label: "Texture Depth",
    type: "number",
    numeric: "float",
    min: -100,
    max: 200,
    step: 0.1,
  },
  textureZoom: {
    label: "Texture Zoom",
    type: "number",
    numeric: "float",
    min: 1,
    max: 400,
    step: 0.1,
  },
  diffuse: {
    label: "Diffuse",
    type: "number",
    numeric: "float",
    min: -100,
    max: 200,
    step: 0.1,
  },
  fontSize: {
    label: "Font Size",
    type: "number",
    numeric: "float",
    min: 1,
    max: 300,
    step: 0.1,
  },
  textAlign: {
    label: "Text Align",
    type: "select",
    numeric: "int",
    options: [
      { value: 0, label: "Center" },
      { value: 1, label: "Left" },
      { value: 2, label: "Right" },
    ],
  },
  frameAlign: {
    label: "Frame Align",
    type: "select",
    numeric: "int",
    options: [
      { value: 0, label: "Vertical Strip" },
      { value: 1, label: "Horizontal Strip" },
      { value: 2, label: "Files" },
    ],
  },
  numFrame: {
    label: "Frames",
    type: "number",
    numeric: "int",
    min: 1,
    max: 256,
    step: 1,
  },
  autoFit: { label: "Auto Fit", type: "checkbox" },
  transparent: {
    label: "Transparent",
    type: "select",
    numeric: "int",
    options: [
      { value: 0, label: "File Alpha" },
      { value: 1, label: "Force Opaque" },
      { value: 2, label: "Color Key" },
    ],
  },
  intelliAlpha: {
    label: "IntelliAlpha",
    type: "number",
    numeric: "int",
    min: 0,
    max: 100,
    step: 1,
  },
};

const PARAMS_BY_PRIM_TYPE = {
  0: [],
  1: [
    "embeddedImage",
    "file",
    "autoFit",
    "intelliAlpha",
    "numFrame",
    "frameAlign",
    "transparent",
  ],
  2: [
    "color",
    "width",
    "round",
    "diffuse",
    "emboss",
    "embossDiffuse",
    "ambient",
    "lightDir",
    "specular",
    "specularWidth",
    "textureDepth",
    "textureZoom",
    "textureFile",
  ],
  3: [
    "color",
    "aspect",
    "diffuse",
    "ambient",
    "lightDir",
    "specular",
    "specularWidth",
    "textureDepth",
    "textureZoom",
    "textureFile",
  ],
  4: [
    "color",
    "aspect",
    "diffuse",
    "ambient",
    "lightDir",
    "specular",
    "specularWidth",
    "textureDepth",
    "textureZoom",
    "textureFile",
  ],
  5: [
    "color",
    "width",
    "step",
    "length",
    "diffuse",
    "emboss",
    "embossDiffuse",
    "ambient",
    "lightDir",
    "specular",
    "specularWidth",
  ],
  6: ["color", "aspect", "diffuse", "step", "angleStep"],
  7: [
    "color",
    "width",
    "round",
    "length",
    "aspect",
    "diffuse",
    "emboss",
    "embossDiffuse",
    "ambient",
    "lightDir",
    "specular",
    "specularWidth",
  ],
  8: [
    "color",
    "round",
    "aspect",
    "diffuse",
    "emboss",
    "embossDiffuse",
    "ambient",
    "lightDir",
    "specular",
    "specularWidth",
    "textureDepth",
    "textureZoom",
    "textureFile",
  ],
  9: ["color", "width", "round", "length", "fill", "diffuse"],
  10: ["color", "width", "length", "lightDir"],
  11: ["color", "width", "length", "angleStep", "step"],
  12: ["color", "width", "step"],
  13: ["color", "width", "step"],
  14: [
    "color",
    "text",
    "font",
    "fontSize",
    "textAlign",
    "frameAlign",
    "bold",
    "italic",
  ],
  15: ["color", "shape", "fill", "round", "diffuse", "width"],
};

const CURVE_OPTIONS = [
  { value: 0, label: "Off" },
  { value: 1, label: "Linear" },
  { value: 2, label: "Curve 1" },
  { value: 3, label: "Curve 2" },
  { value: 4, label: "Curve 3" },
  { value: 5, label: "Curve 4" },
  { value: 6, label: "Curve 5" },
  { value: 7, label: "Curve 6" },
  { value: 8, label: "Curve 7" },
  { value: 9, label: "Curve 8" },
];

const EFFECT_DEFS = {
  antiAlias: { label: "AntiAlias", type: "checkbox" },
  unfold: { label: "Unfold", type: "checkbox" },
  animStep: {
    label: "AnimStep",
    type: "number",
    numeric: "int",
    min: 0,
    max: 1024,
    step: 1,
  },
  zoomXYSepa: { label: "Zoom XY Sepa", type: "checkbox" },
  zoomXF: {
    label: "Zoom X From",
    type: "number",
    numeric: "float",
    min: -500,
    max: 500,
    step: 0.1,
  },
  zoomXT: {
    label: "Zoom X To",
    type: "number",
    numeric: "float",
    min: -500,
    max: 500,
    step: 0.1,
  },
  zoomXAnim: {
    label: "Zoom X Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },
  zoomYF: {
    label: "Zoom Y From",
    type: "number",
    numeric: "float",
    min: -500,
    max: 500,
    step: 0.1,
  },
  zoomYT: {
    label: "Zoom Y To",
    type: "number",
    numeric: "float",
    min: -500,
    max: 500,
    step: 0.1,
  },
  zoomYAnim: {
    label: "Zoom Y Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },
  offXF: {
    label: "Offset X From",
    type: "number",
    numeric: "float",
    min: -500,
    max: 500,
    step: 0.1,
  },
  offXT: {
    label: "Offset X To",
    type: "number",
    numeric: "float",
    min: -500,
    max: 500,
    step: 0.1,
  },
  offXAnim: {
    label: "Offset X Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },
  offYF: {
    label: "Offset Y From",
    type: "number",
    numeric: "float",
    min: -500,
    max: 500,
    step: 0.1,
  },
  offYT: {
    label: "Offset Y To",
    type: "number",
    numeric: "float",
    min: -500,
    max: 500,
    step: 0.1,
  },
  offYAnim: {
    label: "Offset Y Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },
  keepDir: { label: "Keep Dir", type: "checkbox" },
  centerX: {
    label: "Center X",
    type: "number",
    numeric: "float",
    min: -500,
    max: 500,
    step: 0.1,
  },
  centerY: {
    label: "Center Y",
    type: "number",
    numeric: "float",
    min: -500,
    max: 500,
    step: 0.1,
  },
  angleF: {
    label: "Angle From",
    type: "number",
    numeric: "float",
    min: -3600,
    max: 3600,
    step: 0.1,
  },
  angleT: {
    label: "Angle To",
    type: "number",
    numeric: "float",
    min: -3600,
    max: 3600,
    step: 0.1,
  },
  angleAnim: {
    label: "Angle Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },

  alphaF: {
    label: "Alpha From",
    type: "number",
    numeric: "float",
    min: 0,
    max: 100,
    step: 0.1,
  },
  alphaT: {
    label: "Alpha To",
    type: "number",
    numeric: "float",
    min: 0,
    max: 100,
    step: 0.1,
  },
  alphaAnim: {
    label: "Alpha Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },
  brightF: {
    label: "Brightness From",
    type: "number",
    numeric: "float",
    min: -100,
    max: 100,
    step: 0.1,
  },
  brightT: {
    label: "Brightness To",
    type: "number",
    numeric: "float",
    min: -100,
    max: 100,
    step: 0.1,
  },
  brightAnim: {
    label: "Brightness Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },
  contrastF: {
    label: "Contrast From",
    type: "number",
    numeric: "float",
    min: -100,
    max: 100,
    step: 0.1,
  },
  contrastT: {
    label: "Contrast To",
    type: "number",
    numeric: "float",
    min: -100,
    max: 100,
    step: 0.1,
  },
  contrastAnim: {
    label: "Contrast Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },
  saturationF: {
    label: "Saturation From",
    type: "number",
    numeric: "float",
    min: -100,
    max: 100,
    step: 0.1,
  },
  saturationT: {
    label: "Saturation To",
    type: "number",
    numeric: "float",
    min: -100,
    max: 100,
    step: 0.1,
  },
  saturationAnim: {
    label: "Saturation Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },
  hueF: {
    label: "Hue From",
    type: "number",
    numeric: "float",
    min: -360,
    max: 360,
    step: 0.1,
  },
  hueT: {
    label: "Hue To",
    type: "number",
    numeric: "float",
    min: -360,
    max: 360,
    step: 0.1,
  },
  hueAnim: {
    label: "Hue Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },

  mask1Ena: { label: "Enable", type: "checkbox" },
  mask1Type: {
    label: "Type",
    type: "select",
    numeric: "int",
    options: [
      { value: 0, label: "Rotation" },
      { value: 1, label: "Radial" },
      { value: 2, label: "Horizontal" },
      { value: 3, label: "Vertical" },
    ],
  },
  mask1Grad: {
    label: "Gradation",
    type: "number",
    numeric: "float",
    min: 0,
    max: 100,
    step: 0.1,
  },
  mask1GradDir: {
    label: "Grad Dir",
    type: "number",
    numeric: "int",
    min: 0,
    max: 8,
    step: 1,
  },
  mask1StartF: {
    label: "Start From",
    type: "number",
    numeric: "float",
    min: -360,
    max: 360,
    step: 0.1,
  },
  mask1StartT: {
    label: "Start To",
    type: "number",
    numeric: "float",
    min: -360,
    max: 360,
    step: 0.1,
  },
  mask1StartAnim: {
    label: "Start Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },
  mask1StopF: {
    label: "Stop From",
    type: "number",
    numeric: "float",
    min: -360,
    max: 360,
    step: 0.1,
  },
  mask1StopT: {
    label: "Stop To",
    type: "number",
    numeric: "float",
    min: -360,
    max: 360,
    step: 0.1,
  },
  mask1StopAnim: {
    label: "Stop Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },

  mask2Ena: { label: "Enable", type: "checkbox" },
  mask2Op: {
    label: "Operation",
    type: "select",
    numeric: "int",
    options: [
      { value: 0, label: "AND" },
      { value: 1, label: "OR" },
    ],
  },
  mask2Type: {
    label: "Type",
    type: "select",
    numeric: "int",
    options: [
      { value: 0, label: "Rotation" },
      { value: 1, label: "Radial" },
      { value: 2, label: "Horizontal" },
      { value: 3, label: "Vertical" },
    ],
  },
  mask2Grad: {
    label: "Gradation",
    type: "number",
    numeric: "float",
    min: 0,
    max: 100,
    step: 0.1,
  },
  mask2GradDir: {
    label: "Grad Dir",
    type: "number",
    numeric: "int",
    min: 0,
    max: 8,
    step: 1,
  },
  mask2StartF: {
    label: "Start From",
    type: "number",
    numeric: "float",
    min: -360,
    max: 360,
    step: 0.1,
  },
  mask2StartT: {
    label: "Start To",
    type: "number",
    numeric: "float",
    min: -360,
    max: 360,
    step: 0.1,
  },
  mask2StartAnim: {
    label: "Start Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },
  mask2StopF: {
    label: "Stop From",
    type: "number",
    numeric: "float",
    min: -360,
    max: 360,
    step: 0.1,
  },
  mask2StopT: {
    label: "Stop To",
    type: "number",
    numeric: "float",
    min: -360,
    max: 360,
    step: 0.1,
  },
  mask2StopAnim: {
    label: "Stop Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },

  fMaskEna: {
    label: "Mode",
    type: "select",
    numeric: "int",
    options: [
      { value: 0, label: "Off" },
      { value: 1, label: "Range" },
      { value: 2, label: "Bitmask" },
    ],
  },
  fMaskStart: {
    label: "Range Start",
    type: "number",
    numeric: "float",
    min: 0,
    max: 100,
    step: 0.1,
  },
  fMaskStop: {
    label: "Range Stop",
    type: "number",
    numeric: "float",
    min: 0,
    max: 100,
    step: 0.1,
  },
  fMaskBits: { label: "Bitmask", type: "text" },

  sLightDirF: {
    label: "LightDir From",
    type: "number",
    numeric: "float",
    min: -360,
    max: 360,
    step: 0.1,
  },
  sLightDirT: {
    label: "LightDir To",
    type: "number",
    numeric: "float",
    min: -360,
    max: 360,
    step: 0.1,
  },
  sLightDirAnim: {
    label: "LightDir Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },
  sDensityF: {
    label: "Density From",
    type: "number",
    numeric: "float",
    min: 0,
    max: 100,
    step: 0.1,
  },
  sDensityT: {
    label: "Density To",
    type: "number",
    numeric: "float",
    min: 0,
    max: 100,
    step: 0.1,
  },
  sDensityAnim: {
    label: "Density Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },

  dLightDirEna: { label: "Enable", type: "checkbox" },
  dLightDirF: {
    label: "LightDir From",
    type: "number",
    numeric: "float",
    min: -360,
    max: 360,
    step: 0.1,
  },
  dLightDirT: {
    label: "LightDir To",
    type: "number",
    numeric: "float",
    min: -360,
    max: 360,
    step: 0.1,
  },
  dLightDirAnim: {
    label: "LightDir Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },
  dOffsetF: {
    label: "Offset From",
    type: "number",
    numeric: "float",
    min: -500,
    max: 500,
    step: 0.1,
  },
  dOffsetT: {
    label: "Offset To",
    type: "number",
    numeric: "float",
    min: -500,
    max: 500,
    step: 0.1,
  },
  dOffsetAnim: {
    label: "Offset Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },
  dDensityF: {
    label: "Density From",
    type: "number",
    numeric: "float",
    min: 0,
    max: 100,
    step: 0.1,
  },
  dDensityT: {
    label: "Density To",
    type: "number",
    numeric: "float",
    min: 0,
    max: 100,
    step: 0.1,
  },
  dDensityAnim: {
    label: "Density Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },
  dDiffuseF: {
    label: "Diffuse From",
    type: "number",
    numeric: "float",
    min: 0,
    max: 100,
    step: 0.1,
  },
  dDiffuseT: {
    label: "Diffuse To",
    type: "number",
    numeric: "float",
    min: 0,
    max: 100,
    step: 0.1,
  },
  dDiffuseAnim: {
    label: "Diffuse Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },
  dsType: {
    label: "Shadow Type",
    type: "select",
    numeric: "int",
    options: [
      { value: 0, label: "Soft" },
      { value: 1, label: "Hard" },
    ],
  },
  dsGrad: {
    label: "Gradient",
    type: "number",
    numeric: "float",
    min: 0,
    max: 100,
    step: 0.1,
  },

  iLightDirEna: { label: "Enable", type: "checkbox" },
  iLightDirF: {
    label: "LightDir From",
    type: "number",
    numeric: "float",
    min: -360,
    max: 360,
    step: 0.1,
  },
  iLightDirT: {
    label: "LightDir To",
    type: "number",
    numeric: "float",
    min: -360,
    max: 360,
    step: 0.1,
  },
  iLightDirAnim: {
    label: "LightDir Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },
  iOffsetF: {
    label: "Offset From",
    type: "number",
    numeric: "float",
    min: -500,
    max: 500,
    step: 0.1,
  },
  iOffsetT: {
    label: "Offset To",
    type: "number",
    numeric: "float",
    min: -500,
    max: 500,
    step: 0.1,
  },
  iOffsetAnim: {
    label: "Offset Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },
  iDensityF: {
    label: "Density From",
    type: "number",
    numeric: "float",
    min: 0,
    max: 100,
    step: 0.1,
  },
  iDensityT: {
    label: "Density To",
    type: "number",
    numeric: "float",
    min: 0,
    max: 100,
    step: 0.1,
  },
  iDensityAnim: {
    label: "Density Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },
  iDiffuseF: {
    label: "Diffuse From",
    type: "number",
    numeric: "float",
    min: 0,
    max: 100,
    step: 0.1,
  },
  iDiffuseT: {
    label: "Diffuse To",
    type: "number",
    numeric: "float",
    min: 0,
    max: 100,
    step: 0.1,
  },
  iDiffuseAnim: {
    label: "Diffuse Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },

  eLightDirEna: { label: "Enable", type: "checkbox" },
  eLightDirF: {
    label: "LightDir From",
    type: "number",
    numeric: "float",
    min: -360,
    max: 360,
    step: 0.1,
  },
  eLightDirT: {
    label: "LightDir To",
    type: "number",
    numeric: "float",
    min: -360,
    max: 360,
    step: 0.1,
  },
  eLightDirAnim: {
    label: "LightDir Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },
  eOffsetF: {
    label: "Offset From",
    type: "number",
    numeric: "float",
    min: -500,
    max: 500,
    step: 0.1,
  },
  eOffsetT: {
    label: "Offset To",
    type: "number",
    numeric: "float",
    min: -500,
    max: 500,
    step: 0.1,
  },
  eOffsetAnim: {
    label: "Offset Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },
  eDensityF: {
    label: "Density From",
    type: "number",
    numeric: "float",
    min: 0,
    max: 100,
    step: 0.1,
  },
  eDensityT: {
    label: "Density To",
    type: "number",
    numeric: "float",
    min: 0,
    max: 100,
    step: 0.1,
  },
  eDensityAnim: {
    label: "Density Curve",
    type: "select",
    numeric: "int",
    options: CURVE_OPTIONS,
  },
};

const EFFECT_SECTIONS = [
  {
    title: "Transform",
    open: true,
    fields: [
      "antiAlias",
      "unfold",
      "animStep",
      "zoomXYSepa",
      "zoomXF",
      "zoomXT",
      "zoomXAnim",
      "zoomYF",
      "zoomYT",
      "zoomYAnim",
      "offXF",
      "offXT",
      "offXAnim",
      "offYF",
      "offYT",
      "offYAnim",
      "keepDir",
      "centerX",
      "centerY",
      "angleF",
      "angleT",
      "angleAnim",
    ],
  },
  {
    title: "Color",
    open: true,
    fields: [
      "alphaF",
      "alphaT",
      "alphaAnim",
      "brightF",
      "brightT",
      "brightAnim",
      "contrastF",
      "contrastT",
      "contrastAnim",
      "saturationF",
      "saturationT",
      "saturationAnim",
      "hueF",
      "hueT",
      "hueAnim",
    ],
  },
  {
    title: "Mask 1",
    fields: [
      "mask1Ena",
      "mask1Type",
      "mask1Grad",
      "mask1GradDir",
      "mask1StartF",
      "mask1StartT",
      "mask1StartAnim",
      "mask1StopF",
      "mask1StopT",
      "mask1StopAnim",
    ],
  },
  {
    title: "Mask 2",
    fields: [
      "mask2Ena",
      "mask2Op",
      "mask2Type",
      "mask2Grad",
      "mask2GradDir",
      "mask2StartF",
      "mask2StartT",
      "mask2StartAnim",
      "mask2StopF",
      "mask2StopT",
      "mask2StopAnim",
    ],
  },
  {
    title: "Frame Mask",
    fields: ["fMaskEna", "fMaskStart", "fMaskStop", "fMaskBits"],
  },
  {
    title: "Specular Highlight",
    fields: [
      "sLightDirF",
      "sLightDirT",
      "sLightDirAnim",
      "sDensityF",
      "sDensityT",
      "sDensityAnim",
    ],
  },
  {
    title: "Drop Shadow",
    fields: [
      "dLightDirEna",
      "dLightDirF",
      "dLightDirT",
      "dLightDirAnim",
      "dOffsetF",
      "dOffsetT",
      "dOffsetAnim",
      "dDensityF",
      "dDensityT",
      "dDensityAnim",
      "dDiffuseF",
      "dDiffuseT",
      "dDiffuseAnim",
      "dsType",
      "dsGrad",
    ],
  },
  {
    title: "Inner Shadow",
    fields: [
      "iLightDirEna",
      "iLightDirF",
      "iLightDirT",
      "iLightDirAnim",
      "iOffsetF",
      "iOffsetT",
      "iOffsetAnim",
      "iDensityF",
      "iDensityT",
      "iDensityAnim",
      "iDiffuseF",
      "iDiffuseT",
      "iDiffuseAnim",
    ],
  },
  {
    title: "Emboss",
    fields: [
      "eLightDirEna",
      "eLightDirF",
      "eLightDirT",
      "eLightDirAnim",
      "eOffsetF",
      "eOffsetT",
      "eOffsetAnim",
      "eDensityF",
      "eDensityT",
      "eDensityAnim",
    ],
  },
];

function clampFloat(v, min, max) {
  return Math.max(min, Math.min(max, v));
}

function clampInt(v, min, max) {
  if (!Number.isFinite(v)) return min;
  return Math.max(min, Math.min(max, Math.round(v)));
}

function isCurveSelectorField(key) {
  return typeof key === "string" && key.endsWith("Anim");
}

function isDetachedPreviewOpen() {
  return Boolean(detachedPreviewWindow && !detachedPreviewWindow.closed);
}

function detachedPreviewRenderFrames() {
  return Math.max(
    1,
    parseInt(document.getElementById("prefFrames").value, 10) || 1,
  );
}

function detachedPreviewDelayMs() {
  const duration = Math.max(
    1,
    parseInt(document.getElementById("prefDuration").value, 10) || 100,
  );
  const frames = detachedPreviewRenderFrames();
  return Math.max(16, Math.round(duration / Math.max(1, frames)));
}

function cleanupDetachedPreviewWindow() {
  if (detachedPreviewTimer) {
    clearInterval(detachedPreviewTimer);
    detachedPreviewTimer = null;
  }
  detachedPreviewPlaying = false;
  detachedPreviewWindow = null;
  const btn = document.getElementById("btnPreviewWin");
  if (btn) btn.textContent = "Preview";
}

function closeDetachedPreviewWindow() {
  if (isDetachedPreviewOpen()) {
    detachedPreviewWindow.close();
  }
  cleanupDetachedPreviewWindow();
}

function syncCanvasElementSize(canvas, width, height, setStyleSize = true) {
  if (!canvas) return false;
  const w = Math.max(1, Math.round(Number(width) || 0));
  const h = Math.max(1, Math.round(Number(height) || 0));
  let resized = false;
  if (canvas.width !== w) {
    canvas.width = w;
    resized = true;
  }
  if (canvas.height !== h) {
    canvas.height = h;
    resized = true;
  }
  if (setStyleSize) {
    const cssW = `${w}px`;
    const cssH = `${h}px`;
    if (canvas.style.width !== cssW) canvas.style.width = cssW;
    if (canvas.style.height !== cssH) canvas.style.height = cssH;
  }
  return resized;
}

function syncCanvasBackingToDisplaySize(canvas) {
  if (!canvas) return false;
  const rect = canvas.getBoundingClientRect();
  return syncCanvasElementSize(canvas, rect.width, rect.height, false);
}

function renderDetachedPreviewFrame(frame) {
  if (!isDetachedPreviewOpen() || !window.knobman_renderFrameRaw) return;
  const raw = window.knobman_renderFrameRaw(frame);
  if (!raw || !raw.data) return;
  const doc = detachedPreviewWindow.document;
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
    detachedPreviewFrame = 0;
    return;
  }
  if (document.getElementById("prefBiDir").checked) {
    detachedPreviewFrame += detachedPreviewDir;
    if (detachedPreviewFrame >= total - 1) {
      detachedPreviewFrame = total - 1;
      detachedPreviewDir = -1;
    } else if (detachedPreviewFrame <= 0) {
      detachedPreviewFrame = 0;
      detachedPreviewDir = 1;
    }
    return;
  }
  detachedPreviewFrame = (detachedPreviewFrame + 1) % total;
}

function detachedPreviewTick() {
  if (!isDetachedPreviewOpen()) {
    cleanupDetachedPreviewWindow();
    return;
  }
  renderDetachedPreviewFrame(detachedPreviewFrame);
  advanceDetachedPreviewFrame();
}

function setDetachedPreviewPlaying(playing) {
  if (!isDetachedPreviewOpen()) return;
  detachedPreviewPlaying = Boolean(playing);
  if (detachedPreviewTimer) {
    clearInterval(detachedPreviewTimer);
    detachedPreviewTimer = null;
  }
  if (detachedPreviewPlaying) {
    detachedPreviewTimer = setInterval(
      detachedPreviewTick,
      detachedPreviewDelayMs(),
    );
  }
  const toggle = detachedPreviewWindow.document.getElementById(
    "detachedPreviewToggle",
  );
  if (toggle) {
    toggle.textContent = detachedPreviewPlaying ? "Pause" : "Play";
  }
}

function refreshDetachedPreviewNow() {
  if (!isDetachedPreviewOpen()) return;
  if (detachedPreviewPlaying) {
    setDetachedPreviewPlaying(true);
    return;
  }
  detachedPreviewFrame = clampInt(
    currentFrame,
    0,
    detachedPreviewRenderFrames() - 1,
  );
  renderDetachedPreviewFrame(detachedPreviewFrame);
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

  detachedPreviewWindow = win;
  detachedPreviewFrame = clampInt(
    currentFrame,
    0,
    detachedPreviewRenderFrames() - 1,
  );
  detachedPreviewDir = 1;

  const toggle = win.document.getElementById("detachedPreviewToggle");
  const syncBtn = win.document.getElementById("detachedPreviewSync");
  const closeBtn = win.document.getElementById("detachedPreviewClose");
  if (toggle) {
    toggle.addEventListener("click", () => {
      setDetachedPreviewPlaying(!detachedPreviewPlaying);
    });
  }
  if (syncBtn) {
    syncBtn.addEventListener("click", () => {
      detachedPreviewFrame = clampInt(
        currentFrame,
        0,
        detachedPreviewRenderFrames() - 1,
      );
      detachedPreviewDir = 1;
      renderDetachedPreviewFrame(detachedPreviewFrame);
    });
  }
  if (closeBtn) {
    closeBtn.addEventListener("click", closeDetachedPreviewWindow);
  }

  win.addEventListener("beforeunload", cleanupDetachedPreviewWindow);

  setDetachedPreviewPlaying(true);
  renderDetachedPreviewFrame(detachedPreviewFrame);
  const btn = document.getElementById("btnPreviewWin");
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
  layerPreviewCache.forEach((entry) => {
    if (entry && entry.url) URL.revokeObjectURL(entry.url);
  });
  layerPreviewCache.clear();
}

function invalidateLayerPreviews() {
  layerPreviewRevision += 1;
  releaseLayerPreviewCache();
}

function layerPreviewCacheKey(layerIndex, frame) {
  return `${layerPreviewRevision}:${layerIndex}:${frame}`;
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

function canvasToBlobSync(canvas) {
  const dataUrl = canvas.toDataURL("image/png");
  const comma = dataUrl.indexOf(",");
  if (comma < 0) return null;
  const b64 = dataUrl.slice(comma + 1);
  const bin = atob(b64);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return new Blob([bytes], { type: "image/png" });
}

function getLayerPreviewCached(layerIndex, frame) {
  const key = layerPreviewCacheKey(layerIndex, frame);
  const cached = layerPreviewCache.get(key);
  if (cached) return cached;
  if (!window.knobman_getLayerPreview) return null;
  const raw = window.knobman_getLayerPreview(
    layerIndex,
    frame,
    LAYER_PREVIEW_SIZE,
  );
  const built = buildLayerPreviewFromRaw(raw);
  if (!built) return null;
  layerPreviewCache.set(key, built);
  return built;
}

function renderLayerPreviewsAsync(jobs) {
  const token = ++layerPreviewToken;
  (async () => {
    for (let i = 0; i < jobs.length; i++) {
      if (token !== layerPreviewToken) return;
      if (i > 0) await new Promise((resolve) => setTimeout(resolve, 0));
      const job = jobs[i];
      if (!job || !job.img || !job.img.isConnected) continue;
      const preview = getLayerPreviewCached(job.layerIndex, job.frame);
      if (!preview) continue;
      if (token !== layerPreviewToken || !job.img.isConnected) return;
      job.img.src = preview.url;
      job.img.classList.add("ready");
    }
  })();
}

function curveCanvasMetrics() {
  const w = curveCanvas ? curveCanvas.width : 320;
  const h = curveCanvas ? curveCanvas.height : 200;
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
  syncCanvasBackingToDisplaySize(curveCanvas);
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
    if (unique.length === 0 || unique[unique.length - 1].t !== p.t)
      unique.push(p);
  });

  if (unique.length === 0) {
    unique.push({ t: 0, l: 0 }, { t: 100, l: 100 });
  }
  if (unique[0].t !== 0) unique.unshift({ t: 0, l: unique[0].l });
  if (unique[unique.length - 1].t !== 100)
    unique.push({ t: 100, l: unique[unique.length - 1].l });

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
    ? window.knobman_getCurve(selectedCurve)
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

  const ok = window.knobman_setCurve(selectedCurve, {
    tm,
    lv,
    stepReso: clampInt(Number(stepReso), 0, 64),
  });
  if (ok) markDirty();
  return ok;
}

function frameRatioForCurve() {
  const renderFrames =
    parseInt(document.getElementById("prefFrames").value, 10) || 1;
  if (renderFrames <= 1) return 0;
  return clampFloat(currentFrame / (renderFrames - 1), 0, 1);
}

function syncCurveTabState() {
  const tabs = document.getElementById("curveTabs");
  if (!tabs) return;
  tabs.querySelectorAll("button").forEach((btn) => {
    const idx = parseInt(btn.dataset.curve, 10) || 1;
    btn.classList.toggle("active", idx === selectedCurve);
  });
}


function drawCurveEditor() {
  if (!curveCanvas || !curveCtx) return;
  syncCurveCanvasSize();
  const m = curveCanvasMetrics();
  const state = readSelectedCurve();

  const stepInput = document.getElementById("curveStepReso");
  if (stepInput && document.activeElement !== stepInput) {
    stepInput.value = String(state.stepReso);
  }

  curveCtx.clearRect(0, 0, m.w, m.h);
  curveCtx.fillStyle = "#141414";
  curveCtx.fillRect(0, 0, m.w, m.h);

  curveCtx.strokeStyle = "#2f2f2f";
  curveCtx.lineWidth = 1;
  for (let i = 0; i <= 4; i++) {
    const x = m.pad.left + (i / 4) * m.plotW;
    const y = m.pad.top + (i / 4) * m.plotH;
    curveCtx.beginPath();
    curveCtx.moveTo(x, m.pad.top);
    curveCtx.lineTo(x, m.h - m.pad.bottom);
    curveCtx.stroke();
    curveCtx.beginPath();
    curveCtx.moveTo(m.pad.left, y);
    curveCtx.lineTo(m.w - m.pad.right, y);
    curveCtx.stroke();
  }

  curveCtx.strokeStyle = "#4b4b4b";
  curveCtx.strokeRect(m.pad.left, m.pad.top, m.plotW, m.plotH);

  const ratio = frameRatioForCurve();
  const evalLv = clampFloat(
    Number(
      window.knobman_evalCurve
        ? window.knobman_evalCurve(selectedCurve, ratio)
        : 0,
    ),
    0,
    100,
  );
  const frameX = curveToCanvasPoint(ratio * 100, 0, m).x;
  const evalY = curveToCanvasPoint(0, evalLv, m).y;

  curveCtx.save();
  curveCtx.setLineDash([4, 4]);
  curveCtx.strokeStyle = "#f3b24a";
  curveCtx.beginPath();
  curveCtx.moveTo(frameX, m.pad.top);
  curveCtx.lineTo(frameX, m.h - m.pad.bottom);
  curveCtx.stroke();

  curveCtx.strokeStyle = "#5db2ff";
  curveCtx.beginPath();
  curveCtx.moveTo(m.pad.left, evalY);
  curveCtx.lineTo(m.w - m.pad.right, evalY);
  curveCtx.stroke();
  curveCtx.restore();

  curveCtx.strokeStyle = "#7ec7ff";
  curveCtx.lineWidth = 2;
  curveCtx.beginPath();
  state.points.forEach((p, i) => {
    const c = curveToCanvasPoint(p.t, p.l, m);
    if (i === 0) curveCtx.moveTo(c.x, c.y);
    else curveCtx.lineTo(c.x, c.y);
  });
  curveCtx.stroke();

  state.points.forEach((p, i) => {
    const c = curveToCanvasPoint(p.t, p.l, m);
    const isEndpoint = i === 0 || i === state.points.length - 1;
    curveCtx.beginPath();
    curveCtx.arc(c.x, c.y, i === curveSelectedPoint ? 5 : 4, 0, Math.PI * 2);
    curveCtx.fillStyle = isEndpoint ? "#e4e4e4" : "#f3b24a";
    curveCtx.fill();
    curveCtx.strokeStyle = "#000";
    curveCtx.lineWidth = 1;
    curveCtx.stroke();
  });

  curveCtx.fillStyle = "#a0a0a0";
  curveCtx.font = "10px system-ui, sans-serif";
  curveCtx.fillText("0", m.pad.left - 6, m.h - 6);
  curveCtx.fillText("100", m.w - m.pad.right - 18, m.h - 6);
  curveCtx.fillText("100", 3, m.pad.top + 3);
  curveCtx.fillStyle = "#7d7d7d";
  curveCtx.fillText(`Frame ${currentFrame}`, m.pad.left + 6, m.h - 6);
  curveCtx.fillText(`Value ${evalLv.toFixed(1)}`, m.pad.left + 78, m.h - 6);
}

function refreshCurveEditor() {
  syncCurveTabState();
  drawCurveEditor();
}

function focusCurve(curveIdx) {
  selectedCurve = clampInt(Number(curveIdx), 1, 8);
  curveSelectedPoint = -1;
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
  if (!curveCanvas) return { x: 0, y: 0 };
  const rect = curveCanvas.getBoundingClientRect();
  const scaleX = rect.width > 0 ? curveCanvas.width / rect.width : 1;
  const scaleY = rect.height > 0 ? curveCanvas.height / rect.height : 1;
  return {
    x: (e.clientX - rect.left) * scaleX,
    y: (e.clientY - rect.top) * scaleY,
  };
}

function onCurvePointerDown(e) {
  if (e.button !== 0 || !curveCanvas) return;
  const m = curveCanvasMetrics();
  const state = readSelectedCurve();
  const pos = curveEventToCanvasXY(e);
  const hit = curveHitPoint(state.points, pos.x, pos.y);

  if (hit >= 0) {
    curveDragPoint = hit;
    curveSelectedPoint = hit;
    curveCanvas.setPointerCapture(e.pointerId);
    refreshCurveEditor();
    return;
  }

  if (state.points.length >= CURVE_POINT_LIMIT) {
    setStatus("Curve has reached the 12-point limit");
    return;
  }

  const p = canvasToCurvePoint(pos.x, pos.y, m);
  if (p.t <= 0 || p.t >= 100) return;
  let insertAt = state.points.findIndex((pt) => pt.t > p.t);
  if (insertAt < 0) insertAt = state.points.length - 1;
  state.points.splice(insertAt, 0, p);
  curveSelectedPoint = insertAt;
  writeSelectedCurve(state.points, state.stepReso);
  refreshCurveEditor();
}

function onCurvePointerMove(e) {
  if (curveDragPoint < 0 || !curveCanvas) return;
  const m = curveCanvasMetrics();
  const state = readSelectedCurve();
  if (curveDragPoint >= state.points.length) {
    curveDragPoint = -1;
    return;
  }

  const pos = curveEventToCanvasXY(e);
  const p = canvasToCurvePoint(pos.x, pos.y, m);
  const i = curveDragPoint;
  if (i === 0) {
    p.t = 0;
  } else if (i === state.points.length - 1) {
    p.t = 100;
  } else {
    p.t = clampInt(p.t, state.points[i - 1].t + 1, state.points[i + 1].t - 1);
  }
  state.points[i] = p;
  writeSelectedCurve(state.points, state.stepReso);
  refreshCurveEditor();
}

function onCurvePointerUp(e) {
  if (!curveCanvas) return;
  if (curveDragPoint >= 0) {
    curveCanvas.releasePointerCapture(e.pointerId);
  }
  curveDragPoint = -1;
}

function onCurveContextMenu(e) {
  e.preventDefault();
  const state = readSelectedCurve();
  const pos = curveEventToCanvasXY(e);
  const hit = curveHitPoint(state.points, pos.x, pos.y);
  if (hit <= 0 || hit >= state.points.length - 1) return;
  state.points.splice(hit, 1);
  curveSelectedPoint = -1;
  writeSelectedCurve(state.points, state.stepReso);
  refreshCurveEditor();
}

function deleteSelectedCurvePoint() {
  const state = readSelectedCurve();
  if (curveSelectedPoint <= 0 || curveSelectedPoint >= state.points.length - 1)
    return;
  state.points.splice(curveSelectedPoint, 1);
  curveSelectedPoint = -1;
  writeSelectedCurve(state.points, state.stepReso);
  refreshCurveEditor();
}

function initCurveEditor() {
  const tabs = document.getElementById("curveTabs");
  curveCanvas = document.getElementById("curveCanvas");
  if (!tabs || !curveCanvas) return;
  curveCtx = curveCanvas.getContext("2d");

  const details = document.getElementById("curveEditorDetails");
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

  const stepInput = document.getElementById("curveStepReso");
  if (stepInput) {
    stepInput.addEventListener("change", () => {
      const state = readSelectedCurve();
      const step = clampInt(parseInt(stepInput.value, 10) || 0, 0, 64);
      writeSelectedCurve(state.points, step);
      refreshCurveEditor();
    });
  }

  const deleteBtn = document.getElementById("curveDeletePoint");
  if (deleteBtn) deleteBtn.addEventListener("click", deleteSelectedCurvePoint);

  curveCanvas.addEventListener("pointerdown", onCurvePointerDown);
  curveCanvas.addEventListener("pointermove", onCurvePointerMove);
  curveCanvas.addEventListener("pointerup", onCurvePointerUp);
  curveCanvas.addEventListener("pointercancel", onCurvePointerUp);
  curveCanvas.addEventListener("contextmenu", onCurveContextMenu);

  syncCurveTabState();
  refreshCurveEditor();
}

function shapeCommandArity(cmd) {
  switch (cmd) {
    case "M":
      return 2;
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
    const clamped = vals.map((v) => clampFloat(Number(v), 0, 100));
    out.push({ cmd, values: clamped });
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
  return Number(window.knobman_getParam(selectedLayer, "primType") || 0) === 15;
}

function shapeEditorEnabled() {
  return isShapeLayerSelected() && shapeCanvas && shapeCtx;
}

function setShapeOverlayVisible(visible) {
  if (!shapeCanvas) return;
  shapeCanvas.style.display = visible ? "block" : "none";
  shapeCanvas.style.pointerEvents = visible ? "auto" : "none";
}

function shapeMetrics() {
  const w = shapeCanvas ? shapeCanvas.width : canvasW;
  const h = shapeCanvas ? shapeCanvas.height : canvasH;
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
  const raw = String(window.knobman_getParam(selectedLayer, "shape") || "");
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
  const ok = window.knobman_setParam(selectedLayer, "shape", path);
  if (!ok) return false;
  shapeCommands = normalized;
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
  if (!shapeCanvas) return { x: 0, y: 0 };
  const rect = shapeCanvas.getBoundingClientRect();
  const scaleX = rect.width > 0 ? shapeCanvas.width / rect.width : 1;
  const scaleY = rect.height > 0 ? shapeCanvas.height / rect.height : 1;
  return {
    x: (e.clientX - rect.left) * scaleX,
    y: (e.clientY - rect.top) * scaleY,
  };
}

function applyShapeModeButtons() {
  const toolbar = document.getElementById("shapeToolbar");
  if (!toolbar) return;
  toolbar.querySelectorAll("button[data-shape-mode]").forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.shapeMode === shapeMode);
  });
}

function addShapeCommandAt(mode, x, y) {
  const cmds = normalizeShapeCommands(shapeCommands);
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
  const cmds = normalizeShapeCommands(shapeCommands);
  if (cmds.length === 0) return;
  if (cmds[cmds.length - 1].cmd === "Z") return;
  cmds.push({ cmd: "Z", values: [] });
  writeShapeCommands(cmds);
  drawShapeOverlay();
}

function deleteShapeSelection() {
  if (!shapeEditorEnabled() || !shapeSelectedHandle) return;
  const cmds = normalizeShapeCommands(shapeCommands);
  const idx = shapeSelectedHandle.cmdIndex;
  if (idx <= 0 || idx >= cmds.length) {
    setStatus("Initial move point cannot be deleted");
    return;
  }
  cmds.splice(idx, 1);
  shapeSelectedHandle = null;
  shapeDragHandle = null;
  writeShapeCommands(cmds);
  drawShapeOverlay();
}

function drawShapeOverlay() {
  if (!shapeCtx || !shapeCanvas) return;
  shapeCtx.clearRect(0, 0, shapeCanvas.width, shapeCanvas.height);
  if (!shapeEditorEnabled()) return;

  const m = shapeMetrics();
  const fillEnabled = Boolean(window.knobman_getParam(selectedLayer, "fill"));
  shapeCtx.save();
  shapeCtx.strokeStyle = "rgba(220,220,220,0.25)";
  shapeCtx.lineWidth = 1;
  for (let i = 0; i <= 4; i++) {
    const x = m.pad + (i / 4) * m.plotW;
    const y = m.pad + (i / 4) * m.plotH;
    shapeCtx.beginPath();
    shapeCtx.moveTo(x, m.pad);
    shapeCtx.lineTo(x, m.h - m.pad);
    shapeCtx.stroke();
    shapeCtx.beginPath();
    shapeCtx.moveTo(m.pad, y);
    shapeCtx.lineTo(m.w - m.pad, y);
    shapeCtx.stroke();
  }

  shapeCtx.beginPath();
  let cur = null;
  let start = null;
  shapeCommands.forEach((c) => {
    if (c.cmd === "M") {
      const p = shapeToCanvas(c.values[0], c.values[1], m);
      shapeCtx.moveTo(p.x, p.y);
      cur = { x: c.values[0], y: c.values[1] };
      start = { x: cur.x, y: cur.y };
    } else if (c.cmd === "L" && cur) {
      const p = shapeToCanvas(c.values[0], c.values[1], m);
      shapeCtx.lineTo(p.x, p.y);
      cur = { x: c.values[0], y: c.values[1] };
    } else if (c.cmd === "Q" && cur) {
      const c1 = shapeToCanvas(c.values[0], c.values[1], m);
      const p = shapeToCanvas(c.values[2], c.values[3], m);
      shapeCtx.quadraticCurveTo(c1.x, c1.y, p.x, p.y);
      cur = { x: c.values[2], y: c.values[3] };
    } else if (c.cmd === "C" && cur) {
      const c1 = shapeToCanvas(c.values[0], c.values[1], m);
      const c2 = shapeToCanvas(c.values[2], c.values[3], m);
      const p = shapeToCanvas(c.values[4], c.values[5], m);
      shapeCtx.bezierCurveTo(c1.x, c1.y, c2.x, c2.y, p.x, p.y);
      cur = { x: c.values[4], y: c.values[5] };
    } else if (c.cmd === "Z" && start) {
      const p = shapeToCanvas(start.x, start.y, m);
      shapeCtx.lineTo(p.x, p.y);
      cur = { x: start.x, y: start.y };
    }
  });
  if (fillEnabled) {
    shapeCtx.fillStyle = "rgba(14,99,156,0.15)";
    shapeCtx.fill();
  }
  shapeCtx.strokeStyle = "#f6d06f";
  shapeCtx.lineWidth = 2;
  shapeCtx.stroke();

  let prev = null;
  let subStart = null;
  shapeCommands.forEach((c) => {
    if (c.cmd === "M") {
      prev = { x: c.values[0], y: c.values[1] };
      subStart = { x: prev.x, y: prev.y };
      return;
    }
    if (c.cmd === "Q" && prev) {
      const a = shapeToCanvas(prev.x, prev.y, m);
      const ctrl = shapeToCanvas(c.values[0], c.values[1], m);
      const end = shapeToCanvas(c.values[2], c.values[3], m);
      shapeCtx.setLineDash([4, 4]);
      shapeCtx.strokeStyle = "rgba(98,180,255,0.85)";
      shapeCtx.beginPath();
      shapeCtx.moveTo(a.x, a.y);
      shapeCtx.lineTo(ctrl.x, ctrl.y);
      shapeCtx.lineTo(end.x, end.y);
      shapeCtx.stroke();
      shapeCtx.setLineDash([]);
      prev = { x: c.values[2], y: c.values[3] };
      return;
    }
    if (c.cmd === "C" && prev) {
      const a = shapeToCanvas(prev.x, prev.y, m);
      const c1 = shapeToCanvas(c.values[0], c.values[1], m);
      const c2 = shapeToCanvas(c.values[2], c.values[3], m);
      const end = shapeToCanvas(c.values[4], c.values[5], m);
      shapeCtx.setLineDash([4, 4]);
      shapeCtx.strokeStyle = "rgba(98,180,255,0.85)";
      shapeCtx.beginPath();
      shapeCtx.moveTo(a.x, a.y);
      shapeCtx.lineTo(c1.x, c1.y);
      shapeCtx.moveTo(end.x, end.y);
      shapeCtx.lineTo(c2.x, c2.y);
      shapeCtx.stroke();
      shapeCtx.setLineDash([]);
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

  const handles = shapeHandles(shapeCommands);
  handles.forEach((h) => {
    const p = shapeToCanvas(h.x, h.y, m);
    const selected = Boolean(
      shapeSelectedHandle &&
      shapeSelectedHandle.cmdIndex === h.cmdIndex &&
      shapeSelectedHandle.valueIndex === h.valueIndex,
    );
    shapeCtx.beginPath();
    shapeCtx.arc(p.x, p.y, selected ? 5 : 4, 0, Math.PI * 2);
    shapeCtx.fillStyle = h.role === "anchor" ? "#ffe28a" : "#78c9ff";
    shapeCtx.fill();
    shapeCtx.strokeStyle = "#101010";
    shapeCtx.lineWidth = 1;
    shapeCtx.stroke();
  });

  shapeCtx.fillStyle = "#fff";
  shapeCtx.font = "11px system-ui, sans-serif";
  shapeCtx.fillText("Shape Edit Overlay", m.pad + 6, m.pad + 14);
  shapeCtx.restore();
}

function refreshShapeEditor() {
  const panel = document.getElementById("shapeEditorPanel");
  if (!panel) return;
  const active = isShapeLayerSelected();
  panel.classList.toggle("disabled", !active);
  setShapeOverlayVisible(active);

  const hint = document.getElementById("shapeHelp");
  if (!active) {
    shapeCommands = [];
    shapeDragHandle = null;
    shapeSelectedHandle = null;
    if (hint)
      hint.textContent =
        "Select a layer with Primitive = Shape to edit its path.";
    if (shapeCtx && shapeCanvas)
      shapeCtx.clearRect(0, 0, shapeCanvas.width, shapeCanvas.height);
    return;
  }
  if (hint)
    hint.textContent =
      "Click to add points. Drag handles to edit. Right click or Delete removes selected segment.";
  shapeCommands = parseShapeFromLayer();
  drawShapeOverlay();
}

function onShapePointerDown(e) {
  if (!shapeEditorEnabled() || e.button !== 0) return;
  const m = shapeMetrics();
  const pos = shapeEventToCanvasXY(e);
  const handles = shapeHandles(shapeCommands);
  const hit = shapeHitHandle(handles, pos.x, pos.y, m);
  if (hit) {
    shapeSelectedHandle = {
      cmdIndex: hit.cmdIndex,
      valueIndex: hit.valueIndex,
    };
    shapeDragHandle = { cmdIndex: hit.cmdIndex, valueIndex: hit.valueIndex };
    shapeCanvas.setPointerCapture(e.pointerId);
    drawShapeOverlay();
    return;
  }
  const p = canvasToShape(pos.x, pos.y, m);
  const next = addShapeCommandAt(shapeMode, p.x, p.y);
  writeShapeCommands(next);
  drawShapeOverlay();
}

function onShapePointerMove(e) {
  if (!shapeEditorEnabled() || !shapeDragHandle) return;
  const m = shapeMetrics();
  const pos = shapeEventToCanvasXY(e);
  const p = canvasToShape(pos.x, pos.y, m);
  const cmds = normalizeShapeCommands(shapeCommands);
  const cmd = cmds[shapeDragHandle.cmdIndex];
  if (!cmd || cmd.cmd === "Z") return;
  const i = shapeDragHandle.valueIndex;
  if (i < 0 || i + 1 >= cmd.values.length) return;
  cmd.values[i] = p.x;
  cmd.values[i + 1] = p.y;
  writeShapeCommands(cmds);
  drawShapeOverlay();
}

function onShapePointerUp(e) {
  if (!shapeCanvas) return;
  if (shapeDragHandle) {
    shapeCanvas.releasePointerCapture(e.pointerId);
  }
  shapeDragHandle = null;
}

function onShapeContextMenu(e) {
  if (!shapeEditorEnabled()) return;
  e.preventDefault();
  const m = shapeMetrics();
  const pos = shapeEventToCanvasXY(e);
  const hit = shapeHitHandle(shapeHandles(shapeCommands), pos.x, pos.y, m);
  if (!hit) return;
  shapeSelectedHandle = { cmdIndex: hit.cmdIndex, valueIndex: hit.valueIndex };
  deleteShapeSelection();
}

function initShapeEditor() {
  shapeCanvas = document.getElementById("shapeOverlay");
  if (!shapeCanvas) return;
  shapeCtx = shapeCanvas.getContext("2d");
  setShapeOverlayVisible(false);

  shapeCanvas.addEventListener("pointerdown", onShapePointerDown);
  shapeCanvas.addEventListener("pointermove", onShapePointerMove);
  shapeCanvas.addEventListener("pointerup", onShapePointerUp);
  shapeCanvas.addEventListener("pointercancel", onShapePointerUp);
  shapeCanvas.addEventListener("contextmenu", onShapeContextMenu);

  const toolbar = document.getElementById("shapeToolbar");
  if (toolbar) {
    toolbar.querySelectorAll("button[data-shape-mode]").forEach((btn) => {
      btn.addEventListener("click", () => {
        shapeMode = btn.dataset.shapeMode || "L";
        applyShapeModeButtons();
      });
    });
  }
  applyShapeModeButtons();

  const closeBtn = document.getElementById("shapeClosePath");
  if (closeBtn) closeBtn.addEventListener("click", closeShapePath);
  const delBtn = document.getElementById("shapeDeleteSeg");
  if (delBtn) delBtn.addEventListener("click", deleteShapeSelection);
}

async function fetchBuiltinTextureBytes(filename) {
  const paths = [
    "../assets/textures/" + filename,
    "/assets/textures/" + filename,
    "assets/textures/" + filename,
  ];
  for (const path of paths) {
    try {
      const res = await fetch(path);
      if (!res.ok) continue;
      const buf = await res.arrayBuffer();
      if (buf.byteLength > 0) return new Uint8Array(buf);
    } catch (_err) {
      // Try next candidate path.
    }
  }
  return null;
}

async function ensureBuiltinTextures() {
  if (builtinTextureLoadPromise) return builtinTextureLoadPromise;
  builtinTextureLoadPromise = (async () => {
    const existing = window.knobman_getTextureList
      ? window.knobman_getTextureList() || []
      : [];
    const byName = new Set(
      existing.map((t) => String(t.name || "").toLowerCase()),
    );
    for (const filename of BUILTIN_TEXTURE_FILES) {
      if (byName.has(filename.toLowerCase())) continue;
      const data = await fetchBuiltinTextureBytes(filename);
      if (!data) continue;
      const idx = window.knobman_addTexture(filename, data);
      if (idx > 0) byName.add(filename.toLowerCase());
    }
  })().finally(() => {
    builtinTextureLoadPromise = null;
  });
  return builtinTextureLoadPromise;
}

function sampleLabelFromFileName(fileName) {
  return stripFileExtension(fileName).replace(/[_-]+/g, " ").trim();
}

// ── Welcome screen ────────────────────────────────────────────────────────────

const WELCOME_SUPPRESS_KEY = "knobman_welcome_suppress";

function shouldShowWelcome() {
  return localStorage.getItem(WELCOME_SUPPRESS_KEY) !== "1";
}

function closeWelcomeOverlay() {
  const overlay = document.getElementById("welcomeOverlay");
  if (!overlay) return;
  const check = document.getElementById("welcomeSuppressCheck");
  if (check && check.checked) {
    localStorage.setItem(WELCOME_SUPPRESS_KEY, "1");
  }
  overlay.hidden = true;
}

function openWelcomeOverlay() {
  const overlay = document.getElementById("welcomeOverlay");
  if (!overlay) return;
  const check = document.getElementById("welcomeSuppressCheck");
  if (check) check.checked = false;
  renderWelcomeSampleList();
  const search = document.getElementById("welcomeSampleSearch");
  if (search) { search.value = ""; search.focus(); }
  overlay.hidden = false;
}

function renderWelcomeSampleList() {
  const list = document.getElementById("welcomeSampleList");
  const input = document.getElementById("welcomeSampleSearch");
  if (!list) return;
  const query = String(input && input.value ? input.value : "").trim().toLowerCase();

  list.innerHTML = "";
  const matches = SAMPLE_PROJECT_FILES.filter((file) => {
    if (!query) return true;
    const label = sampleLabelFromFileName(file).toLowerCase();
    return file.toLowerCase().includes(query) || label.includes(query);
  });

  if (matches.length === 0) {
    const empty = document.createElement("p");
    empty.className = "placeholder";
    empty.textContent = "No matching sample projects.";
    list.appendChild(empty);
    return;
  }

  matches.forEach((file) => {
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "sample-item";
    const label = document.createElement("strong");
    label.textContent = sampleLabelFromFileName(file);
    btn.appendChild(label);
    const meta = document.createElement("small");
    meta.textContent = file;
    btn.appendChild(meta);
    btn.addEventListener("click", () => {
      closeWelcomeOverlay();
      void loadSampleProject(file);
    });
    list.appendChild(btn);
  });
}

function wireWelcomeOverlay() {
  const overlay = document.getElementById("welcomeOverlay");
  const btnCancel = document.getElementById("btnWelcomeCancel");
  const btnOpen = document.getElementById("btnWelcomeOpenFile");
  const search = document.getElementById("welcomeSampleSearch");

  if (btnCancel) btnCancel.addEventListener("click", closeWelcomeOverlay);
  if (btnOpen) {
    btnOpen.addEventListener("click", () => {
      closeWelcomeOverlay();
      document.getElementById("fileInput").click();
    });
  }
  if (search) search.addEventListener("input", renderWelcomeSampleList);
  if (overlay) {
    overlay.addEventListener("click", (e) => {
      if (e.target === overlay) closeWelcomeOverlay();
    });
  }
}

function isSamplesOverlayOpen() {
  const overlay = document.getElementById("samplesOverlay");
  return Boolean(overlay && !overlay.hidden);
}

function closeSamplesOverlay() {
  const overlay = document.getElementById("samplesOverlay");
  if (!overlay) return;
  overlay.hidden = true;
}

function renderSampleList() {
  const list = document.getElementById("sampleList");
  const input = document.getElementById("sampleSearch");
  if (!list) return;
  const query = String(input && input.value ? input.value : "")
    .trim()
    .toLowerCase();

  list.innerHTML = "";
  const matches = SAMPLE_PROJECT_FILES.filter((file) => {
    if (!query) return true;
    const label = sampleLabelFromFileName(file).toLowerCase();
    return file.toLowerCase().includes(query) || label.includes(query);
  });

  if (matches.length === 0) {
    const empty = document.createElement("p");
    empty.className = "placeholder";
    empty.textContent = "No matching sample projects.";
    list.appendChild(empty);
    return;
  }

  matches.forEach((file) => {
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "sample-item";

    const label = document.createElement("strong");
    label.textContent = sampleLabelFromFileName(file);
    btn.appendChild(label);

    const meta = document.createElement("small");
    meta.textContent = file;
    btn.appendChild(meta);

    btn.addEventListener("click", () => {
      void loadSampleProject(file);
    });
    list.appendChild(btn);
  });
}

function openSamplesOverlay() {
  const overlay = document.getElementById("samplesOverlay");
  const input = document.getElementById("sampleSearch");
  if (!overlay) return;
  overlay.hidden = false;
  renderSampleList();
  if (input) input.focus();
}

async function fetchSampleProjectBytes(fileName) {
  const paths = [
    "../assets/samples/" + fileName,
    "/assets/samples/" + fileName,
    "assets/samples/" + fileName,
  ];
  for (const path of paths) {
    try {
      const res = await fetch(path);
      if (!res.ok) continue;
      const buf = await res.arrayBuffer();
      if (buf.byteLength > 0) return new Uint8Array(buf);
    } catch (_err) {
      // Try next candidate path.
    }
  }
  return null;
}

// ── Initialise after WASM load ────────────────────────────────────────────────

function onWasmReady() {
  document.getElementById("loading").style.display = "none";
  document.getElementById("app").style.display = "flex";

  wireControls();
  wireWelcomeOverlay();
  renderSampleList();

  window.knobman_init(64, 64, zoomFactor);
  initCurveEditor();
  initShapeEditor();

  const restored = restoreSession();
  refreshFromDoc();
  ensureBuiltinTextures().then(() => {
    refreshParamPanel();
    markDirty();
  });
  scheduleRender();

  if (!restored && shouldShowWelcome()) {
    openWelcomeOverlay();
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
  refreshShapeEditor();

  const renderFrames =
    parseInt(document.getElementById("prefFrames").value, 10) || 1;
  const visibleFrames = Math.max(1, renderFrames);
  const maxPreviewFrame = Math.max(0, visibleFrames - 1);
  if (currentFrame > maxPreviewFrame) currentFrame = maxPreviewFrame;
  if (currentFrame < 0) currentFrame = 0;
  document.getElementById("frameSlider").max = Math.max(0, visibleFrames - 1);
  document.getElementById("frameSlider").value = currentFrame;
  document.getElementById("frameValue").textContent = String(currentFrame);
  window.knobman_setPreviewFrame(currentFrame);
  refreshCurveEditor();
  refreshDetachedPreviewNow();
  markDirty();
}

// ── Canvas ────────────────────────────────────────────────────────────────────

function syncCanvasSize() {
  const dims = window.knobman_getDimensions();
  if (!dims) return;
  const targetW = Math.max(1, Math.round(Number(dims.width) || 0));
  const targetH = Math.max(1, Math.round(Number(dims.height) || 0));
  const docSizeChanged = canvasW !== targetW || canvasH !== targetH;

  const canvas = document.getElementById("knobCanvas");
  syncCanvasElementSize(canvas, targetW, targetH);

  const overlay = document.getElementById("shapeOverlay");
  if (overlay) {
    syncCanvasElementSize(overlay, targetW, targetH);
  }

  canvasW = targetW;
  canvasH = targetH;
  if (
    !docSizeChanged &&
    pixelBuf &&
    imageData &&
    imageData.width === canvasW &&
    imageData.height === canvasH
  )
    return;

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
  const t0 = performance.now();
  window.knobman_render(pixelBuf);
  lastRenderMs = Math.round(performance.now() - t0);
  imageData.data.set(pixelBuf);

  const canvas = document.getElementById("knobCanvas");
  const ctx = canvas.getContext("2d");
  if (!ctx) return;
  ctx.imageSmoothingEnabled = false;
  ctx.putImageData(imageData, 0, 0);

  const layers = window.knobman_getLayerList ? window.knobman_getLayerList() || [] : [];
  const active = layers.find((l) => l.selected);
  updateStatusMetrics(active ? active.name || `Layer ${active.index + 1}` : "");
  saveSession();
}

function markDirty() {
  dirty = true;
  scheduleRender();
  if (isDetachedPreviewOpen() && !detachedPreviewPlaying) {
    refreshDetachedPreviewNow();
  }
  updateUndoRedoButtons();
}

// ── Controls ──────────────────────────────────────────────────────────────────

function wireControls() {
  // Frame scrubber
  const slider = document.getElementById("frameSlider");
  slider.addEventListener("input", () => {
    currentFrame = parseInt(slider.value, 10) || 0;
    const renderFrames =
      parseInt(document.getElementById("prefFrames").value, 10) || 1;
    if (currentFrame >= renderFrames) currentFrame = renderFrames - 1;
    if (currentFrame < 0) currentFrame = 0;
    slider.value = String(currentFrame);
    document.getElementById("frameValue").textContent = String(currentFrame);
    window.knobman_setPreviewFrame(currentFrame);
    refreshCurveEditor();
    refreshDetachedPreviewNow();
    markDirty();
  });

  // Zoom
  const zoomSelect = document.getElementById("zoomSelect");
  zoomSelect.addEventListener("change", () => {
    const z = parseInt(zoomSelect.value, 10) || 8;
    zoomFactor = z;
    if (window.knobman_setZoom) window.knobman_setZoom(z);
    syncCanvasSize();
    markDirty();
  });

  // Prefs bar
  document
    .getElementById("prefWidth")
    .addEventListener("change", onPrefWidthChange);
  document
    .getElementById("prefHeight")
    .addEventListener("change", onPrefHeightChange);
  document
    .getElementById("prefLockAspect")
    .addEventListener("change", onPrefLockAspectChange);
  document
    .getElementById("prefFrames")
    .addEventListener("change", onPrefsChange);
  document
    .getElementById("prefPreviewFrames")
    .addEventListener("change", onPrefsChange);
  document
    .getElementById("prefBgColor")
    .addEventListener("input", onPrefsChange);
  document
    .getElementById("prefBgAlpha")
    .addEventListener("input", onPrefsChange);
  document
    .getElementById("prefOversample")
    .addEventListener("change", onPrefsChange);
  document
    .getElementById("prefExport")
    .addEventListener("change", onPrefsChange);
  document
    .getElementById("prefAlign")
    .addEventListener("change", onPrefsChange);
  document
    .getElementById("prefDuration")
    .addEventListener("change", onPrefsChange);
  document.getElementById("prefLoop").addEventListener("change", onPrefsChange);
  document
    .getElementById("prefBiDir")
    .addEventListener("change", onPrefsChange);

  // Toolbar
  document.getElementById("btnNew").addEventListener("click", onNew);
  document
    .getElementById("btnOpen")
    .addEventListener("click", () =>
      document.getElementById("fileInput").click(),
    );
  document
    .getElementById("btnSamples")
    .addEventListener("click", openSamplesOverlay);
  document.getElementById("fileInput").addEventListener("change", onFileOpen);
  document.getElementById("btnSave").addEventListener("click", onSave);
  document.getElementById("btnExport").addEventListener("click", onExport);
  document
    .getElementById("btnPreviewWin")
    .addEventListener("click", toggleDetachedPreviewWindow);
  document.getElementById("btnUndo").addEventListener("click", onUndo);
  document.getElementById("btnRedo").addEventListener("click", onRedo);

  // Layer controls
  document.getElementById("btnAddLayer").addEventListener("click", onAddLayer);
  document
    .getElementById("btnDeleteLayer")
    .addEventListener("click", onDeleteLayer);
  document.getElementById("btnMoveUp").addEventListener("click", onMoveUp);
  document.getElementById("btnMoveDown").addEventListener("click", onMoveDown);
  document
    .getElementById("btnDuplicate")
    .addEventListener("click", onDuplicate);

  const samplesOverlay = document.getElementById("samplesOverlay");
  const sampleSearch = document.getElementById("sampleSearch");
  const closeSamples = document.getElementById("btnCloseSamples");
  if (samplesOverlay) {
    samplesOverlay.addEventListener("click", (e) => {
      if (e.target === samplesOverlay) closeSamplesOverlay();
    });
  }
  if (sampleSearch) {
    sampleSearch.addEventListener("input", renderSampleList);
  }
  if (closeSamples) {
    closeSamples.addEventListener("click", closeSamplesOverlay);
  }

  document.addEventListener("keydown", onKeyDown);
  window.addEventListener("resize", () => {
    syncCanvasSize();
    refreshCurveEditor();
    drawShapeOverlay();
    markDirty();
  });
}

function syncAspectRatioFromInputs() {
  const w = parseInt(document.getElementById("prefWidth").value, 10) || 64;
  const h = parseInt(document.getElementById("prefHeight").value, 10) || 64;
  prefAspectRatio = Math.max(0.01, w / Math.max(1, h));
}

function onPrefLockAspectChange() {
  prefAspectLock = document.getElementById("prefLockAspect").checked;
  if (prefAspectLock) {
    syncAspectRatioFromInputs();
  }
}

function onPrefWidthChange() {
  if (prefAspectLock) {
    const w = parseInt(document.getElementById("prefWidth").value, 10) || 64;
    const h = Math.max(1, Math.round(w / Math.max(0.01, prefAspectRatio)));
    document.getElementById("prefHeight").value = h;
  }
  syncAspectRatioFromInputs();
  onPrefsChange();
}

function onPrefHeightChange() {
  if (prefAspectLock) {
    const h = parseInt(document.getElementById("prefHeight").value, 10) || 64;
    const w = Math.max(1, Math.round(h * Math.max(0.01, prefAspectRatio)));
    document.getElementById("prefWidth").value = w;
  }
  syncAspectRatioFromInputs();
  onPrefsChange();
}

function onPrefsChange() {
  const renderFrames =
    parseInt(document.getElementById("prefFrames").value, 10) || 1;
  const previewFrames =
    parseInt(document.getElementById("prefPreviewFrames").value, 10) ||
    renderFrames;
  const prefs = {
    width: parseInt(document.getElementById("prefWidth").value, 10) || 64,
    height: parseInt(document.getElementById("prefHeight").value, 10) || 64,
    frames: renderFrames,
    renderFrames: renderFrames,
    previewFrames: previewFrames,
    oversampling:
      parseInt(document.getElementById("prefOversample").value, 10) || 0,
    alignHorizontal:
      parseInt(document.getElementById("prefAlign").value, 10) || 0,
    exportOption:
      parseInt(document.getElementById("prefExport").value, 10) || 0,
    duration:
      parseInt(document.getElementById("prefDuration").value, 10) || 100,
    loop: parseInt(document.getElementById("prefLoop").value, 10) || 0,
    biDir: document.getElementById("prefBiDir").checked,
    bgAlpha: parseInt(document.getElementById("prefBgAlpha").value, 10) || 0,
    bgColor: document.getElementById("prefBgColor").value,
  };
  window.knobman_setPrefs(prefs);
  refreshFromDoc();
  refreshDetachedPreviewNow();
}

function syncPrefsFromGo() {
  const p = window.knobman_getPrefs();
  if (!p) return;
  if (p.width != null) document.getElementById("prefWidth").value = p.width;
  if (p.height != null) document.getElementById("prefHeight").value = p.height;
  if (p.renderFrames != null)
    document.getElementById("prefFrames").value = p.renderFrames;
  else if (p.frames != null)
    document.getElementById("prefFrames").value = p.frames;
  if (p.previewFrames != null)
    document.getElementById("prefPreviewFrames").value = p.previewFrames;
  if (p.oversampling != null)
    document.getElementById("prefOversample").value = p.oversampling;
  if (p.alignHorizontal != null)
    document.getElementById("prefAlign").value = p.alignHorizontal;
  if (p.exportOption != null)
    document.getElementById("prefExport").value = p.exportOption;
  if (p.duration != null)
    document.getElementById("prefDuration").value = p.duration;
  if (p.loop != null) document.getElementById("prefLoop").value = p.loop;
  if (p.biDir != null)
    document.getElementById("prefBiDir").checked = Boolean(p.biDir);
  if (p.bgAlpha != null)
    document.getElementById("prefBgAlpha").value = p.bgAlpha;
  if (p.bgColor) document.getElementById("prefBgColor").value = p.bgColor;
  prefAspectLock = document.getElementById("prefLockAspect").checked;
  syncAspectRatioFromInputs();
}

// ── Layers ────────────────────────────────────────────────────────────────────

function refreshLayerList() {
  const layerList = document.getElementById("layerList");
  layerList.innerHTML = "";

  const layers = window.knobman_getLayerList() || [];
  if (selectedLayer >= layers.length)
    selectedLayer = Math.max(0, layers.length - 1);
  const previewJobs = [];
  layerPreviewToken += 1;

  layers.forEach((layer) => {
    const li = document.createElement("li");
    li.dataset.layerIndex = String(layer.index);
    if (layer.selected) {
      li.classList.add("active");
      selectedLayer = layer.index;
    }

    const vis = document.createElement("span");
    vis.className = "layer-vis" + (layer.visible ? " on" : "");
    vis.textContent = "V";
    vis.title = "Toggle visibility";
    vis.addEventListener("click", (e) => {
      e.stopPropagation();
      window.knobman_setLayerVisible(layer.index, !layer.visible);
      refreshLayerList();
      markDirty();
    });

    const solo = document.createElement("span");
    solo.className = "layer-solo" + (layer.solo ? " on" : "");
    solo.textContent = "S";
    solo.title = "Toggle solo";
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
      selectedLayer = window.knobman_selectLayer(layer.index);
      refreshLayerList();
      refreshParamPanel();
      markDirty();
    });

    layerList.appendChild(li);
  });

  renderLayerPreviewsAsync(previewJobs);
}

// ── Primitive parameter panel ─────────────────────────────────────────────────

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
    const v = parseInt(input.value, 10);
    return Number.isFinite(v) ? v : 0;
  }
  if (def.numeric === "float") {
    const v = parseFloat(input.value);
    return Number.isFinite(v) ? v : 0;
  }
  return input.value;
}

function applyParamChange(key, value) {
  const ok = window.knobman_setParam(selectedLayer, key, value);
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
    refreshShapeEditor();
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
    const a = window.knobman_getParam(selectedLayer, "colorAlpha");
    alphaInput.value = String(a == null ? 255 : a);

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
      window.knobman_getParam(selectedLayer, "hasEmbeddedImage"),
    );
    const name = String(window.knobman_getParam(selectedLayer, "file") || "");
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
      const el = document.createElement("option");
      el.value = String(opt.value);
      el.textContent = opt.label;
      input.appendChild(el);
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
  input.addEventListener(eventName, () => {
    const v = coerceParamValue(def, input);
    applyParamChange(key, v);
  });

  if (def.type === "checkbox") {
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

  const row = document.createElement("div");
  row.className = "param-row";
  if (def.type === "checkbox") row.classList.add("checkbox");

  const caption = document.createElement("span");
  caption.textContent = def.label;

  let input;
  if (def.type === "select") {
    input = document.createElement("select");
    (def.options || []).forEach((opt) => {
      const el = document.createElement("option");
      el.value = String(opt.value);
      el.textContent = opt.label;
      input.appendChild(el);
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

  const eventName =
    def.type === "select" || def.type === "checkbox" ? "change" : "input";
  input.addEventListener(eventName, () => {
    const v = coerceParamValue(def, input);
    applyEffectParamChange(key, v);
    if (isCurveSelectorField(key) && Number(v) > 0) {
      focusCurve(Number(v));
    }
  });

  if (def.type === "checkbox") {
    row.appendChild(caption);
    row.appendChild(input);
  } else {
    row.appendChild(caption);
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
    section.fields.forEach((key) => {
      const value = window.knobman_getEffectParam(selectedLayer, key);
      const row = buildEffectRow(key, value);
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
    window.knobman_getParam(selectedLayer, "textureFile") ?? 0;

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
    const opt = document.createElement("option");
    opt.value = String(tex.index);
    opt.textContent = `${tex.index}: ${tex.name} (${tex.width}x${tex.height})`;
    select.appendChild(opt);
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
      img.addEventListener("load", () => URL.revokeObjectURL(url), {
        once: true,
      });
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
  const content = document.getElementById("paramContent");
  content.innerHTML = "";

  const layers = window.knobman_getLayerList() || [];
  if (layers.length === 0) {
    const p = document.createElement("p");
    p.className = "placeholder";
    p.textContent = "No layer selected.";
    content.appendChild(p);
    return;
  }
  selectedLayer = Math.max(0, Math.min(selectedLayer, layers.length - 1));

  const primType = window.knobman_getParam(selectedLayer, "primType") ?? 0;
  const fields = ["name", "primType", ...fieldsForPrimType(primType)];

  fields.forEach((key) => {
    const value = window.knobman_getParam(selectedLayer, key);
    const row = buildParamRow(key, value);
    if (row) content.appendChild(row);
  });

  appendTexturePanel(content, primType);
  appendEffectSections(content);
  refreshShapeEditor();
}

// ── Toolbar handlers ──────────────────────────────────────────────────────────

function onNew() {
  window.knobman_newDocument();
  projectBaseName = "project";
  invalidateLayerPreviews();
  currentFrame = 0;
  refreshFromDoc();
  ensureBuiltinTextures().then(() => refreshParamPanel());
  localStorage.removeItem(SESSION_KEY);
  setStatus("New document");
}

async function onSave() {
  const data = window.knobman_saveFile();
  if (!data || data.length === 0) {
    setStatus("Save failed");
    return;
  }
  const fileName = buildDownloadName("project", "knob");
  const mode = await saveBytes(
    data,
    fileName,
    "application/octet-stream",
    "knobman-save",
  );
  if (mode === "canceled") {
    setStatus("Save canceled");
    return;
  }
  setStatus(mode === "picker" ? `Saved ${fileName}` : `Downloaded ${fileName}`);
}

function downloadBytes(fileName, mimeType, bytes) {
  const blob = new Blob([bytes], {
    type: mimeType || "application/octet-stream",
  });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = fileName;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

function buildPickerTypes(fileName, mimeType) {
  const extIdx = fileName.lastIndexOf(".");
  if (extIdx < 0 || extIdx === fileName.length - 1) return undefined;
  const ext = fileName.slice(extIdx).toLowerCase();
  if (!ext || !mimeType) return undefined;
  return [
    {
      description: ext.slice(1).toUpperCase() + " File",
      accept: { [mimeType]: [ext] },
    },
  ];
}

function sanitizeFileBaseName(name) {
  const raw = String(name || "").trim();
  const mapped = raw
    .replace(/[^A-Za-z0-9._-]+/g, "_")
    .replace(/_+/g, "_")
    .replace(/^[_\. -]+|[_\. -]+$/g, "");
  if (!mapped) return "project";
  return mapped.slice(0, 64);
}

function stripFileExtension(name) {
  const s = String(name || "");
  const idx = s.lastIndexOf(".");
  if (idx <= 0) return s;
  return s.slice(0, idx);
}

function setProjectBaseNameFromFileName(fileName) {
  projectBaseName = sanitizeFileBaseName(stripFileExtension(fileName));
}

function filenameTimestampNow() {
  const d = new Date();
  const pad2 = (n) => String(n).padStart(2, "0");
  return (
    [d.getFullYear(), pad2(d.getMonth() + 1), pad2(d.getDate())].join("") +
    "-" +
    [pad2(d.getHours()), pad2(d.getMinutes()), pad2(d.getSeconds())].join("")
  );
}

function buildDownloadName(tag, ext) {
  const base = sanitizeFileBaseName(projectBaseName || "project");
  const suffix = sanitizeFileBaseName(tag || "file");
  const ts = filenameTimestampNow();
  const extension = String(ext || "").replace(/^\./, "");
  return extension
    ? `${base}_${ts}_${suffix}.${extension}`
    : `${base}_${ts}_${suffix}`;
}

async function saveBytes(bytes, fileName, mimeType, pickerId) {
  if (
    window.isSecureContext &&
    typeof window.showSaveFilePicker === "function"
  ) {
    try {
      const handle = await window.showSaveFilePicker({
        id: pickerId || "knobman-download",
        suggestedName: fileName,
        types: buildPickerTypes(fileName, mimeType),
      });
      const writable = await handle.createWritable();
      await writable.write(bytes);
      await writable.close();
      return "picker";
    } catch (err) {
      if (err && err.name === "AbortError") return "canceled";
      console.warn(
        "showSaveFilePicker failed, falling back to download link:",
        err,
      );
    }
  }
  downloadBytes(fileName, mimeType, bytes);
  return "download";
}

async function onExport() {
  const option = parseInt(document.getElementById("prefExport").value, 10) || 0;
  if (option === 0 || option === 1) {
    if (!window.knobman_exportPNGStrip) {
      setStatus("PNG strip export unavailable");
      return;
    }
    const horizontal = option === 1;
    const out = window.knobman_exportPNGStrip(horizontal);
    if (!out || out.length === 0) {
      setStatus("PNG strip export failed");
      return;
    }
    const suffix = horizontal ? "strip_h" : "strip_v";
    const fileName = buildDownloadName(suffix, "png");
    const mode = await saveBytes(
      out,
      fileName,
      "image/png",
      "knobman-export-png-strip",
    );
    if (mode === "canceled") {
      setStatus("PNG strip export canceled");
      return;
    }
    setStatus(
      mode === "picker" ? `Exported ${fileName}` : `Downloaded ${fileName}`,
    );
    return;
  }
  if (option === 2) {
    if (!window.knobman_exportPNGFramesZip) {
      setStatus("PNG frames export unavailable");
      return;
    }
    const out = window.knobman_exportPNGFramesZip();
    if (!out || out.length === 0) {
      setStatus("PNG frames export failed");
      return;
    }
    const fileName = buildDownloadName("frames", "zip");
    const mode = await saveBytes(
      out,
      fileName,
      "application/zip",
      "knobman-export-frames-zip",
    );
    if (mode === "canceled") {
      setStatus("PNG frames export canceled");
      return;
    }
    setStatus(
      mode === "picker" ? `Exported ${fileName}` : `Downloaded ${fileName}`,
    );
    return;
  }
  if (option === 3) {
    if (!window.knobman_exportGIF) {
      setStatus("GIF export unavailable");
      return;
    }
    const out = window.knobman_exportGIF();
    if (!out || out.length === 0) {
      setStatus("GIF export failed");
      return;
    }
    const fileName = buildDownloadName("anim", "gif");
    const mode = await saveBytes(
      out,
      fileName,
      "image/gif",
      "knobman-export-gif",
    );
    if (mode === "canceled") {
      setStatus("GIF export canceled");
      return;
    }
    setStatus(
      mode === "picker" ? `Exported ${fileName}` : `Downloaded ${fileName}`,
    );
    return;
  }
  if (option === 4) {
    if (!window.knobman_exportAPNG) {
      setStatus("APNG export unavailable");
      return;
    }
    const out = window.knobman_exportAPNG();
    if (!out || out.length === 0) {
      setStatus("APNG export failed");
      return;
    }
    const fileName = buildDownloadName("anim", "apng");
    const mode = await saveBytes(
      out,
      fileName,
      "image/apng",
      "knobman-export-apng",
    );
    if (mode === "canceled") {
      setStatus("APNG export canceled");
      return;
    }
    setStatus(
      mode === "picker" ? `Exported ${fileName}` : `Downloaded ${fileName}`,
    );
    return;
  }
  setStatus("Unknown export option");
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
  const btnUndo = document.getElementById("btnUndo");
  const btnRedo = document.getElementById("btnRedo");
  if (btnUndo)
    btnUndo.disabled = !(window.knobman_canUndo && window.knobman_canUndo());
  if (btnRedo)
    btnRedo.disabled = !(window.knobman_canRedo && window.knobman_canRedo());
}

function onAddLayer() {
  selectedLayer = window.knobman_addLayer();
  invalidateLayerPreviews();
  refreshLayerList();
  refreshParamPanel();
  markDirty();
}

function onDeleteLayer() {
  window.knobman_deleteLayer(selectedLayer);
  invalidateLayerPreviews();
  refreshLayerList();
  refreshParamPanel();
  markDirty();
}

function onMoveUp() {
  selectedLayer = window.knobman_moveLayer(-1);
  invalidateLayerPreviews();
  refreshLayerList();
  refreshParamPanel();
  markDirty();
}

function onMoveDown() {
  selectedLayer = window.knobman_moveLayer(1);
  invalidateLayerPreviews();
  refreshLayerList();
  refreshParamPanel();
  markDirty();
}

function onDuplicate() {
  selectedLayer = window.knobman_duplicateLayer();
  invalidateLayerPreviews();
  refreshLayerList();
  refreshParamPanel();
  markDirty();
}

function applyLoadedProjectBytes(bytes, fileName, statusPrefix) {
  const ok = window.knobman_loadFile(bytes);
  if (!ok) return false;
  setProjectBaseNameFromFileName(fileName);
  invalidateLayerPreviews();
  currentFrame = 0;
  refreshFromDoc();
  ensureBuiltinTextures().then(() => refreshParamPanel());
  markDirty();
  setStatus((statusPrefix || "Loaded") + " " + fileName);
  return true;
}

async function loadSampleProject(fileName) {
  setStatus("Loading sample " + fileName + "...");
  const bytes = await fetchSampleProjectBytes(fileName);
  if (!bytes || bytes.length === 0) {
    setStatus("Failed to load sample " + fileName);
    return;
  }
  if (!applyLoadedProjectBytes(bytes, fileName, "Loaded sample")) {
    setStatus("Failed to load sample " + fileName);
    return;
  }
  closeSamplesOverlay();
}

function onFileOpen(e) {
  const file = e.target.files[0];
  if (!file) return;

  const reader = new FileReader();
  reader.onload = () => {
    const data = new Uint8Array(reader.result);
    if (!applyLoadedProjectBytes(data, file.name, "Loaded")) {
      setStatus("Failed to load " + file.name);
      return;
    }
  };
  reader.readAsArrayBuffer(file);
  e.target.value = "";
}

function onKeyDown(e) {
  if (e.key === "Escape") {
    const welcomeOverlay = document.getElementById("welcomeOverlay");
    if (welcomeOverlay && !welcomeOverlay.hidden) {
      e.preventDefault();
      closeWelcomeOverlay();
      return;
    }
  }
  if (e.key === "Escape" && isSamplesOverlayOpen()) {
    e.preventDefault();
    closeSamplesOverlay();
    return;
  }
  const mod = e.ctrlKey || e.metaKey;
  if (mod && e.key === "z") {
    e.preventDefault();
    onUndo();
    return;
  }
  if (mod && (e.key === "y" || (e.shiftKey && e.key === "z"))) {
    e.preventDefault();
    onRedo();
    return;
  }
  if (mod && e.key === "s") {
    e.preventDefault();
    onSave();
    return;
  }
  if (mod && e.key === "o") {
    e.preventDefault();
    document.getElementById("fileInput").click();
    return;
  }
  if (mod && e.key === "e") {
    e.preventDefault();
    onExport();
    return;
  }
  if (mod && e.key === "d") {
    e.preventDefault();
    onDuplicate();
    return;
  }
  if (e.key === "Delete") {
    e.preventDefault();
    if (isShapeLayerSelected() && shapeSelectedHandle) {
      deleteShapeSelection();
    } else {
      onDeleteLayer();
    }
    return;
  }
  if (e.key === "ArrowUp") {
    e.preventDefault();
    onMoveUp();
    return;
  }
  if (e.key === "ArrowDown") {
    e.preventDefault();
    onMoveDown();
    return;
  }
}

// ── Status bar ────────────────────────────────────────────────────────────────

let lastRenderMs = 0;

function setStatus(msg) {
  document.getElementById("statusMsg").textContent = msg;
}

function updateStatusMetrics(layerName) {
  const prefs = window.knobman_getPrefs ? window.knobman_getPrefs() : null;
  const w = prefs ? prefs.width : 0;
  const h = prefs ? prefs.height : 0;
  const frames = prefs ? prefs.frames : 0;
  const parts = [];
  if (w && h) parts.push(`${w}×${h}`);
  if (frames) parts.push(`${frames} fr`);
  parts.push(`F${currentFrame}`);
  if (layerName) parts.push(`L: ${layerName}`);
  if (lastRenderMs > 0) parts.push(`${lastRenderMs}ms`);
  document.getElementById("statusMetrics").textContent = parts.join(" | ");
}

// ── Session persistence (localStorage) ───────────────────────────────────────

const SESSION_KEY = "knobman_session";

function saveSession() {
  if (!window.knobman_saveFile) return;
  try {
    const bytes = window.knobman_saveFile();
    if (!bytes || !bytes.length) return;
    const b64 = btoa(String.fromCharCode(...new Uint8Array(bytes)));
    localStorage.setItem(SESSION_KEY, b64);
  } catch (_) {}
}

function restoreSession() {
  if (!window.knobman_loadFile) return false;
  try {
    const b64 = localStorage.getItem(SESSION_KEY);
    if (!b64) return false;
    const bin = atob(b64);
    const bytes = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
    window.knobman_loadFile(bytes);
    return true;
  } catch (_) {
    return false;
  }
}

