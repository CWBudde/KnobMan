# KnobMan (Go + WebAssembly)

KnobMan is a **fully in-browser** knob / UI-asset renderer and exporter, implemented as a **Go** application compiled to **WebAssembly**.
It aims for feature parity with the original KnobMan / JKnobMan 1.3.3 workflow while being **offline-capable** and **static-hostable** (no server required).

Try online: https://cwbudde.github.io/KnobMan/

## Highlights

- **Runs locally or on static hosting** (GitHub Pages, S3, …) — just HTML/CSS/JS + a WASM binary.
- **Full .knob compatibility**: open, edit, and save projects.
- **Complete renderer**: all primitives, effect stack, masks, lighting/shadows, and compositing.
- **Animation ready**: curves, stepped animation, dynamic text counters, multi-frame image strips.
- **Advanced editors**: curve editor and shape path editor.
- **Export formats**:
  - PNG strip (vertical / horizontal)
  - PNG frames (download as a zip)
  - Animated GIF
  - APNG
- **Productivity**: undo/redo, keyboard shortcuts, samples browser, recent files, session persistence.
- **Quality**: visual regression tests against explicit Go and Java parity baselines.

## Quickstart

### Requirements

- Go **1.24+**
- A modern browser with WebAssembly support
- Optional: [`just`](https://github.com/casey/just) (recommended)

### Build + run the web app

Using `just`:

```bash
just serve
```

Then open:

```text
http://localhost:8080
```

Manual build (no `just`):

```bash
GOOS=js GOARCH=wasm go build -o web/knobman.wasm ./cmd/wasm/
python3 -m http.server 8080 --directory web
```

### Run tests

```bash
just test
```

## Using KnobMan

### Workflow

1. **New / Open** a `.knob` project.
2. Manage layers in the **Layers** panel (add, delete, reorder, duplicate, rename; visibility/solo toggles).
3. Edit the selected layer in the **Parameters** panel:
   - Primitive type + primitive parameters
   - Effect stack parameters (transform, color, masks, shadows, emboss, etc.)
   - Animation controls for animatable values (From/To + curve selection)
4. Use the **curve editor** for precise motion shaping and the **shape editor** for drawing custom paths.
5. Adjust document preferences in the bottom bar (size, frames, background, oversampling, export format).
6. **Export** a PNG strip / PNG frames / GIF / APNG.

### Keyboard shortcuts

- `Ctrl+Z` / `Cmd+Z`: Undo
- `Ctrl+Shift+Z` / `Cmd+Shift+Z`: Redo
- `Ctrl+O` / `Cmd+O`: Open
- `Ctrl+S` / `Cmd+S`: Save
- `Ctrl+E`: Export
- `Ctrl+D`: Duplicate layer
- `Delete`: Delete selected layer
- `↑` / `↓`: Move layer up/down

## Project layout

- `cmd/wasm/` — Go/WASM entry point and JS bridge (functions exported via `syscall/js`)
- `internal/model/` — document model (layers, primitives, effects, prefs, curves, history)
- `internal/fileio/` — `.knob` load/save
- `internal/render/` — software renderer (primitives, effects, compositing, animation evaluation)
- `internal/export/` — exporters (PNG strip/frames, GIF, APNG)
- `web/` — static UI shell (HTML/CSS/JS + WASM loader)
- `assets/` — embedded textures and sample `.knob` projects
- `tests/parity/` — parity suites, baselines, and transient render artifacts
- `legacy/` — archived legacy materials

## Parity Baselines

- `tests/parity/*/baseline-java/` contains the authoritative golden images rendered by legacy JKnobMan.
- `tests/parity/*/baseline-go/` contains Go-rendered regression baselines used to catch accidental behavior changes during refactors.
- `tests/parity/*/artifacts/` holds generated comparison outputs and should not be treated as committed source data.

## Deployment (static hosting)

This project is designed to be deployed as static files. A GitHub Pages workflow is included:

- `.github/workflows/deploy-pages.yml`

It builds the WASM binary and publishes the contents of `web/dist/`.
If your default branch is not `main`, adjust the workflow trigger accordingly.

## License

MIT — see `LICENSE`.

## Notes

- The original KnobMan / JKnobMan resources are preserved under `legacy/` for reference.
