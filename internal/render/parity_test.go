package render

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

const parityTolerance uint8 = 2

type paritySuite struct {
	sampleDir       string
	baselineGoDir   string
	baselineJavaDir string
	artifactsDir    string
}

func TestParityRegressionSamplesFrame0(t *testing.T) {
	root := testRepoRoot(t)
	runParitySuite(t, root, sampleParitySuite(root), "baseline-go")
}

func TestParityRegressionPrimitiveFixturesFrame0(t *testing.T) {
	root := testRepoRoot(t)
	runParitySuite(t, root, primitiveParitySuite(root), "baseline-go")
}

func TestParityGoldenPrimitiveFixturesFrame0(t *testing.T) {
	root := testRepoRoot(t)
	runParitySuite(t, root, primitiveParitySuite(root), "baseline-java")
}

func sampleParitySuite(root string) paritySuite {
	return paritySuite{
		sampleDir:       filepath.Join(root, "assets", "samples"),
		baselineGoDir:   filepath.Join(root, "tests", "parity", "samples", "baseline-go"),
		baselineJavaDir: filepath.Join(root, "tests", "parity", "samples", "baseline-java"),
		artifactsDir:    filepath.Join(root, "tests", "parity", "samples", "artifacts"),
	}
}

func primitiveParitySuite(root string) paritySuite {
	return paritySuite{
		sampleDir:       filepath.Join(root, "tests", "parity", "primitives", "inputs"),
		baselineGoDir:   filepath.Join(root, "tests", "parity", "primitives", "baseline-go"),
		baselineJavaDir: filepath.Join(root, "tests", "parity", "primitives", "baseline-java"),
		artifactsDir:    filepath.Join(root, "tests", "parity", "primitives", "artifacts"),
	}
}

func runParitySuite(t *testing.T, root string, suite paritySuite, baseline string) {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(suite.sampleDir, "*.knob"))
	if err != nil {
		t.Fatalf("glob samples: %v", err)
	}
	if len(paths) == 0 {
		t.Fatalf("no sample .knob files found in %s", suite.sampleDir)
	}

	sort.Strings(paths)
	refDir := suite.baselineDir(t, baseline)
	artifactsDir := filepath.Join(suite.artifactsDir, baseline)

	for _, sample := range paths {
		samplePath := sample
		name := strings.TrimSuffix(filepath.Base(sample), filepath.Ext(sample))
		refPath := filepath.Join(refDir, name+".png")

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			doc, textures, err := LoadParityDocument(samplePath, root)
			if err != nil {
				t.Fatalf("load sample: %v", err)
			}

			if _, err := os.Stat(refPath); err != nil {
				t.Skipf("missing reference: %s", refPath)
			}

			out := NewPixBuf(doc.Prefs.PWidth.Val, doc.Prefs.PHeight.Val)
			if out == nil {
				t.Fatal("failed to allocate output buffer")
			}

			RenderFrame(out, doc, 0, textures)
			if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
				t.Fatalf("mkdir artifacts: %v", err)
			}
			actualPath := filepath.Join(artifactsDir, name+".png")
			if err := WritePixBufPNG(actualPath, out); err != nil {
				t.Fatalf("write artifact: %v", err)
			}
			ref, err := ReadPNGAsRGBA(refPath)
			if err != nil {
				t.Fatalf("read reference: %v", err)
			}
			if err := comparePixBufWithRef(out, ref, parityTolerance); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func (s paritySuite) baselineDir(t *testing.T, baseline string) string {
	t.Helper()
	switch baseline {
	case "baseline-go":
		return s.baselineGoDir
	case "baseline-java":
		return s.baselineJavaDir
	default:
		t.Fatalf("unknown baseline kind %q", baseline)
		return ""
	}
}

func comparePixBufWithRef(actual *PixBuf, want *image.RGBA, tol uint8) error {
	if actual == nil {
		return fmt.Errorf("nil actual buffer")
	}
	if want == nil {
		return fmt.Errorf("nil reference image")
	}
	if actual.Width != want.Bounds().Dx() || actual.Height != want.Bounds().Dy() {
		return fmt.Errorf("size mismatch: actual %dx%d vs ref %dx%d", actual.Width, actual.Height, want.Bounds().Dx(), want.Bounds().Dy())
	}

	var bad, maxDelta int
	for y := 0; y < actual.Height; y++ {
		for x := 0; x < actual.Width; x++ {
			i := y*actual.Stride + x*4
			wi := y*want.Stride + x*4

			r := delta(actual.Data[i+0], want.Pix[wi+0], tol)
			g := delta(actual.Data[i+1], want.Pix[wi+1], tol)
			b := delta(actual.Data[i+2], want.Pix[wi+2], tol)
			a := delta(actual.Data[i+3], want.Pix[wi+3], tol)
			m := int(max4(r, g, b, a))
			if m > maxDelta {
				maxDelta = m
			}
			if r != 0 || g != 0 || b != 0 || a != 0 {
				bad++
			}
		}
	}

	if bad == 0 {
		return nil
	}

	total := actual.Width * actual.Height
	pct := float64(bad) * 100 / float64(total)
	return fmt.Errorf("parity mismatch: %d/%d pixels differ (%.2f%%), maxDelta=%d", bad, total, pct, maxDelta)
}

func delta(a, b, tol uint8) uint8 {
	if a > b {
		d := a - b
		if d <= tol {
			return 0
		}
		return d
	}
	d := b - a
	if d <= tol {
		return 0
	}
	return d
}

func max4(a, b, c, d uint8) uint8 {
	if a < b {
		a = b
	}
	if a < c {
		a = c
	}
	if a < d {
		a = d
	}
	return a
}

func testRepoRoot(t *testing.T) string {
	_, f, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(f), "..", ".."))
}
