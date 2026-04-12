# KnobMan Go Rewrite — Plan (Compact)

## Goal

- Reach behavior parity with Java JKnobMan 1.3.3 for rendering, effects, animation, and export.
- Keep parity checks deterministic and fast enough for CI.

## Status

- Current date: April 12, 2026
- Architecture: Go/WASM + parser + renderer + JS shell
- Current milestone: `Subphase A` in progress

## Phase 1 — Baseline Parity (in progress)

- [x] Scaffold parity harness and sample fixture iteration.
- [x] Implement frame-0 test loop over `assets/samples`.
- [x] Implement Go reference renderer (`cmd/parityref`) and test data path.
- [x] Keep texture loading compatible with legacy BMP textures.
- [x] Normalize PNG readback (`ReadPNGAsRGBA`) to unpremultiplied RGBA for all decoders.
- [x] Regenerate `tests/parity/*/baseline-go` from the Go renderer.
- [ ] Record and enforce baseline mismatches:
  - [ ] 0x mismatch count check with current tolerance.
  - [ ] Add per-sample allowlist only if intentional.

### Phase 1.1 task split

- [x] Task A-1: make failing samples pass at frame 0.
- [ ] Task A-2: document any remaining known-failing classes (lighting, texture, effects).
- [ ] Task A-3: decide strict vs tolerance-based pass policy.

## Phase 2 — Primitive + Layer Effects (next)

- [ ] Create isolated fixture tests for primitive raster math edge cases.
- [ ] Verify primitive families:
  - [ ] Geometry shape rasterization
  - [ ] Texture mapping + tiling + wrap
  - [ ] Opacity/composite behavior on semi-transparent pixels
- [ ] Add small fixture groups before running the full sample sweep.

## Phase 3 — Effect Stack (next)

- [ ] Add focused tests for transform, shadow, mask, color, and bloom/blur chains.
- [ ] Add combined-stack fixtures representing real sample order.
- [ ] Extend comparison to multiple frames for samples with animation.

## Phase 4 — Export + Animation Parity (next)

- [ ] Confirm frame export strip and GIF/APNG timing alignment.
- [ ] Confirm animated layer/prim/effect interpolation matches frame boundaries.
- [ ] Add animated parity fixtures for first/mid/last frame.

## Phase 5 — App Completion (next)

- [ ] Finalize undo/redo mutation coverage.
- [ ] Finish keyboard and history recovery paths.
- [ ] Persist session/state recovery and recent docs.
- [ ] Harden JS/WASM integration edge cases.

## Phase 6 — CI / Release (next)

- [ ] Add CI gate for parity reference generation/checking.
- [ ] Add `parity-test` command and artifact reporting.
- [ ] Add export perf guard for `RenderAll` and invalidation hot path.
- [ ] Add stable publish path for `web/`.

## Delivery key

- `[x]` done, `[ ]` pending, `[-]` in progress

## Phase 7 — Expand agg_go Adoption

### Phase 7.1 — Audit and boundary design

- Add a focused inventory pass that maps current custom rendering paths against available `agg_go` APIs and defines a safe migration order.
- Define parity requirements for image transforms, interpolation, clipping behavior, color conversion, and bounds handling per phase.
- Add short ADR-style notes in-plan for each component deciding whether to migrate now or defer.

### Phase 7.2 — Migrate transform-heavy raster operations to agg_go

- Replace `TransformBilinear` implementation with agg_go image transformation and interpolation paths.
- Verify mapping between existing transform conventions (translation/rotation/scaling order, invert handling) and agg_go matrix usage.
- Ensure compositing/alpha behavior remains unchanged at the same quality level.

### Phase 7.3 — Replace manual pixel pipeline blocks

- Move image-based primitive rasterization from manual `PixBuf` loops to agg_go draw/image workflows where safe.
- Replace custom downsample logic with agg_go-resident resizing/downsizing/filtering when behavior is equivalent or acceptable.
- Consolidate image conversion and blitting in one shared helper to reduce hand-rolled conversion paths.

### Phase 7.4 — Evaluate and expand path/API surface

- Assess each remaining primitive shape path for `agg.Path` alternatives and migrate incrementally by parity tier.
- Keep non-trivial custom blur/shadow pipelines isolated and only migrate after explicit visual equivalence checks.
- Document known gaps where agg_go lacks behavior parity or where manual fallback is required.

### Phase 7.5 — Verification and rollout

- Add focused render-regression checks for each migrated area before/after (exactly matching current reference sets).
- Add performance checkpoints for transform and full-frame render time after each subphase.
- Update `PLAN.md` progress markers and create follow-up Phase entries once this phase completes.
