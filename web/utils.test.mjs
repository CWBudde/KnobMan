import test from "node:test";
import assert from "node:assert/strict";

import {
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
      ["zoomYAnim", "Zoom Y Curve", true],
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
