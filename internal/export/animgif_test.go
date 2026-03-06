package export

import (
	"bytes"
	"image/gif"
	"testing"

	"knobman/internal/model"
)

func TestExportGIFBasic(t *testing.T) {
	doc := model.NewDocument()
	doc.Prefs.PWidth.Val = 9
	doc.Prefs.PHeight.Val = 5
	doc.Prefs.Width = 9
	doc.Prefs.Height = 5
	doc.Prefs.RenderFrames.Val = 4
	doc.Prefs.Duration.Val = 40
	doc.Prefs.Loop.Val = 2
	doc.Prefs.BiDir.Val = 0

	data, err := ExportGIF(doc, nil)
	if err != nil {
		t.Fatalf("gif export failed: %v", err)
	}

	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("gif decode failed: %v", err)
	}

	if got, want := len(g.Image), 4; got != want {
		t.Fatalf("gif frame count: got %d want %d", got, want)
	}

	for i, img := range g.Image {
		if got, want := img.Bounds().Dx(), 9; got != want {
			t.Fatalf("frame %d width: got %d want %d", i, got, want)
		}

		if got, want := img.Bounds().Dy(), 5; got != want {
			t.Fatalf("frame %d height: got %d want %d", i, got, want)
		}
	}

	if got, want := g.Delay[0], 4; got != want {
		t.Fatalf("gif delay: got %d want %d", got, want)
	}

	if got, want := g.LoopCount, 2; got != want {
		t.Fatalf("gif loop count: got %d want %d", got, want)
	}
}

func TestGIFFrameSequenceBiDir(t *testing.T) {
	seq := gifFrameSequence(5, true)

	want := []int{0, 1, 2, 3, 4, 3, 2, 1}
	if len(seq) != len(want) {
		t.Fatalf("seq len: got %d want %d", len(seq), len(want))
	}

	for i := range want {
		if seq[i] != want[i] {
			t.Fatalf("seq[%d]: got %d want %d", i, seq[i], want[i])
		}
	}
}
