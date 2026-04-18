package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"html"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"knobman/internal/fileio"
	"knobman/internal/render"
)

type caseEntry struct {
	Suite       string
	Baseline    string
	Name        string
	DocBG       string
	RefDocB64   string
	RefWhiteB64 string
	RefDarkB64  string
	RefCheckB64 string
	ActDocB64   string
	ActWhiteB64 string
	ActDarkB64  string
	ActCheckB64 string
	RMSE        float64
	AvgDiff     float64
	MaxDiff     uint8
	DiffPixels  int
	TotalPixels int
	DiffRatio   float64
	RefWidth    int
	RefHeight   int
	ActWidth    int
	ActHeight   int
	RefB64      string
	ActB64      string
	RawDiffB64  string
	AmpDiffB64  string
}

type loadResult struct {
	Cases              []caseEntry
	ComparedCount      int
	MissingArtifactCnt int
}

type metrics struct {
	RMSE        float64
	AvgDiff     float64
	MaxDiff     uint8
	DiffPixels  int
	TotalPixels int
	DiffRatio   float64
}

func main() {
	port := flag.String("port", envOr("PORT", "8090"), "Port to listen on")
	parityDirFlag := flag.String("parity-dir", filepath.Join("tests", "parity"), "Parity directory to inspect")

	flag.Parse()

	root, err := detectRepoRoot()
	if err != nil {
		log.Fatalf("detect repo root: %v", err)
	}

	parityDir := *parityDirFlag
	if !filepath.IsAbs(parityDir) {
		parityDir = filepath.Join(root, parityDir)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		result, err := loadCases(parityDir)
		if err != nil {
			http.Error(w, fmt.Sprintf("load parity cases: %v", err), http.StatusInternalServerError)
			return
		}

		renderPage(w, result)
	})
	http.HandleFunc("/rerender", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		err := rerenderArtifact(root, parityDir, r.FormValue("suite"), r.FormValue("name"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})

	addr := ":" + *port
	log.Printf("Parity viewer running at http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func loadCases(parityDir string) (loadResult, error) {
	repoRoot, _ := detectRepoRoot()

	suiteDirs, err := os.ReadDir(parityDir)
	if err != nil {
		return loadResult{}, fmt.Errorf("read parity dir %s: %w", parityDir, err)
	}

	var result loadResult

	for _, suiteDir := range suiteDirs {
		if !suiteDir.IsDir() {
			continue
		}

		suiteName := suiteDir.Name()
		suitePath := filepath.Join(parityDir, suiteName)

		children, err := os.ReadDir(suitePath)
		if err != nil {
			return loadResult{}, fmt.Errorf("read suite dir %s: %w", suitePath, err)
		}

		for _, child := range children {
			if !child.IsDir() || !strings.HasPrefix(child.Name(), "baseline-") {
				continue
			}

			baselineName := child.Name()
			baselineDir := filepath.Join(suitePath, baselineName)
			artifactDir := filepath.Join(suitePath, "artifacts")

			baselines, err := filepath.Glob(filepath.Join(baselineDir, "*.png"))
			if err != nil {
				return loadResult{}, fmt.Errorf("glob baselines in %s: %w", baselineDir, err)
			}

			sort.Strings(baselines)

			for _, baselinePath := range baselines {
				name := strings.TrimSuffix(filepath.Base(baselinePath), filepath.Ext(baselinePath))

				artifactPath := filepath.Join(artifactDir, baselineName, name+".png")

				_, err := os.Stat(artifactPath)
				if err != nil {
					artifactPath = filepath.Join(artifactDir, name+".png")

					_, err = os.Stat(artifactPath)
					if err != nil {
						result.MissingArtifactCnt++
						continue
					}
				}

				entry, err := buildEntry(repoRoot, suiteName, baselineName, name, baselinePath, artifactPath)
				if err != nil {
					return loadResult{}, fmt.Errorf("build entry %s/%s/%s: %w", suiteName, baselineName, name, err)
				}

				result.Cases = append(result.Cases, entry)
				result.ComparedCount++
			}
		}
	}

	sort.SliceStable(result.Cases, func(i, j int) bool {
		if result.Cases[i].RMSE == result.Cases[j].RMSE {
			if result.Cases[i].Suite == result.Cases[j].Suite {
				if result.Cases[i].Baseline == result.Cases[j].Baseline {
					return result.Cases[i].Name < result.Cases[j].Name
				}

				return result.Cases[i].Baseline < result.Cases[j].Baseline
			}

			return result.Cases[i].Suite < result.Cases[j].Suite
		}

		return result.Cases[i].RMSE > result.Cases[j].RMSE
	})

	return result, nil
}

func buildEntry(repoRoot, suite, baseline, name, baselinePath, artifactPath string) (caseEntry, error) {
	ref, err := readPNGAsRGBA(baselinePath)
	if err != nil {
		return caseEntry{}, fmt.Errorf("read baseline: %w", err)
	}

	act, err := readPNGAsRGBA(artifactPath)
	if err != nil {
		return caseEntry{}, fmt.Errorf("read artifact: %w", err)
	}

	rawDiff := rawDiffImage(ref, act)
	ampDiff := amplifiedDiffImage(ref, act)
	stats := compareImages(ref, act)

	refB64, err := pngToBase64(ref)
	if err != nil {
		return caseEntry{}, fmt.Errorf("encode baseline: %w", err)
	}

	actB64, err := pngToBase64(act)
	if err != nil {
		return caseEntry{}, fmt.Errorf("encode artifact: %w", err)
	}

	rawDiffB64, err := pngToBase64(rawDiff)
	if err != nil {
		return caseEntry{}, fmt.Errorf("encode raw diff: %w", err)
	}

	ampDiffB64, err := pngToBase64(ampDiff)
	if err != nil {
		return caseEntry{}, fmt.Errorf("encode amplified diff: %w", err)
	}

	docBGColor := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	docBG := "#ffffff"

	if repoRoot != "" {
		bg, css, bgErr := documentBackground(repoRoot, suite, name)
		if bgErr == nil {
			docBGColor = bg
			docBG = css
		}
	}

	refDocB64, err := pngToBase64(compositeOverSolid(ref, docBGColor))
	if err != nil {
		return caseEntry{}, fmt.Errorf("encode baseline doc matte: %w", err)
	}

	refWhiteB64, err := pngToBase64(compositeOverSolid(ref, color.RGBA{R: 255, G: 255, B: 255, A: 255}))
	if err != nil {
		return caseEntry{}, fmt.Errorf("encode baseline white matte: %w", err)
	}

	refDarkB64, err := pngToBase64(compositeOverSolid(ref, color.RGBA{R: 12, G: 16, B: 22, A: 255}))
	if err != nil {
		return caseEntry{}, fmt.Errorf("encode baseline dark matte: %w", err)
	}

	refCheckB64, err := pngToBase64(compositeOverCheckerboard(ref))
	if err != nil {
		return caseEntry{}, fmt.Errorf("encode baseline checkerboard matte: %w", err)
	}

	actDocB64, err := pngToBase64(compositeOverSolid(act, docBGColor))
	if err != nil {
		return caseEntry{}, fmt.Errorf("encode artifact doc matte: %w", err)
	}

	actWhiteB64, err := pngToBase64(compositeOverSolid(act, color.RGBA{R: 255, G: 255, B: 255, A: 255}))
	if err != nil {
		return caseEntry{}, fmt.Errorf("encode artifact white matte: %w", err)
	}

	actDarkB64, err := pngToBase64(compositeOverSolid(act, color.RGBA{R: 12, G: 16, B: 22, A: 255}))
	if err != nil {
		return caseEntry{}, fmt.Errorf("encode artifact dark matte: %w", err)
	}

	actCheckB64, err := pngToBase64(compositeOverCheckerboard(act))
	if err != nil {
		return caseEntry{}, fmt.Errorf("encode artifact checkerboard matte: %w", err)
	}

	return caseEntry{
		Suite:       suite,
		Baseline:    baseline,
		Name:        name,
		DocBG:       docBG,
		RefDocB64:   refDocB64,
		RefWhiteB64: refWhiteB64,
		RefDarkB64:  refDarkB64,
		RefCheckB64: refCheckB64,
		ActDocB64:   actDocB64,
		ActWhiteB64: actWhiteB64,
		ActDarkB64:  actDarkB64,
		ActCheckB64: actCheckB64,
		RMSE:        stats.RMSE,
		AvgDiff:     stats.AvgDiff,
		MaxDiff:     stats.MaxDiff,
		DiffPixels:  stats.DiffPixels,
		TotalPixels: stats.TotalPixels,
		DiffRatio:   stats.DiffRatio,
		RefWidth:    ref.Bounds().Dx(),
		RefHeight:   ref.Bounds().Dy(),
		ActWidth:    act.Bounds().Dx(),
		ActHeight:   act.Bounds().Dy(),
		RefB64:      refB64,
		ActB64:      actB64,
		RawDiffB64:  rawDiffB64,
		AmpDiffB64:  ampDiffB64,
	}, nil
}

func compareImages(ref, act *image.RGBA) metrics {
	bounds := unionBounds(ref.Bounds(), act.Bounds())

	totalPixels := bounds.Dx() * bounds.Dy()
	if totalPixels == 0 {
		return metrics{}
	}

	var sumSq float64
	var totalDiff float64
	var diffPixels int
	var maxDiff uint8

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rr, rg, rb, ra := rgbaAt(ref, x, y)
			ar, ag, ab, aa := rgbaAt(act, x, y)

			dr := absDiff8(rr, ar)
			dg := absDiff8(rg, ag)
			db := absDiff8(rb, ab)
			da := absDiff8(ra, aa)

			pixelMax := max4(dr, dg, db, da)
			if pixelMax > maxDiff {
				maxDiff = pixelMax
			}

			if pixelMax != 0 {
				diffPixels++
			}

			totalDiff += float64(dr) + float64(dg) + float64(db) + float64(da)
			sumSq += sqDiff(rr, ar) + sqDiff(rg, ag) + sqDiff(rb, ab) + sqDiff(ra, aa)
		}
	}

	return metrics{
		RMSE:        math.Sqrt(sumSq / float64(totalPixels*4)),
		AvgDiff:     totalDiff / float64(totalPixels*4),
		MaxDiff:     maxDiff,
		DiffPixels:  diffPixels,
		TotalPixels: totalPixels,
		DiffRatio:   float64(diffPixels) / float64(totalPixels),
	}
}

