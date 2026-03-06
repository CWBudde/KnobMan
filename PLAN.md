# KnobMan Go Rewrite — Implementation Plan

**Goal:** Complete rewrite of KnobMan in Go using `agg_go` as the rendering backend, compiled to WebAssembly and running fully in the browser. Feature parity with the original Java 1.3.3 release.

**Architecture:** Go/WASM binary for all rendering logic + state management, with an HTML/CSS/JS frontend shell for the UI (following the same pattern as `agg_go/cmd/wasm`). No server required — fully static, runs offline.

---

## TODO Checklist (Current Progress)

- [x] **Phase 0.1** — Legacy code moved to `legacy/` (`src`, `res`, `pom.xml`, jar archive)
- [x] **Phase 0.2** — Go module skeleton created (`cmd/wasm`, `internal/*`, `web/*`, `assets/*`)
- [x] **Phase 0.3** — Build system created (`justfile` with `build-wasm`, `serve`, `test`, `clean`)
- [x] **Phase 0.4** — WASM skeleton rendering through `agg_go` implemented

- [x] **Phase 1.1** — Parameter types implemented in `internal/model/params.go`
- [x] **Phase 1.2** — AnimCurve interpolation implemented in `internal/model/animcurve.go`
- [x] **Phase 1.3** — Prefs model implemented in `internal/model/prefs.go`
- [x] **Phase 1.4** — Primitive model implemented in `internal/model/primitive.go`
- [x] **Phase 1.5** — Effect stack model implemented in `internal/model/effect.go`
- [x] **Phase 1.6** — Layer model implemented in `internal/model/layer.go`
- [x] **Phase 1.7** — Document model implemented in `internal/model/document.go`
- [x] **Phase 1.8** — `.knob` load/save + sample round-trip tests implemented in `internal/fileio/`

- [x] **Phase 2** — Rendering engine primitives (`internal/render/*`) (baseline implementation done; parity pending)
- [ ] **Phase 3** — Effect stack renderer (`internal/render/*`)
- [ ] **Phase 4.2** — AnimStep support
- [ ] **Phase 4.3** — DynamicText implementation
- [ ] **Phase 4.4** — Multi-frame image strip support

- [ ] **Phase 5.1** — Full JS API surface from WASM (currently partial)
- [x] **Phase 5.2** — Basic three-column HTML/CSS UI shell created
- [ ] **Phase 5.3** — Layer panel behavior (list/selection/reorder) (currently stubbed)
- [ ] **Phase 5.4** — Parameter panel behavior (currently placeholder/stubbed)
- [ ] **Phase 5.5** — End-to-end live preview with document state wiring (currently partial)

- [ ] **Phase 6** — Complete primitive/effect parameter panels
- [ ] **Phase 7** — Advanced editors (curve, shape, preview tools)
- [ ] **Phase 8** — Export pipeline (PNG strip/frames, GIF, APNG)

- [ ] **Phase 9.1** — Undo/redo integrated into app mutations (history model exists)
- [ ] **Phase 9.2** — Complete shortcut set (currently partial)
- [ ] **Phase 9.3** — Browser file open/save fully wired (currently stubbed)
- [ ] **Phase 9.4** — Sample project browser
- [ ] **Phase 9.5** — Recent files/session persistence
- [ ] **Phase 9.6** — Full status bar metrics
- [ ] **Phase 9.7** — Localization (optional)

- [ ] **Phase 10.1** — Visual regression suite
- [ ] **Phase 10.2** — Full unit test matrix (currently partial: model + fileio + render baseline)
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

```
legacy/
  src/           ← move src/main/java/* here
  res/           ← move res/* here (samples, textures, lang, icons)
  pom.xml
  JKnobMan133-jar.zip
```

Keep at root: `LICENSE`, `README.md`, `PLAN.md`, `.gitignore`.

### [x] 0.2 — Go Module Setup

