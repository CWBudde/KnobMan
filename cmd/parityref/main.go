package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"knobman/internal/render"
)

func main() {
	inputPath := flag.String("input", "", "Single .knob file to render")
	outputPath := flag.String("output", "", "Single PNG file to write when --input is used")
	samplesDir := flag.String("samples", filepath.Join("assets", "samples"), "Directory with sample .knob files")
	refDir := flag.String("refs", filepath.Join("tests", "parity", "samples", "baseline-go"), "Directory to write reference PNGs")
	names := flag.String("names", "", "Comma-separated .knob basenames to render from --samples")
	keyframes := flag.String("keyframes", "", "Comma-separated keyframes to render: first,mid,last")
	frame := flag.Int("frame", 0, "Frame index to render")
	overwrite := flag.Bool("overwrite", false, "Overwrite existing reference images")
	transparentBG := flag.Bool("transparent-bg", false, "Force document background alpha to 0 before rendering")

	flag.Parse()

	root, err := detectRepoRoot()
	if err != nil {
		log.Fatalf("detect repo root: %v", err)
	}

	keyframeSpecs, err := parseKeyframes(*keyframes)
	if err != nil {
		log.Fatalf("parse keyframes: %v", err)
	}

	if *inputPath != "" {
		if *outputPath == "" {
			log.Fatal("--output is required when --input is used")
		}

		if len(keyframeSpecs) != 0 {
			err := renderKeyframes(root, *inputPath, *outputPath, keyframeSpecs, *transparentBG)
			if err != nil {
				log.Fatalf("render %s: %v", *inputPath, err)
			}

			for _, spec := range keyframeSpecs {
				fmt.Println(keyframeOutputPath(*outputPath, spec.name))
			}

			return
		}

		err := renderOne(root, *inputPath, *outputPath, *frame, *transparentBG)
		if err != nil {
			log.Fatalf("render %s: %v", *inputPath, err)
		}

		fmt.Println(*outputPath)

		return
	}

	paths, err := filepath.Glob(filepath.Join(*samplesDir, "*.knob"))
	if err != nil {
		log.Fatalf("glob samples: %v", err)
	}

	if len(paths) == 0 {
		log.Fatalf("no sample .knob files found in %s", *samplesDir)
	}

	paths = filterSamplePaths(paths, parseNames(*names))
	sort.Strings(paths)

	for _, sample := range paths {
		name := strings.TrimSuffix(filepath.Base(sample), filepath.Ext(sample))
		if len(keyframeSpecs) != 0 {
			err := renderSampleKeyframes(root, sample, *refDir, name, keyframeSpecs, *overwrite, *transparentBG)
			if err != nil {
				log.Fatalf("render %s: %v", sample, err)
			}

			for _, spec := range keyframeSpecs {
				fmt.Println(filepath.Join(*refDir, name+"__"+spec.name+".png"))
			}

			continue
		}

		refPath := filepath.Join(*refDir, name+".png")
		if !*overwrite && fileExists(refPath) {
			continue
		}

		err := renderOne(root, sample, refPath, *frame, *transparentBG)
		if err != nil {
			log.Fatalf("render %s: %v", sample, err)
		}

		fmt.Println(refPath)
	}
}

type keyframeSpec struct {
	name string
}

func renderOne(root, samplePath, outputPath string, frame int, transparentBG bool) error {
	doc, textures, err := render.LoadParityDocument(samplePath, root)
	if err != nil {
		return fmt.Errorf("load sample: %w", err)
	}

	if transparentBG {
		doc.Prefs.BkColor.Val.A = 0
	}

	out := render.NewPixBuf(doc.Prefs.PWidth.Val, doc.Prefs.PHeight.Val)
	if out == nil {
		return errors.New("allocate buffer")
	}

	render.RenderFrame(out, doc, frame, textures)

	err = render.WritePixBufPNG(outputPath, out)
	if err != nil {
		return fmt.Errorf("write png: %w", err)
	}

	return nil
}

func renderSampleKeyframes(root, samplePath, outputDir, name string, keyframes []keyframeSpec, overwrite, transparentBG bool) error {
	doc, textures, err := render.LoadParityDocument(samplePath, root)
	if err != nil {
		return fmt.Errorf("load sample: %w", err)
	}

	if transparentBG {
		doc.Prefs.BkColor.Val.A = 0
	}

	for _, spec := range keyframes {
		outputPath := filepath.Join(outputDir, name+"__"+spec.name+".png")
		if !overwrite && fileExists(outputPath) {
			continue
		}

		out := render.NewPixBuf(doc.Prefs.PWidth.Val, doc.Prefs.PHeight.Val)
		if out == nil {
			return errors.New("allocate buffer")
		}

		render.RenderFrame(out, doc, spec.frameIndex(doc.Prefs.RenderFrames.Val), textures)

		err := render.WritePixBufPNG(outputPath, out)
		if err != nil {
			return fmt.Errorf("write png: %w", err)
		}
	}

	return nil
}

func renderKeyframes(root, samplePath, outputPath string, keyframes []keyframeSpec, transparentBG bool) error {
	doc, textures, err := render.LoadParityDocument(samplePath, root)
	if err != nil {
		return fmt.Errorf("load sample: %w", err)
	}

	if transparentBG {
		doc.Prefs.BkColor.Val.A = 0
	}

	for _, spec := range keyframes {
		out := render.NewPixBuf(doc.Prefs.PWidth.Val, doc.Prefs.PHeight.Val)
		if out == nil {
			return errors.New("allocate buffer")
		}

		render.RenderFrame(out, doc, spec.frameIndex(doc.Prefs.RenderFrames.Val), textures)

		err := render.WritePixBufPNG(keyframeOutputPath(outputPath, spec.name), out)
		if err != nil {
			return fmt.Errorf("write png: %w", err)
		}
	}

	return nil
}

func parseKeyframes(raw string) ([]keyframeSpec, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")

	out := make([]keyframeSpec, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		switch name {
		case "first", "mid", "last":
			out = append(out, keyframeSpec{name: name})
		case "":
			continue
		default:
			return nil, fmt.Errorf("unsupported keyframe %q", name)
		}
	}

	return out, nil
}

func parseNames(raw string) map[string]struct{} {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	out := make(map[string]struct{})

	for part := range strings.SplitSeq(raw, ",") {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}

		out[name] = struct{}{}
	}

	return out
}

func filterSamplePaths(paths []string, allowed map[string]struct{}) []string {
	if len(allowed) == 0 {
		return paths
	}

	out := make([]string, 0, len(paths))
	for _, path := range paths {
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if _, ok := allowed[name]; ok {
			out = append(out, path)
		}
	}

	return out
}

func keyframeOutputPath(baseOutput, keyframe string) string {
	ext := filepath.Ext(baseOutput)
	if ext == "" {
		return baseOutput + "__" + keyframe + ".png"
	}

	stem := strings.TrimSuffix(baseOutput, ext)

	return stem + "__" + keyframe + ext
}

func (k keyframeSpec) frameIndex(totalFrames int) int {
	if totalFrames <= 1 {
		return 0
	}

	switch k.name {
	case "mid":
		return totalFrames / 2
	case "last":
		return totalFrames - 1
	default:
		return 0
	}
}

func detectRepoRoot() (string, error) {
	wd, err := filepath.Abs(".")
	if err != nil {
		return "", err
	}

	for range 8 {
		if fileExists(filepath.Join(wd, "go.mod")) {
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
