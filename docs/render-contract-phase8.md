# Phase 8 Render Contract

This note locks the target renderer contract for Phase 8 before further code
movement. The goal is to restore a Java-like storage boundary while still using
`agg_go` as the primary math and rasterization toolbox.

## Target contract

`PixBuf` is the renderer-owned image boundary.

- Storage contract: straight-alpha RGBA bytes.
- Read contract: `PixBuf.At` returns straight-alpha `color.RGBA`.
- Write contract: local renderer math such as `Set`, `Clear`, `FillRect`, and
  `BlendOver` read and write straight-alpha values.
- Export contract: PNG/export helpers should be able to treat `PixBuf` as
  straight-alpha storage without needing whole-image AGG-aware cleanup.

As of the current Phase 8 work, `PixBufToNRGBA` is expected to export `PixBuf`
bytes directly into `image.NRGBA`. Any premultiply/demultiply cleanup still
required by AGG must happen before data is written back into `PixBuf`, not as a
whole-image export fixup.

This matches the original Java `Bitmap` boundary more closely than the current
shared-buffer Agg2D path.

## Java reference contract

The legacy Java implementation uses `BufferedImage(..., 6)` plus `getRGB` /
`setRGB` as the public bitmap boundary:

- `legacy/src/main/java/Bitmap.java`
  - construction: `new BufferedImage(..., 6)`
  - write path: `setRGB(...)`
  - read path: `getRGB(...)`

For KnobMan parity work, the important consequence is:

- Java `Bitmap` callers observe straight-alpha ARGB values at the storage
  boundary.
- Java2D may use premultiplied math internally, but `Bitmap` itself does not
  expose a premultiplied contract to the rest of the renderer.

The Go rewrite should converge on the same boundary rule: premultiplication is
an implementation detail only at explicit rendering adapters, never the general
`PixBuf` contract.

## Current AGG boundary status

The sections below distinguish between:

- explicit AGG boundaries that already convert between straight-alpha `PixBuf`
  storage and premultiplied AGG image buffers, and
- remaining shared-buffer adapters that still allow direct AGG attachment to
  `PixBuf.Data`.

### Image paths now back on the custom engine by default

- `internal/render/transform.go`
  - `TransformBilinear`
- `internal/render/primitive.go`
  - `drawPixBufToRect`

These paths now use custom nearest and bilinear sampling over straight-alpha
`PixBuf` storage. Bilinear interpolation still uses premultiplied math
internally, but AGG no longer owns the production image-render path.

### Explicit AGG-backed primitive boundaries still in place

- `internal/render/primitive.go`
  - `renderText`

These paths now render into an off-screen AGG buffer and blend the result back
into `PixBuf`, instead of letting AGG write directly into `PixBuf.Data`. Text
is still an intentional retained AGG dependency, but the active GSV fallback no
longer uses `Agg2D`'s text engine helpers (`FontGSV`, `TextWidth`, `Text`).
Instead it uses a public GSV path source exposed from `agg_go` and feeds
ordinary AGG path stroking. If a font/text adapter is missing, the adapter
should be added locally in KnobMan or promoted into `agg_go` rather than
replacing the text renderer wholesale.

### Primitive families now back on the custom engine by default

- `internal/render/primitive_rect_triangle.go`
  - `renderRectOutline`
  - `renderRectFill`
  - `renderTriangle`
- `internal/render/primitive_line_families.go`
  - `renderLine`
  - `renderRadiateLines`
  - `renderParallelLines`
- `internal/render/primitive_circle_families.go`
  - `renderCircleOutline`
  - `renderCircleFill`
- `internal/render/primitive.go`
  - filled-shape mask generation in `renderShape`
  - shape-outline mask generation in `renderShapeOutlineMask`

The old Agg helper variants for these families have been removed; the active
render path now uses the custom rasterizers directly.

### Remaining shared-buffer adapters

- `internal/render/image_adapter.go`
  - `AggImageForPixBuf`
  - `AggContextForPixBuf`
  - `Agg2DForPixBuf`

These helpers still rely on a shared `PixBuf` <-> Agg2D buffer contract, but
there are no remaining production call sites in `internal/render`. They remain
only as fenced legacy/test adapters until Phase 8 decides whether to delete or
replace them outright.

## Transition rules

These rules apply until Phase 8 finishes.

### Allowed during transition

- Text remains on AGG, but the preferred direction is local text adapters plus
  ordinary AGG path/raster operations rather than `Agg2D` text helpers.
- The current build's GSV fallback already follows that rule; the remaining
  optional FreeType-backed path still needs its own non-`Agg2D` adapter if it
  must stay enabled.
- Existing Agg-based primitive fill/stroke helpers may remain temporarily if
  they stay behind explicit adapter functions and parity coverage exists for the
  affected primitive family.
- Lower-level `agg_go` affine transforms, rasterizers, paths, and pixel-format
  helpers are preferred over custom math whenever they can be used without
  exposing a premultiplied storage contract.

### Not allowed during transition

- No new direct shared-buffer `PixBuf` -> `AggImage` / `AggContext` / `Agg2D`
  call sites outside the existing adapter layer.
- No new code may assume that general `PixBuf` storage is premultiplied.
- No new export/readback logic may depend on whole-image demultiplication as the
  normal `PixBuf` contract.

## Migration order implied by this contract

1. Move image transform/blit paths off the shared-buffer contract first.
2. Add explicit straight-alpha <-> premultiplied adapter boundaries where AGG
   image rendering still requires them.
3. Keep text on AGG, but move fallback text off `Agg2D` text helpers first and
   add any missing local adapters in KnobMan.
4. Replace remaining convenience-only `Agg2D` primitive paths with lower-level
   `agg_go` building blocks once the image boundary is stable.

## Success criteria for Phase 8.1

Phase 8.1 is considered complete when:

- the target `PixBuf` contract is documented and used as the reference for new
  work,
- all current AGG shared-buffer touchpoints are inventoried,
- the transition rules above are the default review standard for renderer
  changes.