```
go.mod           module knobman
cmd/
  wasm/
    main.go      ← WASM entry point (js && wasm build tag)
    main_stub.go ← non-WASM stub for native builds/tests
internal/
  model/         ← data model (Phase 1)
  render/        ← rendering engine (Phases 2–4)
  fileio/        ← .knob file format (Phase 1)
  export/        ← PNG/GIF/APNG export (Phase 8)
web/
  index.html
  style.css
  main.js
  wasm_exec.js
assets/
  textures/      ← 18 built-in texture images (copied from legacy/res/Texture/)
  samples/       ← sample .knob files (copied from legacy/res/Samples/)
  icons/         ← UI icons (copied from legacy/res/Resource/Images/)
```

Module declaration:
```go
module knobman

go 1.24

require agg_go v0.0.0
replace agg_go => ../agg_go
```

### [x] 0.3 — Build System

`Makefile` (or `Justfile`) with targets:
- `build-wasm` — `GOOS=js GOARCH=wasm go build -o web/knobman.wasm ./cmd/wasm/`
- `serve` — local HTTP server for the web directory
- `test` — run all Go tests (native, no WASM needed)
- `clean`

### [x] 0.4 — Minimal Skeleton

`cmd/wasm/main.go` that compiles to WASM, opens a blank canvas, and proves the agg_go integration works. `web/index.html` with just the canvas and WASM loader.

**Deliverable:** `go build ./...` passes, WASM builds, blank white canvas appears in browser.

---

## Phase 1 — Core Data Model & File Format

**Goal:** Go types for every parameter from the original, plus .knob file load/save. No rendering yet.
**Status:** [x] Completed

### [x] 1.1 — Parameter Types (`internal/model/params.go`)

```go
// Mirrors Java Param* types
type FloatParam  struct { Val float64 }
type IntParam    struct { Val int }
type BoolParam   struct { Val bool }
type SelectParam struct { Val int }
type StringParam struct { Val string }
type ColorParam  struct { Val color.RGBA }
```

All parameter types support JSON and INI serialization.

### [x] 1.2 — AnimCurve (`internal/model/animcurve.go`)

- Up to 12 keypoints: `[12]AnimPoint` where `AnimPoint{Time, Level float64}` (0–100 range)
- `Eval(t float64) float64` — piecewise-linear interpolation
- `EvalStepped(t float64, steps int) float64` — quantized version
- 8 global curves per document, indexed 0–7

### [x] 1.3 — Prefs (`internal/model/prefs.go`)

```go
type Prefs struct {
    Width, Height   int       // canvas size (default 64×64)
    Oversampling    int       // 0=1x 1=2x 2=4x 3=8x
    PreviewFrames   int       // default 5
    ExportFrames    int       // default 31
    BgColor         color.RGBA
    AlignHorz       bool      // strip orientation
    ExportOption    int       // 0=strip 1=individual 2=gif 3=apng
    Duration        int       // ms per frame (APNG/GIF)
    Loop            bool
    BiDir           bool      // ping-pong animation
}
```

### [x] 1.4 — Primitive (`internal/model/primitive.go`)

One struct holding ALL parameters for all 16 primitive types:

```go
type PrimitiveType int
const (
    PrimNone PrimitiveType = iota
    PrimImage
    PrimCircle
    PrimCircleFill
    PrimMetalCircle
    PrimWaveCircle
    PrimGradientCircle
    PrimRect
    PrimRectFill
    PrimTriangle
    PrimLine
    PrimRadiateLine
    PrimHLines
    PrimVLines
    PrimText
    PrimShape
)

type Primitive struct {
    Type           SelectParam
    Color          ColorParam
    TextureFile    SelectParam
    Transparent    SelectParam
    Font           SelectParam
    TextAlign      SelectParam
    FrameAlign     SelectParam
    Aspect         FloatParam
    Round          FloatParam
    Width          FloatParam
    Length         FloatParam
    Step           FloatParam
    AngleStep      FloatParam
    Emboss         FloatParam
    EmbossDiffuse  FloatParam
    Ambient        FloatParam
    LightDir       FloatParam
    Specular       FloatParam
    SpecularWidth  FloatParam
    TextureDepth   FloatParam
    TextureZoom    FloatParam
    Diffuse        FloatParam
    File           StringParam  // image file path
    Text           StringParam  // dynamic text content
    Shape          StringParam  // SVG-like path data
    AutoFit        BoolParam
    Bold           BoolParam
    Italic         BoolParam
    Fill           BoolParam
    IntelliAlpha   SelectParam
    NumFrame       IntParam
}
```

