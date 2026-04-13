# KnobMan Go Rewrite — Plan (Compact)

## Goal

- Reach behavior parity with Java JKnobMan 1.3.3 for rendering, effects, animation, and export.
- Keep parity checks deterministic and fast enough for CI.

## Status

- Current date: April 12, 2026
- Architecture: Go/WASM + parser + renderer + JS shell
- Current milestone: `Phase 7.8` in progress

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

### Phase 7.2 — Transform pipeline migration (done)

- [x] Add focused transform-only coverage in renderer tests:
- [x] translation
- [x] rotation around effect center
- [x] non-uniform scale
- [x] `KeepDir`
- [x] out-of-bounds transparent clipping
- [x] Replace `TransformBilinear` sampling with `agg_go` image-transform/interpolation APIs.
- [x] Keep matrix construction entirely on `agg_go` types and helpers.
- [x] Preserve current transform semantics through focused matrix/image tests.
- [x] Verify output does not introduce wrap, mirror, or edge extension outside source bounds.
- [x] Record any unavoidable transform differences in the plan before moving on: current Java-golden drift remains small but non-zero on the dedicated `vu3_line_transform_flip_probe*` fixtures, so the agg_go path is kept and the residual difference is tracked rather than blocking Phase 7.

### Phase 7.3 — Image adapter and conversion layer (done)

- [x] Add a small shared adapter between `PixBuf` and `agg_go` image/context types.
- [x] Centralize straight-alpha RGBA conversion in one place.
- [x] Remove duplicated ad hoc `agg_go` image wrapping from call sites.
- [x] Add focused checks for round-trip image conversion and alpha preservation.
- [x] Keep `PixBuf` as the renderer-owned outer image contract.

### Phase 7.4 — Downsample and blit consolidation (done)

- [x] Audit current manual image copy/downsample paths.
- [x] Decide which paths can move to `agg_go` without changing output semantics.
- [x] Replace the safest path first:
- [x] oversampling downsample review: keep the current local box downsample after focused agg_go minification checks regressed behavior
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
- [ ] Re-evaluate shadows and blur after the current transform/image migration settles against the remaining parity probes.
- [ ] Re-evaluate color adjustment only if a shared `agg_go` image pipeline makes it simpler.
- [ ] Keep lighting local unless there is a clear `agg_go` win without parity risk.
- [ ] Keep text deferred until its separate parity track is ready.

### Phase 7.8 — Verification and rollout (in progress)

- [x] Add focused regression checks for each migrated area before changing the next one.
- [ ] Add before/after parity checkpoints for:
- [x] transform fixtures
- [x] primitive fixtures
- [ ] sample sweep deltas
- [ ] Add lightweight performance checkpoints for transform-heavy scenes.
- [x] Update milestone markers as each subphase completes.
- [x] Create the follow-up phase once the remaining custom fallbacks are explicit.

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

## Phase 8 — Renderer Contract Realignment over Lower-Level AGG (in progress)

Goal: move from the current mixed `PixBuf`/`Agg2D` contract toward a renderer-owned
straight-alpha `PixBuf` model closer to the original Java `Bitmap` behavior, while
still preferring lower-level `agg_go` math, transforms, rasterizers, and pixel formats
over custom geometry/math code.

### Phase 8.1 — Lock the target contract (in progress)

- [-] Make `PixBuf` the authoritative renderer boundary again and define it explicitly as straight-alpha storage.
- [x] Document the Java reference contract around `Bitmap` read/write/composite behavior and map each relevant Go boundary to it in `docs/render-contract-phase8.md`.
- [x] Record every place where AGG currently writes premultiplied data into `PixBuf` in `docs/render-contract-phase8.md`.
- [x] Define which operations remain allowed to use `Agg2D` during transition and which must move to lower-level `agg_go` building blocks in `docs/render-contract-phase8.md`.

### Phase 8.2 — Move premultiply/demultiply to explicit AGG boundaries (done)

