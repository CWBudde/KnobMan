package render

import (
	"errors"
	"fmt"
	"image"
	"math"
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

type parityCheckpointSummary struct {
	Compared        int
	DiffCases       int
	TotalPixels     int
	TotalDiffPixels int
	MeanRMSE        float64
	MaxRMSE         float64
	WorstCase       string
}

type parityCheckpointBudget struct {
	ComparedMaxRMSE  float64
	ComparedMeanRMSE float64
	ComparedDiffRate float64
	ComparedCases    int
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

func TestSampleSweepDeltaCheckpoints(t *testing.T) {
	root := testRepoRoot(t)
	checks := []struct {
		name     string
		suite    paritySuite
		baseline string
		budget   parityCheckpointBudget
	}{
		{
			name:     "samples_vs_baseline_go",
			suite:    sampleParitySuite(root),
			baseline: "baseline-go",
			budget: parityCheckpointBudget{
				ComparedCases:    38,
				ComparedMaxRMSE:  62,
				ComparedMeanRMSE: 24.3,
				ComparedDiffRate: 0.515,
			},
		},
		{
			name:     "samples_vs_baseline_java",
			suite:    sampleParitySuite(root),
			baseline: "baseline-java",
			budget: parityCheckpointBudget{
				ComparedCases:    38,
				ComparedMaxRMSE:  40,
				ComparedMeanRMSE: 18.4,
				ComparedDiffRate: 0.645,
			},
		},
	}

	for _, check := range checks {
		check := check
		t.Run(check.name, func(t *testing.T) {
			summary := collectParityCheckpointSummary(t, root, check.suite, check.baseline)
			diffRate := 0.0
			if summary.TotalPixels > 0 {
				diffRate = float64(summary.TotalDiffPixels) / float64(summary.TotalPixels)
			}

			t.Logf(
				"parity checkpoint baseline=%s compared=%d diffCases=%d meanRMSE=%.4f maxRMSE=%.4f diffRate=%.4f worst=%s",
				check.baseline,
				summary.Compared,
				summary.DiffCases,
				summary.MeanRMSE,
				summary.MaxRMSE,
				diffRate,
				summary.WorstCase,
			)

			if summary.Compared != check.budget.ComparedCases {
				t.Fatalf("compared cases = %d, want %d", summary.Compared, check.budget.ComparedCases)
			}

			if summary.MaxRMSE > check.budget.ComparedMaxRMSE {
				t.Fatalf("max RMSE %.4f exceeded checkpoint %.4f", summary.MaxRMSE, check.budget.ComparedMaxRMSE)
			}

			if summary.MeanRMSE > check.budget.ComparedMeanRMSE {
				t.Fatalf("mean RMSE %.4f exceeded checkpoint %.4f", summary.MeanRMSE, check.budget.ComparedMeanRMSE)
			}

			if diffRate > check.budget.ComparedDiffRate {
				t.Fatalf("diff rate %.4f exceeded checkpoint %.4f", diffRate, check.budget.ComparedDiffRate)
			}
		})
	}
}

func TestNumberHSwitchUnfoldRendersAllFourSlots(t *testing.T) {
	root := testRepoRoot(t)
	samplePath := filepath.Join(root, "assets", "samples", "Number_HSwitch.knob")

	doc, textures, err := LoadParityDocument(samplePath, root)
	if err != nil {
		t.Fatalf("load sample: %v", err)
	}

	out := NewPixBuf(doc.Prefs.PWidth.Val, doc.Prefs.PHeight.Val)
	RenderFrame(out, doc, 0, textures)

	minX, maxX, ok := nonBackgroundBoundsX(out)
	if !ok {
		t.Fatal("rendered no visible content")
	}

	if minX != 1 || maxX != 63 {
		t.Fatalf("unfolded bounds = [%d,%d], want [1,63]", minX, maxX)
	}

	for _, slot := range [][2]int{{1, 15}, {17, 31}, {33, 47}, {49, 63}} {
		if count, ok := nonBackgroundCountXRange(out, slot[0], slot[1]); !ok || count == 0 {
			t.Fatalf("slot [%d,%d] has no visible unfolded content", slot[0], slot[1])
		}
	}
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
		return errors.New("nil actual buffer")
	}

	if want == nil {
		return errors.New("nil reference image")
	}

	if actual.Width != want.Bounds().Dx() || actual.Height != want.Bounds().Dy() {
		return fmt.Errorf("size mismatch: actual %dx%d vs ref %dx%d", actual.Width, actual.Height, want.Bounds().Dx(), want.Bounds().Dy())
	}

	var bad, maxDelta int

	for y := range actual.Height {
		for x := range actual.Width {
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

func nonBackgroundBoundsX(buf *PixBuf) (minX, maxX int, ok bool) {
	if buf == nil || buf.Width == 0 || buf.Height == 0 {
		return 0, 0, false
	}

	bg := buf.At(0, 0)
	minX = buf.Width
	maxX = -1

	for y := range buf.Height {
		for x := range buf.Width {
			if buf.At(x, y) == bg {
				continue
			}

			if x < minX {
				minX = x
			}

			if x > maxX {
				maxX = x
			}
		}
	}

	if maxX < minX {
		return 0, 0, false
	}

	return minX, maxX, true
}

func nonBackgroundCountXRange(buf *PixBuf, startX, endX int) (count int, ok bool) {
	if buf == nil || buf.Width == 0 || buf.Height == 0 {
		return 0, false
	}

	if startX < 0 {
		startX = 0
	}

	if endX >= buf.Width {
		endX = buf.Width - 1
	}

	if startX > endX {
		return 0, false
	}

	bg := buf.At(0, 0)
	for y := range buf.Height {
		for x := startX; x <= endX; x++ {
			if buf.At(x, y) != bg {
				count++
			}
		}
	}

	return count, true
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

func collectParityCheckpointSummary(t *testing.T, root string, suite paritySuite, baseline string) parityCheckpointSummary {
	t.Helper()

	paths, err := filepath.Glob(filepath.Join(suite.sampleDir, "*.knob"))
	if err != nil {
		t.Fatalf("glob samples: %v", err)
	}

	sort.Strings(paths)

	refDir := suite.baselineDir(t, baseline)
	summary := parityCheckpointSummary{}

	for _, samplePath := range paths {
		name := strings.TrimSuffix(filepath.Base(samplePath), filepath.Ext(samplePath))
		refPath := filepath.Join(refDir, name+".png")

		doc, textures, err := LoadParityDocument(samplePath, root)
		if err != nil {
			t.Fatalf("load sample %s: %v", samplePath, err)
		}

		ref, err := ReadPNGAsRGBA(refPath)
		if err != nil {
			t.Fatalf("read reference %s: %v", refPath, err)
		}

		out := NewPixBuf(doc.Prefs.PWidth.Val, doc.Prefs.PHeight.Val)
		RenderFrame(out, doc, 0, textures)

		rmse, diffPixels, totalPixels, diffRatio := comparePixBufMetrics(out, ref)
		summary.Compared++
		summary.TotalPixels += totalPixels
		summary.TotalDiffPixels += diffPixels
		summary.MeanRMSE += rmse

		if diffPixels != 0 {
			summary.DiffCases++
		}

		if rmse > summary.MaxRMSE {
			summary.MaxRMSE = rmse
			summary.WorstCase = fmt.Sprintf("%s (rmse=%.4f, diff=%.4f)", name, rmse, diffRatio)
		}
	}

	if summary.Compared > 0 {
		summary.MeanRMSE /= float64(summary.Compared)
	}

	return summary
}

func comparePixBufMetrics(actual *PixBuf, want *image.RGBA) (rmse float64, diffPixels, totalPixels int, diffRatio float64) {
	if actual == nil || want == nil {
		return 0, 0, 0, 0
	}

	totalPixels = actual.Width * actual.Height
	if totalPixels == 0 {
		return 0, 0, 0, 0
	}

	var sumSq float64

	for y := 0; y < actual.Height; y++ {
		for x := 0; x < actual.Width; x++ {
			i := y*actual.Stride + x*4
			wi := y*want.Stride + x*4

			dr := absDiff(actual.Data[i+0], want.Pix[wi+0])
			dg := absDiff(actual.Data[i+1], want.Pix[wi+1])
			db := absDiff(actual.Data[i+2], want.Pix[wi+2])
			da := absDiff(actual.Data[i+3], want.Pix[wi+3])

			if max4(dr, dg, db, da) != 0 {
				diffPixels++
			}

			sumSq += sqDiff(actual.Data[i+0], want.Pix[wi+0])
			sumSq += sqDiff(actual.Data[i+1], want.Pix[wi+1])
			sumSq += sqDiff(actual.Data[i+2], want.Pix[wi+2])
			sumSq += sqDiff(actual.Data[i+3], want.Pix[wi+3])
		}
	}

	rmse = math.Sqrt(sumSq / float64(totalPixels*4))
	diffRatio = float64(diffPixels) / float64(totalPixels)

	return rmse, diffPixels, totalPixels, diffRatio
}

func absDiff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}

	return b - a
}

func sqDiff(a, b uint8) float64 {
	d := float64(absDiff(a, b))
	return d * d
}

func testRepoRoot(t *testing.T) string {
	_, f, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(f), "..", ".."))
}
