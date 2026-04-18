const PREVIEW_SOURCE = "knobman-sample-preview";

const go = new Go();
let bootError = null;
let wasmReadyResolve = null;
const wasmReady = new Promise((resolve) => {
  wasmReadyResolve = resolve;
});
let renderQueue = Promise.resolve();

function post(type, payload) {
  window.parent.postMessage(
    {
      source: PREVIEW_SOURCE,
      type,
      ...payload,
    },
    "*",
  );
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
    } catch (_err) {}
  }
  return null;
}

function imageDataUrlFromRaw(raw) {
  if (!raw || !raw.data || !raw.width || !raw.height) return null;
  const width = Number(raw.width) || 0;
  const height = Number(raw.height) || 0;
  if (width <= 0 || height <= 0) return null;

  const pixels = new Uint8ClampedArray(raw.data.length);
  pixels.set(raw.data);

  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;
  const ctx = canvas.getContext("2d");
  if (!ctx) return null;
  ctx.putImageData(new ImageData(pixels, width, height), 0, 0);
  return {
    dataUrl: canvas.toDataURL("image/png"),
    width,
    height,
  };
}

async function renderSample(fileName) {
  await wasmReady;
  if (bootError) throw bootError;

  const bytes = await fetchSampleProjectBytes(fileName);
  if (!bytes || !bytes.length) {
    throw new Error("sample-not-found");
  }
  if (!window.knobman_loadFile || !window.knobman_renderFrameRaw) {
    throw new Error("preview-runtime-unavailable");
  }
  if (!window.knobman_loadFile(bytes)) {
    throw new Error("sample-load-failed");
  }
  const raw = window.knobman_renderFrameRaw(0);
  const image = imageDataUrlFromRaw(raw);
  if (!image) {
    throw new Error("sample-render-failed");
  }
  return image;
}

window.addEventListener("message", (event) => {
  const msg = event.data;
  if (!msg || msg.source !== PREVIEW_SOURCE || msg.type !== "render") return;

  renderQueue = renderQueue
    .then(async () => {
      try {
        const image = await renderSample(msg.fileName);
        post("result", {
          id: msg.id,
          fileName: msg.fileName,
          dataUrl: image.dataUrl,
          width: image.width,
          height: image.height,
        });
      } catch (err) {
        post("result", {
          id: msg.id,
          fileName: msg.fileName,
          error: String(err && err.message ? err.message : err),
        });
      }
    })
    .catch(() => {});
});

WebAssembly.instantiateStreaming(
  fetch("knobman.wasm", { cache: "no-store" }),
  go.importObject,
)
  .then((result) => {
    go.run(result.instance);
    wasmReadyResolve();
    post("ready", {});
  })
  .catch((err) => {
    bootError = err instanceof Error ? err : new Error(String(err));
    wasmReadyResolve();
    post("ready", { error: String(bootError.message || bootError) });
  });
