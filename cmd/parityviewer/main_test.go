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

	"knobman/internal/fileio"
	"knobman/internal/model"
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
		Name:       "triangle_plain",
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

func TestPageHeaderIncludesAnimatedSuiteFilters(t *testing.T) {
	for _, want := range []string{
		`<option value="animated">Suite: animated</option>`,
		`<option value="animated-samples">Suite: animated-samples</option>`,
	} {
		if !strings.Contains(pageHeader, want) {
			t.Fatalf("pageHeader missing %q", want)
		}
	}
}

func TestPageHeaderIncludesBulkRerenderButton(t *testing.T) {
	for _, want := range []string{
		`<button id="rerender-all-btn" class="rerender-btn" type="button">Re-render Artifacts</button>`,
	} {
		if !strings.Contains(pageHeader, want) {
			t.Fatalf("pageHeader missing %q", want)
		}
	}
}

func TestPageFooterIncludesBulkRerenderHelpers(t *testing.T) {
	for _, want := range []string{
		`function rerenderArtifact(suite, name) {`,
		`function setRerenderButtonsDisabled(disabled) {`,
		`var bulkButton = document.getElementById('rerender-all-btn');`,
		`var buttons = document.querySelectorAll('.rerender-btn');`,
		`return rerenderArtifact(suite, name).then(function() {`,
		`var cards = Array.from(document.querySelectorAll('.card'));`,
		`return rerenderArtifact(card.dataset.suite || '', card.dataset.name || '');`,
	} {
		if !strings.Contains(pageFooter, want) {
			t.Fatalf("pageFooter missing %q", want)
		}
	}
}

func TestParityCaseSpec(t *testing.T) {
	root := "/repo"

	got, frame, err := parityCaseSpec(root, "samples", "Aqua")
	if err != nil {
		t.Fatalf("samples parityCaseSpec: %v", err)
	}

	if want := filepath.Join(root, "assets", "samples", "Aqua.knob"); got != want {
		t.Fatalf("samples path mismatch: got %q want %q", got, want)
	}

	if frame != 0 {
		t.Fatalf("samples frame mismatch: got %d want 0", frame)
	}

	got, frame, err = parityCaseSpec(root, "primitives", "triangle_plain")
	if err != nil {
		t.Fatalf("primitives parityCaseSpec: %v", err)
	}

	if want := filepath.Join(root, "tests", "parity", "primitives", "inputs", "triangle_plain.knob"); got != want {
		t.Fatalf("primitives path mismatch: got %q want %q", got, want)
	}

	if frame != 0 {
		t.Fatalf("primitives frame mismatch: got %d want 0", frame)
	}

	_, _, err = parityCaseSpec(root, "unknown", "x")
	if err == nil {
		t.Fatal("expected error for unsupported suite")
	}
}

func TestParityCaseSpecAnimatedKeyframes(t *testing.T) {
	root := t.TempDir()
	doc := model.NewDocument()
	doc.Prefs.RenderFrames.Val = 5

	mustWriteKnob(t, filepath.Join(root, "tests", "parity", "animated", "inputs", "fixture.knob"), doc)
	mustWriteKnob(t, filepath.Join(root, "assets", "samples", "sample.knob"), doc)

	got, frame, err := parityCaseSpec(root, "animated", "fixture__mid")
	if err != nil {
		t.Fatalf("animated parityCaseSpec: %v", err)
	}

	if want := filepath.Join(root, "tests", "parity", "animated", "inputs", "fixture.knob"); got != want {
		t.Fatalf("animated path mismatch: got %q want %q", got, want)
	}

	if frame != 2 {
		t.Fatalf("animated frame mismatch: got %d want 2", frame)
	}

	got, frame, err = parityCaseSpec(root, "animated-samples", "sample__last")
	if err != nil {
		t.Fatalf("animated-samples parityCaseSpec: %v", err)
	}

	if want := filepath.Join(root, "assets", "samples", "sample.knob"); got != want {
		t.Fatalf("animated-samples path mismatch: got %q want %q", got, want)
	}

	if frame != 4 {
		t.Fatalf("animated-samples frame mismatch: got %d want 4", frame)
	}

	_, _, err = parityCaseSpec(root, "animated", "fixture")
	if err == nil {
		t.Fatal("expected error for animated case without keyframe suffix")
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

	err := os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer f.Close()

	err = png.Encode(f, img)
	if err != nil {
		t.Fatalf("encode %s: %v", path, err)
	}
}

func mustWriteKnob(t *testing.T, path string, doc *model.Document) {
	t.Helper()

	data, err := fileio.Save(doc)
	if err != nil {
		t.Fatalf("save %s: %v", path, err)
	}

	err = os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}

	err = os.WriteFile(path, data, 0o644)
	if err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