func rawDiffImage(ref, act *image.RGBA) *image.RGBA {
	bounds := unionBounds(ref.Bounds(), act.Bounds())
	out := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))

	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			rr, rg, rb, ra := rgbaAt(ref, bounds.Min.X+x, bounds.Min.Y+y)
			ar, ag, ab, aa := rgbaAt(act, bounds.Min.X+x, bounds.Min.Y+y)

			dr := absDiff8(rr, ar)
			dg := absDiff8(rg, ag)
			db := absDiff8(rb, ab)
			da := absDiff8(ra, aa)

			if dr == 0 && dg == 0 && db == 0 && da == 0 {
				out.SetRGBA(x, y, color.RGBA{R: 0, G: 0xaa, B: 0, A: 255})
				continue
			}

			out.SetRGBA(x, y, color.RGBA{R: dr, G: dg, B: db, A: clampAlpha(da)})
		}
	}

	return out
}

func amplifiedDiffImage(ref, act *image.RGBA) *image.RGBA {
	bounds := unionBounds(ref.Bounds(), act.Bounds())
	out := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))

	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			rr, rg, rb, ra := rgbaAt(ref, bounds.Min.X+x, bounds.Min.Y+y)
			ar, ag, ab, aa := rgbaAt(act, bounds.Min.X+x, bounds.Min.Y+y)

			dr := absDiff8(rr, ar)
			dg := absDiff8(rg, ag)
			db := absDiff8(rb, ab)
			da := absDiff8(ra, aa)
			pixelMax := max4(dr, dg, db, da)

			if pixelMax == 0 {
				out.SetRGBA(x, y, color.RGBA{R: 0, G: 0xaa, B: 0, A: 255})
				continue
			}

			intensity := uint8(255)
			if pixelMax < 255 {
				intensity = uint8((float64(pixelMax) / 255.0) * 255.0)
			}

			out.SetRGBA(x, y, color.RGBA{R: intensity, G: 0, B: 0, A: 255})
		}
	}

	return out
}

