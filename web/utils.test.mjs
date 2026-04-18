import test from "node:test";
import assert from "node:assert/strict";

import {
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
