# KnobMan Go Rewrite — Plan (Compact)

## Goal

- Reach behavior parity with Java JKnobMan 1.3.3 for rendering, effects, animation, and export.
- Keep parity checks deterministic and fast enough for CI.

## Status

- Current date: April 12, 2026
- Architecture: Go/WASM + parser + renderer + JS shell
- Current milestone: `Phase 7.1` in progress

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

## Phase 7 — Expand agg_go Adoption (in progress)

### Phase 7.1 — Lock scope and migration boundaries (done)

- [x] Inventory the current render pipeline and mark where `agg_go` is already in use.
- [x] Define the migration boundary: keep renderer orchestration local, move transform/image/path internals to `agg_go` where safe.
- [x] Define parity constraints for transforms, interpolation, clipping, color conversion, and bounds handling.
- [x] Decide explicit deferrals for text, masks, shadows, lighting, and parity-sensitive texture sampling.

### Phase 7.2 — Transform pipeline migration (in progress)

- [x] Add focused transform-only coverage in renderer tests:
- [x] translation
- [x] rotation around effect center
- [x] non-uniform scale
- [x] `KeepDir`
- [x] out-of-bounds transparent clipping
- [x] Replace `TransformBilinear` sampling with `agg_go` image-transform/interpolation APIs.
- [x] Keep matrix construction entirely on `agg_go` types and helpers.
- [x] Preserve current transform semantics through focused matrix/image tests.
- [ ] Verify output does not introduce wrap, mirror, or edge extension outside source bounds.
- [ ] Record any unavoidable transform differences in the plan before moving on.

### Phase 7.3 — Image adapter and conversion layer (in progress)

- [x] Add a small shared adapter between `PixBuf` and `agg_go` image/context types.
- [x] Centralize straight-alpha RGBA conversion in one place.
- [x] Remove duplicated ad hoc `agg_go` image wrapping from call sites.
- [x] Add focused checks for round-trip image conversion and alpha preservation.
- [x] Keep `PixBuf` as the renderer-owned outer image contract.

### Phase 7.4 — Downsample and blit consolidation (in progress)

- [x] Audit current manual image copy/downsample paths.
- [x] Decide which paths can move to `agg_go` without changing output semantics.
- [x] Replace the safest path first:
- [ ] oversampling downsample
- [x] image blit/scale helper used by primitive image rendering
- [x] Add focused checks for semi-transparent edges and alpha accumulation.
- [x] Keep any path local if `agg_go` changes output beyond accepted tolerance.

### Phase 7.5 — Primitive path expansion (done)

- [x] Split primitives into migration tiers and use `PrimShape`/`PrimImage` as the baseline adapter pattern; keep outline mask, keyed transparency, and `IntelliAlpha` local.
- [x] `Tier 1`: `PrimRectFill`, `PrimRect`, `PrimTriangle`; add fill/outline fixtures, re-check AA edge placement and clipping after each step, and try `PrimCircleFill` only if plain path fill matches baseline geometry before lighting/texture branches.
- [x] `Tier 2`: `PrimLine`, `PrimHLines`, `PrimVLines`, `PrimRadiateLine`; add line fixtures, share one `agg_go` stroke builder where possible, and keep any family local if thickness or spacing drifts from parity fixtures.
- [x] `Tier 3`: `PrimCircle`, deferred `PrimCircleFill`, `PrimMetalCircle`, `PrimWaveCircle`, `PrimSphere`; add texture/lighting fixtures, separate shell geometry from shading math, move only safe shell geometry to `agg_go`, and defer the rest until boundaries are explicit.
- [x] Execute migration in tier order: Tier 1 and Tier 2 are complete, and Tier 3 shell reassessment is reflected in the current circle-family split.
- [x] Keep `PrimText` out of scope for 7.5 and defer primitives whose legacy raster math is still unclear.

### Phase 7.6 — Texture path review (done)

- [x] Add focused fixtures for texture wrap, zoom, low-zoom reduction, and tiling seams.
- [x] Compare the current legacy-style texture sampling against available `agg_go` image filter/resample options; no direct equivalent was found for tiled grayscale+alpha sampling with explicit low-zoom reduction.
- [x] Migrate only the non-legacy-sensitive pieces, if any: none identified yet, so the active texture sampler stays local.
- [x] Keep `SampleHeightAlpha` local unless parity coverage proves equivalence.
- [x] Document the texture behaviors that remain custom for now: tiled wrap, zoom-frequency mapping, `SampleHeightAlpha`'s `<= 50%` 2x reduction, and seam continuity.

### Phase 7.7 — Deferred custom domains review (next)

- [ ] Re-evaluate masks after dedicated fixture coverage exists.
- [ ] Re-evaluate shadows and blur only after transform/image migration stabilizes.
- [ ] Re-evaluate color adjustment only if a shared `agg_go` image pipeline makes it simpler.
- [ ] Keep lighting local unless there is a clear `agg_go` win without parity risk.
- [ ] Keep text deferred until its separate parity track is ready.

### Phase 7.8 — Verification and rollout (next)

- [ ] Add focused regression checks for each migrated area before changing the next one.
- [ ] Add before/after parity checkpoints for:
- [ ] transform fixtures
- [ ] primitive fixtures
- [ ] sample sweep deltas
- [ ] Add lightweight performance checkpoints for transform-heavy scenes.
- [ ] Update milestone markers as each subphase completes.
- [ ] Create the follow-up phase only after the remaining custom fallbacks are explicit.

### Phase 7 component decisions

- [x] `transform.go`: migrate now; highest leverage and already partially on `agg_go`.
- [x] `primitive.go` geometry: migrate incrementally after transform and adapter work.
- [x] `primitive.go` text: defer.
- [x] `texture.go`: defer partial migration until texture fixtures exist.
- [x] `mask.go`: defer.
- [x] `shadow.go`: defer.
- [x] `coloradj.go`: defer unless shared image pipeline reduces risk.
- [x] `lighting.go`: defer.
- [x] `buffer.go` and `render.go`: keep local boundary; migrate through adapters, not wholesale replacement.
