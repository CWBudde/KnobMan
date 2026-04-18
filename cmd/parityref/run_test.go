package main

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"knobman/internal/fileio"
	"knobman/internal/model"
	"knobman/internal/render"
)

func TestRunParityRefRendersSingleInput(t *testing.T) {
	t.Parallel()

	root := writeParityRepoRoot(t)
	input := writeParitySampleDoc(t, root, "single.knob", makeRenderDoc(1))
	output := filepath.Join(t.TempDir(), "out.png")

	if err := runParityRef([]string{"-input", input, "-output", output}, root); err != nil {
		t.Fatalf("runParityRef: %v", err)
	}

	img, err := render.ReadPNGAsRGBA(output)
	if err != nil {
		t.Fatalf("ReadPNGAsRGBA(%q): %v", output, err)
	}

	if img.Bounds().Dx() != 16 || img.Bounds().Dy() != 16 {
		t.Fatalf("unexpected output bounds %v", img.Bounds())
	}
}

func TestRunParityRefRendersNamedSampleKeyframes(t *testing.T) {
	t.Parallel()

	root := writeParityRepoRoot(t)
	samplesDir := filepath.Join(root, "assets", "samples")
	if err := os.MkdirAll(samplesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", samplesDir, err)
	}

	writeParitySampleDoc(t, root, filepath.Join("assets", "samples", "alpha.knob"), makeRenderDoc(5))
	writeParitySampleDoc(t, root, filepath.Join("assets", "samples", "beta.knob"), makeRenderDoc(5))

	refsDir := filepath.Join(t.TempDir(), "refs")
	if err := runParityRef([]string{
		"-samples", samplesDir,
		"-refs", refsDir,
		"-names", "beta",
		"-keyframes", "first,mid,last",
		"-overwrite",
	}, root); err != nil {
		t.Fatalf("runParityRef: %v", err)
	}

	for _, name := range []string{"beta__first.png", "beta__mid.png", "beta__last.png"} {
		if _, err := os.Stat(filepath.Join(refsDir, name)); err != nil {
			t.Fatalf("expected keyframe output %q: %v", name, err)
		}
	}

	if _, err := os.Stat(filepath.Join(refsDir, "alpha__first.png")); !os.IsNotExist(err) {
		t.Fatalf("expected filtered sample alpha to be skipped, got err=%v", err)
	}
}

func TestRunParityRefRequiresOutputForSingleInput(t *testing.T) {
	t.Parallel()

	root := writeParityRepoRoot(t)
	input := writeParitySampleDoc(t, root, "single.knob", makeRenderDoc(1))

	if err := runParityRef([]string{"-input", input}, root); err == nil {
		t.Fatal("expected missing output to fail")
	}
}

func TestDetectRepoRootFrom(t *testing.T) {
	t.Parallel()

	root := writeParityRepoRoot(t)
	nested := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", nested, err)
	}

	got, err := detectRepoRootFrom(nested)
	if err != nil {
		t.Fatalf("detectRepoRootFrom: %v", err)
	}

	if got != root {
		t.Fatalf("detectRepoRootFrom(%q) = %q, want %q", nested, got, root)
	}
}

func writeParityRepoRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module knobman\n\ngo 1.24.0\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(go.mod): %v", err)
	}

	return root
}

func writeParitySampleDoc(t *testing.T, root, rel string, doc *model.Document) string {
	t.Helper()

	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}

	data, err := fileio.Save(doc)
	if err != nil {
		t.Fatalf("fileio.Save: %v", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}

	return path
}

func makeRenderDoc(frames int) *model.Document {
	doc := model.NewDocument()
	doc.Prefs.Width = 16
	doc.Prefs.Height = 16
	doc.Prefs.PWidth.Val = 16
	doc.Prefs.PHeight.Val = 16
	doc.Prefs.RenderFrames.Val = frames
	doc.Prefs.PreviewFrames.Val = frames
	doc.Prefs.BkColor.Val = color.RGBA{}
	doc.Layers = []model.Layer{model.NewLayer()}

	ly := &doc.Layers[0]
	ly.Visible.Val = 1
	ly.Prim.Type.Val = int(model.PrimRectFill)
	ly.Prim.Color.Val = color.RGBA{R: 200, G: 40, B: 30, A: 255}
	ly.Prim.Length.Val = 100
	ly.Prim.Aspect.Val = 100

	return doc
}
