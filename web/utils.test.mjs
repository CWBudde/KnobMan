import test from "node:test";
import assert from "node:assert/strict";

import {
  getColorEffectRows,
  getDropShadowEffectRows,
  getEmbossEffectRows,
  getInnerShadowEffectRows,
  getSpecularHighlightRows,
  getTransformEffectRows,
  getLayerControlLabel,
  getLayerToggleLabel,
  hasBoundedRangeControl,
  isPristineSessionPayload,
  resolveAssetUrl,
} from "./utils.js";

test("resolveAssetUrl uses sibling assets when app is served from web/", () => {
  const url = resolveAssetUrl(
    "samples",
    "Ticked_HSlider.knob",
    "https://example.test/web/index.html",
  );

  assert.equal(url, "https://example.test/assets/samples/Ticked_HSlider.knob");
});

test("resolveAssetUrl keeps repo subpath when app is served from dist root", () => {
  const url = resolveAssetUrl(
    "samples",
    "Ticked_HSlider.knob",
    "https://example.test/KnobMan/index.html",
  );

  assert.equal(
    url,
    "https://example.test/KnobMan/assets/samples/Ticked_HSlider.knob",
  );
});

test("isPristineSessionPayload matches an untouched default session", () => {
  const pristine = isPristineSessionPayload(
    {
      data: "blank",
      currentFrame: 0,
      zoomFactor: 8,
      selectedLayer: 0,
      selectedCurve: 1,
      prefAspectLock: false,
      projectBaseName: "project",
    },
    {
      data: "blank",
      currentFrame: 0,
      zoomFactor: 8,
      selectedLayer: 0,
      selectedCurve: 1,
      prefAspectLock: false,
      projectBaseName: "project",
    },
  );

  assert.equal(pristine, true);
});

test("isPristineSessionPayload keeps non-default sessions restorable", () => {
  const pristine = isPristineSessionPayload(
    {
      data: "custom",
      currentFrame: 0,
      zoomFactor: 8,
      selectedLayer: 0,
      selectedCurve: 1,
      prefAspectLock: false,
      projectBaseName: "project",
    },
    {
      data: "blank",
      currentFrame: 0,
      zoomFactor: 8,
      selectedLayer: 0,
      selectedCurve: 1,
      prefAspectLock: false,
      projectBaseName: "project",
    },
  );

  assert.equal(pristine, false);
});

test("getLayerToggleLabel returns state-aware visibility and solo labels", () => {
  assert.equal(getLayerToggleLabel("visibility", true), "Hide layer");
  assert.equal(getLayerToggleLabel("visibility", false), "Show layer");
  assert.equal(getLayerToggleLabel("solo", true), "Disable solo");
  assert.equal(getLayerToggleLabel("solo", false), "Solo layer");
});

test("getLayerControlLabel returns descriptive layer toolbar labels", () => {
  assert.equal(getLayerControlLabel("add"), "Add layer");
  assert.equal(getLayerControlLabel("delete"), "Delete selected layer");
  assert.equal(getLayerControlLabel("up"), "Move selected layer up");
  assert.equal(getLayerControlLabel("down"), "Move selected layer down");
  assert.equal(getLayerControlLabel("duplicate"), "Duplicate selected layer");
});

test("hasBoundedRangeControl is true only for bounded numeric defs", () => {
  assert.equal(
    hasBoundedRangeControl({ type: "number", min: -100, max: 100 }),
    true,
  );
  assert.equal(
    hasBoundedRangeControl({ type: "number", min: 0, max: 0 }),
    false,
  );
  assert.equal(
    hasBoundedRangeControl({ type: "number", min: 0 }),
    false,
  );
  assert.equal(
    hasBoundedRangeControl({ type: "text", min: 0, max: 10 }),
    false,
  );
});

test("getTransformEffectRows collapses zoom labels and hides inactive value rows", () => {
  const rows = getTransformEffectRows({
    zoomXYSepa: false,
    zoomXAnim: 1,
    zoomYAnim: 3,
    offXAnim: 0,
    offYAnim: 2,
  });

  assert.deepEqual(
    rows.map((row) => [row.key, row.label, row.disabled]),
    [
      ["antiAlias", "AntiAlias", false],
      ["unfold", "Unfold", false],
      ["animStep", "AnimStep", false],
      ["zoomXYSepa", "Zoom XY Sepa", false],
      ["zoomXAnim", "Zoom Curve", false],
      ["zoomXF", "Zoom From", false],
      ["zoomXT", "Zoom To", false],
      ["offXAnim", "Offset X Curve", false],
      ["offYAnim", "Offset Y Curve", false],
      ["offYF", "Offset Y From", false],
      ["offYT", "Offset Y To", false],
      ["keepDir", "Keep Dir", false],
      ["centerX", "Center X", false],
      ["centerY", "Center Y", false],
      ["angleF", "Angle From", false],
      ["angleT", "Angle To", false],
      ["angleAnim", "Angle Curve", false],
    ],
  );
});

