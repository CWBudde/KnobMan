## Parity Layout

`tests/parity/` is organized by suite first, then by file role:

- `samples/` uses the real sample projects from `assets/samples/`.
- `primitives/inputs/` contains focused fixture `.knob` files for isolated renderer coverage.
- `animated/inputs/` contains generated animation/export fixtures used for deterministic first/mid/last-frame parity checks.
- `animated-samples/` contains first/mid/last-frame baselines and artifacts for a targeted subset of real animated sample projects.
- `baseline-java/` directories contain the authoritative golden renders produced by legacy JKnobMan. These files are the source of truth for cross-implementation parity.
- `baseline-go/` directories contain Go-rendered regression baselines. They are kept to catch accidental output changes during refactorings, even when a suite is not yet fully gated against the Java golden set.
- `artifacts/` directories contain the current Go-rendered outputs for each suite. They are transient and ignored by git.

## Commands

- `just parity-generate` regenerates `tests/parity/samples/artifacts/`.
- `just parity-baseline-go-generate` regenerates `tests/parity/samples/baseline-go/`.
- `just java-parity-generate` regenerates `tests/parity/samples/baseline-java/`.
- `just primitive-fixtures-generate` regenerates `tests/parity/primitives/inputs/`.
- `just animated-fixtures-generate` regenerates `tests/parity/animated/inputs/`.
- `just parity-primitives-generate` regenerates `tests/parity/primitives/artifacts/`.
- `just parity-primitives-baseline-go-generate` regenerates `tests/parity/primitives/baseline-go/`.
- `just java-parity-primitives-generate` regenerates `tests/parity/primitives/baseline-java/`.
- `just phase2-parity-test` runs the full Phase 2 primitive parity slice, including the Java checkpoint for the expanded primitive suite.
- `just parity-animated-generate` regenerates `tests/parity/animated/artifacts/` for first/mid/last keyframes.
- `just parity-animated-baseline-go-generate` regenerates `tests/parity/animated/baseline-go/` for first/mid/last keyframes.
- `just java-parity-animated-generate` regenerates `tests/parity/animated/baseline-java/` for first/mid/last keyframes.
- `just parity-animated-samples-generate` regenerates `tests/parity/animated-samples/artifacts/` for the selected real animated samples.
- `just parity-animated-samples-baseline-go-generate` regenerates `tests/parity/animated-samples/baseline-go/` for the selected real animated samples.
- `just java-parity-animated-samples-generate` regenerates `tests/parity/animated-samples/baseline-java/` for the selected real animated samples.
- `just parity-animated-test` runs the animated fixture regression suite against Go keyframe baselines.
- `just parity-animated-golden-test` runs the animated fixture parity suite against Java keyframe baselines.
- `just parity-animated-samples-test` runs the selected animated sample regression suite plus the Java checkpoint test.
- `just phase3-parity-test` runs the full Phase 3 effect-stack/keyframe parity slice in one command.
- `just parity-viewer` starts a local browser-facing report that compares baselines against the latest artifacts and sorts cases by RMSE or other diff metrics.
  The viewer defaults to `baseline-java`, keeps a baseline switch, and can re-render the currently shown artifact from the underlying `.knob` file.

## Pass Policy

Phase 1 formalized a two-track pass policy. Both tracks run on every parity test invocation.

- **`baseline-go` — strict (0x mismatch at `parityTolerance=2`).** The Go regression tripwire. `TestParityRegressionSamplesFrame0`, `TestParityRegressionPrimitiveFixturesFrame0`, and `TestParityStrictZeroMismatchBaselineGo` fail if _any_ pixel differs from the Go baseline. When intentionally changing renderer output, regenerate via `just parity-baseline-go-generate` (+ `just parity-primitives-baseline-go-generate` / `just parity-animated-baseline-go-generate` as applicable) and commit the baseline PNG diff in the same PR as the renderer change.
- **`baseline-java` — tolerance budgets (max RMSE, mean RMSE, diff-rate).** The cross-implementation parity checkpoint. `TestPrimitiveFixtureJavaCheckpoints` and `TestSampleSweepDeltaCheckpoints` enforce RMSE / diff-rate budgets that reflect known small deltas from legacy JKnobMan. Budgets live next to each test and are tightened as parity improves.
- **`parityAllowlist` (in `internal/render/parity_test.go`) skips intentionally diverging samples with a written reason.** Empty today — adding an entry requires a PR that references the tracking issue. Key format: `"<suite>/<baseline>/<sample>"` (e.g. `"samples/baseline-java/Green_Radar"`). The allowlist should be a last resort, used only when the delta has a documented cause that falls outside the current budget.

## Known Deltas vs `baseline-java`

These are the categories currently inside the tolerance budgets. Each bullet names the effect class and cross-references the budget constant.

- **Primitive fixtures** — budget: `maxRMSE ≤ 27`, `meanRMSE ≤ 2.6`, `diffRate ≤ 0.42` (`internal/render/parity_test.go:87–91`). Dominant sources: subpixel anti-aliasing edges on outline primitives (circle/rect outlines, radiate lines), texture tile seams on tiling + wrap cases, text baseline alignment on `text_basic_center`.
- **Full samples sweep** — budget: `maxRMSE ≤ 40`, `meanRMSE ≤ 18.4`, `diffRate ≤ 0.645` (`internal/render/parity_test.go:134–139`). Dominant classes: lighting / sphere shading with non-linear color ramps (`Aqua`, `Black_Gear`, `Red_Gear`), complex effect stacks combining shadow + blur ordering (`LineShadow`, `Green_Radar`), font metric drift on small numeric glyphs (`Number`, `NumberedTick`).
- **Animated fixtures (keyframes)** — budget: `maxRMSE ≤ 132`, `meanRMSE ≤ 88`, `diffRate ≤ 0.91` (`internal/render/parity_animation_test.go:42–46`). Dominant sources: animstep interpolation rounding at frame boundaries, frame mask edge pixels, keyframe snapshot timing for layer/prim animation steps.
- **Animated samples (keyframes)** — covered by the Phase 4 budget in `collectNamedAnimatedKeyframeCheckpointSummary` (`internal/render/parity_sample_animation_test.go`). Same dominant classes as full samples plus animstep interpolation.

When a budget is tightened, or a new class is driven to parity, remove the corresponding bullet. When a new known delta is accepted, add a bullet with the effect class, tracking issue, and the budget value it fits under.

## Testing Intent

- Use `baseline-java` when you want to measure parity against the legacy implementation.
- Use `baseline-go` when you want a stable regression tripwire during renderer refactors.
- Run the relevant parity test before opening the viewer so `tests/parity/*/artifacts/` contains fresh Go-rendered images for comparison against either baseline set.
