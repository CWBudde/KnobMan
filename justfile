set shell := ["bash", "-uc"]

goroot := `go env GOROOT`
wasm_exec := "{{goroot}}/lib/wasm/wasm_exec.js"

# Default recipe - show available commands
default:
    @just --list

# Build the WASM binary into web/
build-wasm:
    GOOS=js GOARCH=wasm go build -o web/knobman.wasm ./cmd/wasm/
    @echo "Built web/knobman.wasm ($(du -sh web/knobman.wasm | cut -f1))"

# Sync wasm_exec.js from the Go toolchain (run after updating Go)
sync-wasm-exec:
    cp {{wasm_exec}} web/wasm_exec.js
    @echo "Copied wasm_exec.js from {{wasm_exec}}"

# Start a local HTTP server for the web/ directory
serve: build-wasm
    @echo "Serving on http://localhost:8080"
    @go run golang.org/x/net/http2/h2c@latest 2>/dev/null || python3 -m http.server 8080 --directory web

# Build for native (no WASM) - useful for testing non-UI packages
build:
    go build ./...

# Run all Go tests (native build, no WASM)
test:
    go test ./...

# Run tests with verbose output
test-v:
    go test -v ./...

# Run tests with race detector
test-race:
    go test -race ./...

# Run tests with coverage
test-coverage:
    go test -v -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html

# Render frame-0 Go regression baselines from sample .knob files.
parity-generate FLAGS="":
    go run ./cmd/parityref {{FLAGS}}

# Render frame-0 authoritative Java baselines from sample .knob files.
java-parity-generate FLAGS="":
    ./legacy/run-render-cli.sh --samples assets/samples --output-dir tests/parity/samples/baseline-java --frame 0 --overwrite {{FLAGS}}

# Generate minimal primitive parity fixtures as .knob files.
primitive-fixtures-generate FLAGS="":
    go run ./cmd/primitivefixtures {{FLAGS}}

# Render frame-0 Go regression baselines for the primitive fixtures.
parity-primitives-generate FLAGS="":
    go run ./cmd/parityref --samples tests/parity/primitives/inputs --refs tests/parity/primitives/baseline-go --frame 0 --overwrite {{FLAGS}}

# Render authoritative Java baselines for the primitive fixtures.
java-parity-primitives-generate FLAGS="":
    ./legacy/run-render-cli.sh --samples tests/parity/primitives/inputs --output-dir tests/parity/primitives/baseline-java --frame 0 --overwrite {{FLAGS}}

# Run the primitive regression suite against Go baselines.
parity-primitives-test:
    go test ./internal/render -run TestParityRegressionPrimitiveFixturesFrame0 -count=1

# Run the primitive parity suite against authoritative Java baselines.
parity-primitives-golden-test:
    go test ./internal/render -run TestParityGoldenPrimitiveFixturesFrame0 -count=1

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
