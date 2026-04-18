set shell := ["bash", "-uc"]

goroot := `go env GOROOT`

# Default recipe - show available commands
default:
    @just --list

# Build the WASM binary into web/
build-wasm:
    GOOS=js GOARCH=wasm go build -o web/knobman.wasm ./cmd/wasm/
    cp "{{goroot}}/lib/wasm/wasm_exec.js" web/wasm_exec.js
    @echo "Built web/knobman.wasm ($(du -sh web/knobman.wasm | cut -f1))"
    @echo "Copied wasm_exec.js from {{goroot}}/lib/wasm/wasm_exec.js"

# Sync wasm_exec.js from the Go toolchain without rebuilding the WASM binary
sync-wasm-exec:
    cp "{{goroot}}/lib/wasm/wasm_exec.js" web/wasm_exec.js
    @echo "Copied wasm_exec.js from {{goroot}}/lib/wasm/wasm_exec.js"

# Start a local HTTP server for the web/ directory
serve: build-wasm
    @echo "Serving on http://localhost:8080"
    @go run golang.org/x/net/http2/h2c@latest 2>/dev/null || python3 -m http.server 8080 --directory web

# Build for native (no WASM) - useful for testing non-UI packages
build:
    go build -tags freetype ./...

# Run all Go tests (native build, no WASM)
test:
    go test -tags freetype ./...

# Run tests with verbose output
test-v:
    go test -v -tags freetype ./...

# Run tests with race detector
test-race:
    go test -race -tags freetype ./...

# Run tests with coverage
test-coverage:
    go test -v -tags freetype -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html

# Render frame-0 Go artifacts from sample .knob files.
parity-generate FLAGS="":
    go run -tags freetype ./cmd/parityref {{FLAGS}}

# Regenerate frame-0 Go regression baselines from sample .knob files.
parity-baseline-go-generate FLAGS="":
    go run -tags freetype ./cmd/parityref --refs tests/parity/samples/baseline-go --overwrite {{FLAGS}}

# Render frame-0 authoritative Java baselines from sample .knob files.
java-parity-generate FLAGS="":
    ./legacy/run-render-cli.sh --samples assets/samples --output-dir tests/parity/samples/baseline-java --frame 0 --overwrite {{FLAGS}}

# Generate minimal primitive parity fixtures as .knob files.
primitive-fixtures-generate FLAGS="":
    go run ./cmd/primitivefixtures {{FLAGS}}

# Generate focused animated/effect-stack parity fixtures as .knob files.
animated-fixtures-generate FLAGS="":
    go run ./cmd/primitivefixtures --suite animated {{FLAGS}}

# Render frame-0 Go artifacts for the primitive fixtures.
parity-primitives-generate FLAGS="":
    go run -tags freetype ./cmd/parityref --samples tests/parity/primitives/inputs --refs tests/parity/primitives/artifacts --frame 0 --overwrite {{FLAGS}}

# Regenerate frame-0 Go regression baselines for the primitive fixtures.
parity-primitives-baseline-go-generate FLAGS="":
    go run -tags freetype ./cmd/parityref --samples tests/parity/primitives/inputs --refs tests/parity/primitives/baseline-go --frame 0 --overwrite {{FLAGS}}

# Render authoritative Java baselines for the primitive fixtures.
java-parity-primitives-generate FLAGS="":
    ./legacy/run-render-cli.sh --samples tests/parity/primitives/inputs --output-dir tests/parity/primitives/baseline-java --frame 0 --overwrite {{FLAGS}}

# Render Go artifacts for animated/effect-stack fixture keyframes.
parity-animated-generate FLAGS="":
    go run -tags freetype ./cmd/parityref --samples tests/parity/animated/inputs --refs tests/parity/animated/artifacts --keyframes first,mid,last --overwrite {{FLAGS}}

# Regenerate Go regression baselines for animated/effect-stack fixture keyframes.
parity-animated-baseline-go-generate FLAGS="":
    go run -tags freetype ./cmd/parityref --samples tests/parity/animated/inputs --refs tests/parity/animated/baseline-go --keyframes first,mid,last --overwrite {{FLAGS}}

# Render authoritative Java baselines for animated/effect-stack fixture keyframes.
java-parity-animated-generate FLAGS="":
    ./legacy/run-render-cli.sh --samples tests/parity/animated/inputs --output-dir tests/parity/animated/baseline-java --keyframes first,mid,last --overwrite {{FLAGS}}

# Render Go artifacts for selected animated sample keyframes.
parity-animated-samples-generate FLAGS="":
    go run -tags freetype ./cmd/parityref --samples assets/samples --refs tests/parity/animated-samples/artifacts --names Green_Radar,LineShadow,White_Vol --keyframes first,mid,last --overwrite {{FLAGS}}

