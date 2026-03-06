# KnobMan Go Rewrite — Implementation Plan

**Goal:** Complete rewrite of KnobMan in Go using `agg_go` as the rendering backend, compiled to WebAssembly and running fully in the browser. Feature parity with the original Java 1.3.3 release.

**Architecture:** Go/WASM binary for all rendering logic + state management, with an HTML/CSS/JS frontend shell for the UI (following the same pattern as `agg_go/cmd/wasm`). No server required — fully static, runs offline.

---

## TODO Checklist (Current Progress)

- [x] **Completed (condensed)**
    - Phases 0–1: repo skeleton + full model + `.knob` load/save + tests
    - Phase 5–7: WASM + web UI (layers/params), editors (curve/shape), preview tooling
    - Phase 4 (partial items): AnimStep, DynamicText, multi-frame image-strip support
    - Phase 8: full export pipeline complete (PNG strip/frames, GIF, APNG + download flow)

- [ ] **Still partial / pending**
    - Phase 2: primitive render parity against Java reference output
    - Phase 3: effect stack parity against Java reference output
    - Phase 4: complete render-side animation pipeline integration
    - Phase 9: app-level undo/redo integration, shortcuts, persistence, status metrics
    - Phase 10: regression tests, perf targets, deployment automation

---

## Dependency Reference

