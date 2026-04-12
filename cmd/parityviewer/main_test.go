package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCasesSortsByRMSEDescending(t *testing.T) {
	parityDir := filepath.Join(t.TempDir(), "parity")
	mustWritePNG(t, filepath.Join(parityDir, "samples", "baseline-go", "same.png"), solidImage(2, 2, color.RGBA{R: 10, G: 20, B: 30, A: 255}))
	mustWritePNG(t, filepath.Join(parityDir, "samples", "artifacts", "same.png"), solidImage(2, 2, color.RGBA{R: 10, G: 20, B: 30, A: 255}))
	mustWritePNG(t, filepath.Join(parityDir, "samples", "baseline-go", "different.png"), solidImage(2, 2, color.RGBA{R: 0, G: 0, B: 0, A: 255}))
	mustWritePNG(t, filepath.Join(parityDir, "samples", "artifacts", "different.png"), solidImage(2, 2, color.RGBA{R: 255, G: 255, B: 255, A: 255}))

	result, err := loadCases(parityDir)
	if err != nil {
		t.Fatalf("loadCases: %v", err)
	}

	if len(result.Cases) != 2 {
		t.Fatalf("expected 2 cases, got %d", len(result.Cases))
	}

	if got := result.Cases[0].Name; got != "different" {
		t.Fatalf("expected highest-RMSE case first, got %q", got)
	}

	if result.Cases[0].RMSE <= result.Cases[1].RMSE {
		t.Fatalf("expected descending RMSE order, got %.2f then %.2f", result.Cases[0].RMSE, result.Cases[1].RMSE)
	}

	if result.Cases[0].Suite != "samples" || result.Cases[0].Baseline != "baseline-go" {
		t.Fatalf("unexpected case metadata: %+v", result.Cases[0])
	}
}

func TestRenderCardIncludesFiltersAndMetrics(t *testing.T) {
	entry := caseEntry{
		Suite:      "primitives",
		Baseline:   "baseline-java",
		Name:       "triangle_basic",
		RMSE:       12.3456,
		AvgDiff:    3.21,
		MaxDiff:    17,
		DiffPixels: 42,
		DiffRatio:  0.125,
		RefWidth:   64,
		RefHeight:  64,
		ActWidth:   64,
		ActHeight:  64,
		RefB64:     "ref",
		ActB64:     "act",
		RawDiffB64: "raw",
		AmpDiffB64: "amp",
	}

	var buf bytes.Buffer
	renderCard(&buf, &entry)

	html := buf.String()
	for _, want := range []string{
		`data-suite="primitives"`,
		`data-baseline="baseline-java"`,
		`Re-render Artifact`,
		`data-rmse="12.3456"`,
		`data-avg-diff="3.2100"`,
		`data-max-diff="17"`,
		`data-diff-pixels="42"`,
		`Java Golden`,
		`class="img-col col-ref"`,
		`class="img-col col-artifact"`,
		`class="parity-image"`,
		`sort-metric-badge`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("renderCard missing %q in output:\n%s", want, html)
		}
	}
}

func TestPageHeaderOriginalSizeStylesExpandImageColumns(t *testing.T) {
	for _, want := range []string{
		`.original-size .img-grid { grid-template-columns: repeat(5, max-content); }`,
		`.original-size .img-col { min-width: max-content; }`,
		`.original-size .parity-image, .original-size .slider-wrap { align-self: flex-start; }`,
		`.original-size .slider-wrap { width: auto; }`,
		`<select id="resample-mode" onchange="setResampleMode(this.value)">`,
		`<option value="smooth">Scaling: smooth</option>`,
		`<option value="pixelated">Scaling: pixelated</option>`,
		`.resample-pixelated .parity-image { image-rendering: pixelated; }`,
		`.resample-pixelated .slider-wrap img.base { image-rendering: pixelated; }`,
		`.resample-pixelated .slider-overlay img { image-rendering: pixelated; }`,
	} {
		if !strings.Contains(pageHeader, want) {
			t.Fatalf("pageHeader missing %q", want)
		}
	}
}

func TestPageFooterInitializesResampleMode(t *testing.T) {
	for _, want := range []string{
		`function setResampleMode(mode) {`,
		`container.classList.remove('resample-smooth', 'resample-pixelated');`,
		`container.classList.add(mode === 'pixelated' ? 'resample-pixelated' : 'resample-smooth');`,
		`window.setResampleMode = setResampleMode;`,
		`setResampleMode(document.getElementById('resample-mode').value);`,
	} {
		if !strings.Contains(pageFooter, want) {
			t.Fatalf("pageFooter missing %q", want)
		}
	}
}

func TestParityInputPath(t *testing.T) {
	root := "/repo"

	got, err := parityInputPath(root, "samples", "Aqua")
	if err != nil {
		t.Fatalf("samples parityInputPath: %v", err)
	}

	if want := filepath.Join(root, "assets", "samples", "Aqua.knob"); got != want {
		t.Fatalf("samples path mismatch: got %q want %q", got, want)
	}

	got, err = parityInputPath(root, "primitives", "triangle_basic")
	if err != nil {
		t.Fatalf("primitives parityInputPath: %v", err)
	}

	if want := filepath.Join(root, "tests", "parity", "primitives", "inputs", "triangle_basic.knob"); got != want {
		t.Fatalf("primitives path mismatch: got %q want %q", got, want)
	}

	if _, err := parityInputPath(root, "unknown", "x"); err == nil {
		t.Fatal("expected error for unsupported suite")
	}
}

func solidImage(width, height int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.SetRGBA(x, y, c)
		}
	}

	return img
}

func mustWritePNG(t *testing.T, path string, img image.Image) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode %s: %v", path, err)
	}
}