### [x] 1.5 — Effect Stack (`internal/model/effect.go`)

All animatable fields exist as From/To pairs (e.g. `ZoomXFrom`, `ZoomXTo`) plus an anim curve index. This matches the Java Eff struct exactly.

```go
type Effect struct {
    // Transform
    ZoomXFrom, ZoomXTo   FloatParam; ZoomXAnim   SelectParam
    ZoomYFrom, ZoomYTo   FloatParam; ZoomYAnim   SelectParam
    ZoomXYSeparate        BoolParam
    OffsetXFrom, OffsetXTo FloatParam; OffsetXAnim SelectParam
    OffsetYFrom, OffsetYTo FloatParam; OffsetYAnim SelectParam
    AngleFrom, AngleTo    FloatParam; AngleAnim   SelectParam
    CenterX, CenterY      FloatParam
    KeepDir               BoolParam
    AntiAlias             BoolParam
    Unfold                SelectParam
    AnimStep              IntParam

    // Color
    AlphaFrom, AlphaTo       FloatParam; AlphaAnim      SelectParam
    BrightnessFrom, BrightnessTo FloatParam; BrightnessAnim SelectParam
    ContrastFrom, ContrastTo  FloatParam; ContrastAnim   SelectParam
    SaturationFrom, SaturationTo FloatParam; SaturationAnim SelectParam
    HueFrom, HueTo            FloatParam; HueAnim        SelectParam

    // Masks 1 & 2
    Mask1Enable, Mask2Enable BoolParam
    Mask1Type, Mask2Type     SelectParam // 0=Rotation 1=Radial 2=Horizontal 3=Vertical
    Mask1Grad, Mask2Grad     FloatParam
    Mask1GradDir, Mask2GradDir FloatParam
    Mask1StartFrom, Mask1StartTo FloatParam; Mask1StartAnim SelectParam
    Mask1StopFrom, Mask1StopTo  FloatParam; Mask1StopAnim  SelectParam
    Mask2StartFrom, Mask2StartTo FloatParam; Mask2StartAnim SelectParam
    Mask2StopFrom, Mask2StopTo  FloatParam; Mask2StopAnim  SelectParam
    Mask2Op                   SelectParam // AND/OR
    FrameMaskEnable           BoolParam
    FrameMaskBits             StringParam
    FrameMaskStart, FrameMaskStop IntParam

    // Specular/Highlight
    SpecLightDirFrom, SpecLightDirTo FloatParam; SpecLightDirAnim SelectParam
    SpecDensityFrom, SpecDensityTo   FloatParam; SpecDensityAnim  SelectParam

    // Drop Shadow
    DropShadowLightDirEnable  BoolParam
    DropShadowLightDirFrom, DropShadowLightDirTo FloatParam; DropShadowLightDirAnim SelectParam
    DropShadowOffsetFrom, DropShadowOffsetTo     FloatParam; DropShadowOffsetAnim   SelectParam
    DropShadowDensityFrom, DropShadowDensityTo   FloatParam; DropShadowDensityAnim  SelectParam
    DropShadowDiffuseFrom, DropShadowDiffuseTo   FloatParam; DropShadowDiffuseAnim  SelectParam
    DropShadowGrad, DropShadowType FloatParam

    // Inner Shadow
    InnerShadowLightDirEnable BoolParam
    // ... same pattern as drop shadow

    // Emboss Effect
    EmbossLightDirEnable BoolParam
    // ... same pattern
}
```

### [x] 1.6 — Layer (`internal/model/layer.go`)

