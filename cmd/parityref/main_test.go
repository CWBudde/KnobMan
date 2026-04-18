package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultRefsDirForSamplesDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		samplesDir string
		want       string
	}{
		{
			name:       "samples",
			samplesDir: filepath.Join("assets", "samples"),
			want:       filepath.Join("tests", "parity", "samples", "artifacts"),
		},
		{
			name:       "primitives",
			samplesDir: filepath.Join("tests", "parity", "primitives", "inputs"),
			want:       filepath.Join("tests", "parity", "primitives", "artifacts"),
		},
		{
			name:       "animated",
			samplesDir: filepath.Join("tests", "parity", "animated", "inputs"),
			want:       filepath.Join("tests", "parity", "animated", "artifacts"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := defaultRefsDirForSamplesDir(tt.samplesDir); got != tt.want {
				t.Fatalf("defaultRefsDirForSamplesDir(%q) = %q, want %q", tt.samplesDir, got, tt.want)
			}
		})
	}
}

func TestJustParityGenerateRecipesWriteArtifacts(t *testing.T) {
	t.Parallel()

	justfilePath := filepath.Join("..", "..", "justfile")

	data, err := os.ReadFile(justfilePath)
	if err != nil {
		t.Fatalf("read justfile: %v", err)
	}

	content := string(data)
	wants := []string{
		"parity-primitives-generate FLAGS=\"\":\n    go run -tags freetype ./cmd/parityref --samples tests/parity/primitives/inputs --refs tests/parity/primitives/artifacts --frame 0 --overwrite {{FLAGS}}",
		"parity-animated-generate FLAGS=\"\":\n    go run -tags freetype ./cmd/parityref --samples tests/parity/animated/inputs --refs tests/parity/animated/artifacts --keyframes first,mid,last --overwrite {{FLAGS}}",
		"parity-animated-samples-generate FLAGS=\"\":\n    go run -tags freetype ./cmd/parityref --samples assets/samples --refs tests/parity/animated-samples/artifacts --names Green_Radar,LineShadow,White_Vol --keyframes first,mid,last --overwrite {{FLAGS}}",
		"parity-baseline-go-generate FLAGS=\"\":\n    go run -tags freetype ./cmd/parityref --refs tests/parity/samples/baseline-go --overwrite {{FLAGS}}",
		"parity-primitives-baseline-go-generate FLAGS=\"\":\n    go run -tags freetype ./cmd/parityref --samples tests/parity/primitives/inputs --refs tests/parity/primitives/baseline-go --frame 0 --overwrite {{FLAGS}}",
		"parity-animated-baseline-go-generate FLAGS=\"\":\n    go run -tags freetype ./cmd/parityref --samples tests/parity/animated/inputs --refs tests/parity/animated/baseline-go --keyframes first,mid,last --overwrite {{FLAGS}}",
		"parity-animated-samples-baseline-go-generate FLAGS=\"\":\n    go run -tags freetype ./cmd/parityref --samples assets/samples --refs tests/parity/animated-samples/baseline-go --names Green_Radar,LineShadow,White_Vol --keyframes first,mid,last --overwrite {{FLAGS}}",
	}

	for _, want := range wants {
		if !strings.Contains(content, want) {
			t.Fatalf("justfile missing expected recipe snippet:\n%s", want)
		}
	}
}

func TestParseRenderOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "default empty", raw: ""},
		{name: "default explicit", raw: "default"},
		{name: "java triangle raster", raw: "java-triangle-raster"},
		{name: "invalid", raw: "java", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts, err := parseRenderOptions(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseRenderOptions(%q) succeeded, want error", tt.raw)
				}

				return
			}

			if err != nil {
				t.Fatalf("parseRenderOptions(%q): %v", tt.raw, err)
			}

			switch tt.raw {
			case "java-triangle-raster":
				if opts.Compatibility != "java-triangle-raster" {
					t.Fatalf("compatibility = %q, want %q", opts.Compatibility, "java-triangle-raster")
				}
			default:
				if opts.Compatibility != "" {
					t.Fatalf("compatibility = %q, want default empty mode", opts.Compatibility)
				}
			}
		})
	}
}
