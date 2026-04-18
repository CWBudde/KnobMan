import { EFFECT_DEFS, HANDLE_DB_NAME, HANDLE_STORE_NAME } from "./state.js";

export function clampFloat(v, min, max) {
  return Math.max(min, Math.min(max, v));
}

export function clampInt(v, min, max) {
  if (!Number.isFinite(v)) return min;
  return Math.max(min, Math.min(max, Math.round(v)));
}

export function storageGet(key) {
  try {
    return window.localStorage.getItem(key);
  } catch (_err) {
    return null;
  }
}

export function storageSet(key, value) {
  try {
    window.localStorage.setItem(key, value);
    return true;
  } catch (_err) {
    return false;
  }
}

export function storageRemove(key) {
  try {
    window.localStorage.removeItem(key);
  } catch (_err) {}
}

export function bytesToBase64(bytes) {
  if (!bytes || !bytes.length) return "";
  const chunkSize = 0x8000;
  let binary = "";
  for (let i = 0; i < bytes.length; i += chunkSize) {
    const slice = bytes.subarray(i, Math.min(bytes.length, i + chunkSize));
    binary += String.fromCharCode.apply(null, slice);
  }
  return btoa(binary);
}

export function base64ToBytes(b64) {
  const bin = atob(String(b64 || ""));
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return bytes;
}

export function supportsOpenPicker() {
  return (
    window.isSecureContext &&
    typeof window.showOpenFilePicker === "function" &&
    typeof window.FileSystemFileHandle === "function"
  );
}

export function supportsStoredHandles() {
  return (
    typeof window.indexedDB !== "undefined" &&
    typeof window.FileSystemFileHandle === "function"
  );
}

export function openRecentHandlesDb(state) {
  if (!supportsStoredHandles()) return Promise.resolve(null);
  if (state.recentHandlesDbPromise) return state.recentHandlesDbPromise;
  state.recentHandlesDbPromise = new Promise((resolve) => {
    try {
      const req = indexedDB.open(HANDLE_DB_NAME, 1);
      req.onupgradeneeded = () => {
        const db = req.result;
        if (!db.objectStoreNames.contains(HANDLE_STORE_NAME)) {
          db.createObjectStore(HANDLE_STORE_NAME);
        }
      };
      req.onsuccess = () => resolve(req.result);
      req.onerror = () => resolve(null);
    } catch (_err) {
      resolve(null);
    }
  });
  return state.recentHandlesDbPromise;
}

export async function idbPutHandle(state, key, handle) {
  const db = await openRecentHandlesDb(state);
  if (!db || !key || !handle) return false;
  return new Promise((resolve) => {
    try {
      const tx = db.transaction(HANDLE_STORE_NAME, "readwrite");
      tx.objectStore(HANDLE_STORE_NAME).put(handle, key);
      tx.oncomplete = () => resolve(true);
      tx.onerror = () => resolve(false);
      tx.onabort = () => resolve(false);
    } catch (_err) {
      resolve(false);
    }
  });
}

export async function idbGetHandle(state, key) {
  const db = await openRecentHandlesDb(state);
  if (!db || !key) return null;
  return new Promise((resolve) => {
    try {
      const tx = db.transaction(HANDLE_STORE_NAME, "readonly");
      const req = tx.objectStore(HANDLE_STORE_NAME).get(key);
      req.onsuccess = () => resolve(req.result || null);
      req.onerror = () => resolve(null);
    } catch (_err) {
      resolve(null);
    }
  });
}

export async function idbDeleteHandle(state, key) {
  const db = await openRecentHandlesDb(state);
  if (!db || !key) return false;
  return new Promise((resolve) => {
    try {
      const tx = db.transaction(HANDLE_STORE_NAME, "readwrite");
      tx.objectStore(HANDLE_STORE_NAME).delete(key);
      tx.oncomplete = () => resolve(true);
      tx.onerror = () => resolve(false);
      tx.onabort = () => resolve(false);
    } catch (_err) {
      resolve(false);
    }
  });
}

export function uniqueId(prefix) {
  if (window.crypto && typeof window.crypto.randomUUID === "function") {
    return `${prefix}_${window.crypto.randomUUID()}`;
  }
  return `${prefix}_${Date.now()}_${Math.random().toString(36).slice(2, 10)}`;
}

export function isCurveSelectorField(key) {
  return typeof key === "string" && key.endsWith("Anim");
}

