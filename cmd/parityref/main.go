package main

import (
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
	frame := flag.Int("frame", 0, "Frame index to render")
	overwrite := flag.Bool("overwrite", false, "Overwrite existing reference images")
	transparentBG := flag.Bool("transparent-bg", false, "Force document background alpha to 0 before rendering")
	flag.Parse()

	paths, err := filepath.Glob(filepath.Join(*samplesDir, "*.knob"))
	if err != nil {
		log.Fatalf("glob samples: %v", err)
	}
	if len(paths) == 0 {
		log.Fatalf("no sample .knob files found in %s", *samplesDir)
	}
	sort.Strings(paths)

	root, err := detectRepoRoot()
	if err != nil {
		log.Fatalf("detect repo root: %v", err)
	}

	if *inputPath != "" {
		if *outputPath == "" {
			log.Fatal("--output is required when --input is used")
		}
		if err := renderOne(root, *inputPath, *outputPath, *frame, *transparentBG); err != nil {
			log.Fatalf("render %s: %v", *inputPath, err)
		}
		fmt.Println(*outputPath)
		return
	}

	for _, sample := range paths {
		name := strings.TrimSuffix(filepath.Base(sample), filepath.Ext(sample))
		refPath := filepath.Join(*refDir, name+".png")

		if !*overwrite {
			if fileExists(refPath) {
				continue
			}
		}

		if err := renderOne(root, sample, refPath, *frame, *transparentBG); err != nil {
			log.Fatalf("render %s: %v", sample, err)
		}
		fmt.Println(refPath)
	}
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
		return fmt.Errorf("allocate buffer")
	}

	render.RenderFrame(out, doc, frame, textures)
	if err := render.WritePixBufPNG(outputPath, out); err != nil {
		return fmt.Errorf("write png: %w", err)
	}
	return nil
}

func detectRepoRoot() (string, error) {
	wd, err := filepath.Abs(".")
	if err != nil {
		return "", err
	}

	for i := 0; i < 8; i++ {
		if fileExists(filepath.Join(wd, "go.mod")) {
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
