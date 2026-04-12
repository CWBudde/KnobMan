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
	samplesDir := flag.String("samples", filepath.Join("assets", "samples"), "Directory with sample .knob files")
	refDir := flag.String("refs", filepath.Join("tests", "parity", "samples", "baseline-go"), "Directory to write reference PNGs")
	frame := flag.Int("frame", 0, "Frame index to render")
	overwrite := flag.Bool("overwrite", false, "Overwrite existing reference images")
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

	for _, sample := range paths {
		name := strings.TrimSuffix(filepath.Base(sample), filepath.Ext(sample))
		refPath := filepath.Join(*refDir, name+".png")

		doc, textures, err := render.LoadParityDocument(sample, root)
		if err != nil {
			log.Fatalf("load sample %s: %v", sample, err)
		}

		out := render.NewPixBuf(doc.Prefs.PWidth.Val, doc.Prefs.PHeight.Val)
		if out == nil {
			log.Fatalf("allocate buffer for %s", sample)
		}

		render.RenderFrame(out, doc, *frame, textures)
		if !*overwrite {
			if fileExists(refPath) {
				continue
			}
		}

		if err := render.WritePixBufPNG(refPath, out); err != nil {
			log.Fatalf("write %s: %v", refPath, err)
		}
		fmt.Println(refPath)
	}
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
