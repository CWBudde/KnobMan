# KnobMan Go Rewrite — Implementation Plan

**Goal:** Complete rewrite of KnobMan in Go using `agg_go` as the rendering backend, compiled to WebAssembly and running fully in the browser. Feature parity with the original Java 1.3.3 release.

**Architecture:** Go/WASM binary for all rendering logic + state management, with an HTML/CSS/JS frontend shell for the UI (following the same pattern as `agg_go/cmd/wasm`). No server required — fully static, runs offline.

---

## TODO Checklist (Current Progress)

- [x] **Done (condensed)**
    - Phase 0 (0.1–0.4): legacy archived, Go skeleton + build system, WASM boots with `agg_go`
    - Phase 1 (1.1–1.8): model + `.knob` load/save + round-trip tests
    - Phase 2: baseline primitive rendering pipeline implemented (parity pending)
    - Phase 5.2: basic three-column HTML/CSS UI shell
    - Phase 9.1: undo/redo history model (integration pending)

- [x] **Phase 3** — Effect stack renderer (`internal/render/*`) (baseline implementation done; parity pending)
- [x] **Phase 4.2** — AnimStep support
- [x] **Phase 4.3** — DynamicText implementation
- [x] **Phase 4.4** — Multi-frame image strip support

- [x] **Phase 5.1** — Full JS API surface from WASM (currently partial)
- [x] **Phase 5.3** — Layer panel behavior (list/selection/reorder) (currently stubbed)
- [x] **Phase 5.4** — Parameter panel behavior (basic primitive panel wired)
- [x] **Phase 5.5** — End-to-end live preview with document state wiring (currently partial)

- [ ] **Phase 6** — Complete primitive/effect parameter panels (partial: 6.1 done)
- [x] **Phase 6.1** — Primitive panel per type (JS + WASM primitive param bridge)
- [x] **Phase 6.2** — Effect stack panel (JS + WASM effect param bridge)
- [ ] **Phase 7** — Advanced editors (curve, shape, preview tools)
- [ ] **Phase 8** — Export pipeline (PNG strip/frames, GIF, APNG)

- [ ] **Phase 9.1** — Undo/redo integrated into app mutations (history model exists)
- [ ] **Phase 9.2** — Complete shortcut set (currently partial)
- [x] **Phase 9.3** — Browser file open/save fully wired (currently stubbed)
- [ ] **Phase 9.4** — Sample project browser
- [ ] **Phase 9.5** — Recent files/session persistence
- [ ] **Phase 9.6** — Full status bar metrics
- [ ] **Phase 9.7** — Localization (optional)

- [ ] **Phase 10.1** — Visual regression suite
- [ ] **Phase 10.2** — Full unit test matrix (currently partial: model + fileio + render baseline + phase3 pipeline)
- [ ] **Phase 10.3** — Performance tuning targets
- [ ] **Phase 10.4** — Deployment automation

---

## Dependency Reference

