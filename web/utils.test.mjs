import test from "node:test";
import assert from "node:assert/strict";

import { isPristineSessionPayload, resolveAssetUrl } from "./utils.js";

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