- **agg_go** at `../agg_go` (module path `agg_go`) — rendering backend
- **Go 1.24+** — WASM target: `GOOS=js GOARCH=wasm`
- **wasm_exec.js** — copied from Go toolchain (or TinyGo's variant)

---

## Phase 0 — Repository Reorganization

**Goal:** Archive the original Java source, create the new Go project skeleton, and establish the build system. No rendering code yet.
**Status:** [x] Completed

**Completed (condensed):**

- [x] 0.1 Legacy archived under `legacy/`
- [x] 0.2 Go module + folder layout (`cmd/wasm`, `internal/*`, `web/*`, `assets/*`) + `agg_go` wiring
- [x] 0.3 Build/test/dev loop via `justfile`
- [x] 0.4 Minimal WASM skeleton (canvas boot + agg_go proof)

**Where:** `cmd/wasm/`, `web/`, `justfile`, `go.mod`

**Deliverable:** `go build ./...` passes; WASM boots and renders a blank canvas.

---

## Phase 1 — Core Data Model & File Format

**Goal:** Go types for every parameter from the original, plus .knob file load/save. No rendering yet.
**Status:** [x] Completed

**Completed (condensed):**

- [x] 1.1 Params + serialization: `internal/model/params.go`
- [x] 1.2 AnimCurve model + eval: `internal/model/animcurve.go`
- [x] 1.3 Prefs: `internal/model/prefs.go`
- [x] 1.4 Primitives: `internal/model/primitive.go`
- [x] 1.5 Effects: `internal/model/effect.go`
- [x] 1.6 Layers: `internal/model/layer.go`
- [x] 1.7 Document root: `internal/model/document.go`
- [x] 1.8 `.knob` load/save + round-trip tests: `internal/fileio/`

---

## Phase 2 — Rendering Engine: Primitives

**Goal:** Implement all 16 primitive types as pure-Go software renderers using agg_go for path/shape work and direct per-pixel math for the lighting models. All rendering operates on an RGBA pixel buffer.
**Status:** [ ] Partial (baseline renderer + image strip/transparency semantics + shape parser improvements; Java parity pending)

**Completed (condensed):**

- [x] 2.1 Pixel buffer + oversampling/downsample pipeline
- [x] 2.2 Texture load/sample (built-ins, tiling, zoom)
- [x] 2.3 Primitive render pipeline (all primitive types wired; baseline behavior)
- [x] 2.4 Shared per-pixel lighting/blur utilities

**Pending:**

- [ ] Pixel parity against Java for all primitives (including edge-case semantics)
- [ ] Visual regression fixtures covering the sample set

**Where:** `internal/render/buffer.go`, `internal/render/texture.go`, `internal/render/primitive.go`, `internal/render/lighting.go`

**Deliverable:** Unit-testable primitive renderer with reference-image comparisons.

---

## Phase 3 — Rendering Engine: Effect Stack

**Goal:** Port the full `Eff.Apply()` pipeline.
**Status:** [ ] Partial (baseline transform/color/mask/shadow/composite/frame renderer implemented; parity pending)

**Completed (condensed):**

- [x] 3.1 Affine transform + bilinear sampling
- [x] 3.2 Color adjustments (alpha/brightness/contrast/sat/hue)
- [x] 3.3 Mask generation/application (incl. combine + frame mask)
- [x] 3.4 Shadow/highlight helpers
- [x] 3.5 Composite orchestration
- [x] 3.6 Frame render entrypoints (single/all frames) + oversampling path

**Pending:**

- [ ] Pixel parity vs Java across the full effect stack (including ordering/edge cases)
- [ ] Reference-image regression suite on the sample set

**Where:** `internal/render/transform.go`, `internal/render/coloradj.go`, `internal/render/mask.go`, `internal/render/shadow.go`, `internal/render/composite.go`, `internal/render/render.go`

**Deliverable:** `RenderAll` matches Java reference output for all samples.

---

## Phase 4 — Animation System

**Goal:** Full animation curve evaluation and frame-based parameter interpolation.
**Status:** [ ] Partial (curve model done; render-side animation pipeline pending)

**Completed (condensed):**

- [x] 4.1 Animatable parameter evaluation helper: `internal/render/animeval.go` (`EvalAnim`)
- [x] 4.2 AnimStep semantics (layer-local frame count)
- [x] 4.3 DynamicText frame counter substitution: `internal/render/dyntext.go`
- [x] 4.4 Image-strip frame extraction: `internal/render/imagestrip.go`

**Pending:**

- [ ] End-to-end animation integration through the full render/export pipeline (all animatable params)
- [ ] Verify semantics against Java across sample animations

**Where:** `internal/render/animeval.go`, `internal/render/dyntext.go`, `internal/render/imagestrip.go`

**Deliverable:** Animated exports advance frames correctly with interpolated parameters.

---

## Phase 5 — WASM Shell & Basic Web UI

**Goal:** A functioning web UI with canvas preview, layer list, and primitive parameter panel. Establishes the JS↔Go communication pattern.
**Status:** [x] Completed (basic WASM shell UI wired end-to-end)

**Completed (condensed):**

- [x] 5.1 WASM entrypoint + JS API bridge: `cmd/wasm/main.go`
- [x] 5.2 Three-column web UI shell + responsive layout: `web/index.html`, `web/style.css`
- [x] 5.3 Layer panel (list, selection, reorder, basic actions): `web/main.js`
- [x] 5.4 Primitive parameter panel wired to Go state: `web/main.js`
- [x] 5.5 Canvas preview with dirty rendering + frame scrubber: `web/main.js`

**Deliverable:** Load a `.knob`, edit layers/params, and see a live preview in the browser.

---

## Phase 6 — Complete Parameter Panels

**Goal:** Full fidelity of all parameter controls for all 16 primitive types and the full effect stack.
**Status:** [x] Completed

**Completed (condensed):**

- [x] 6.1 Primitive parameter panels for all primitive types (show/hide per type)
- [x] 6.2 Effect stack panel with animatable From/To + curve selection
- [x] 6.3 Prefs panel (size/frames/bg/export options) wired to document
- [x] 6.4 Texture UI (built-ins + upload + assignment)
- [x] 6.5 Color picking with alpha support

**Where:** UI wiring in `web/main.js`; model/behavior lives in `internal/model/*` and `cmd/wasm/main.go` bridge.

**Deliverable:** All primitive + effect parameters are editable from the web UI.

---

## Phase 7 — Advanced Editors

**Goal:** Visual editors for animation curves and shape paths, completing the feature set of `CurveEditor` and `ShapeEditor`.
**Status:** [x] Completed

**Completed (condensed):**

- [x] 7.1 Curve editor (canvas, drag keypoints, 8 tabs)
- [x] 7.2 Shape/path editor (M/L/C/Q/Z tools + serialization)
- [x] 7.3 Per-layer thumbnails (frame 0/1 previews)
- [x] 7.4 Floating/detached animated preview

**Where:** Frontend behavior in `web/main.js` (and related `web/*` assets); render support in `internal/render/*`.

**Deliverable:** Visual curve + shape authoring and improved preview tooling.

---

## Phase 8 — Export

**Goal:** All four export formats from the original.
**Status:** [x] Completed

### Completed (condensed)

- [x] 8.1 PNG strip export: `internal/export/pngstrip.go` (`ExportPNGStrip`)
- [x] 8.2 PNG frames export: `internal/export/pngframes.go` (`ExportPNGFrames`) (zipped on the JS side)
- [x] 8.3 Animated GIF export: `internal/export/animgif.go` (`ExportGIF`)
- [x] 8.4 APNG export: `internal/export/apng.go` (`ExportAPNG`)
- [x] 8.5 Download mechanism: unified JS save/download path + WASM byte transfer bridges

---

## Phase 9 — State Management, Undo/Redo & Polish

**Goal:** Complete the application with undo/redo, keyboard shortcuts, file management, and session persistence.
**Status:** [ ] Partial (history model done; file open/save wired; undo/redo/session integration pending)

### [x] 9.1 — Undo/Redo (`internal/model/history.go`)

```go
type History struct {
    Past   []*Document  // deep copies
    Future []*Document
    MaxLen int          // default 50
}

func (h *History) Push(doc *Document)
func (h *History) Undo() *Document
func (h *History) Redo() *Document
```

Every mutating operation (`setParam`, `addLayer`, etc.) pushes the current document state before modification. Deep copy is necessary since Documents contain slices.

### [ ] 9.2 — Keyboard Shortcuts (JS)

```
Ctrl+Z / Cmd+Z      Undo
Ctrl+Shift+Z / Cmd+Shift+Z  Redo
Ctrl+S / Cmd+S      Save
Ctrl+O / Cmd+O      Open
Ctrl+E              Export
Ctrl+D              Duplicate Layer
Delete              Delete Selected Layer
↑ / ↓              Move Layer Up/Down
```

### [x] 9.3 — File Open/Save (Browser)

- **Open**: `<input type="file" accept=".knob">` → `FileReader.readAsArrayBuffer()` → pass bytes to `knobman_loadFile(bytes)`
- **Save**: `knobman_saveFile()` returns `[]byte` → save via `showSaveFilePicker` when available, fallback to browser download
- **Export Downloads**: unified save path for PNG strip, PNG frames ZIP, GIF, and APNG (picker + fallback)

Auto-save to `localStorage` every 30 seconds (serialize current document as base64).

### [x] 9.4 — Sample Projects

Sample `.knob` files from `assets/samples/` are available via a "Samples" browser popup in the toolbar. Users can filter and load any sample project directly in-browser.

### [ ] 9.5 — Recent Files

Since there's no filesystem in the browser, "recent files" stores document snapshots in `localStorage` (last 10 documents). A "Recent" dropdown in the toolbar.

### [ ] 9.6 — Status Bar

A bottom status bar showing: current canvas size, frame count, render time, last save time, active layer name.

### [ ] 9.7 — Localization (Optional)

If desired, port the INI-based localization system. All UI strings defined in a `lang` map (default English). Additional language files loadable at runtime. Lower priority — English-only for initial release.

**Deliverable:** Full application with undo/redo, keyboard shortcuts, file open/save, sample library, and session persistence.

---

## Phase 10 — Testing, Performance & Deployment

**Goal:** Quality assurance, optimization, and production-ready deployment.
**Status:** [ ] Not started

### [ ] 10.1 — Visual Regression Tests

For each of the sample `.knob` files in `assets/samples/`:
1. Render frame 0 and frame N/2 with the Go engine
2. Compare against reference PNGs rendered by the Java original
3. Accept images within a per-pixel tolerance of ±2 (RGBA each channel)

Framework: Go test + `image` package comparison. Reference images committed to `tests/reference/`.

### [ ] 10.2 — Unit Tests

- `AnimCurve.Eval` against known values
- `DynamicText.SubstituteFrameCounters` against expected outputs
- `EvalAnim` for boundary conditions (frame 0, frame N-1)
- File format round-trip for all sample `.knob` files
- Per-pixel lighting math (PhongLighting, SphereNormal, etc.)
- Mask generation (all 4 mask types)

### [ ] 10.3 — Performance

Target: render a 64×64 knob with 10 layers at 4× oversampling in < 50ms in WASM.

Optimization strategies:
- Parallel layer rendering using goroutines (Web Workers do not apply to WASM goroutines, but Go's scheduler still benefits from multi-core via `GOMAXPROCS`)
- Cache primitive renders (only re-render a layer if its `Primitive` params changed)
- Incremental preview: render at 1× oversampling for interactive preview, then upgrade to full quality asynchronously
- agg_go's pixel formats are already optimized; avoid extra buffer copies

WASM binary size: aim for < 5MB. Use TinyGo if standard Go produces an unacceptably large binary (though TinyGo has limited goroutine support — evaluate tradeoffs).

### [ ] 10.4 — Deployment

Static files only:
```
web/
    index.html
    style.css
    main.js
    wasm_exec.js
    knobman.wasm     ← built artifact
    assets/          ← embedded in WASM via go:embed
```

Deploy to GitHub Pages (`gh-pages` branch) via GitHub Actions:
- On push to `master`: build WASM, copy to `web/`, publish `web/` to `gh-pages`

`Makefile` target: `make deploy`

---

## Appendix A — Mapping: Java Class → Go Package

| Java Class | Go Location |
|------------|-------------|
| `AnimCurve.java` | `internal/model/animcurve.go` |
| `Prefs.java` | `internal/model/prefs.go` |
| `Primitive.java` | `internal/model/primitive.go` + `internal/render/primitive_*.go` |
| `Eff.java` | `internal/model/effect.go` + `internal/render/composite.go` |
| `Layer.java` | `internal/model/layer.go` |
| `Control.java` | `cmd/wasm/state.go` (global document state) |
| `Render.java` | `internal/render/render.go` |
| `Bitmap.java` | `internal/render/buffer.go` |
| `Col.java` | `internal/render/coloradj.go` |
| `Tex.java` | `internal/render/texture.go` |
| `XYMatrix.java` | `internal/render/transform.go` |
| `DynamicText.java` | `internal/render/dyntext.go` |
| `IntelliAlpha.java` | `internal/render/intellialpha.go` |
| `ProfileReader/Writer` | `internal/fileio/knob.go` |
| `APng.java` | `internal/export/apng.go` |
| `AnimGif.java` | `internal/export/animgif.go` |
| `History.java` | `internal/model/history.go` |
| `GUIEditor.java` | `web/index.html` + `web/main.js` |
| `LayerPanel.java` | `web/main.js` (layer panel JS) |
| `PrimPanel.java` | `web/main.js` (prim params JS) |
| `EffPanel.java` | `web/main.js` (effect stack JS) |
| `CurveEditor.java` | `web/main.js` (curve editor canvas) |
| `ShapeEditor.java` | `web/main.js` (shape editor canvas) |
| `PrefsPanel.java` | `web/main.js` (prefs bar JS) |

---

## Appendix B — Phase Summary

| Phase | Status | Deliverable | Dependencies |
|-------|--------|-------------|--------------|
| **0** | [x] Completed | Go skeleton, legacy archived, WASM builds, blank canvas | — |
| **1** | [x] Completed | Full data model, file format load/save, round-trip tests | Phase 0 |
| **2** | [ ] Partial | All 16 primitives render correctly (native tests) | Phase 1 |
| **3** | [ ] Partial | Full effect stack (transform, color, masks, shadows) | Phase 2 |
| **4** | [ ] Partial | Animation interpolation, dynamic text, image strips | Phase 3 |
| **5** | [x] Completed | Web UI shell: canvas, layer list, basic param panel | Phase 4 |
| **6** | [x] Completed | All parameter controls in the web UI | Phase 5 |
| **7** | [x] Completed | Curve editor, shape editor, layer previews, floating preview | Phase 6 |
| **8** | [x] Completed | All 4 export formats (PNG strip, frames, GIF, APNG) | Phase 4 |
| **9** | [ ] Partial | Undo/redo, shortcuts, file open/save, samples, session | Phase 6, 8 |
| **10** | [ ] Not started | Visual regression tests, performance, GitHub Pages deploy | All |

---

## Appendix C — Key Design Decisions

1. **No CGO in the critical path.** agg_go's GSV font (pure Go stroke-vector font) is used for text rendering. FreeType2 (CGO) is excluded — it cannot compile to WASM. For font variety, additional embedded stroke/bitmap fonts can be added.

2. **Per-pixel lighting in Go vs. agg_go paths.** The Phong sphere, metal circle, and emboss effects require per-pixel math that doesn't map to AGG's path-based rendering model. These primitives render to a `PixBuf` directly in Go. agg_go is used for: circle/rect outlines, polygon fills, text rendering, shape paths, and all line primitives.

3. **Blur for shadows.** The Java shadow system uses a Gaussian blur of the layer's alpha channel. Go implements this as a separable 1D Gaussian blur (horizontal pass + vertical pass). agg_go's internal blur effects (`internal/effects/`) may be leveraged here.

4. **State management.** All document state lives in Go. JavaScript is a thin UI shell — it sends user events to Go and receives rendered pixel data + serialized state (layer list as JSON, parameter values as JSON) to update the DOM.

5. **WASM binary.** Standard `go build GOOS=js GOARCH=wasm` is the primary target. If binary size becomes a problem (> 8MB), evaluate TinyGo or a tree-shaking step. The agg_go dependency is significant but its `internal/` packages that are unused (e.g., platform backends, SDL2) will be eliminated by the linker.

6. **No localization in initial release.** The Java original has 7 languages, but since this is a ground-up rewrite of a niche tool, English-only for initial release simplifies development. Localization can be layered on later via a JS i18n system.

7. **Texture embedding.** All 18 built-in textures are embedded in the WASM binary via `//go:embed assets/textures/*`. This keeps deployment simple (single WASM file + HTML). User-uploaded textures are held in JS memory and passed to Go as `[]byte`.