- **agg_go** at `../agg_go` (module path `agg_go`) — rendering backend
- **Go 1.24+** — WASM target: `GOOS=js GOARCH=wasm`
- **wasm_exec.js** — copied from Go toolchain (or TinyGo's variant)

---

## Phase 0 — Repository Reorganization

**Goal:** Archive the original Java source, create the new Go project skeleton, and establish the build system. No rendering code yet.
**Status:** [x] Completed

### [x] 0.1 — Move Legacy Code

Done — legacy Java sources/resources archived under `legacy/`.

### [x] 0.2 — Go Module Setup

Done — Go module + folder layout created (`cmd/wasm`, `internal/*`, `web/*`, `assets/*`), including `agg_go` replace wiring.

### [x] 0.3 — Build System

Done — build/test/dev loop scripted via `justfile`.

### [x] 0.4 — Minimal Skeleton

`cmd/wasm/main.go` that compiles to WASM, opens a blank canvas, and proves the agg_go integration works. `web/index.html` with just the canvas and WASM loader.

**Deliverable:** `go build ./...` passes, WASM builds, blank white canvas appears in browser.

---

## Phase 1 — Core Data Model & File Format

**Goal:** Go types for every parameter from the original, plus .knob file load/save. No rendering yet.
**Status:** [x] Completed

### [x] 1.1 — Parameter Types (`internal/model/params.go`)

Done — parameter types implemented with JSON/INI serialization.

### [x] 1.2 — AnimCurve (`internal/model/animcurve.go`)

- Up to 12 keypoints: `[12]AnimPoint` where `AnimPoint{Time, Level float64}` (0–100 range)
- `Eval(t float64) float64` — piecewise-linear interpolation
- `EvalStepped(t float64, steps int) float64` — quantized version
- 8 global curves per document, indexed 0–7

### [x] 1.3 — Prefs (`internal/model/prefs.go`)

Done — prefs model implemented (canvas size, oversampling, frames, export options).

### [x] 1.4 — Primitive (`internal/model/primitive.go`)

Done — primitive model implemented (all primitive types + all parameters).

### [x] 1.5 — Effect Stack (`internal/model/effect.go`)

Done — effect stack model implemented (animatable from/to parameters + curve selectors; Java field parity).

### [x] 1.6 — Layer (`internal/model/layer.go`)

Done — layer model implemented.

### [x] 1.7 — Document (`internal/model/document.go`)

Done — document model implemented (prefs + curves + layers + texture entries).

### [x] 1.8 — File Format (`internal/fileio/`)

Done — `.knob` load/save implemented with round-trip tests.

---

## Phase 2 — Rendering Engine: Primitives

**Goal:** Implement all 16 primitive types as pure-Go software renderers using agg_go for path/shape work and direct per-pixel math for the lighting models. All rendering operates on an RGBA pixel buffer.
**Status:** [ ] Partial (baseline renderer + image strip/transparency semantics + shape parser improvements; Java parity pending)

### [x] 2.1 — Buffer Management (`internal/render/buffer.go`)

Done — pixel buffer abstraction + oversampling/downsample pipeline.

### [x] 2.2 — Texture System (`internal/render/texture.go`)

Done — texture loading/sampling (built-ins embedded; tiling + zoom).

### [x] 2.3 — Primitive Renderer (`internal/render/primitive.go`)

Done — primitive rendering pipeline implemented (all primitive types wired; baseline behavior present, parity still tracked under Phase 2).

### [x] 2.4 — Per-Pixel Math Utilities (`internal/render/lighting.go`)

Done — shared per-pixel lighting + blur utilities ported.

**Deliverable:** Unit-testable renderer. Each primitive type has a test that renders to a PNG and compares against a reference image.

---

## Phase 3 — Rendering Engine: Effect Stack

**Goal:** Port the full `Eff.Apply()` pipeline.
**Status:** [ ] Partial (baseline transform/color/mask/shadow/composite/frame renderer implemented; parity pending)

### [x] 3.1 — Affine Transform with Bilinear Sampling (`internal/render/transform.go`)

Done — affine transform + bilinear sampling.

### [x] 3.2 — Color Adjustments (`internal/render/coloradj.go`)

Done — alpha/brightness/contrast/saturation/hue adjustments.

### [x] 3.3 — Mask System (`internal/render/mask.go`)

Done — mask generation/application (including combining masks + frame mask support).

### [x] 3.4 — Shadow and Highlight (`internal/render/shadow.go`)

Done — shadow/highlight rendering helpers.

### [x] 3.5 — Compositing (`internal/render/composite.go`)

Done — compositing primitives and baseline effect-application orchestration.

### [x] 3.6 — Frame Renderer (`internal/render/render.go`)

Done — frame rendering entrypoints (single frame + all frames), including oversampling path.

**Deliverable:** Given a loaded `.knob` document, `RenderAll` produces pixel-accurate output matching the Java reference output for all sample files.

---

## Phase 4 — Animation System

**Goal:** Full animation curve evaluation and frame-based parameter interpolation.
**Status:** [ ] Partial (curve model done; render-side animation pipeline pending)

### [x] 4.1 — Parameter Evaluation (`internal/render/animeval.go`)

```go
// Evaluate an animatable from/to parameter pair at the given frame.
// frameFrac is in [0.0, 1.0] representing position in animation.
// The anim SelectParam selects which of the 8 AnimCurves to use (0=linear).
func EvalAnim(from, to float64, animCurveIdx int, curves [8]*model.AnimCurve, frameFrac float64) float64
```

The Java convention: `frameFrac = frame / (totalFrames-1)`. The AnimCurve maps 0–100 time to 0–100 level, then the level is remapped to `[from, to]`.

### [x] 4.2 — AnimStep

When `Effect.AnimStep > 0`, the layer uses an independent frame count (`Effect.AnimStep` instead of `doc.Prefs.ExportFrames`) for its animation timeline. This allows a layer to complete its animation in a sub-range.

### [x] 4.3 — DynamicText (`internal/render/dyntext.go`)

Port `DynamicText.java` and `TextCounter.java`:

```go
// Substitute frame counter patterns in text.
// Format: "(start:end)" → left-zero-padded frame number in [start,end]
// Multiple patterns can appear in the same string.
func SubstituteFrameCounters(text string, frame, totalFrames int) string
```

Example: `"Frame (1:99)"` on frame 5 of 31 → `"Frame 05"` (maps 0..30 → 01..99 proportionally).

### [x] 4.4 — Multi-Frame Image Strip (`internal/render/imagestrip.go`)

For `PrimImage` with `NumFrame > 1`, the external image file is a strip of frames. Load the strip and extract the correct sub-image for the current render frame:

```go
func ExtractFrame(strip *PixBuf, numFrames, frame, totalFrames int) *PixBuf
```

**Deliverable:** Animated export (GIF/APNG) correctly moves through frames with all animatable parameters interpolated.

---

## Phase 5 — WASM Shell & Basic Web UI

**Goal:** A functioning web UI with canvas preview, layer list, and primitive parameter panel. Establishes the JS↔Go communication pattern.
**Status:** [x] Completed (basic WASM shell UI wired end-to-end)

### [x] 5.1 — WASM Entry Point (`cmd/wasm/main.go`)

```go
//go:build js && wasm

package main

import "syscall/js"

func main() {
    // Expose all Go functions to JS
    js.Global().Set("knobman_init",         js.FuncOf(init_))
    js.Global().Set("knobman_render",       js.FuncOf(render))
    js.Global().Set("knobman_getLayerList", js.FuncOf(getLayerList))
    js.Global().Set("knobman_addLayer",     js.FuncOf(addLayer))
    js.Global().Set("knobman_deleteLayer",  js.FuncOf(deleteLayer))
    js.Global().Set("knobman_moveLayer",    js.FuncOf(moveLayer))
    js.Global().Set("knobman_setLayerVisible", js.FuncOf(setLayerVisible))
    js.Global().Set("knobman_setLayerSolo",    js.FuncOf(setLayerSolo))
    js.Global().Set("knobman_duplicateLayer",  js.FuncOf(duplicateLayer))
    js.Global().Set("knobman_selectLayer",     js.FuncOf(selectLayer))
    js.Global().Set("knobman_setParam",        js.FuncOf(setParam))
    js.Global().Set("knobman_getParam",        js.FuncOf(getParam))
    js.Global().Set("knobman_setPrefs",        js.FuncOf(setPrefs))
    js.Global().Set("knobman_getPrefs",        js.FuncOf(getPrefs))
    js.Global().Set("knobman_loadFile",        js.FuncOf(loadFile))
    js.Global().Set("knobman_saveFile",        js.FuncOf(saveFile))
    js.Global().Set("knobman_exportPNG",       js.FuncOf(exportPNG))
    js.Global().Set("knobman_exportGIF",       js.FuncOf(exportGIF))
    js.Global().Set("knobman_exportAPNG",      js.FuncOf(exportAPNG))
    js.Global().Set("knobman_setTexture",      js.FuncOf(setTexture))
    js.Global().Set("knobman_setAnimCurve",    js.FuncOf(setAnimCurve))
    js.Global().Set("knobman_getAnimCurve",    js.FuncOf(getAnimCurve))
    js.Global().Set("knobman_setPreviewFrame", js.FuncOf(setPreviewFrame))
    js.Global().Set("knobman_undo",            js.FuncOf(undo_))
    js.Global().Set("knobman_redo",            js.FuncOf(redo_))
    select {}
}
```

Go-side state: a single global `*model.Document` + `[]*render.Texture` + undo stack.

The `render` function renders the current document at the current preview frame and copies the pixel buffer to a JS `Uint8ClampedArray` for drawing to `<canvas>`.

### [x] 5.2 — HTML/CSS Layout (`web/index.html`, `web/style.css`)

Three-column layout mirroring the Java Swing UI:

```
┌─────────────────────────────────────────────────┐
│  Toolbar: New | Open | Save | Export | Undo/Redo │
├──────────┬─────────────────────┬─────────────────┤
│  Layer   │   Canvas Preview    │  Parameters     │
│  Panel   │                     │  Panel          │
│          │   [canvas element]  │                 │
│  ┌─┐Name │                     │  [Prim Type]    │
│  ├─┤Name │   Frame scrubber    │  [Prim Params]  │
│  ├─┤Name │   ← ──────── →      │  ─────────────  │
│  └─┘     │                     │  [Effect Params]│
│  [+][-]↑↓│                     │                 │
├──────────┴─────────────────────┴─────────────────┤
│  Prefs: Size | Frames | BgColor | Export Options  │
└─────────────────────────────────────────────────-─┘
```

Responsive: on narrow viewports, panels stack vertically.

### [x] 5.3 — Layer Panel (JS)

- Renders the layer list as a `<ul>` with drag-to-reorder
- Each row: visibility toggle (eye icon), solo toggle, layer name (dblclick to rename), layer type indicator
- Buttons: Add, Delete, Move Up, Move Down, Duplicate
- Click to select — triggers parameter panel refresh

### [x] 5.4 — Basic Parameter Panel (JS)

- Primitive type selector (dropdown with 16 options)
- On type change: show/hide relevant parameter fields
- Each parameter rendered as: label + appropriate input (range slider, number input, color picker, text input, file picker, checkbox, select)
- All changes call `knobman_setParam(layerIdx, paramKey, value)` → triggers re-render

### [x] 5.5 — Canvas & Preview

- HTML5 `<canvas>` element
- `requestAnimationFrame`-based render loop (only re-renders when dirty)
- Frame scrubber: range input 0..N → calls `knobman_setPreviewFrame(n)`
- Zoom: pixel-doubled preview for small canvases (e.g. 64×64 at 4× zoom)
- Background checkerboard pattern shows transparency

**Deliverable:** Can load a `.knob` file, see the layers, select them, and see a rendered preview updating in real time as parameters change.

---

## Phase 6 — Complete Parameter Panels

**Goal:** Full fidelity of all parameter controls for all 16 primitive types and the full effect stack.
**Status:** [ ] Partial (6.1/6.2 done; 6.3+ pending)

### [x] 6.1 — Primitive Panel per Type

Each PrimitiveType maps to a set of visible parameters. The JS panel shows/hides parameter groups based on selected type:

| Type | Visible Parameters |
|------|--------------------|
| None | (none) |
| Image | File picker, AutoFit, IntelliAlpha, NumFrame |
| Circle | Color, Width, Round, Diffuse, Emboss, EmbossDiffuse, Ambient, LightDir, Specular, SpecularWidth, TextureDepth, TextureZoom, TextureFile |
| CircleFill | Color, Aspect, Diffuse, Ambient, LightDir, Specular, SpecularWidth, TextureDepth, TextureZoom, TextureFile |
| MetalCircle | Color, Aspect, Diffuse, Ambient, LightDir, Specular, SpecularWidth, TextureDepth, TextureZoom, TextureFile |
| WaveCircle | Color, Width, Step, Length, Diffuse, Emboss, ... |
| GradientCircle | Color, Aspect, Diffuse, Step, AngleStep |
| Rect | Color, Width, Round, Length, Aspect, Diffuse, Emboss, ... |
| RectFill | Color, Round, Aspect, Diffuse, Emboss, EmbossDiffuse, Ambient, LightDir, Specular, SpecularWidth, TextureDepth, TextureZoom, TextureFile |
| Triangle | Color, Width, Round, Length, Diffuse, Fill |
| Line | Color, Width, Length, LightDir (angle) |
| RadiateLine | Color, Width, Length, AngleStep, Step |
| HLines/VLines | Color, Width, Step |
| Text | Color, Text, Font, TextAlign, FrameAlign, Bold, Italic |
| Shape | Color, Shape (path editor), Fill, Round, Diffuse |

### [x] 6.2 — Effect Stack Panel

Collapsible sections (mirroring Java's EffPanel):
- **Transform** — ZoomX, ZoomY, OffsetX, OffsetY, Angle, Center, KeepDir, Unfold, AnimStep, AntiAlias
- **Color** — Alpha, Brightness, Contrast, Saturation, Hue
- **Mask 1** — Enable, Type, Start, Stop, Gradient, GradDir
- **Mask 2** — Enable, Type, Start, Stop, Gradient, GradDir, Operation
- **Frame Mask** — Enable, Type, Start/Stop or Bitmask
- **Specular Highlight** — Enable, LightDir, Density
- **Drop Shadow** — Enable, LightDir, Offset, Density, Diffuse, Grad, Type
- **Inner Shadow** — Enable, LightDir, Offset, Density, Diffuse
- **Emboss** — Enable, LightDir, Offset, Density

Each animatable parameter shows:
- A From/To range (two inputs or a dual-handle range slider)
- An AnimCurve selector (dropdown: Linear, Curve 1..8)

### [ ] 6.3 — Prefs Panel

Bottom bar with: Width × Height (with lock-aspect option), Oversampling selector, PreviewFrames, ExportFrames, Background Color, Strip Orientation (H/V), Export Format, Duration (ms), Loop, BiDir.

Canvas resize: triggers full document re-render.

### [ ] 6.4 — Texture Panel

A dropdown or palette showing the 18 built-in textures with thumbnail previews. Plus an "Add Texture" button that opens a file picker (PNG/JPG/BMP). Loaded textures are stored in the document and embedded in saved `.knob` files or referenced by path (to be decided).

### [ ] 6.5 — Color Picker

Custom `<canvas>`-based HSV color picker (or use the browser's native `<input type="color">` with an additional alpha slider). Used for all `ColorParam` inputs.

**Deliverable:** All parameters from all 16 primitive types and the full effect stack are editable via the web UI.

---

## Phase 7 — Advanced Editors

**Goal:** Visual editors for animation curves and shape paths, completing the feature set of `CurveEditor` and `ShapeEditor`.
**Status:** [ ] Not started

### [ ] 7.1 — Animation Curve Editor

A `<canvas>`-based interactive editor:
- Displays the piecewise-linear curve as a polyline in a 100×100 normalized space
- Up to 12 draggable keypoints
- Click to add a keypoint, right-click (or button) to delete
- Displays current frame position as a vertical line
- Shows evaluated value as a horizontal line
- Applied to any Effect parameter that has an `AnimCurve` selector
- 8 global curves displayed as tabs

This mirrors `CurveEditor.java` in the original.

### [ ] 7.2 — Shape Path Editor

For the `PrimShape` shape string (SVG-like M/L/C/Q/Z path):

- `<canvas>`-based interactive editor overlaid on the current canvas preview
- Shows control points (Move, Line, Cubic Bezier, Quadratic Bezier) as draggable handles
- Bezier control arms shown as dashed lines
- Toolbar: M (move), L (line), C (cubic), Q (quadratic), Z (close), delete point
- Serializes to the SVG path string format

Mirrors `ShapeEditor.java`.

### [ ] 7.3 — Layer Bitmap Preview

Two small preview thumbnails per layer (frame 0 and frame 1 of that layer's primitive, before effects), rendered asynchronously. Shown in the layer panel row. Mirrors `BitmapView.java`.

This uses a secondary render pass with `RenderFrame` called for just that layer in isolation.

### [ ] 7.4 — Preview Window / Floating Preview

An optional detachable preview panel (or popup window via `window.open()`) that shows the rendered knob at its actual export size, animating through frames. Mirrors `TransparentIcon.java`.

**Deliverable:** Users can design custom shapes and fully animate all parameters graphically.

---

## Phase 8 — Export

**Goal:** All four export formats from the original.
**Status:** [ ] Not started

### [ ] 8.1 — PNG Strip (`internal/export/pngstrip.go`)

```go
// Render all frames and stitch into a vertical or horizontal PNG strip.
func ExportPNGStrip(doc *model.Document, textures []*render.Texture, horizontal bool) ([]byte, error)
```

Encodes using Go's `image/png`. Returns the PNG bytes for browser download.

### [ ] 8.2 — Individual PNG Frames (`internal/export/pngframes.go`)

```go
// Render all frames, return as a slice of PNG []byte (one per frame).
// The JS layer zips these for download.
func ExportPNGFrames(doc *model.Document, textures []*render.Texture) ([][]byte, error)
```

Since WASM can't write to the filesystem, the JS layer receives all PNG bytes and packages them as a zip for download (using JSZip or a Go zip implementation).

### [ ] 8.3 — Animated GIF (`internal/export/animgif.go`)

Port or wrap the Java `AnimGif.java` GIF encoder logic in pure Go (no external deps). Or use a pure-Go GIF encoder library.

```go
func ExportGIF(doc *model.Document, textures []*render.Texture) ([]byte, error)
```

### [ ] 8.4 — APNG (`internal/export/apng.go`)

Port the Java `APng.java` APNG encoder in pure Go. APNG is a PNG extension — each frame is a PNG chunk sequence.

```go
func ExportAPNG(doc *model.Document, textures []*render.Texture) ([]byte, error)
```

Respects: `Duration`, `Loop`, `BiDir` (ping-pong: append reversed frames).

### [ ] 8.5 — Download Mechanism (JS)

```js
function downloadBytes(filename, mimeType, bytes) {
    const blob = new Blob([bytes], { type: mimeType });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url; a.download = filename; a.click();
    URL.revokeObjectURL(url);
}
```

Go passes `[]byte` to JS via `js.CopyBytesToJS` into a preallocated `Uint8Array`.

**Deliverable:** All four export formats work from the browser, producing files that match the Java output.

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
- **Save**: `knobman_saveFile()` returns `[]byte` → trigger browser download of `.knob` file

Auto-save to `localStorage` every 30 seconds (serialize current document as base64).

### [ ] 9.4 — Sample Projects

Embed sample `.knob` files from `assets/samples/`. A "Samples" menu or gallery popup lets users load any sample as a starting point. Samples are embedded via `//go:embed assets/samples/*.knob`.

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

For each of the 35 sample `.knob` files in `assets/samples/`:
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
| **6** | [ ] Partial | All parameter controls in the web UI | Phase 5 |
| **7** | [ ] Not started | Curve editor, shape editor, layer previews | Phase 6 |
| **8** | [ ] Not started | All 4 export formats (PNG strip, frames, GIF, APNG) | Phase 4 |
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
