## Parity Layout

`tests/parity/` is organized by suite first, then by file role:

- `samples/` uses the real sample projects from `assets/samples/`.
- `primitives/inputs/` contains focused fixture `.knob` files for isolated renderer coverage.
- `baseline-java/` directories contain the authoritative golden renders produced by legacy JKnobMan. These files are the source of truth for cross-implementation parity.
- `baseline-go/` directories contain Go-rendered regression baselines. They are kept to catch accidental output changes during refactorings, even when a suite is not yet fully gated against the Java golden set.
- `artifacts/` directories contain the current Go-rendered outputs for each suite. They are transient and ignored by git.

## Commands

- `just parity-generate` regenerates `tests/parity/samples/baseline-go/`.
- `just java-parity-generate` regenerates `tests/parity/samples/baseline-java/`.
- `just primitive-fixtures-generate` regenerates `tests/parity/primitives/inputs/`.
- `just parity-primitives-generate` regenerates `tests/parity/primitives/baseline-go/`.
- `just java-parity-primitives-generate` regenerates `tests/parity/primitives/baseline-java/`.
- `just parity-viewer` starts a local browser-facing report that compares baselines against the latest artifacts and sorts cases by RMSE or other diff metrics.
  The viewer defaults to `baseline-java`, keeps a baseline switch, and can re-render the currently shown artifact from the underlying `.knob` file.

## Testing Intent

- Use `baseline-java` when you want to measure parity against the legacy implementation.
- Use `baseline-go` when you want a stable regression tripwire during renderer refactors.
- Run the relevant parity test before opening the viewer so `tests/parity/*/artifacts/` contains fresh Go-rendered images for comparison against either baseline set.
