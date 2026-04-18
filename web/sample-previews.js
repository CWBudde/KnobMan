const PREVIEW_SOURCE = "knobman-sample-preview";

export function createSamplePreviewService({ renderer }) {
  const cache = new Map();
  const inflight = new Map();

  async function getPreview(fileName) {
    if (cache.has(fileName)) {
      return cache.get(fileName);
    }
    if (inflight.has(fileName)) {
      return inflight.get(fileName);
    }

    const pending = Promise.resolve()
      .then(() => renderer(fileName))
      .then((preview) => {
        const value = preview ?? null;
        cache.set(fileName, value);
        inflight.delete(fileName);
        return value;
      })
      .catch((_err) => {
        cache.set(fileName, null);
        inflight.delete(fileName);
        return null;
      });

    inflight.set(fileName, pending);
    return pending;
  }

  return {
    getPreview,
  };
}

export function createIframeSamplePreviewRenderer({
  frameUrl = "sample-preview-frame.html",
} = {}) {
  let frame = null;
  let readyPromise = null;
  let requestId = 0;
  const pending = new Map();

  function ensureFrame() {
    if (readyPromise) return readyPromise;

    frame = document.createElement("iframe");
    frame.hidden = true;
    frame.tabIndex = -1;
    frame.setAttribute("aria-hidden", "true");
    frame.className = "sample-preview-frame";
    frame.src = frameUrl;
    document.body.appendChild(frame);

    readyPromise = new Promise((resolve, reject) => {
      function onMessage(event) {
        if (!frame || event.source !== frame.contentWindow) return;
        const msg = event.data;
        if (!msg || msg.source !== PREVIEW_SOURCE) return;

        if (msg.type === "ready") {
          if (msg.error) {
            window.removeEventListener("message", onMessage);
            reject(new Error(msg.error));
            return;
          }
          resolve(frame.contentWindow);
          return;
        }

        if (msg.type !== "result") return;
        const entry = pending.get(msg.id);
        if (!entry) return;
        pending.delete(msg.id);
        if (msg.error) {
          entry.reject(new Error(msg.error));
          return;
        }
        entry.resolve({
          fileName: msg.fileName,
          url: msg.dataUrl,
          width: msg.width,
          height: msg.height,
        });
      }

      window.addEventListener("message", onMessage);
    });

    return readyPromise;
  }

  return async function renderSamplePreview(fileName) {
    const target = await ensureFrame();
    const id = `preview_${++requestId}`;
    return new Promise((resolve, reject) => {
      pending.set(id, { resolve, reject });
      target.postMessage(
        {
          source: PREVIEW_SOURCE,
          type: "render",
          id,
          fileName,
        },
        "*",
      );
    });
  };
}