```go
type Layer struct {
    Name    string
    Visible bool
    Solo    bool
    Prim    Primitive
    Eff     Effect
}
```

### [x] 1.7 — Document (`internal/model/document.go`)

```go
type Document struct {
    Prefs   Prefs
    Curves  [8]AnimCurve
    Layers  []Layer
    Textures []TextureEntry  // loaded texture images + metadata
}
```

### [x] 1.8 — File Format (`internal/fileio/`)

Port the `ProfileReader`/`ProfileWriter`/`Parse` logic:

- **`knob.go`** — `Load(path string) (*Document, error)` and `Save(doc *Document, path string) error`
- Binary header: magic `KM` + 4-byte little-endian length prefix before the INI text
- INI sections and keys map to struct fields via reflection/tag-based approach
- Full backward compatibility with v1.3.1+ format
- For WASM: operate on `[]byte` (file upload/download via JS) rather than filesystem paths

**Deliverable:** Round-trip test: load a sample `.knob` file from `assets/samples/`, marshal back, diff against original.

---

## Phase 2 — Rendering Engine: Primitives

**Goal:** Implement all 16 primitive types as pure-Go software renderers using agg_go for path/shape work and direct per-pixel math for the lighting models. All rendering operates on an RGBA pixel buffer.
**Status:** [ ] Partial (baseline renderer + image strip/transparency semantics + shape parser improvements; Java parity pending)

### [x] 2.1 — Buffer Management (`internal/render/buffer.go`)

```go
type PixBuf struct {
    Data          []uint8   // RGBA, row-major
    Width, Height int
    Stride        int       // bytes per row = Width*4
}

func NewPixBuf(w, h int) *PixBuf
func (b *PixBuf) Clear(c color.RGBA)
func (b *PixBuf) At(x, y int) color.RGBA
func (b *PixBuf) Set(x, y int, c color.RGBA)
func (b *PixBuf) CopyFrom(src *PixBuf)
```

Oversampling: primitive renders into an internal buffer scaled by the oversampling factor (1×/2×/4×/8×), then box-filter-downsampled to the document size.

### [x] 2.2 — Texture System (`internal/render/texture.go`)

```go
type Texture struct {
    Data []uint8
    W, H int
}

// Sample with tiling and zoom
func (t *Texture) Sample(u, v, zoom float64) color.RGBA
```

- Load 18 built-in textures from `assets/textures/` (embed with `//go:embed`)
- Texture zoom and tiling (wrapping UV)
- User-supplied textures loaded via file upload (Phase 5)

### [x] 2.3 — Primitive Renderer (`internal/render/primitive.go`)

Interface:
```go
type PrimRenderer interface {
    Render(dst *PixBuf, p *model.Primitive, textures []*Texture, frame, totalFrames int)
}
```

Implementations (one per PrimitiveType):

#### Prim 0: None
Empty — leave `dst` transparent.

#### Prim 1: Image
- Load external image from `File` parameter
- Handle multi-frame strips (N columns or rows, indexed by frame/totalFrames)
- `IntelliAlpha`: generate alpha from luminance if enabled
- `AutoFit`: scale to canvas

#### Prim 2: Circle (Outline Ring)
Use agg_go's `RasterizerScanlineAA` + `Ellipse` shape with stroke.
- Hollow ring with `Width` controlling ring thickness
- Emboss: render two offset rings (bright/dark) using `EmbossDiffuse` for blur radius → simulate the `MakeShadow` Gaussian blur from Java
- `Diffuse`: feather the ring edges with alpha gradient

#### Prim 3: Circle Fill (Phong Sphere)
Per-pixel Phong lighting — port the Java per-pixel math directly:
```
for each pixel (x,y) in canvas:
    nx, ny, nz = sphere normal at (x,y)  // project 2D→sphere normal
    diffuse_light = dot(normal, light_dir)
    specular = pow(dot(reflect, view), specularWidth) * specular
    final_color = ambient*color + diffuse*diffuse_light*color + specular*white
```
- `Aspect` distorts the sphere's X/Y extent
- `Diffuse` controls edge feathering (alpha attenuation near sphere boundary)
- Texture overlay at `TextureDepth`/`TextureZoom`

