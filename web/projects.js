import {
  BUILTIN_TEXTURE_FILES,
  LEGACY_SESSION_KEY,
  RECENT_DOCS_KEY,
  RECENT_DOC_LIMIT,
  SAMPLE_PROJECT_FILES,
  SESSION_KEY,
  WELCOME_SUPPRESS_KEY,
} from "./state.js";
import {
  base64ToBytes,
  buildPickerTypes,
  bytesToBase64,
  clampInt,
  downloadBytes,
  filenameTimestampNow,
  idbDeleteHandle,
  idbGetHandle,
  idbPutHandle,
  resolveAssetUrl,
  sanitizeFileBaseName,
  storageGet,
  storageRemove,
  storageSet,
  stripFileExtension,
  supportsOpenPicker,
  uniqueId,
} from "./utils.js";
import {
  createIframeSamplePreviewRenderer,
  createSamplePreviewService,
} from "./sample-previews.js";

export function createProjects({
  state,
  el,
  invalidateLayerPreviews,
  markDirty,
  refreshFromDoc,
  refreshParamPanel,
  setStatus,
  syncAspectRatioFromInputs,
}) {
  const samplePreviewService = createSamplePreviewService({
    renderer: createIframeSamplePreviewRenderer(),
  });

  async function fetchBuiltinTextureBytes(filename) {
    try {
      const res = await fetch(resolveAssetUrl("textures", filename));
      if (!res.ok) return null;
      const buf = await res.arrayBuffer();
      if (buf.byteLength > 0) return new Uint8Array(buf);
    } catch (_err) {}
    return null;
  }

  async function ensureBuiltinTextures() {
    if (state.builtinTextureLoadPromise) return state.builtinTextureLoadPromise;
    state.builtinTextureLoadPromise = (async () => {
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
      state.builtinTextureLoadPromise = null;
    });
    return state.builtinTextureLoadPromise;
  }

  function sampleLabelFromFileName(fileName) {
    return stripFileExtension(fileName).replace(/[_-]+/g, " ").trim();
  }

  function buildSampleCard(fileName, onClick) {
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "sample-item sample-preview-item";

    const layout = document.createElement("div");
    layout.className = "sample-preview-layout";

    const textCol = document.createElement("div");
    textCol.className = "sample-preview-text";
    const label = document.createElement("strong");
    label.textContent = sampleLabelFromFileName(fileName);
    textCol.appendChild(label);
    const meta = document.createElement("small");
    meta.textContent = fileName;
    textCol.appendChild(meta);

    const preview = document.createElement("div");
    preview.className = "sample-preview loading";
    const status = document.createElement("span");
    status.className = "sample-preview-status";
    status.textContent = "Loading preview…";
    preview.appendChild(status);

    layout.appendChild(textCol);
    layout.appendChild(preview);
    btn.appendChild(layout);
    btn.addEventListener("click", onClick);

    void samplePreviewService.getPreview(fileName).then((result) => {
      if (!btn.isConnected) return;
      preview.classList.remove("loading");
      if (!result || !result.url) {
        preview.classList.add("failed");
        status.textContent = "No preview";
        return;
      }
      preview.classList.add("ready");
      const img = document.createElement("img");
      img.className = "sample-preview-image";
      img.alt = `${sampleLabelFromFileName(fileName)} preview`;
      img.src = result.url;
      preview.textContent = "";
      preview.appendChild(img);
    });

    return btn;
  }

  function recentDocsLoad() {
    try {
      const raw = storageGet(RECENT_DOCS_KEY);
      if (!raw) return [];
      const parsed = JSON.parse(raw);
      return Array.isArray(parsed) ? parsed : [];
    } catch (_err) {
      return [];
    }
  }

  function recentDocsSave(entries) {
    storageSet(RECENT_DOCS_KEY, JSON.stringify(entries || []));
  }

  async function rememberRecentDoc(entry) {
    if (!entry || !entry.kind || !entry.fileName) return;
    const now = Date.now();
    const removed = [];
    const items = recentDocsLoad().filter((item) => {
      let keep = true;
      if (!item || item.kind !== entry.kind) return true;
      if (entry.kind === "sample") {
        keep = item.fileName !== entry.fileName;
      } else {
        keep = item.id !== entry.id && item.fileName !== entry.fileName;
      }
      if (!keep && item.handleKey) removed.push(item.handleKey);
      return keep;
    });

    const next = {
      id: entry.id || uniqueId(entry.kind),
      kind: entry.kind,
      fileName: entry.fileName,
      label:
        entry.label ||
        (entry.kind === "sample"
          ? sampleLabelFromFileName(entry.fileName)
          : stripFileExtension(entry.fileName) || entry.fileName),
      ts: now,
      handleKey: entry.handleKey || null,
      reopenable:
        entry.reopenable !== false &&
        (entry.kind === "sample" || !!entry.handleKey),
    };

    items.unshift(next);
    const truncated = items.slice(RECENT_DOC_LIMIT);
    for (const item of truncated) {
      if (item && item.handleKey) removed.push(item.handleKey);
    }
    recentDocsSave(items.slice(0, RECENT_DOC_LIMIT));
    await Promise.all(removed.map((key) => idbDeleteHandle(state, key)));
    if (entry.handleKey && entry.handle) {
      await idbPutHandle(state, entry.handleKey, entry.handle);
    }
  }

  async function pruneRecentDoc(entry) {
    if (!entry) return;
    const items = recentDocsLoad().filter((item) => item && item.id !== entry.id);
    recentDocsSave(items);
    if (entry.handleKey) {
      await idbDeleteHandle(state, entry.handleKey);
    }
  }

  function formatRecentTimestamp(ts) {
    if (!Number.isFinite(ts) || ts <= 0) return "";
    try {
      return new Date(ts).toLocaleString();
    } catch (_err) {
      return "";
    }
  }

  function closeRecentOverlay() {
    const overlay = el("recentOverlay");
    if (overlay) overlay.hidden = true;
  }

  function isRecentOverlayOpen() {
    const overlay = el("recentOverlay");
    return Boolean(overlay && !overlay.hidden);
  }

  function renderRecentList() {
    const list = el("recentList");
    if (!list) return;
    list.innerHTML = "";

    const entries = recentDocsLoad();
    if (entries.length === 0) {
      const empty = document.createElement("p");
      empty.className = "placeholder";
      empty.textContent = "No recent projects yet.";
      list.appendChild(empty);
      return;
    }

    entries.forEach((entry) => {
      const btn = document.createElement("button");
      btn.type = "button";
      btn.className = "sample-item";
      btn.disabled = entry.reopenable === false;

      const label = document.createElement("strong");
      label.textContent = entry.label || entry.fileName || "Untitled";
      btn.appendChild(label);

      const meta = document.createElement("small");
      meta.textContent = entry.fileName || "";
      btn.appendChild(meta);

      const metaRow = document.createElement("div");
      metaRow.className = "sample-meta-row";
      const tag = document.createElement("span");
      tag.className = "sample-tag";
      tag.textContent = entry.kind === "sample" ? "Sample" : "Local";
      metaRow.appendChild(tag);
      const stamp = document.createElement("small");
      stamp.textContent =
        formatRecentTimestamp(entry.ts) ||
        (entry.reopenable === false ? "Reopen unavailable" : "");
      metaRow.appendChild(stamp);
      btn.appendChild(metaRow);

      btn.addEventListener("click", () => {
        void openRecentDoc(entry);
      });
      list.appendChild(btn);
    });
  }

  function openRecentOverlay() {
    const overlay = el("recentOverlay");
    if (!overlay) return;
    renderRecentList();
    overlay.hidden = false;
  }

  function shouldShowWelcome() {
    return storageGet(WELCOME_SUPPRESS_KEY) !== "1";
  }

  function closeWelcomeOverlay() {
    const overlay = el("welcomeOverlay");
    if (!overlay) return;
    const check = el("welcomeSuppressCheck");
    if (check && check.checked) {
      storageSet(WELCOME_SUPPRESS_KEY, "1");
    }
    overlay.hidden = true;
  }

  function openWelcomeOverlay() {
    const overlay = el("welcomeOverlay");
    if (!overlay) return;
    const check = el("welcomeSuppressCheck");
    if (check) check.checked = false;
    renderWelcomeSampleList();
    const search = el("welcomeSampleSearch");
    if (search) {
      search.value = "";
      search.focus();
    }
    overlay.hidden = false;
  }

  function renderWelcomeSampleList() {
    const list = el("welcomeSampleList");
    const input = el("welcomeSampleSearch");
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
      const btn = buildSampleCard(file, () => {
        closeWelcomeOverlay();
        void loadSampleProject(file);
      });
      list.appendChild(btn);
    });
  }

  function wireWelcomeOverlay() {
    const overlay = el("welcomeOverlay");
    const btnCancel = el("btnWelcomeCancel");
    const btnOpen = el("btnWelcomeOpenFile");
    const search = el("welcomeSampleSearch");

    if (btnCancel) btnCancel.addEventListener("click", closeWelcomeOverlay);
    if (btnOpen) {
      btnOpen.addEventListener("click", () => {
        closeWelcomeOverlay();
        el("fileInput").click();
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
    const overlay = el("samplesOverlay");
    return Boolean(overlay && !overlay.hidden);
  }

  function closeSamplesOverlay() {
    const overlay = el("samplesOverlay");
    if (!overlay) return;
    overlay.hidden = true;
  }

  function renderSampleList() {
    const list = el("sampleList");
    const input = el("sampleSearch");
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
      const btn = buildSampleCard(file, () => {
        void loadSampleProject(file);
      });
      list.appendChild(btn);
    });
  }

  function openSamplesOverlay() {
    const overlay = el("samplesOverlay");
    const input = el("sampleSearch");
    if (!overlay) return;
    overlay.hidden = false;
    renderSampleList();
    if (input) input.focus();
  }

  async function fetchSampleProjectBytes(fileName) {
    try {
      const res = await fetch(resolveAssetUrl("samples", fileName));
      if (!res.ok) return null;
      const buf = await res.arrayBuffer();
      if (buf.byteLength > 0) return new Uint8Array(buf);
    } catch (_err) {}
    return null;
  }

  function setProjectBaseNameFromFileName(fileName) {
    state.projectBaseName = sanitizeFileBaseName(stripFileExtension(fileName));
  }

  function buildDownloadName(tag, ext) {
    const base = sanitizeFileBaseName(state.projectBaseName || "project");
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
        return { mode: "picker", handle };
      } catch (err) {
        if (err && err.name === "AbortError") return { mode: "canceled" };
        console.warn(
          "showSaveFilePicker failed, falling back to download link:",
          err,
        );
      }
    }
    downloadBytes(fileName, mimeType, bytes);
    return { mode: "download" };
  }

  function applyLoadedProjectBytes(bytes, fileName, statusPrefix) {
    const ok = window.knobman_loadFile(bytes);
    if (!ok) return false;
    setProjectBaseNameFromFileName(fileName);
    state.modifiedSinceSave = false;
    invalidateLayerPreviews();
    state.currentFrame = 0;
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
    await rememberRecentDoc({
      kind: "sample",
      fileName,
      label: sampleLabelFromFileName(fileName),
    });
    closeSamplesOverlay();
    closeRecentOverlay();
  }

  async function loadProjectFromHandle(handle) {
    if (!handle || typeof handle.getFile !== "function") return false;
    const file = await handle.getFile();
    const data = new Uint8Array(await file.arrayBuffer());
    if (!applyLoadedProjectBytes(data, file.name, "Loaded")) return false;
    const handleKey = uniqueId("handle");
    await rememberRecentDoc({
      id: handle.name || file.name,
      kind: "file",
      fileName: file.name,
      label: stripFileExtension(file.name),
      handleKey,
      handle,
    });
    closeRecentOverlay();
    return true;
  }

  async function openProjectWithPicker() {
    if (!supportsOpenPicker()) {
      el("fileInput").click();
      return;
    }
    try {
      const handles = await window.showOpenFilePicker({
        id: "knobman-open",
        multiple: false,
        types: [
          {
            description: "KnobMan Project",
            accept: { "application/octet-stream": [".knob"] },
          },
        ],
      });
      const handle = handles && handles[0];
      if (!handle) return;
      const ok = await loadProjectFromHandle(handle);
      if (!ok) setStatus("Failed to load selected file");
    } catch (err) {
      if (err && err.name === "AbortError") return;
      console.warn("showOpenFilePicker failed, falling back to file input:", err);
      el("fileInput").click();
    }
  }

  async function openRecentDoc(entry) {
    if (!entry) return;
    try {
      if (entry.kind === "sample") {
        await loadSampleProject(entry.fileName);
        return;
      }
      if (!entry.handleKey) {
        setStatus("Reopen unavailable for this project");
        return;
      }
      const handle = await idbGetHandle(state, entry.handleKey);
      if (!handle) {
        await pruneRecentDoc(entry);
        renderRecentList();
        setStatus("Recent file handle is no longer available");
        return;
      }
      if (!(await loadProjectFromHandle(handle))) {
        setStatus("Failed to reopen " + entry.fileName);
      }
    } catch (err) {
      console.warn("Failed to reopen recent project:", err);
      setStatus("Failed to reopen " + (entry.fileName || "recent project"));
    }
  }

  function onNew() {
    window.knobman_newDocument();
    state.projectBaseName = "project";
    state.modifiedSinceSave = false;
    invalidateLayerPreviews();
    state.currentFrame = 0;
    refreshFromDoc();
    ensureBuiltinTextures().then(() => refreshParamPanel());
    storageRemove(SESSION_KEY);
    storageRemove(LEGACY_SESSION_KEY);
    setStatus("New document");
  }

  async function onSave() {
    const data = window.knobman_saveFile();
    if (!data || data.length === 0) {
      setStatus("Save failed");
      return;
    }
    const fileName = buildDownloadName("project", "knob");
    const result = await saveBytes(
      data,
      fileName,
      "application/octet-stream",
      "knobman-save",
    );
    if (result.mode === "canceled") {
      setStatus("Save canceled");
      return;
    }
    if (result.mode === "picker" && result.handle) {
      const savedName = result.handle.name || fileName;
      setProjectBaseNameFromFileName(savedName);
      await rememberRecentDoc({
        kind: "file",
        fileName: savedName,
        label: stripFileExtension(savedName),
        handleKey: uniqueId("handle"),
        handle: result.handle,
      });
    }
    state.modifiedSinceSave = false;
    setStatus(
      result.mode === "picker" ? `Saved ${fileName}` : `Downloaded ${fileName}`,
    );
  }

  async function onExport() {
    const option = parseInt(el("prefExport").value, 10) || 0;
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
      const result = await saveBytes(
        out,
        fileName,
        "image/png",
        "knobman-export-png-strip",
      );
      if (result.mode === "canceled") {
        setStatus("PNG strip export canceled");
        return;
      }
      setStatus(
        result.mode === "picker"
          ? `Exported ${fileName}`
          : `Downloaded ${fileName}`,
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
      const result = await saveBytes(
        out,
        fileName,
        "application/zip",
        "knobman-export-frames-zip",
      );
      if (result.mode === "canceled") {
        setStatus("PNG frames export canceled");
        return;
      }
      setStatus(
        result.mode === "picker"
          ? `Exported ${fileName}`
          : `Downloaded ${fileName}`,
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
      const result = await saveBytes(
        out,
        fileName,
        "image/gif",
        "knobman-export-gif",
      );
      if (result.mode === "canceled") {
        setStatus("GIF export canceled");
        return;
      }
      setStatus(
        result.mode === "picker"
          ? `Exported ${fileName}`
          : `Downloaded ${fileName}`,
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
      const result = await saveBytes(
        out,
        fileName,
        "image/apng",
        "knobman-export-apng",
      );
      if (result.mode === "canceled") {
        setStatus("APNG export canceled");
        return;
      }
      setStatus(
        result.mode === "picker"
          ? `Exported ${fileName}`
          : `Downloaded ${fileName}`,
      );
      return;
    }
    setStatus("Unknown export option");
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
      void rememberRecentDoc({
        kind: "file",
        fileName: file.name,
        label: stripFileExtension(file.name),
        reopenable: false,
      });
      closeRecentOverlay();
    };
    reader.readAsArrayBuffer(file);
    e.target.value = "";
  }

  function saveSession() {
    if (!window.knobman_saveFile) return;
    try {
      const bytes = window.knobman_saveFile();
      if (!bytes || !bytes.length) return;
      const payload = {
        version: 2,
        ts: Date.now(),
        data: bytesToBase64(new Uint8Array(bytes)),
        currentFrame: state.currentFrame,
        zoomFactor: state.zoomFactor,
        selectedLayer: state.selectedLayer,
        selectedCurve: state.selectedCurve,
        prefAspectLock: state.prefAspectLock,
        projectBaseName: state.projectBaseName,
      };
      storageSet(SESSION_KEY, JSON.stringify(payload));
      storageRemove(LEGACY_SESSION_KEY);
    } catch (_err) {}
  }

  function applyRestoredSessionState(sessionState) {
    if (!sessionState || typeof sessionState !== "object") return;
    if (sessionState.projectBaseName) {
      state.projectBaseName = sanitizeFileBaseName(sessionState.projectBaseName);
    }
    state.zoomFactor = clampInt(Number(sessionState.zoomFactor) || state.zoomFactor, 1, 16);
    const zoomSelect = el("zoomSelect");
    if (zoomSelect) zoomSelect.value = String(state.zoomFactor);
    if (window.knobman_setZoom) window.knobman_setZoom(state.zoomFactor);

    state.currentFrame = Math.max(0, Number(sessionState.currentFrame) || 0);
    state.selectedCurve = clampInt(Number(sessionState.selectedCurve) || 1, 1, 8);
    state.prefAspectLock = Boolean(sessionState.prefAspectLock);
    const lock = el("prefLockAspect");
    if (lock) lock.checked = state.prefAspectLock;
    syncAspectRatioFromInputs();

    if (window.knobman_selectLayer) {
      state.selectedLayer = window.knobman_selectLayer(
        Number(sessionState.selectedLayer) || 0,
      );
    }
  }

  function restoreSessionPayload() {
    const raw = storageGet(SESSION_KEY);
    if (raw) {
      try {
        const payload = JSON.parse(raw);
        if (
          payload &&
          typeof payload.data === "string" &&
          payload.data.length > 0
        ) {
          return payload;
        }
        console.warn("Session payload missing or has invalid data field");
        storageRemove(SESSION_KEY);
      } catch (_err) {
        console.warn("Corrupt session removed from storage");
        storageRemove(SESSION_KEY);
      }
    }

    const legacy = storageGet(LEGACY_SESSION_KEY);
    if (!legacy) return null;
    return { version: 1, data: legacy };
  }

  function restoreSession() {
    if (!window.knobman_loadFile) return false;
    try {
      const payload = restoreSessionPayload();
      if (!payload || !payload.data) return false;
      const bytes = base64ToBytes(payload.data);
      if (!window.knobman_loadFile(bytes)) {
        storageRemove(SESSION_KEY);
        storageRemove(LEGACY_SESSION_KEY);
        return false;
      }
      applyRestoredSessionState(payload);
      return true;
    } catch (_err) {
      storageRemove(SESSION_KEY);
      storageRemove(LEGACY_SESSION_KEY);
      return false;
    }
  }

  return {
    closeRecentOverlay,
    closeSamplesOverlay,
    closeWelcomeOverlay,
    ensureBuiltinTextures,
    isRecentOverlayOpen,
    isSamplesOverlayOpen,
    onExport,
    onFileOpen,
    onNew,
    onSave,
    openProjectWithPicker,
    openRecentOverlay,
    openSamplesOverlay,
    openWelcomeOverlay,
    renderRecentList,
    renderSampleList,
    restoreSession,
    saveSession,
    shouldShowWelcome,
    wireWelcomeOverlay,
  };
}