test("getTransformEffectRows shows both zoom axes and value rows when enabled", () => {
  const rows = getTransformEffectRows({
    zoomXYSepa: true,
    zoomXAnim: 2,
    zoomYAnim: 1,
    offXAnim: 3,
    offYAnim: 0,
  });

  assert.deepEqual(
    rows.map((row) => [row.key, row.label, row.disabled]),
    [
      ["antiAlias", "AntiAlias", false],
      ["unfold", "Unfold", false],
      ["animStep", "AnimStep", false],
      ["zoomXYSepa", "Zoom XY Sepa", false],
      ["zoomXAnim", "Zoom X Curve", false],
      ["zoomXF", "Zoom X From", false],
      ["zoomXT", "Zoom X To", false],
      ["zoomYAnim", "Zoom Y Curve", false],
      ["zoomYF", "Zoom Y From", false],
      ["zoomYT", "Zoom Y To", false],
      ["offXAnim", "Offset X Curve", false],
      ["offXF", "Offset X From", false],
      ["offXT", "Offset X To", false],
      ["offYAnim", "Offset Y Curve", false],
      ["keepDir", "Keep Dir", false],
      ["centerX", "Center X", false],
      ["centerY", "Center Y", false],
      ["angleF", "Angle From", false],
      ["angleT", "Angle To", false],
      ["angleAnim", "Angle Curve", false],
    ],
  );
});

test("getColorEffectRows moves curves first and hides off value rows", () => {
  const rows = getColorEffectRows({
    alphaAnim: 0,
    brightAnim: 2,
    contrastAnim: 1,
    saturationAnim: 0,
    hueAnim: 3,
  });

  assert.deepEqual(
    rows.map((row) => [row.key, row.label, row.disabled]),
    [
      ["alphaAnim", "Alpha Curve", false],
      ["brightAnim", "Brightness Curve", false],
      ["brightF", "Brightness From", false],
      ["brightT", "Brightness To", false],
      ["contrastAnim", "Contrast Curve", false],
      ["contrastF", "Contrast From", false],
      ["contrastT", "Contrast To", false],
      ["saturationAnim", "Saturation Curve", false],
      ["hueAnim", "Hue Curve", false],
      ["hueF", "Hue From", false],
      ["hueT", "Hue To", false],
    ],
  );
});

test("getSpecularHighlightRows keeps static values visible and puts curves last", () => {
  const rows = getSpecularHighlightRows({
    sLightDirAnim: 0,
    sDensityAnim: 2,
  });

  assert.deepEqual(
    rows.map((row) => [row.key, row.label, row.disabled]),
    [
      ["sLightDirF", "LightDir", false],
      ["sLightDirAnim", "LightDir Curve", false],
      ["sDensityF", "Density From", false],
      ["sDensityT", "Density To", false],
      ["sDensityAnim", "Density Curve", false],
    ],
  );
});

test("getDropShadowEffectRows preserves fixed controls and only expands animated groups when enabled", () => {
  const rows = getDropShadowEffectRows({
    dLightDirAnim: 0,
    dOffsetAnim: 1,
    dDensityAnim: 0,
    dDiffuseAnim: 3,
  });

  assert.deepEqual(
    rows.map((row) => [row.key, row.label, row.disabled]),
    [
      ["dLightDirEna", "Enable", false],
      ["dLightDirF", "LightDir", false],
      ["dLightDirAnim", "LightDir Curve", false],
      ["dOffsetF", "Offset From", false],
      ["dOffsetT", "Offset To", false],
      ["dOffsetAnim", "Offset Curve", false],
      ["dDensityF", "Density", false],
      ["dDensityAnim", "Density Curve", false],
      ["dDiffuseF", "Diffuse From", false],
      ["dDiffuseT", "Diffuse To", false],
      ["dDiffuseAnim", "Diffuse Curve", false],
      ["dsType", "Shadow Type", false],
      ["dsGrad", "Gradient", false],
    ],
  );
});

test("getEmbossEffectRows applies the same static-vs-animated rule", () => {
  const rows = getEmbossEffectRows({
    eLightDirAnim: 1,
    eOffsetAnim: 0,
    eDensityAnim: 0,
  });

  assert.deepEqual(
    rows.map((row) => [row.key, row.label, row.disabled]),
    [
      ["eLightDirEna", "Enable", false],
      ["eLightDirF", "LightDir From", false],
      ["eLightDirT", "LightDir To", false],
      ["eLightDirAnim", "LightDir Curve", false],
      ["eOffsetF", "Offset", false],
      ["eOffsetAnim", "Offset Curve", false],
      ["eDensityF", "Density", false],
      ["eDensityAnim", "Density Curve", false],
    ],
  );
});

test("getInnerShadowEffectRows keeps curve last and only shows To when animated", () => {
  const rows = getInnerShadowEffectRows({
    iLightDirAnim: 0,
    iOffsetAnim: 0,
    iDensityAnim: 2,
    iDiffuseAnim: 0,
  });

  assert.deepEqual(
    rows.map((row) => [row.key, row.label, row.disabled]),
    [
      ["iLightDirEna", "Enable", false],
      ["iLightDirF", "LightDir", false],
      ["iLightDirAnim", "LightDir Curve", false],
      ["iOffsetF", "Offset", false],
      ["iOffsetAnim", "Offset Curve", false],
      ["iDensityF", "Density From", false],
      ["iDensityT", "Density To", false],
      ["iDensityAnim", "Density Curve", false],
      ["iDiffuseF", "Diffuse", false],
      ["iDiffuseAnim", "Diffuse Curve", false],
    ],
  );
});