#### Prim 4: Metal Circle
Same as CircleFill but using the metal specular model (sharp linear specular band, `spectype=1` in Java). The specular is a cosine-based horizontal band rather than point highlight.

#### Prim 5: Wave Circle
Filled circle with radial sine-wave distortion of the boundary. The wave parameters derive from `Step` (frequency), `Length` (amplitude), and `Width` (ring thickness).

#### Prim 6: Gradient Circle
Filled circle with angular or radial gradient instead of Phong lighting. Use agg_go linear/radial gradient span generators.

#### Prim 7: Rect (Outline)
agg_go path: rectangle stroke with `Width` and `Round` (corner radius via RoundedRect).

#### Prim 8: Rect Fill
Per-pixel: fill rectangle, then apply emboss (raised/inset 3D effect using two-pass blur for highlight/shadow), specular highlight stripe, texture overlay. Port Java's `RenderRectFill()` per-pixel math.

#### Prim 9: Triangle
agg_go triangle path (3 vertices), filled and/or stroked. `Round` smooths vertices. `Diffuse` feathers edges.

#### Prim 10: Line
agg_go single antialiased line segment. `Width` = stroke width, endpoints derived from `Length` and `LightDir` (angle).

#### Prim 11: Radiate Line
Multiple spokes radiating from center. `AngleStep` controls angular spacing. Each spoke is an agg_go line. `Width` = thickness. `Length` = spoke length ratio.

#### Prim 12 & 13: H-Lines / V-Lines
Parallel horizontal (or vertical) lines. `Step` = spacing, `Width` = line thickness. Use agg_go repeated line paths or direct pixel fill.

#### Prim 14: Text
agg_go GSV stroke-vector font (no CGO, WASM-safe). Falls back to embedded bitmap font if needed.
- Font selection mapped to embedded font files
- Bold/Italic variants
- `TextAlign` (L/C/R) and `FrameAlign` (top/center/bottom)
- `DynamicText`: parse `(start:end)` substitution syntax for frame counter rendering (e.g. `(1:99)` → `01` on frame 1, `99` on frame 99)

#### Prim 15: Shape
Parse SVG-like path string (M, L, C, Q, Z commands) via a small path parser. Feed parsed vertices into agg_go `PathStorage`. Fill and/or stroke based on `Fill` parameter. `Round` applies agg_go contour smoothing.

### [x] 2.4 — Per-Pixel Math Utilities (`internal/render/lighting.go`)

Port from Java `Primitive.java`:
- `SphereNormal(x, y, cx, cy, rx, ry float64) (nx, ny, nz float64)`
- `PhongLighting(normal [3]float64, lightDir, ambient, diffuse, specular, specWidth float64, baseColor color.RGBA) color.RGBA`
- `MetalSpecular(x, y, cx, cy, lightDir, specular, specWidth float64) float64`
- `EmbossLighting(normal [3]float64, emboss, diffuse float64) float64`
- `TextureBlend(base, tex color.RGBA, depth float64) color.RGBA`
- `Gaussian1D(radius float64) []float64` — for shadow/highlight blur kernels
- `BlurH(src *PixBuf, kernel []float64) *PixBuf`
- `BlurV(src *PixBuf, kernel []float64) *PixBuf`

**Deliverable:** Unit-testable renderer. Each primitive type has a test that renders to a PNG and compares against a reference image.

---

## Phase 3 — Rendering Engine: Effect Stack

**Goal:** Port the full `Eff.Apply()` pipeline.
**Status:** [ ] Not started

### [ ] 3.1 — Affine Transform with Bilinear Sampling (`internal/render/transform.go`)

```go
// Build the 2D affine matrix for zoom+rotate+offset
func BuildMatrix(zoomX, zoomY, angle, offX, offY, centerX, centerY float64) [6]float64

// Apply transform: sample src into dst using inverse mapping + bilinear interpolation
func TransformBlit(dst, src *PixBuf, m [6]float64, antiAlias bool)
```

