goroot := `go env GOROOT`
wasm_exec := "{{goroot}}/lib/wasm/wasm_exec.js"

all: build-wasm

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

# Run all Go tests (native build, no WASM)
test:
    go test ./...

# Run tests with verbose output
test-v:
    go test -v ./...

# Build for native (no WASM) - useful for testing non-UI packages
build:
    go build ./...

clean:
    rm -f web/knobman.wasm
