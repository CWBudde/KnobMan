# KnobMan Go Rewrite — Plan (Compact)

## Goal
- Reach behavior parity with Java JKnobMan 1.3.3 for rendering, effects, animation, and export.
- Keep parity checks deterministic and fast enough for CI.

## Status
- Current date: April 12, 2026
- Architecture: Go/WASM + parser + renderer + JS shell
- Current milestone: `Subphase A` in progress

## Subphase A — Baseline Parity (in progress)
- [x] Scaffold parity harness and sample fixture iteration.
- [x] Implement frame-0 test loop over `assets/samples`.
- [x] Implement Go reference renderer (`cmd/parityref`) and test data path.
- [x] Keep texture loading compatible with legacy BMP textures.
- [x] Normalize PNG readback (`ReadPNGAsRGBA`) to unpremultiplied RGBA for all decoders.
- [x] Regenerate `tests/parity/*/baseline-go` from the Go renderer.
- [ ] Record and enforce baseline mismatches:
  - [ ] 0x mismatch count check with current tolerance.
  - [ ] Add per-sample allowlist only if intentional.

### Subphase A task split
- [x] Task A-1: make failing samples pass at frame 0.
- [ ] Task A-2: document any remaining known-failing classes (lighting, texture, effects).
- [ ] Task A-3: decide strict vs tolerance-based pass policy.

## Subphase B — Primitive + Layer Effects (next)
- [ ] Create isolated fixture tests for primitive raster math edge cases.
- [ ] Verify primitive families:
  - [ ] Geometry shape rasterization
  - [ ] Texture mapping + tiling + wrap
  - [ ] Opacity/composite behavior on semi-transparent pixels
- [ ] Add small fixture groups before running the full sample sweep.

## Subphase C — Effect Stack (next)
- [ ] Add focused tests for transform, shadow, mask, color, and bloom/blur chains.
- [ ] Add combined-stack fixtures representing real sample order.
- [ ] Extend comparison to multiple frames for samples with animation.

## Subphase D — Export + Animation Parity (next)
- [ ] Confirm frame export strip and GIF/APNG timing alignment.
- [ ] Confirm animated layer/prim/effect interpolation matches frame boundaries.
- [ ] Add animated parity fixtures for first/mid/last frame.

## Subphase E — App Completion (next)
- [ ] Finalize undo/redo mutation coverage.
- [ ] Finish keyboard and history recovery paths.
- [ ] Persist session/state recovery and recent docs.
- [ ] Harden JS/WASM integration edge cases.

## Subphase F — CI / Release (next)
- [ ] Add CI gate for parity reference generation/checking.
- [ ] Add `parity-test` command and artifact reporting.
- [ ] Add export perf guard for `RenderAll` and invalidation hot path.
- [ ] Add stable publish path for `web/`.

## Delivery key
- `[x]` done, `[ ]` pending, `[-]` in progress
