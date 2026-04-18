import test from "node:test";
import assert from "node:assert/strict";

import { createSamplePreviewService } from "./sample-previews.js";

test("sample preview service caches completed previews and dedupes inflight requests", async () => {
  let calls = 0;
  const service = createSamplePreviewService({
    renderer: async (fileName) => {
      calls += 1;
      await Promise.resolve();
      return { fileName, url: `blob:${fileName}` };
    },
  });

  const [first, second] = await Promise.all([
    service.getPreview("Aqua.knob"),
    service.getPreview("Aqua.knob"),
  ]);

  assert.equal(calls, 1);
  assert.equal(first.url, "blob:Aqua.knob");
  assert.equal(second, first);

  const third = await service.getPreview("Aqua.knob");
  assert.equal(calls, 1);
  assert.equal(third, first);
});

test("sample preview service stores failures and returns null on later requests", async () => {
  let calls = 0;
  const service = createSamplePreviewService({
    renderer: async () => {
      calls += 1;
      throw new Error("render failed");
    },
  });

  const first = await service.getPreview("Broken.knob");
  const second = await service.getPreview("Broken.knob");

  assert.equal(first, null);
  assert.equal(second, null);
  assert.equal(calls, 1);
});
