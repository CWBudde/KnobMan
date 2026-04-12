package main

import (
	"bytes"
	"encoding/base64"
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
	"path/filepath"
	"sort"
	"strings"
)

type caseEntry struct {
	Suite       string
	Baseline    string
	Name        string
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

	addr := ":" + *port
	log.Printf("Parity viewer running at http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func loadCases(parityDir string) (loadResult, error) {
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
			artifactDir := filepath.Join(suitePath, "artifacts", baselineName)

			baselines, err := filepath.Glob(filepath.Join(baselineDir, "*.png"))
			if err != nil {
				return loadResult{}, fmt.Errorf("glob baselines in %s: %w", baselineDir, err)
			}
			sort.Strings(baselines)

			for _, baselinePath := range baselines {
				name := strings.TrimSuffix(filepath.Base(baselinePath), filepath.Ext(baselinePath))
				artifactPath := filepath.Join(artifactDir, name+".png")
				if _, err := os.Stat(artifactPath); err != nil {
					result.MissingArtifactCnt++
					continue
				}

				entry, err := buildEntry(suiteName, baselineName, name, baselinePath, artifactPath)
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

func buildEntry(suite, baseline, name, baselinePath, artifactPath string) (caseEntry, error) {
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

	return caseEntry{
		Suite:       suite,
		Baseline:    baseline,
		Name:        name,
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
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func readPNGAsRGBA(path string) (*image.RGBA, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	return imageToRGBA(img), nil
}

func imageToRGBA(img image.Image) *image.RGBA {
	if img == nil {
		return nil
	}
	b := img.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))

	switch src := img.(type) {
	case *image.RGBA:
		for y := 0; y < b.Dy(); y++ {
			srcY := b.Min.Y + y
			for x := 0; x < b.Dx(); x++ {
				srcX := b.Min.X + x
				srcOffset := srcY*src.Stride + srcX*4
				a := uint32(src.Pix[srcOffset+3])
				dstOffset := y*out.Stride + x*4
				if a == 0 {
					out.Pix[dstOffset+0] = 0
					out.Pix[dstOffset+1] = 0
					out.Pix[dstOffset+2] = 0
					out.Pix[dstOffset+3] = 0
					continue
				}
				r := uint32(src.Pix[srcOffset+0])
				g := uint32(src.Pix[srcOffset+1])
				bl := uint32(src.Pix[srcOffset+2])
				out.Pix[dstOffset+0] = uint8((r*255 + a/2) / a)
				out.Pix[dstOffset+1] = uint8((g*255 + a/2) / a)
				out.Pix[dstOffset+2] = uint8((bl*255 + a/2) / a)
				out.Pix[dstOffset+3] = uint8(a)
			}
		}
		return out
	case *image.NRGBA:
		for y := 0; y < b.Dy(); y++ {
			srcY := b.Min.Y + y
			srcOffset := srcY*src.Stride + b.Min.X*4
			dstOffset := y * out.Stride
			copy(out.Pix[dstOffset:dstOffset+b.Dx()*4], src.Pix[srcOffset:srcOffset+b.Dx()*4])
		}
		return out
	}

	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := color.NRGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			dst := y*out.Stride + x*4
			out.Pix[dst+0] = c.R
			out.Pix[dst+1] = c.G
			out.Pix[dst+2] = c.B
			out.Pix[dst+3] = c.A
		}
	}
	return out
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

	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd, nil
		}
		next := filepath.Dir(wd)
		if next == wd {
			break
		}
		wd = next
	}

	return "", fmt.Errorf("go.mod not found from cwd")
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
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
	if entry.Baseline == "baseline-java" {
		refLabel = "Java Golden"
	} else if entry.Baseline == "baseline-go" {
		refLabel = "Go Baseline"
	}

	fmt.Fprintf(
		w,
		`<div class="card" data-name="%s" data-suite="%s" data-baseline="%s" data-rmse="%.4f" data-avg-diff="%.4f" data-max-diff="%d" data-diff-pixels="%d" data-diff-ratio="%.6f">`,
		esc(entry.Name),
		esc(entry.Suite),
		esc(entry.Baseline),
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
	fmt.Fprint(w, `</div><div class="img-grid">`)

	fmt.Fprint(w, `<div class="img-col">`)
	fmt.Fprintf(w, `<label>%s</label>`, esc(refLabel))
	fmt.Fprintf(w, `<img src="data:image/png;base64,%s" alt="baseline">`, entry.RefB64)
	fmt.Fprint(w, `</div>`)

	fmt.Fprint(w, `<div class="img-col">`)
	fmt.Fprint(w, `<label>Artifact</label>`)
	fmt.Fprintf(w, `<img src="data:image/png;base64,%s" alt="artifact">`, entry.ActB64)
	fmt.Fprint(w, `</div>`)

	fmt.Fprint(w, `<div class="img-col">`)
	fmt.Fprint(w, `<label>Overlay</label>`)
	fmt.Fprint(w, `<div class="slider-wrap">`)
	fmt.Fprintf(w, `<img class="base" src="data:image/png;base64,%s" alt="base">`, entry.RefB64)
	fmt.Fprintf(w, `<div class="slider-overlay"><img src="data:image/png;base64,%s" alt="overlay"></div>`, entry.ActB64)
	fmt.Fprint(w, `<div class="slider-divider"></div></div></div>`)

	fmt.Fprint(w, `<div class="img-col col-amp">`)
	fmt.Fprint(w, `<label>Diff (amplified)</label>`)
	fmt.Fprintf(w, `<img src="data:image/png;base64,%s" alt="amplified-diff">`, entry.AmpDiffB64)
	fmt.Fprint(w, `</div>`)

	fmt.Fprint(w, `<div class="img-col col-raw" style="display:none">`)
	fmt.Fprint(w, `<label>Diff (raw)</label>`)
	fmt.Fprintf(w, `<img src="data:image/png;base64,%s" alt="raw-diff">`, entry.RawDiffB64)
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
.img-grid { display: grid; grid-template-columns: repeat(5, 1fr); gap: 10px; }
.img-col { display: flex; flex-direction: column; gap: 6px; overflow: auto; }
.img-col label { font-size: 11px; color: #93a1b5; text-align: center; }
.img-col img { display: block; image-rendering: auto; width: 100%; height: auto; border-radius: 6px; background: #0c1016; }
.original-size .img-col img { width: auto; height: auto; max-width: none; }
.col-raw { display: none; }
.slider-wrap { position: relative; overflow: hidden; width: 100%; cursor: col-resize; border-radius: 6px; background: #0c1016; }
.slider-wrap img.base { display: block; image-rendering: auto; width: 100%; height: auto; }
.original-size .slider-wrap img.base { width: auto; height: auto; max-width: none; }
.slider-overlay { position: absolute; top: 0; left: 0; height: 100%; overflow: hidden; width: 50%; }
.slider-overlay img { display: block; position: absolute; top: 0; left: 0; image-rendering: auto; width: 200%; }
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
    </select>
    <select id="baseline-filter" onchange="filterCards()">
      <option value="">Baseline: all</option>
      <option value="baseline-java">Baseline: java</option>
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

  document.querySelectorAll('.card-header').forEach(function(header) {
    header.addEventListener('click', function() {
      header.closest('.card').classList.toggle('open');
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

  window.filterCards = filterCards;
  window.sortCards = sortCards;
  window.setDiffMode = setDiffMode;
  window.setOriginalSize = setOriginalSize;

  updateSortMetricBadges(document.getElementById('sort-select').value);
  updateSummary();
})();
</script>
</body>
</html>
`

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