func rgbaAt(img *image.RGBA, x, y int) (uint8, uint8, uint8, uint8) {
	if img == nil || !image.Pt(x, y).In(img.Bounds()) {
		return 0, 0, 0, 0
	}

	i := img.PixOffset(x, y)

	return img.Pix[i+0], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]
}

func unionBounds(a, b image.Rectangle) image.Rectangle {
	minX := min(a.Min.X, b.Min.X)
	minY := min(a.Min.Y, b.Min.Y)
	maxX := max(a.Max.X, b.Max.X)
	maxY := max(a.Max.Y, b.Max.Y)

	if maxX < minX {
		maxX = minX
	}

	if maxY < minY {
		maxY = minY
	}

	return image.Rect(minX, minY, maxX, maxY)
}

func pngToBase64(img image.Image) (string, error) {
	var buf bytes.Buffer

	err := png.Encode(&buf, img)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func readPNGAsRGBA(path string) (*image.RGBA, error) {
	return render.ReadPNGAsRGBA(path)
}

func absDiff8(a, b uint8) uint8 {
	if a > b {
		return a - b
	}

	return b - a
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

func sqDiff(a, b uint8) float64 {
	d := float64(int(a) - int(b))
	return d * d
}

func clampAlpha(a uint8) uint8 {
	if a == 0 {
		return 255
	}

	return a
}

func detectRepoRoot() (string, error) {
	wd, err := filepath.Abs(".")
	if err != nil {
		return "", err
	}

	for range 8 {
		_, err := os.Stat(filepath.Join(wd, "go.mod"))
		if err == nil {
			return wd, nil
		}

		next := filepath.Dir(wd)
		if next == wd {
			break
		}

		wd = next
	}

	return "", errors.New("go.mod not found from cwd")
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}

	return fallback
}

func rerenderArtifact(repoRoot, parityDir, suite, name string) error {
	if !isSafePathPart(suite) {
		return fmt.Errorf("invalid suite %q", suite)
	}

	if !isSafePathPart(name) {
		return fmt.Errorf("invalid case name %q", name)
	}

	inputPath, frame, err := parityCaseSpec(repoRoot, suite, name)
	if err != nil {
		return err
	}

	artifactDir := filepath.Join(parityDir, suite, "artifacts")

	outputPaths := []string{filepath.Join(artifactDir, name+".png")}
	for _, baselineName := range []string{"baseline-go", "baseline-java"} {
		basePath := filepath.Join(artifactDir, baselineName, name+".png")

		_, err := os.Stat(filepath.Dir(basePath))
		if err == nil {
			outputPaths = append(outputPaths, basePath)
		}
	}

	seen := make(map[string]struct{}, len(outputPaths))

	uniq := make([]string, 0, len(outputPaths))
	for _, path := range outputPaths {
		if _, ok := seen[path]; ok {
			continue
		}

		seen[path] = struct{}{}
		uniq = append(uniq, path)
	}

	outputPaths = uniq

	outputFile, err := os.CreateTemp("", "parityviewer-rerender-*.png")
	if err != nil {
		return fmt.Errorf("create temp output: %w", err)
	}
	defer os.Remove(outputFile.Name())

	err = outputFile.Close()
	if err != nil {
		return fmt.Errorf("close temp output: %w", err)
	}

	outputPath := outputFile.Name()

	cmd := exec.Command(
		"go", "run", "-tags", "freetype", "./cmd/parityref",
		"--input", inputPath,
		"--output", outputPath,
		"--frame", strconv.Itoa(frame),
	)
	cmd.Dir = repoRoot

	cmd.Env = append(os.Environ(), "GOCACHE=/tmp/knobman-gocache")

	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}

		return fmt.Errorf("rerender failed: %s", msg)
	}

	_, err = os.Stat(outputPath)
	if err != nil {
		return fmt.Errorf("rerender did not produce %s: %w", outputPath, err)
	}

	rendered, err := os.ReadFile(outputPath)
	if err != nil {
		return fmt.Errorf("read rerender output: %w", err)
	}

	for _, path := range outputPaths {
		err := os.MkdirAll(filepath.Dir(path), 0o755)
		if err != nil {
			return fmt.Errorf("create artifact dir: %w", err)
		}

		err = os.WriteFile(path, rendered, 0o644)
		if err != nil {
			return fmt.Errorf("write artifact %s: %w", path, err)
		}
	}

	return nil
}

func documentBackground(repoRoot, suite, name string) (color.RGBA, string, error) {
	inputPath, _, err := parityCaseSpec(repoRoot, suite, name)
	if err != nil {
		return color.RGBA{}, "", err
	}

	data, err := os.ReadFile(inputPath)
	if err != nil {
		return color.RGBA{}, "", fmt.Errorf("read input %s: %w", inputPath, err)
	}

	doc, err := fileio.Load(data)
	if err != nil {
		return color.RGBA{}, "", fmt.Errorf("load input %s: %w", inputPath, err)
	}

	bg := color.RGBA{R: doc.Prefs.BkColor.Val.R, G: doc.Prefs.BkColor.Val.G, B: doc.Prefs.BkColor.Val.B, A: 255}

	return bg, fmt.Sprintf("#%02x%02x%02x", bg.R, bg.G, bg.B), nil
}

