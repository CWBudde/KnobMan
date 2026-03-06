package export

import (
	"bytes"
	"image/png"
	"testing"

	"knobman/internal/model"
)

func TestExportPNGStripDimensions(t *testing.T) {
	doc := model.NewDocument()
	doc.Prefs.PWidth.Val = 10
	doc.Prefs.PHeight.Val = 8
	doc.Prefs.Width = 10
	doc.Prefs.Height = 8
	doc.Prefs.RenderFrames.Val = 3

	vert, err := ExportPNGStrip(doc, nil, false)
	if err != nil {
		t.Fatalf("vertical export failed: %v", err)
	}

	imgV, err := png.Decode(bytes.NewReader(vert))
	if err != nil {
		t.Fatalf("vertical png decode failed: %v", err)
	}

	if got, want := imgV.Bounds().Dx(), 10; got != want {
		t.Fatalf("vertical width: got %d want %d", got, want)
	}

	if got, want := imgV.Bounds().Dy(), 24; got != want {
		t.Fatalf("vertical height: got %d want %d", got, want)
	}

	horz, err := ExportPNGStrip(doc, nil, true)
	if err != nil {
		t.Fatalf("horizontal export failed: %v", err)
	}

	imgH, err := png.Decode(bytes.NewReader(horz))
	if err != nil {
		t.Fatalf("horizontal png decode failed: %v", err)
	}

	if got, want := imgH.Bounds().Dx(), 30; got != want {
		t.Fatalf("horizontal width: got %d want %d", got, want)
	}

	if got, want := imgH.Bounds().Dy(), 8; got != want {
		t.Fatalf("horizontal height: got %d want %d", got, want)
	}
}