- [x] Stop relying on implicit shared-buffer assumptions between `PixBuf` and AGG image rendering paths.
- [x] Introduce explicit conversion helpers for straight-alpha `PixBuf` <-> premultiplied AGG image buffers.
- [x] Restrict premultiply/demultiply to the narrowest image-transform/blend boundaries that actually require it.
- [x] Add focused tests for semi-transparent pixels, filtered image edges, and `RGB > A` cases across those boundaries.

### Phase 8.3 — Replace high-level `Agg2D` usage where lower-level AGG is clearer

- [x] Replace the AGG-backed image transform/blit production path with custom nearest and bilinear sampling, keeping `agg_go` only for affine matrix construction helpers.
- [x] Move rectangle / triangle rendering back to the custom raster paths as the primary engine and remove the now-unused Agg helper variants.
- [x] Move line-family rendering (`renderLine`, `renderRadiateLines`, `renderParallelLines`) back to the custom raster paths as the primary engine and remove the now-unused Agg helper variants.
- [x] Move circle-family rendering back to the custom raster paths as the primary engine and remove the now-unused Agg helper variants.
- [x] Replace filled-shape AGG mask generation with a custom supersampled even-odd fill mask in `primitive.go` and delete the now-unused AGG path-construction helper.
- [x] Keep text on AGG intentionally, but isolate it behind an explicit off-screen render-to-temp-and-blend-back adapter instead of direct `PixBuf` attachment.
- [x] Move the active GSV fallback off `Agg2D.Text` / `TextWidth` / `FontGSV` by exposing a public GSV path source in `agg_go` and rendering it through ordinary AGG path stroking.
- [x] Audit `AggContextForPixBuf` / `Agg2DForPixBuf` call sites after the text move; no remaining production call sites are left, so keep the adapters fenced as legacy/test helpers only.
- [-] Treat text as an intentional AGG dependency; if a needed font/text adapter is missing, add it locally in KnobMan rather than forcing a non-AGG text rewrite.
- [ ] Add a dedicated non-`Agg2D` TrueType/FreeType text adapter if the optional FreeType-backed path needs to stay active.
- [ ] Decide whether the fenced legacy/test shared-buffer adapters in `image_adapter.go` should be deleted outright once no comparison/debug use remains.

### Phase 8.3 status snapshot

- [x] Production primitive rasterization is back on the custom engine for rectangles, triangles, line families, circle families, shape outlines, and shape fills.
- [x] Production image draw/scale/transform is back on custom sampling code; `agg_go` remains only for affine matrix helpers in this area.
- [x] Custom primitive rasterization now blends over existing destination pixels instead of overwriting them, so semi-transparent primitives preserve prior content.
- [x] No production render path in `internal/render` attaches AGG directly to `PixBuf.Data`.
- [-] Production AGG usage still exists for text rendering. The default build no longer uses the Agg2D text engine surface for GSV fallback text, but a dedicated non-`Agg2D` TrueType adapter is still pending if the optional FreeType-backed path is enabled.
- [x] Production AGG usage has been removed from image transform/blit rendering.

### Phase 8.4 — Migration order

- [x] Migrate primitive raster paths first where current `Agg2D` use was convenience-only rather than behavior-critical.
- [ ] Revisit masks/shadows/color adjustment as the remaining AGG-heavy paths shrink.
- [x] Revisit image draw/scale/transform next; production rendering now uses custom sampling instead of AGG image rendering.
- [x] Revisit text last; keep it on AGG unless a clearly missing adapter forces local support work.

### Phase 8.5 — Acceptance criteria

- [-] `PixBufToNRGBA` can become a direct straight-alpha export path again, or any remaining conversion is narrowly justified and documented.
- [ ] The renderer no longer depends on AGG-owned premultiplied semantics leaking into general `PixBuf` storage.
- [ ] Java parity improves or at minimum becomes easier to reason about on semi-transparent image fixtures.
- [ ] All remaining production AGG dependencies are explicit, justified, adapter-complete where needed, and covered by parity tests.
