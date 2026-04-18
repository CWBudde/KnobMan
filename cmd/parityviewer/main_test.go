package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
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
		Suite:       "primitives",
		Baseline:    "baseline-java",
		Name:        "triangle_plain",
		DocBG:       "#ffffff",
		RMSE:        12.3456,
		AvgDiff:     3.21,
		MaxDiff:     17,
		DiffPixels:  42,
		DiffRatio:   0.125,
		RefWidth:    64,
		RefHeight:   64,
		ActWidth:    64,
		ActHeight:   64,
		RefB64:      "ref",
		ActB64:      "act",
		RawDiffB64:  "raw",
		AmpDiffB64:  "amp",
		RefDocB64:   "ref-doc",
		RefWhiteB64: "ref-white",
		RefDarkB64:  "ref-dark",
		RefCheckB64: "ref-check",
		ActDocB64:   "act-doc",
		ActWhiteB64: "act-white",
		ActDarkB64:  "act-dark",
		ActCheckB64: "act-check",
	}

	var buf bytes.Buffer
	renderCard(&buf, &entry)

	html := buf.String()
	for _, want := range []string{
		`data-suite="primitives"`,
		`data-baseline="baseline-java"`,
		`data-doc-bg="#ffffff"`,
		`data-matte-document="ref-doc"`,
		`data-matte-white="ref-white"`,
		`data-matte-dark="ref-dark"`,
		`data-matte-checkerboard="ref-check"`,
		`data-matte-document="act-doc"`,
		`data-matte-white="act-white"`,
		`data-matte-dark="act-dark"`,
		`data-matte-checkerboard="act-check"`,
		`class="parity-image matte-target"`,
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

func TestRenderPageEmptyState(t *testing.T) {
	var buf bytes.Buffer

	renderPage(&buf, loadResult{})

	html := buf.String()
	for _, want := range []string{
		`No parity comparisons found.`,
		`<div class="container" id="cards-container">`,
		`0 comparisons loaded.`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("renderPage missing %q in output:\n%s", want, html)
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

func TestPageHeaderIncludesMatteModeSelector(t *testing.T) {
	for _, want := range []string{
		`<select id="matte-mode" onchange="setMatteMode(this.value)">`,
		`<option value="document" selected>Matte: document bg</option>`,
		`<option value="white">Matte: white</option>`,
		`<option value="dark">Matte: dark</option>`,
		`<option value="checkerboard">Matte: checkerboard</option>`,
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

func TestPageFooterInitializesMatteMode(t *testing.T) {
	for _, want := range []string{
		`function setMatteMode(mode) {`,
		`document.querySelectorAll('.matte-target').forEach(function(img) {`,
		`img.src = 'data:image/png;base64,' + b64;`,
		`window.setMatteMode = setMatteMode;`,
		`setMatteMode(document.getElementById('matte-mode').value);`,
	} {
		if !strings.Contains(pageFooter, want) {
			t.Fatalf("pageFooter missing %q", want)
		}
	}
}

func TestCompositeOverSolidMatte(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1, 1))
	src.SetRGBA(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 128})

	got := compositeOverSolid(src, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	if got == nil {
		t.Fatal("compositeOverSolid returned nil")
	}

	if c := got.RGBAAt(0, 0); c != (color.RGBA{R: 255, G: 127, B: 127, A: 255}) {
		t.Fatalf("compositeOverSolid pixel = %+v, want {R:255 G:127 B:127 A:255}", c)
	}
}

func TestEnvOr(t *testing.T) {
	t.Setenv("KNOBMAN_TEST_VALUE", " configured ")
	if got := envOr("KNOBMAN_TEST_VALUE", "fallback"); got != "configured" {
		t.Fatalf("envOr returned %q, want %q", got, "configured")
	}

	t.Setenv("KNOBMAN_TEST_VALUE", "   ")
	if got := envOr("KNOBMAN_TEST_VALUE", "fallback"); got != "fallback" {
		t.Fatalf("envOr blank env = %q, want fallback", got)
	}
}

func TestIsSafePathPart(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{input: "alpha", want: true},
		{input: "alpha-beta_01", want: true},
		{input: "", want: false},
		{input: "  ", want: false},
		{input: "../escape", want: false},
		{input: "nested/path", want: false},
	}

	for _, tt := range tests {
		if got := isSafePathPart(tt.input); got != tt.want {
			t.Fatalf("isSafePathPart(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestBadgeClassThresholds(t *testing.T) {
	if got := badgeClass(5); got != badgeClassOK {
		t.Fatalf("badgeClass(5) = %q, want %q", got, badgeClassOK)
	}

	if got := badgeClass(12); got != badgeClassWarn {
		t.Fatalf("badgeClass(12) = %q, want %q", got, badgeClassWarn)
	}

	if got := badgeClass(21); got != badgeClassBad {
		t.Fatalf("badgeClass(21) = %q, want %q", got, badgeClassBad)
	}

	if got := badgeClassAvgDiff(2); got != badgeClassOK {
		t.Fatalf("badgeClassAvgDiff(2) = %q, want %q", got, badgeClassOK)
	}

	if got := badgeClassAvgDiff(5); got != badgeClassWarn {
		t.Fatalf("badgeClassAvgDiff(5) = %q, want %q", got, badgeClassWarn)
	}

	if got := badgeClassAvgDiff(9); got != badgeClassBad {
		t.Fatalf("badgeClassAvgDiff(9) = %q, want %q", got, badgeClassBad)
	}

	if got := badgeClassMaxDiff(10); got != badgeClassOK {
		t.Fatalf("badgeClassMaxDiff(10) = %q, want %q", got, badgeClassOK)
	}

	if got := badgeClassMaxDiff(20); got != badgeClassWarn {
		t.Fatalf("badgeClassMaxDiff(20) = %q, want %q", got, badgeClassWarn)
	}

	if got := badgeClassMaxDiff(41); got != badgeClassBad {
		t.Fatalf("badgeClassMaxDiff(41) = %q, want %q", got, badgeClassBad)
	}

	if got := badgeClassDiffRatio(0.01); got != badgeClassOK {
		t.Fatalf("badgeClassDiffRatio(0.01) = %q, want %q", got, badgeClassOK)
	}

	if got := badgeClassDiffRatio(0.03); got != badgeClassWarn {
		t.Fatalf("badgeClassDiffRatio(0.03) = %q, want %q", got, badgeClassWarn)
	}

	if got := badgeClassDiffRatio(0.06); got != badgeClassBad {
		t.Fatalf("badgeClassDiffRatio(0.06) = %q, want %q", got, badgeClassBad)
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

func TestRerenderArtifactWritesArtifactAndBaselineOutputs(t *testing.T) {
	root := t.TempDir()
	parityDir := filepath.Join(root, "tests", "parity")
	doc := model.NewDocument()
	doc.Prefs.RenderFrames.Val = 4
	mustWriteKnob(t, filepath.Join(root, "tests", "parity", "animated", "inputs", "fixture.knob"), doc)

	sentinel := solidImage(2, 2, color.RGBA{R: 25, G: 50, B: 75, A: 255})
	mustWritePNG(t, filepath.Join(parityDir, "animated", "artifacts", "baseline-go", "stale.png"), sentinel)
	mustWritePNG(t, filepath.Join(parityDir, "animated", "artifacts", "baseline-java", "stale.png"), sentinel)

	orig := runRerenderCommand
	t.Cleanup(func() { runRerenderCommand = orig })

	var gotRepoRoot, gotInput, gotOutput string
	var gotFrame int
	runRerenderCommand = func(repoRoot, inputPath, outputPath string, frame int) error {
		gotRepoRoot = repoRoot
		gotInput = inputPath
		gotOutput = outputPath
		gotFrame = frame
		mustWritePNG(t, outputPath, solidImage(3, 1, color.RGBA{R: 210, G: 80, B: 40, A: 255}))
		return nil
	}

	if err := rerenderArtifact(root, parityDir, "animated", "fixture__last"); err != nil {
		t.Fatalf("rerenderArtifact: %v", err)
	}

	if gotRepoRoot != root {
		t.Fatalf("repo root = %q, want %q", gotRepoRoot, root)
	}

	if want := filepath.Join(root, "tests", "parity", "animated", "inputs", "fixture.knob"); gotInput != want {
		t.Fatalf("input path = %q, want %q", gotInput, want)
	}

	if gotFrame != 3 {
		t.Fatalf("frame = %d, want 3", gotFrame)
	}

	if !strings.Contains(filepath.Base(gotOutput), "parityviewer-rerender-") {
		t.Fatalf("unexpected temp output path %q", gotOutput)
	}

	for _, path := range []string{
		filepath.Join(parityDir, "animated", "artifacts", "fixture__last.png"),
		filepath.Join(parityDir, "animated", "artifacts", "baseline-go", "fixture__last.png"),
		filepath.Join(parityDir, "animated", "artifacts", "baseline-java", "fixture__last.png"),
	} {
		img, err := readPNGAsRGBA(path)
		if err != nil {
			t.Fatalf("readPNGAsRGBA(%q): %v", path, err)
		}

		if img.Bounds().Dx() != 3 || img.Bounds().Dy() != 1 {
			t.Fatalf("unexpected rerendered bounds for %q: %v", path, img.Bounds())
		}
	}
}

func TestRerenderArtifactRejectsUnsafePathParts(t *testing.T) {
	root := t.TempDir()
	parityDir := filepath.Join(root, "tests", "parity")

	if err := rerenderArtifact(root, parityDir, "../bad", "case"); err == nil {
		t.Fatal("expected unsafe suite to fail")
	}

	if err := rerenderArtifact(root, parityDir, "samples", "../bad"); err == nil {
		t.Fatal("expected unsafe case name to fail")
	}
}

func TestDocumentBackgroundReadsKnobPreferences(t *testing.T) {
	root := t.TempDir()
	doc := model.NewDocument()
	doc.Prefs.BkColor.Val = color.RGBA{R: 1, G: 2, B: 3, A: 99}
	mustWriteKnob(t, filepath.Join(root, "tests", "parity", "primitives", "inputs", "fixture.knob"), doc)

	bg, css, err := documentBackground(root, "primitives", "fixture")
	if err != nil {
		t.Fatalf("documentBackground: %v", err)
	}

	if bg != (color.RGBA{R: 1, G: 2, B: 3, A: 255}) {
		t.Fatalf("background = %+v, want opaque RGB copy", bg)
	}

	if css != "#010203" {
		t.Fatalf("css = %q, want #010203", css)
	}
}

func TestRenderFrameCountAndKeyframeFrameIndex(t *testing.T) {
	root := t.TempDir()
	doc := model.NewDocument()
	doc.Prefs.RenderFrames.Val = 7
	input := filepath.Join(root, "fixture.knob")
	mustWriteKnob(t, input, doc)

	totalFrames, err := renderFrameCount(input)
	if err != nil {
		t.Fatalf("renderFrameCount: %v", err)
	}

	if totalFrames != 7 {
		t.Fatalf("renderFrameCount = %d, want 7", totalFrames)
	}

	tests := []struct {
		key  string
		want int
	}{
		{key: "first", want: 0},
		{key: "mid", want: 3},
		{key: "last", want: 6},
	}

	for _, tt := range tests {
		got, err := keyframeFrameIndex(tt.key, totalFrames)
		if err != nil {
			t.Fatalf("keyframeFrameIndex(%q): %v", tt.key, err)
		}

		if got != tt.want {
			t.Fatalf("keyframeFrameIndex(%q) = %d, want %d", tt.key, got, tt.want)
		}
	}

	if _, err := keyframeFrameIndex("frame-"+strconv.Itoa(totalFrames), totalFrames); err == nil {
		t.Fatal("expected unsupported keyframe to fail")
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