func compositeOverSolid(src *image.RGBA, bg color.RGBA) *image.RGBA {
	if src == nil {
		return nil
	}

	bounds := src.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := rgbaAt(src, x, y)
			out.SetRGBA(x-bounds.Min.X, y-bounds.Min.Y, compositePixel(r, g, b, a, bg))
		}
	}

	return out
}

func compositeOverCheckerboard(src *image.RGBA) *image.RGBA {
	if src == nil {
		return nil
	}

	bounds := src.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	light := color.RGBA{R: 216, G: 222, B: 233, A: 255}
	dark := color.RGBA{R: 195, G: 202, B: 214, A: 255}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			bg := light
			if ((x-bounds.Min.X)/6+(y-bounds.Min.Y)/6)%2 != 0 {
				bg = dark
			}

			r, g, b, a := rgbaAt(src, x, y)
			out.SetRGBA(x-bounds.Min.X, y-bounds.Min.Y, compositePixel(r, g, b, a, bg))
		}
	}

	return out
}

func compositePixel(r, g, b, a uint8, bg color.RGBA) color.RGBA {
	srcA := int(a)
	invA := 255 - srcA

	return color.RGBA{
		R: uint8((int(r)*srcA + int(bg.R)*invA) / 255),
		G: uint8((int(g)*srcA + int(bg.G)*invA) / 255),
		B: uint8((int(b)*srcA + int(bg.B)*invA) / 255),
		A: 255,
	}
}

func isSafePathPart(s string) bool {
	if strings.TrimSpace(s) == "" {
		return false
	}

	if s != filepath.Base(s) {
		return false
	}

	return !strings.Contains(s, "..")
}

func parityCaseSpec(repoRoot, suite, name string) (string, int, error) {
	switch suite {
	case "samples":
		return filepath.Join(repoRoot, "assets", "samples", name+".knob"), 0, nil
	case "primitives":
		return filepath.Join(repoRoot, "tests", "parity", "primitives", "inputs", name+".knob"), 0, nil
	case "animated":
		return animatedCaseSpec(filepath.Join(repoRoot, "tests", "parity", "animated", "inputs"), name)
	case "animated-samples":
		return animatedCaseSpec(filepath.Join(repoRoot, "assets", "samples"), name)
	default:
		return "", 0, fmt.Errorf("unsupported suite %q", suite)
	}
}

func animatedCaseSpec(inputDir, name string) (string, int, error) {
	baseName, keyframe, ok := strings.Cut(name, "__")
	if !ok || strings.TrimSpace(baseName) == "" || strings.TrimSpace(keyframe) == "" {
		return "", 0, fmt.Errorf("animated case %q must use name__keyframe form", name)
	}

	inputPath := filepath.Join(inputDir, baseName+".knob")

	totalFrames, err := renderFrameCount(inputPath)
	if err != nil {
		return "", 0, err
	}

	frame, err := keyframeFrameIndex(keyframe, totalFrames)
	if err != nil {
		return "", 0, err
	}

	return inputPath, frame, nil
}

func renderFrameCount(inputPath string) (int, error) {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return 0, fmt.Errorf("read input %s: %w", inputPath, err)
	}

	doc, err := fileio.Load(data)
	if err != nil {
		return 0, fmt.Errorf("load input %s: %w", inputPath, err)
	}

	if doc.Prefs.RenderFrames.Val <= 1 {
		return 1, nil
	}

	return doc.Prefs.RenderFrames.Val, nil
}

func keyframeFrameIndex(keyframe string, totalFrames int) (int, error) {
	if totalFrames <= 1 {
		return 0, nil
	}

	switch keyframe {
	case "first":
		return 0, nil
	case "mid":
		return totalFrames / 2, nil
	case "last":
		return totalFrames - 1, nil
	default:
		return 0, fmt.Errorf("unsupported keyframe %q", keyframe)
	}
}

func esc(s string) string {
	return html.EscapeString(s)
}

func renderPage(w io.Writer, result loadResult) {
	fmt.Fprint(w, pageHeader)
	fmt.Fprintf(w, `<div class="header-meta">%d comparisons loaded`, result.ComparedCount)

	if result.MissingArtifactCnt > 0 {
		fmt.Fprintf(w, `, %d baselines skipped because no matching artifact exists`, result.MissingArtifactCnt)
	}

	fmt.Fprint(w, `.</div></div>`)
	fmt.Fprint(w, `<div class="container" id="cards-container">`)

	for i := range result.Cases {
		renderCard(w, &result.Cases[i])
	}

	if len(result.Cases) == 0 {
		fmt.Fprint(w, `<div class="empty-state">No parity comparisons found. Run a parity test first so `+
			`the suite writes images under <code>tests/parity/*/artifacts/</code>.</div>`)
	}

	fmt.Fprint(w, pageFooter)
}

