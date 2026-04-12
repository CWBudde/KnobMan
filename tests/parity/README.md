## Parity Layout

`tests/parity/` is organized by suite first, then by file role:

- `samples/` uses the real sample projects from `assets/samples/`.
- `primitives/inputs/` contains focused fixture `.knob` files for isolated renderer coverage.
- `baseline-java/` directories contain the authoritative golden renders produced by legacy JKnobMan. These files are the source of truth for cross-implementation parity.
- `baseline-go/` directories contain Go-rendered regression baselines. They are kept to catch accidental output changes during refactorings, even when a suite is not yet fully gated against the Java golden set.
- `artifacts/` directories contain generated outputs from parity test runs and regeneration commands. They are transient and ignored by git.

## Commands

- `just parity-generate` regenerates `tests/parity/samples/baseline-go/`.
- `just java-parity-generate` regenerates `tests/parity/samples/baseline-java/`.
- `just primitive-fixtures-generate` regenerates `tests/parity/primitives/inputs/`.
- `just parity-primitives-generate` regenerates `tests/parity/primitives/baseline-go/`.
- `just java-parity-primitives-generate` regenerates `tests/parity/primitives/baseline-java/`.

## Testing Intent

- Use `baseline-java` when you want to measure parity against the legacy implementation.
- Use `baseline-go` when you want a stable regression tripwire during renderer refactors.