Port the Java `XYMatrix` logic: inverse-transform each destination pixel to a source coordinate, apply bilinear interpolation.

### [ ] 3.2 — Color Adjustments (`internal/render/coloradj.go`)

In-place operations on a `*PixBuf`:
```go
func AdjustAlpha(b *PixBuf, alpha float64)
func AdjustBrightness(b *PixBuf, brightness float64)
func AdjustContrast(b *PixBuf, contrast float64)
func AdjustSaturation(b *PixBuf, saturation float64)
func AdjustHue(b *PixBuf, hue float64)
```

HSV conversion for saturation/hue (port Java `Col.java` HSV methods).

### [ ] 3.3 — Mask System (`internal/render/mask.go`)

Produce a float32 alpha mask image of the same dimensions as the canvas:

```go
type MaskType int
const (
    MaskRotation   MaskType = iota // angular/rotational band
    MaskRadial                     // radial distance band
    MaskHorizontal                 // horizontal stripe
    MaskVertical                   // vertical stripe
)

func MakeMask(w, h int, mtype MaskType, start, stop, grad, gradDir float64) []float32
func ApplyMask(b *PixBuf, mask []float32)
func CombineMasks(m1, m2 []float32, op int) []float32  // AND/OR
```

Frame mask: zero-out the entire layer for frames outside the mask range/bitmask.

### [ ] 3.4 — Shadow and Highlight (`internal/render/shadow.go`)

Port `Eff.MakeShadow()` and `Eff.Hilight()`:

```go
// Create a blurred alpha channel from the source's alpha channel,
// offset by (dx,dy), attenuated by density, blurred by diffuse radius.
func MakeShadow(src *PixBuf, offset, density, diffuse float64, lightDir float64, shadowColor color.RGBA) *PixBuf

// Inner highlight: invert the shadow for lit-side highlight
func MakeHighlight(src *PixBuf, offset, density, diffuse float64, lightDir float64) *PixBuf
```

Used for: drop shadow, inner shadow, emboss highlight/shadow.

### [ ] 3.5 — Compositing (`internal/render/composite.go`)

```go
// Alpha-blend src over dst (Porter-Duff SrcOver)
func BlendOver(dst, src *PixBuf)

// Full effect apply pipeline: transform + color adjust + masks + shadows → composite onto dst
func ApplyEffect(dst *PixBuf, primBuf *PixBuf, eff *model.Effect, curves [8]*model.AnimCurve, frame, totalFrames int, textures []*Texture)
```

`ApplyEffect` orchestrates:
1. Evaluate all animatable parameters for the given frame
2. Apply zoom/offset/rotate transform
3. Apply color adjustments
4. Build and apply mask(s)
5. Render drop shadow → composite below layer
6. Render inner shadow → composite over layer
7. Render emboss highlight → composite over layer
8. Composite final layer onto `dst`

### [ ] 3.6 — Frame Renderer (`internal/render/render.go`)

```go
// Render a single frame to dst. Iterates layers in order.
func RenderFrame(dst *PixBuf, doc *model.Document, frame int, textures []*Texture)

// Render all export frames. Returns []PixBuf (one per frame).
func RenderAll(doc *model.Document, textures []*Texture) []*PixBuf
```

Oversampling: if `doc.Prefs.Oversampling > 0`, `RenderFrame` uses an internal buffer at 2×/4×/8× resolution and downsamples with a box filter at the end.

**Deliverable:** Given a loaded `.knob` document, `RenderAll` produces pixel-accurate output matching the Java reference output for all sample files.

---

## Phase 4 — Animation System

**Goal:** Full animation curve evaluation and frame-based parameter interpolation.
**Status:** [ ] Partial (curve model done; render-side animation pipeline pending)

### [ ] 4.1 — Parameter Evaluation (`internal/render/animeval.go`)