# Regenerate Go regression baselines for selected animated sample keyframes.
parity-animated-samples-baseline-go-generate FLAGS="":
    go run -tags freetype ./cmd/parityref --samples assets/samples --refs tests/parity/animated-samples/baseline-go --names Green_Radar,LineShadow,White_Vol --keyframes first,mid,last --overwrite {{FLAGS}}

# Render authoritative Java baselines for selected animated sample keyframes.
java-parity-animated-samples-generate FLAGS="":
    ./legacy/run-render-cli.sh --samples assets/samples --output-dir tests/parity/animated-samples/baseline-java --names Green_Radar,LineShadow,White_Vol --keyframes first,mid,last --overwrite {{FLAGS}}

# Run the primitive regression suite against Go baselines.
parity-primitives-test:
    go test -tags freetype ./internal/render -run TestParityRegressionPrimitiveFixturesFrame0 -count=1

# Run the primitive parity suite against authoritative Java baselines.
parity-primitives-golden-test:
    go test -tags freetype ./internal/render -run TestParityGoldenPrimitiveFixturesFrame0 -count=1

# Run the full Phase 2 primitive parity slice.
phase2-parity-test:
    go test -tags freetype ./internal/render -run 'Test(ParityRegressionPrimitiveFixturesFrame0|ParityGoldenPrimitiveFixturesFrame0|PrimitiveFixtureJavaCheckpoints)$' -count=1

# Run animated/effect-stack fixture keyframe parity against Go baselines.
parity-animated-test:
    go test -tags freetype ./internal/render -run TestParityRegressionAnimatedFixturesKeyframes -count=1

# Run animated/effect-stack fixture keyframe parity against Java baselines.
parity-animated-golden-test:
    go test -tags freetype ./internal/render -run TestParityGoldenAnimatedFixturesKeyframes -count=1

# Run animated sample keyframe regression and Java checkpoint tests.
parity-animated-samples-test:
    go test -tags freetype ./internal/render -run 'Test(ParityRegressionAnimatedSamplesKeyframes|AnimatedSampleKeyframeCheckpoints)$' -count=1

# Run the Phase 3 effect-stack/keyframe parity slice.
phase3-parity-test:
    go test -tags freetype ./internal/render -run 'Test(ParityRegressionAnimatedFixturesKeyframes|ParityGoldenAnimatedFixturesKeyframes|AnimatedEffectFixtureCheckpoints|ParityRegressionAnimatedSamplesKeyframes|AnimatedSampleKeyframeCheckpoints)$' -count=1

# Run the focused Phase 7 verification checkpoints.
phase7-checkpoints:
    go test -tags freetype ./internal/render -run 'Test(MaskLegacyCenterAndCombineSemantics|ShadowLegacyDirectionalSweepAndDiffuse|ColorAdjustKeepsLocalHueSaturationAlphaSemantics|EffectMaskPipelineAppliesCombinedMasks|SampleSweepDeltaCheckpoints)$' -count=1 -v

# Run the lightweight Phase 7 transform-heavy performance checkpoint.
phase7-bench BENCHTIME="100ms":
    go test -tags freetype ./internal/render -run '^$' -bench 'BenchmarkRenderFrameVU3Frame0$' -benchmem -count=1 -benchtime={{BENCHTIME}}

# Start the parity comparison viewer on a local web server.
parity-viewer PORT="8090":
    PORT={{PORT}} go run -tags freetype ./cmd/parityviewer --port {{PORT}}

# Format all code using treefmt
fmt:
    treefmt --allow-missing-formatter

# Check if code is formatted correctly
check-formatted:
    treefmt --allow-missing-formatter --fail-on-change

# Run linters
lint:
    GOCACHE="${GOCACHE:-/tmp/gocache}" GOMODCACHE="${GOMODCACHE:-/tmp/gomodcache}" GOLANGCI_LINT_CACHE="${GOLANGCI_LINT_CACHE:-/tmp/golangci-lint-cache}" golangci-lint run --timeout=2m ./...

# Run linters with auto-fix
lint-fix:
    GOCACHE="${GOCACHE:-/tmp/gocache}" GOMODCACHE="${GOMODCACHE:-/tmp/gomodcache}" GOLANGCI_LINT_CACHE="${GOLANGCI_LINT_CACHE:-/tmp/golangci-lint-cache}" golangci-lint run --fix --timeout=2m ./...

# Ensure go.mod is tidy
check-tidy:
    go mod tidy
    git diff --exit-code go.mod go.sum

# Run all checks (formatting, linting, tests, tidiness)
ci: check-formatted test lint check-tidy

# Clean build artifacts
clean:
    rm -f web/knobman.wasm coverage.out coverage.html

fix:
    just lint-fix
    just fmt