func renderCard(w io.Writer, entry *caseEntry) {
	if entry == nil {
		return
	}

	refLabel := "Baseline"

	switch entry.Baseline {
	case "baseline-java":
		refLabel = "Java Golden"
	case "baseline-go":
		refLabel = "Go Baseline"
	}

	fmt.Fprintf(
		w,
		`<div class="card" data-name="%s" data-suite="%s" data-baseline="%s" data-doc-bg="%s" data-rmse="%.4f" data-avg-diff="%.4f" data-max-diff="%d" data-diff-pixels="%d" data-diff-ratio="%.6f">`,
		esc(entry.Name),
		esc(entry.Suite),
		esc(entry.Baseline),
		esc(entry.DocBG),
		entry.RMSE,
		entry.AvgDiff,
		entry.MaxDiff,
		entry.DiffPixels,
		entry.DiffRatio,
	)
	fmt.Fprint(w, `<div class="card-header">`)
	fmt.Fprint(w, `<span class="badge badge-neutral sort-metric-badge" style="display:none"></span>`)
	fmt.Fprintf(w, `<span class="card-title">%s</span>`, esc(entry.Name))
	fmt.Fprint(w, `<div class="right-badges">`)
	fmt.Fprintf(w, `<span class="badge badge-neutral">%s</span>`, esc(entry.Suite))
	fmt.Fprintf(w, `<span class="badge badge-neutral">%s</span>`, esc(entry.Baseline))
	fmt.Fprintf(w, `<span class="badge %s">RMSE %.2f</span>`, badgeClass(entry.RMSE), entry.RMSE)
	fmt.Fprintf(w, `<span class="badge %s">avg %.2f</span>`, badgeClassAvgDiff(entry.AvgDiff), entry.AvgDiff)
	fmt.Fprintf(w, `<span class="badge %s">max %d</span>`, badgeClassMaxDiff(entry.MaxDiff), entry.MaxDiff)
	fmt.Fprintf(w, `<span class="badge %s">diff %.2f%%</span>`, badgeClassDiffRatio(entry.DiffRatio), entry.DiffRatio*100)
	fmt.Fprint(w, `</div></div>`)

	fmt.Fprint(w, `<div class="card-body"><div class="card-meta">`)
	fmt.Fprintf(w, `baseline %dx%d, artifact %dx%d`, entry.RefWidth, entry.RefHeight, entry.ActWidth, entry.ActHeight)
	fmt.Fprintf(
		w,
		` <button class="rerender-btn" data-suite="%s" data-name="%s" type="button">Re-render Artifact</button>`,
		esc(entry.Suite),
		esc(entry.Name),
	)
	fmt.Fprint(w, `</div><div class="img-grid">`)

	fmt.Fprint(w, `<div class="img-col col-ref">`)
	fmt.Fprintf(w, `<label>%s</label>`, esc(refLabel))
	fmt.Fprintf(
		w,
		`<img class="parity-image matte-target" src="data:image/png;base64,%s" alt="baseline" data-matte-document="%s" data-matte-white="%s" data-matte-dark="%s" data-matte-checkerboard="%s">`,
		entry.RefDocB64,
		entry.RefDocB64,
		entry.RefWhiteB64,
		entry.RefDarkB64,
		entry.RefCheckB64,
	)
	fmt.Fprint(w, `</div>`)

	fmt.Fprint(w, `<div class="img-col col-artifact">`)
	fmt.Fprint(w, `<label>Artifact</label>`)
	fmt.Fprintf(
		w,
		`<img class="parity-image matte-target" src="data:image/png;base64,%s" alt="artifact" data-matte-document="%s" data-matte-white="%s" data-matte-dark="%s" data-matte-checkerboard="%s">`,
		entry.ActDocB64,
		entry.ActDocB64,
		entry.ActWhiteB64,
		entry.ActDarkB64,
		entry.ActCheckB64,
	)
	fmt.Fprint(w, `</div>`)

	fmt.Fprint(w, `<div class="img-col col-overlay">`)
	fmt.Fprint(w, `<label>Overlay</label>`)
	fmt.Fprint(w, `<div class="slider-wrap">`)
	fmt.Fprintf(
		w,
		`<img class="base matte-target" src="data:image/png;base64,%s" alt="base" data-matte-document="%s" data-matte-white="%s" data-matte-dark="%s" data-matte-checkerboard="%s">`,
		entry.RefDocB64,
		entry.RefDocB64,
		entry.RefWhiteB64,
		entry.RefDarkB64,
		entry.RefCheckB64,
	)
	fmt.Fprintf(
		w,
		`<div class="slider-overlay"><img class="matte-target" src="data:image/png;base64,%s" alt="overlay" data-matte-document="%s" data-matte-white="%s" data-matte-dark="%s" data-matte-checkerboard="%s"></div>`,
		entry.ActDocB64,
		entry.ActDocB64,
		entry.ActWhiteB64,
		entry.ActDarkB64,
		entry.ActCheckB64,
	)
	fmt.Fprint(w, `<div class="slider-divider"></div></div></div>`)

	fmt.Fprint(w, `<div class="img-col col-amp">`)
	fmt.Fprint(w, `<label>Diff (amplified)</label>`)
	fmt.Fprintf(w, `<img class="parity-image" src="data:image/png;base64,%s" alt="amplified-diff">`, entry.AmpDiffB64)
	fmt.Fprint(w, `</div>`)

	fmt.Fprint(w, `<div class="img-col col-raw" style="display:none">`)
	fmt.Fprint(w, `<label>Diff (raw)</label>`)
	fmt.Fprintf(w, `<img class="parity-image" src="data:image/png;base64,%s" alt="raw-diff">`, entry.RawDiffB64)
	fmt.Fprint(w, `</div>`)

	fmt.Fprint(w, `</div></div></div>`)
}

func badgeClass(rmse float64) string {
	if rmse <= 5 {
		return "badge-ok"
	}

	if rmse <= 20 {
		return "badge-warn"
	}

	return "badge-bad"
}

func badgeClassAvgDiff(v float64) string {
	if v <= 2 {
		return "badge-ok"
	}

	if v <= 8 {
		return "badge-warn"
	}

	return "badge-bad"
}

func badgeClassMaxDiff(v uint8) string {
	if v <= 10 {
		return "badge-ok"
	}

	if v <= 40 {
		return "badge-warn"
	}

	return "badge-bad"
}

func badgeClassDiffRatio(r float64) string {
	if r <= 0.01 {
		return "badge-ok"
	}

	if r <= 0.05 {
		return "badge-warn"
	}

	return "badge-bad"
}

