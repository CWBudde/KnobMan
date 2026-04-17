package render

import (
	"path/filepath"
	"testing"
)

func TestTriangleJavaCompatModeImprovesParityAgainstJavaBaseline(t *testing.T) {
	root := testRepoRoot(t)
	samplePath := filepath.Join(root, "tests", "parity", "primitives", "inputs", "triangle_plain.knob")

	doc, textures, err := LoadParityDocument(samplePath, root)
	if err != nil {
		t.Fatalf("LoadParityDocument: %v", err)
	}

	want, err := ReadPNGAsRGBA(filepath.Join(root, "tests", "parity", "primitives", "baseline-java", "triangle_plain.png"))
	if err != nil {
		t.Fatalf("ReadPNGAsRGBA: %v", err)
	}

	defaultOut := NewPixBuf(doc.Prefs.PWidth.Val, doc.Prefs.PHeight.Val)
	RenderFrame(defaultOut, doc, 0, textures)

	compatOut := NewPixBuf(doc.Prefs.PWidth.Val, doc.Prefs.PHeight.Val)
	RenderFrameWithOptions(compatOut, doc, 0, textures, RenderOptions{
		Compatibility: CompatibilityJavaTriangleRaster,
	})

	defaultRMSE, defaultDiffPixels, _, _ := comparePixBufMetrics(defaultOut, want)
	compatRMSE, compatDiffPixels, _, _ := comparePixBufMetrics(compatOut, want)

	t.Logf("triangle_plain baseline-java rmse default=%.4f compat=%.4f diffPixels default=%d compat=%d",
		defaultRMSE, compatRMSE, defaultDiffPixels, compatDiffPixels)

	if compatRMSE >= defaultRMSE {
		t.Fatalf("compat RMSE %.4f, want lower than default %.4f", compatRMSE, defaultRMSE)
	}

	if defaultRMSE-compatRMSE < 5.0 {
		t.Fatalf("compat RMSE improvement %.4f, want at least 5.0", defaultRMSE-compatRMSE)
	}
}
