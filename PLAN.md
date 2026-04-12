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

### Phase 7.5 — Primitive path expansion (in progress)

- [x] Split primitives into migration tiers.
- [x] Refine primitive expansion into tier-specific subtasks.
- [ ] Execute tiered primitive migration in order.
- [ ] Keep text explicitly out of scope for this phase.
- [ ] Defer any primitive whose legacy raster math is still unclear.

#### Phase 7.5 primitive tiers

- `Tier 0` already partially or fully on the migration path:
- [x] `PrimShape`
  Current status: filled path rendering already uses `agg_go`; outline mask is still custom.
- [x] `PrimImage`
  Current status: image blit/scale helper is partially moved to `agg_go` behind a safe gate; keyed/intelli-alpha paths remain local.

- `Tier 1` path-friendly geometry:
- [x] `PrimRectFill`
- [ ] `PrimRect`
- [ ] `PrimTriangle`
- [ ] optional follow-up: `PrimCircleFill` only if a plain path fill can match baseline geometry before lighting/texture features are involved

- `Tier 2` stroke and repeated-line families:
- [ ] `PrimLine`
- [ ] `PrimHLines`
- [ ] `PrimVLines`
- [ ] `PrimRadiateLine`

- `Tier 3` texture- and lighting-influenced primitives:
- [ ] `PrimCircle`
- [ ] `PrimCircleFill` if deferred from Tier 1
- [ ] `PrimMetalCircle`
- [ ] `PrimWaveCircle`
- [ ] `PrimSphere`

- `Deferred outside 7.5`:
- [ ] `PrimText`

#### Phase 7.5 Tier 0 follow-up tasks

- [ ] Keep `PrimShape` fill on `agg_go` and evaluate only whether the outline mask should migrate later.
- [ ] Keep `PrimImage` keyed transparency and `IntelliAlpha` on the local path.
- [ ] Treat Tier 0 as the baseline adapter pattern for later primitive migrations.

#### Phase 7.5 Tier 1 subtasks

- [x] Add one isolated fixture group for `Tier 1` geometry with fill-only and outline-only cases.
- [x] Migrate `PrimRectFill` first using `agg_go` path fill.
- [ ] Migrate `PrimRect` second using `agg_go` stroke/path handling only if stroke width parity is acceptable.
- [ ] Migrate `PrimTriangle` third using `agg_go` path fill.
- [ ] Re-check anti-aliased edge placement and canvas clipping after each primitive family.
- [ ] Keep any emboss/specular/texture-influenced branch on the custom path until parity is proven.

#### Phase 7.5 Tier 2 subtasks

- [ ] Add one isolated fixture group for line width, cap/join appearance, spacing, and rotation.
- [ ] Identify which Tier 2 families can share one `agg_go` stroke builder.
- [ ] Migrate `PrimLine` first.
- [ ] Migrate `PrimHLines` and `PrimVLines` second via shared repeated-stroke generation.
- [ ] Migrate `PrimRadiateLine` last in Tier 2 because angle stepping compounds placement risk.
- [ ] Keep any family local if stroke thickness or spacing deviates from current parity fixtures.

#### Phase 7.5 Tier 3 subtasks

- [ ] Add one isolated fixture group for texture depth, lighting direction, specular response, and edge diffuse behavior.
- [ ] Separate Tier 3 primitives into:
- [ ] geometry shell that may be moved to `agg_go`
- [ ] shading/texture math that remains local until proven safe
- [ ] Evaluate `PrimCircle` and `PrimCircleFill` first as the simplest Tier 3 shells.
- [ ] Defer `PrimMetalCircle`, `PrimWaveCircle`, and `PrimSphere` until shell-vs-shading boundaries are explicit in code.
- [ ] Keep legacy texture sampling and lighting math local unless parity coverage proves equivalence.

#### Phase 7.5 execution order

1. Complete the rest of Tier 1.
3. Add Tier 2 fixture group and migrate line families.
4. Reassess whether any Tier 3 shell can move without dragging texture/lighting math with it.
5. Leave `PrimText` deferred for its separate parity track.

### Phase 7.6 — Texture path review (next)

- [ ] Add focused fixtures for texture wrap, zoom, low-zoom reduction, and tiling seams.
- [ ] Compare current legacy-style texture sampling against available `agg_go` options.
- [ ] Migrate only the non-legacy-sensitive pieces, if any.
- [ ] Keep `SampleHeightAlpha` local unless parity coverage proves equivalence.
- [ ] Document any texture behaviors that must remain custom.

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