const pageHeader = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>KnobMan Parity Viewer</title>
<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
body { background: #101216; color: #d7dce4; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 13px; }
.sticky-header {
  position: sticky; top: 0; z-index: 100;
  background: rgba(16,18,22,0.96); backdrop-filter: blur(10px);
  border-bottom: 1px solid #2d3440; padding: 10px 12px;
}
.controls { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.controls h1 { font-size: 15px; color: #f4f7fb; margin-right: 8px; }
.controls input, .controls select {
  background: #171b22; color: #d7dce4; border: 1px solid #334155; padding: 5px 8px;
  font-family: inherit; font-size: 12px; border-radius: 6px;
}
.controls label { display: flex; align-items: center; gap: 5px; font-size: 12px; color: #aeb8c8; }
.header-meta { margin-top: 8px; color: #93a1b5; font-size: 12px; }
#summary { margin-left: auto; color: #93a1b5; font-size: 12px; }
.container { padding: 12px; }
.card {
  background: #141922; border: 1px solid #293241; margin-bottom: 10px;
  border-radius: 10px; overflow: hidden; box-shadow: 0 8px 24px rgba(0,0,0,0.2);
}
.card-header {
  padding: 9px 12px; cursor: pointer; display: flex; align-items: center; gap: 8px;
  background: #181f2a; user-select: none;
}
.card-header:hover { background: #1d2531; }
.card-body { display: none; padding: 12px; }
.card.open .card-body { display: block; }
.card-title { font-size: 13px; color: #f4f7fb; flex: 1; }
.card-meta { color: #93a1b5; margin-bottom: 10px; }
.right-badges { display: flex; gap: 6px; align-items: center; flex-wrap: wrap; }
.badge { padding: 2px 7px; border-radius: 999px; font-size: 11px; font-weight: bold; }
.badge-neutral { background: #202734; color: #d7dce4; border: 1px solid #425168; }
.badge-ok { background: #163323; color: #7cf0a5; border: 1px solid #2e7250; }
.badge-warn { background: #3a2d13; color: #ffc66d; border: 1px solid #6f5521; }
.badge-bad { background: #3b1619; color: #ff8a8f; border: 1px solid #7c2d35; }
.rerender-btn {
  margin-left: 10px; padding: 5px 10px; border-radius: 8px; border: 1px solid #425168;
  background: #202734; color: #d7dce4; cursor: pointer; font: inherit;
}
.rerender-btn:hover { background: #273244; }
.rerender-btn:disabled { opacity: 0.6; cursor: wait; }
.img-grid {
  display: grid; grid-template-columns: repeat(5, minmax(0, 1fr)); gap: 10px;
  align-items: start; overflow-x: auto;
}
.img-col { display: flex; flex-direction: column; gap: 6px; min-width: 0; overflow: auto; }
.img-col label { font-size: 11px; color: #93a1b5; text-align: center; }
.parity-image {
  display: block; image-rendering: auto; width: 100%; height: auto; border-radius: 6px;
  background-color: #0c1016; background-image: none; max-width: 100%;
}
.resample-pixelated .parity-image { image-rendering: pixelated; }
.container.matte-document .parity-image, .container.matte-document .slider-wrap { background-color: var(--doc-bg, #ffffff); }
.container.matte-white .parity-image, .container.matte-white .slider-wrap { background-color: #ffffff; }
.container.matte-dark .parity-image, .container.matte-dark .slider-wrap { background-color: #0c1016; }
.container.matte-checkerboard .parity-image, .container.matte-checkerboard .slider-wrap {
  background-color: #d8dee9;
  background-image:
    linear-gradient(45deg, #c3cad6 25%, transparent 25%, transparent 75%, #c3cad6 75%, #c3cad6),
    linear-gradient(45deg, #c3cad6 25%, transparent 25%, transparent 75%, #c3cad6 75%, #c3cad6);
  background-position: 0 0, 6px 6px;
  background-size: 12px 12px;
}
.original-size .img-grid { grid-template-columns: repeat(5, max-content); }
.original-size .img-col { min-width: max-content; }
.original-size .parity-image, .original-size .slider-wrap { align-self: flex-start; }
.original-size .parity-image { width: auto; height: auto; max-width: none; }
.col-raw { display: none; }
.slider-wrap {
  position: relative; overflow: hidden; width: 100%; cursor: col-resize; border-radius: 6px;
  background-color: #0c1016; background-image: none;
}
.slider-wrap img.base { display: block; image-rendering: auto; width: 100%; height: auto; }
.resample-pixelated .slider-wrap img.base { image-rendering: pixelated; }
.original-size .slider-wrap { width: auto; }
.original-size .slider-wrap img.base { width: auto; height: auto; max-width: none; }
.slider-overlay { position: absolute; top: 0; left: 0; height: 100%; overflow: hidden; width: 50%; }
.slider-overlay img { display: block; position: absolute; top: 0; left: 0; image-rendering: auto; width: 200%; }
.resample-pixelated .slider-overlay img { image-rendering: pixelated; }
.original-size .slider-overlay img { width: auto !important; }
.slider-divider {
  position: absolute; top: 0; left: 50%; height: 100%; width: 3px;
  background: #f8fafc; cursor: col-resize; transform: translateX(-50%);
}
.slider-divider::before {
  content: ''; position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%);
  width: 18px; height: 18px; border-radius: 50%; background: #f8fafc; border: 2px solid #111827;
}
.empty-state {
  border: 1px dashed #425168; border-radius: 10px; padding: 24px;
  color: #93a1b5; background: #141922;
}
code { color: #f4f7fb; }
@media (max-width: 1400px) {
  .img-grid { grid-template-columns: repeat(2, 1fr); }
}
@media (max-width: 800px) {
  .img-grid { grid-template-columns: 1fr; }
}
</style>
</head>
<body>
<div class="sticky-header">
  <div class="controls">
    <h1>KnobMan Parity Viewer</h1>
    <input type="text" id="search" placeholder="Search case…" oninput="filterCards()" style="width:180px">
    <select id="suite-filter" onchange="filterCards()">
      <option value="">Suite: all</option>
      <option value="samples">Suite: samples</option>
      <option value="primitives">Suite: primitives</option>
      <option value="animated">Suite: animated</option>
      <option value="animated-samples">Suite: animated-samples</option>
    </select>
    <select id="baseline-filter" onchange="filterCards()">
      <option value="">Baseline: all</option>
      <option value="baseline-java" selected>Baseline: java</option>
      <option value="baseline-go">Baseline: go</option>
    </select>
    <select id="sort-select" onchange="sortCards()">
      <option value="rmse-desc">Sort: RMSE ↓</option>
      <option value="rmse-asc">Sort: RMSE ↑</option>
      <option value="diff-pixels-desc">Sort: Different pixels ↓</option>
      <option value="diff-pixels-asc">Sort: Different pixels ↑</option>
      <option value="diff-ratio-desc">Sort: Different ratio ↓</option>
      <option value="diff-ratio-asc">Sort: Different ratio ↑</option>
      <option value="avg-diff-desc">Sort: Avg diff ↓</option>
      <option value="avg-diff-asc">Sort: Avg diff ↑</option>
      <option value="max-diff-desc">Sort: Max diff ↓</option>
      <option value="max-diff-asc">Sort: Max diff ↑</option>
      <option value="name-asc">Sort: Name ↑</option>
    </select>
    <select id="diff-mode" onchange="setDiffMode(this.value)">
      <option value="amp">Diff: amplified</option>
      <option value="raw">Diff: raw</option>
      <option value="both">Diff: both</option>
    </select>
    <select id="matte-mode" onchange="setMatteMode(this.value)">
      <option value="document" selected>Matte: document bg</option>
      <option value="white">Matte: white</option>
      <option value="dark">Matte: dark</option>
      <option value="checkerboard">Matte: checkerboard</option>
    </select>
    <select id="resample-mode" onchange="setResampleMode(this.value)">
      <option value="smooth">Scaling: smooth</option>
      <option value="pixelated">Scaling: pixelated</option>
    </select>
    <button id="rerender-all-btn" class="rerender-btn" type="button">Re-render Artifacts</button>
    <label><input type="checkbox" id="original-size" onchange="setOriginalSize(this.checked)"> Original size</label>
    <span id="summary"></span>
  </div>
`

const pageFooter = `</div>
<script>
(function() {
  function metric(card, attr) {
    return parseFloat(card.dataset[attr] || 0);
  }

  function filterCards() {
    var q = document.getElementById('search').value.toLowerCase();
    var suite = document.getElementById('suite-filter').value;
    var baseline = document.getElementById('baseline-filter').value;
    document.querySelectorAll('.card').forEach(function(card) {
      var name = (card.dataset.name || '').toLowerCase();
      var cardSuite = card.dataset.suite || '';
      var cardBaseline = card.dataset.baseline || '';
      var visible = name.includes(q);
      if (suite && cardSuite !== suite) visible = false;
      if (baseline && cardBaseline !== baseline) visible = false;
      card.style.display = visible ? '' : 'none';
    });
    updateSummary();
  }

  function sortCards() {
    var mode = document.getElementById('sort-select').value;
    var container = document.getElementById('cards-container');
    var cards = Array.from(container.querySelectorAll('.card'));
    cards.sort(function(a, b) {
      if (mode === 'rmse-desc') return metric(b, 'rmse') - metric(a, 'rmse');
      if (mode === 'rmse-asc') return metric(a, 'rmse') - metric(b, 'rmse');
      if (mode === 'diff-pixels-desc') return metric(b, 'diffPixels') - metric(a, 'diffPixels');
      if (mode === 'diff-pixels-asc') return metric(a, 'diffPixels') - metric(b, 'diffPixels');
      if (mode === 'diff-ratio-desc') return metric(b, 'diffRatio') - metric(a, 'diffRatio');
      if (mode === 'diff-ratio-asc') return metric(a, 'diffRatio') - metric(b, 'diffRatio');
      if (mode === 'avg-diff-desc') return metric(b, 'avgDiff') - metric(a, 'avgDiff');
      if (mode === 'avg-diff-asc') return metric(a, 'avgDiff') - metric(b, 'avgDiff');
      if (mode === 'max-diff-desc') return metric(b, 'maxDiff') - metric(a, 'maxDiff');
      if (mode === 'max-diff-asc') return metric(a, 'maxDiff') - metric(b, 'maxDiff');
      return (a.dataset.name || '').localeCompare(b.dataset.name || '');
    });
    cards.forEach(function(card) { container.appendChild(card); });
    updateSortMetricBadges(mode);
  }

  function badgeColorClass(attr, value, diffRatio) {
    if (attr === 'rmse') {
      if (value <= 5) return 'badge-ok';
      if (value <= 20) return 'badge-warn';
      return 'badge-bad';
    }
    if (attr === 'avgDiff') {
      if (value <= 2) return 'badge-ok';
      if (value <= 8) return 'badge-warn';
      return 'badge-bad';
    }
    if (attr === 'maxDiff') {
      if (value <= 10) return 'badge-ok';
      if (value <= 40) return 'badge-warn';
      return 'badge-bad';
    }
    if (attr === 'diffPixels' || attr === 'diffRatio') {
      var r = parseFloat(diffRatio || 0);
      if (r <= 0.01) return 'badge-ok';
      if (r <= 0.05) return 'badge-warn';
      return 'badge-bad';
    }
    return 'badge-neutral';
  }

  function updateSortMetricBadges(mode) {
    var label = '';
    var attr = '';
    var formatter = function(v) { return String(v); };
    if (mode === 'rmse-desc' || mode === 'rmse-asc') {
      label = 'RMSE'; attr = 'rmse'; formatter = function(v) { return Number(v).toFixed(2); };
    } else if (mode === 'diff-pixels-desc' || mode === 'diff-pixels-asc') {
      label = 'diff px'; attr = 'diffPixels'; formatter = function(v) { return String(Math.round(Number(v))); };
    } else if (mode === 'diff-ratio-desc' || mode === 'diff-ratio-asc') {
      label = 'diff %'; attr = 'diffRatio'; formatter = function(v) { return (Number(v) * 100).toFixed(2); };
    } else if (mode === 'avg-diff-desc' || mode === 'avg-diff-asc') {
      label = 'avg'; attr = 'avgDiff'; formatter = function(v) { return Number(v).toFixed(2); };
    } else if (mode === 'max-diff-desc' || mode === 'max-diff-asc') {
      label = 'max'; attr = 'maxDiff'; formatter = function(v) { return String(Math.round(Number(v))); };
    }

    document.querySelectorAll('.card').forEach(function(card) {
      var badge = card.querySelector('.sort-metric-badge');
      if (!badge) return;
      if (!attr) {
        badge.style.display = 'none';
        return;
      }
      var value = parseFloat(card.dataset[attr] || 0);
      badge.className = 'badge ' + badgeColorClass(attr, value, card.dataset.diffRatio) + ' sort-metric-badge';
      badge.textContent = label + ' ' + formatter(value);
      badge.style.display = '';
    });
  }

  function setDiffMode(mode) {
    document.querySelectorAll('.col-amp').forEach(function(el) {
      el.style.display = (mode === 'amp' || mode === 'both') ? 'flex' : 'none';
    });
    document.querySelectorAll('.col-raw').forEach(function(el) {
      el.style.display = (mode === 'raw' || mode === 'both') ? 'flex' : 'none';
    });
  }

  function updateSummary() {
    var all = document.querySelectorAll('.card');
    var visible = Array.from(all).filter(function(card) { return card.style.display !== 'none'; });
    document.getElementById('summary').textContent = visible.length + ' / ' + all.length + ' cases';
  }

  function setRerenderButtonsDisabled(disabled) {
    var buttons = document.querySelectorAll('.rerender-btn');
    buttons.forEach(function(button) {
      button.disabled = disabled;
    });
  }

  function rerenderArtifact(suite, name) {
    return fetch('/rerender', {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8' },
      body: new URLSearchParams({ suite: suite, name: name }).toString()
    }).then(function(response) {
      if (!response.ok) {
        return response.text().then(function(text) {
          throw new Error(text || 'rerender failed');
        });
      }
    });
  }

  document.querySelectorAll('.card-header').forEach(function(header) {
    header.addEventListener('click', function() {
      header.closest('.card').classList.toggle('open');
    });
  });

  document.querySelectorAll('.rerender-btn').forEach(function(button) {
    button.addEventListener('click', function(event) {
      event.stopPropagation();
      if (button.id === 'rerender-all-btn') {
        return;
      }
      var suite = button.dataset.suite || '';
      var name = button.dataset.name || '';
      setRerenderButtonsDisabled(true);
      return rerenderArtifact(suite, name).then(function() {
        window.location.reload();
      }).catch(function(err) {
        window.alert(err.message);
        setRerenderButtonsDisabled(false);
      });
    });
  });

  var bulkButton = document.getElementById('rerender-all-btn');
  bulkButton.addEventListener('click', function() {
    var cards = Array.from(document.querySelectorAll('.card'));
    setRerenderButtonsDisabled(true);

    var chain = Promise.resolve();
    cards.forEach(function(card) {
      chain = chain.then(function() {
        return rerenderArtifact(card.dataset.suite || '', card.dataset.name || '');
      });
    });

    chain.then(function() {
      window.location.reload();
    }).catch(function(err) {
      window.alert(err.message);
      setRerenderButtonsDisabled(false);
    });
  });

  document.querySelectorAll('.slider-wrap').forEach(function(wrap) {
    var divider = wrap.querySelector('.slider-divider');
    var overlay = wrap.querySelector('.slider-overlay');
    var dragging = false;

    function updateSlider(clientX) {
      var rect = wrap.getBoundingClientRect();
      var pct = Math.max(0, Math.min(1, (clientX - rect.left) / rect.width));
      overlay.style.width = (pct * 100) + '%';
      divider.style.left = (pct * 100) + '%';
      if (pct > 0) {
        overlay.querySelector('img').style.width = (100 / pct) + '%';
      } else {
        overlay.querySelector('img').style.width = '100%';
      }
    }

    divider.addEventListener('mousedown', function(e) {
      dragging = true;
      e.preventDefault();
    });
    document.addEventListener('mousemove', function(e) {
      if (dragging) updateSlider(e.clientX);
    });
    document.addEventListener('mouseup', function() { dragging = false; });
    wrap.addEventListener('click', function(e) { updateSlider(e.clientX); });
  });

  function setOriginalSize(on) {
    var container = document.getElementById('cards-container');
    if (on) {
      container.classList.add('original-size');
    } else {
      container.classList.remove('original-size');
    }
  }

  function setResampleMode(mode) {
    var container = document.getElementById('cards-container');
    container.classList.remove('resample-smooth', 'resample-pixelated');
    container.classList.add(mode === 'pixelated' ? 'resample-pixelated' : 'resample-smooth');
  }

  function setMatteMode(mode) {
    var selected = mode;
    if (selected !== 'document' && selected !== 'white' && selected !== 'dark' && selected !== 'checkerboard') {
      selected = 'document';
    }
    var dataKey = 'matte' + selected.charAt(0).toUpperCase() + selected.slice(1);
    document.querySelectorAll('.matte-target').forEach(function(img) {
      var b64 = img.dataset[dataKey];
      if (!b64) return;
      img.src = 'data:image/png;base64,' + b64;
    });
  }

  window.filterCards = filterCards;
  window.sortCards = sortCards;
  window.setDiffMode = setDiffMode;
  window.setOriginalSize = setOriginalSize;
  window.setResampleMode = setResampleMode;
  window.setMatteMode = setMatteMode;

  sortCards();
  setDiffMode(document.getElementById('diff-mode').value);
  setMatteMode(document.getElementById('matte-mode').value);
  setResampleMode(document.getElementById('resample-mode').value);
  filterCards();
})();
</script>
</body>
</html>
`
