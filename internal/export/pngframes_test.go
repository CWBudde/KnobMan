package export

import (
	"bytes"
	"image/png"
	"testing"

	"knobman/internal/model"
)

func TestExportPNGFramesCountAndDimensions(t *testing.T) {
	doc := model.NewDocument()
	doc.Prefs.PWidth.Val = 12
	doc.Prefs.PHeight.Val = 7
	doc.Prefs.Width = 12
	doc.Prefs.Height = 7
	doc.Prefs.RenderFrames.Val = 4

	frames, err := ExportPNGFrames(doc, nil)
	if err != nil {
		t.Fatalf("png frames export failed: %v", err)
	}
	if got, want := len(frames), 4; got != want {
		t.Fatalf("frame count: got %d want %d", got, want)
	}
	for i, b := range frames {
		img, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			t.Fatalf("decode frame %d failed: %v", i, err)
		}
		if got, want := img.Bounds().Dx(), 12; got != want {
			t.Fatalf("frame %d width: got %d want %d", i, got, want)
		}
		if got, want := img.Bounds().Dy(), 7; got != want {
			t.Fatalf("frame %d height: got %d want %d", i, got, want)
		}
	}
}