export function syncCanvasElementSize(
  canvas,
  width,
  height,
  setStyleSize = true,
) {
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

export function syncCanvasBackingToDisplaySize(canvas) {
  if (!canvas) return false;
  const rect = canvas.getBoundingClientRect();
  return syncCanvasElementSize(canvas, rect.width, rect.height, false);
}

export function canvasToBlobSync(canvas) {
  const dataUrl = canvas.toDataURL("image/png");
  const comma = dataUrl.indexOf(",");
  if (comma < 0) return null;
  const b64 = dataUrl.slice(comma + 1);
  const bin = atob(b64);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return new Blob([bytes], { type: "image/png" });
}

export function downloadBytes(fileName, mimeType, bytes) {
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

export function buildPickerTypes(fileName, mimeType) {
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

export function sanitizeFileBaseName(name) {
  const raw = String(name || "").trim();
  const mapped = raw
    .replace(/[^A-Za-z0-9._-]+/g, "_")
    .replace(/_+/g, "_")
    .replace(/^[_\. -]+|[_\. -]+$/g, "");
  if (!mapped) return "project";
  return mapped.slice(0, 64);
}

export function stripFileExtension(name) {
  const s = String(name || "");
  const idx = s.lastIndexOf(".");
  if (idx <= 0) return s;
  return s.slice(0, idx);
}

export function resolveAssetUrl(assetGroup, fileName, locationHref) {
  const href =
    locationHref ||
    (typeof window !== "undefined" && window.location
      ? window.location.href
      : "http://localhost/");
  const currentUrl = new URL(href);
  const currentDir = new URL(".", currentUrl);
  const fromWebDir = currentDir.pathname.endsWith("/web/");
  const relativePath = fromWebDir
    ? `../assets/${assetGroup}/${fileName}`
    : `assets/${assetGroup}/${fileName}`;
  return new URL(relativePath, currentUrl).href;
}

export function isPristineSessionPayload(payload, baseline) {
  if (!payload || !baseline) return false;
  return (
    payload.data === baseline.data &&
    Number(payload.currentFrame || 0) === Number(baseline.currentFrame || 0) &&
    Number(payload.zoomFactor || 0) === Number(baseline.zoomFactor || 0) &&
    Number(payload.selectedLayer || 0) === Number(baseline.selectedLayer || 0) &&
    Number(payload.selectedCurve || 0) === Number(baseline.selectedCurve || 0) &&
    Boolean(payload.prefAspectLock) === Boolean(baseline.prefAspectLock) &&
    String(payload.projectBaseName || "") === String(baseline.projectBaseName || "")
  );
}

export function getLayerToggleLabel(kind, isActive) {
  if (kind === "visibility") {
    return isActive ? "Hide layer" : "Show layer";
  }
  if (kind === "solo") {
    return isActive ? "Disable solo" : "Solo layer";
  }
  return "Layer control";
}

export function getLayerControlLabel(action) {
  switch (action) {
    case "add":
      return "Add layer";
    case "delete":
      return "Delete selected layer";
    case "up":
      return "Move selected layer up";
    case "down":
      return "Move selected layer down";
    case "duplicate":
      return "Duplicate selected layer";
    default:
      return "Layer control";
  }
}

export function hasBoundedRangeControl(def) {
  return Boolean(
    def &&
      def.type === "number" &&
      Number.isFinite(def.min) &&
      Number.isFinite(def.max) &&
      Number(def.max) > Number(def.min),
  );
}

function getEffectLabel(key) {
  return EFFECT_DEFS[key]?.label || key;
}

function isCurveEnabled(value) {
  return Number(value) > 0;
}

function appendAnimatedEffectRows(rows, curveKey, fromKey, toKey, labelBase, opts) {
  const disabled = Boolean(opts?.disabled);
  rows.push({
    key: curveKey,
    label: `${labelBase} Curve`,
    disabled,
  });
  if (opts?.showValues) {
    rows.push({
      key: fromKey,
      label: `${labelBase} From`,
      disabled,
    });
    rows.push({
      key: toKey,
      label: `${labelBase} To`,
      disabled,
    });
  }
}

export function getTransformEffectRows(effectValues) {
  const values = effectValues || {};
  const rows = [
    { key: "antiAlias", label: getEffectLabel("antiAlias"), disabled: false },
    { key: "unfold", label: getEffectLabel("unfold"), disabled: false },
    { key: "animStep", label: getEffectLabel("animStep"), disabled: false },
    { key: "zoomXYSepa", label: getEffectLabel("zoomXYSepa"), disabled: false },
  ];

  const zoomSeparated = Boolean(values.zoomXYSepa);
  appendAnimatedEffectRows(
    rows,
    "zoomXAnim",
    "zoomXF",
    "zoomXT",
    zoomSeparated ? "Zoom X" : "Zoom",
    { showValues: isCurveEnabled(values.zoomXAnim) },
  );
  if (zoomSeparated) {
    appendAnimatedEffectRows(rows, "zoomYAnim", "zoomYF", "zoomYT", "Zoom Y", {
      showValues: isCurveEnabled(values.zoomYAnim),
    });
  }
  appendAnimatedEffectRows(rows, "offXAnim", "offXF", "offXT", "Offset X", {
    showValues: isCurveEnabled(values.offXAnim),
  });
  appendAnimatedEffectRows(rows, "offYAnim", "offYF", "offYT", "Offset Y", {
    showValues: isCurveEnabled(values.offYAnim),
  });

  [
    "keepDir",
    "centerX",
    "centerY",
    "angleF",
    "angleT",
    "angleAnim",
  ].forEach((key) => {
    rows.push({
      key,
      label: getEffectLabel(key),
      disabled: false,
    });
  });

  return rows;
}

export function getColorEffectRows(effectValues) {
  const values = effectValues || {};
  const rows = [];

  [
    ["alpha", "alphaAnim", "alphaF", "alphaT", "Alpha"],
    ["bright", "brightAnim", "brightF", "brightT", "Brightness"],
    ["contrast", "contrastAnim", "contrastF", "contrastT", "Contrast"],
    ["saturation", "saturationAnim", "saturationF", "saturationT", "Saturation"],
    ["hue", "hueAnim", "hueF", "hueT", "Hue"],
  ].forEach(([, curveKey, fromKey, toKey, labelBase]) => {
    appendAnimatedEffectRows(rows, curveKey, fromKey, toKey, labelBase, {
      showValues: isCurveEnabled(values[curveKey]),
    });
  });

  return rows;
}

export function filenameTimestampNow() {
  const d = new Date();
  const pad2 = (n) => String(n).padStart(2, "0");
  return (
    [d.getFullYear(), pad2(d.getMonth() + 1), pad2(d.getDate())].join("") +
    "-" +
    [pad2(d.getHours()), pad2(d.getMinutes()), pad2(d.getSeconds())].join("")
  );
}
