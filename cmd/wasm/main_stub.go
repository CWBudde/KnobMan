//go:build !js || !wasm

package main

func main() {
	// Native stub — the application runs as WASM only.
	// This file exists so `go build ./...` succeeds on non-WASM platforms.
}
