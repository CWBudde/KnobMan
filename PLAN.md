# KnobMan Go Rewrite — Plan (Compact)

## Goal

- Reach behavior parity with Java JKnobMan 1.3.3 for rendering, effects, animation, and export.
- Keep parity checks deterministic and fast enough for CI.

## Status

- Current date: April 14, 2026
- Architecture: Go/WASM + parser + renderer + JS shell
- Current milestone: `Phase 5 complete / Phase 6 prep`

## Phase 1 — Baseline Parity (done)

- [x] Scaffold parity harness and sample fixture iteration.
- [x] Implement frame-0 test loop over `assets/samples`.
- [x] Implement Go reference renderer (`cmd/parityref`) and test data path.
- [x] Keep texture loading compatible with legacy BMP textures.
- [x] Normalize PNG readback (`ReadPNGAsRGBA`) to unpremultiplied RGBA for all decoders.
- [x] Regenerate `tests/parity/*/baseline-go` from the Go renderer.
- [x] Record and enforce baseline mismatches:
  - [x] 0x mismatch count check with current tolerance. _(enforced by `TestParityStrictZeroMismatchBaselineGo` at `parityTolerance=2`.)_
  - [x] Add per-sample allowlist only if intentional. _(empty `parityAllowlist` wired in `internal/render/parity_test.go` with unit tests.)_

### Phase 1.1 task split

- [x] Task A-1: make failing samples pass at frame 0.
- [x] Task A-2: document any remaining known-failing classes (lighting, texture, effects). _(see `tests/parity/README.md` Known Deltas.)_
- [x] Task A-3: decide strict vs tolerance-based pass policy. _(two-track split documented in `tests/parity/README.md` Pass Policy.)_

## Phase 2 — Primitive + Layer Effects (next)

- [x] Create isolated fixture tests for primitive raster math edge cases.
- [x] Verify primitive families:
  - [x] Geometry shape rasterization
  - [x] Texture mapping + tiling + wrap
  - [x] Opacity/composite behavior on semi-transparent pixels
- [x] Add small fixture groups before running the full sample sweep.

## Phase 3 — Effect Stack (next)

- [x] Add focused tests for transform, shadow, mask, color, and bloom/blur chains.
- [x] Add combined-stack fixtures representing real sample order.
- [x] Extend comparison to multiple frames for samples with animation.

## Phase 4 — Export + Animation Parity (in progress)

- [x] Confirm frame export strip and GIF/APNG timing alignment.
- [x] Confirm animated layer/prim/effect interpolation matches frame boundaries.
- [x] Add animated parity fixtures for first/mid/last frame.

## Phase 5 — App Completion (done)

- [x] Finalize undo/redo mutation coverage.
- [x] Finish keyboard and history recovery paths.
- [x] Persist session/state recovery and recent docs.
- [x] Harden JS/WASM integration edge cases.

## Phase 6 — CI / Release (next)

- [ ] Add CI gate for parity reference generation/checking.
- [ ] Add `parity-test` command and artifact reporting.
- [ ] Add export perf guard for `RenderAll` and invalidation hot path.
- [ ] Add stable publish path for `web/`.