```go
// Evaluate an animatable from/to parameter pair at the given frame.
// frameFrac is in [0.0, 1.0] representing position in animation.
// The anim SelectParam selects which of the 8 AnimCurves to use (0=linear).
func EvalAnim(from, to float64, animCurveIdx int, curves [8]*model.AnimCurve, frameFrac float64) float64
```

The Java convention: `frameFrac = frame / (totalFrames-1)`. The AnimCurve maps 0–100 time to 0–100 level, then the level is remapped to `[from, to]`.

### [ ] 4.2 — AnimStep

When `Effect.AnimStep > 0`, the layer uses an independent frame count (`Effect.AnimStep` instead of `doc.Prefs.ExportFrames`) for its animation timeline. This allows a layer to complete its animation in a sub-range.

### [ ] 4.3 — DynamicText (`internal/render/dyntext.go`)

Port `DynamicText.java` and `TextCounter.java`:

```go
// Substitute frame counter patterns in text.
// Format: "(start:end)" → left-zero-padded frame number in [start,end]
// Multiple patterns can appear in the same string.
func SubstituteFrameCounters(text string, frame, totalFrames int) string
```

Example: `"Frame (1:99)"` on frame 5 of 31 → `"Frame 05"` (maps 0..30 → 01..99 proportionally).

### [ ] 4.4 — Multi-Frame Image Strip (`internal/render/imagestrip.go`)

For `PrimImage` with `NumFrame > 1`, the external image file is a strip of frames. Load the strip and extract the correct sub-image for the current render frame:

```go
func ExtractFrame(strip *PixBuf, numFrames, frame, totalFrames int) *PixBuf
```

**Deliverable:** Animated export (GIF/APNG) correctly moves through frames with all animatable parameters interpolated.

---

## Phase 5 — WASM Shell & Basic Web UI

**Goal:** A functioning web UI with canvas preview, layer list, and primitive parameter panel. Establishes the JS↔Go communication pattern.
**Status:** [ ] Partial (UI shell + basic render loop done; state wiring pending)

### [ ] 5.1 — WASM Entry Point (`cmd/wasm/main.go`)

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

### [ ] 5.3 — Layer Panel (JS)

- Renders the layer list as a `<ul>` with drag-to-reorder
- Each row: visibility toggle (eye icon), solo toggle, layer name (dblclick to rename), layer type indicator
- Buttons: Add, Delete, Move Up, Move Down, Duplicate
- Click to select — triggers parameter panel refresh

### [ ] 5.4 — Basic Parameter Panel (JS)

- Primitive type selector (dropdown with 16 options)
- On type change: show/hide relevant parameter fields
- Each parameter rendered as: label + appropriate input (range slider, number input, color picker, text input, file picker, checkbox, select)
- All changes call `knobman_setParam(layerIdx, paramKey, value)` → triggers re-render

### [ ] 5.5 — Canvas & Preview

- HTML5 `<canvas>` element
- `requestAnimationFrame`-based render loop (only re-renders when dirty)
- Frame scrubber: range input 0..N → calls `knobman_setPreviewFrame(n)`
- Zoom: pixel-doubled preview for small canvases (e.g. 64×64 at 4× zoom)
- Background checkerboard pattern shows transparency

**Deliverable:** Can load a `.knob` file, see the layers, select them, and see a rendered preview updating in real time as parameters change.

---

## Phase 6 — Complete Parameter Panels

**Goal:** Full fidelity of all parameter controls for all 16 primitive types and the full effect stack.
**Status:** [ ] Not started

### [ ] 6.1 — Primitive Panel per Type

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

### [ ] 6.2 — Effect Stack Panel

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
**Status:** [ ] Partial (history model done; integration pending)

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

### [ ] 9.3 — File Open/Save (Browser)

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
| **3** | [ ] Not started | Full effect stack (transform, color, masks, shadows) | Phase 2 |
| **4** | [ ] Partial | Animation interpolation, dynamic text, image strips | Phase 3 |
| **5** | [ ] Partial | Web UI shell: canvas, layer list, basic param panel | Phase 4 |
| **6** | [ ] Not started | All parameter controls in the web UI | Phase 5 |
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
