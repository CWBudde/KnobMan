package export

import (
	"bytes"
	"encoding/binary"
	"testing"

	"knobman/internal/model"
)

func TestExportAPNGBasic(t *testing.T) {
	doc := model.NewDocument()
	doc.Prefs.PWidth.Val = 11
	doc.Prefs.PHeight.Val = 6
	doc.Prefs.Width = 11
	doc.Prefs.Height = 6
	doc.Prefs.RenderFrames.Val = 3
	doc.Prefs.Duration.Val = 25
	doc.Prefs.Loop.Val = 2
	doc.Prefs.BiDir.Val = 0

	data, err := ExportAPNG(doc, nil)
	if err != nil {
		t.Fatalf("apng export failed: %v", err)
	}

	if !bytes.Equal(data[:8], pngSignature) {
		t.Fatalf("missing png signature")
	}

	chunks, err := parsePNGChunks(data)
	if err != nil {
		t.Fatalf("parse apng failed: %v", err)
	}

	actl := firstChunkData(chunks, "acTL")
	if len(actl) != 8 {
		t.Fatalf("missing acTL")
	}

	if got, want := binary.BigEndian.Uint32(actl[0:4]), uint32(3); got != want {
		t.Fatalf("num_frames: got %d want %d", got, want)
	}

	if got, want := binary.BigEndian.Uint32(actl[4:8]), uint32(2); got != want {
		t.Fatalf("num_plays: got %d want %d", got, want)
	}

	fctlCount := 0
	fdatCount := 0

	for _, ch := range chunks {
		if ch.Type == "fcTL" {
			fctlCount++
		}

		if ch.Type == "fdAT" {
			fdatCount++
		}
	}

	if got, want := fctlCount, 3; got != want {
		t.Fatalf("fcTL count: got %d want %d", got, want)
	}

	if fdatCount == 0 {
		t.Fatalf("expected fdAT chunks for animated frames")
	}
}

func TestExportAPNGBiDirFrameCount(t *testing.T) {
	doc := model.NewDocument()
	doc.Prefs.PWidth.Val = 8
	doc.Prefs.PHeight.Val = 8
	doc.Prefs.Width = 8
	doc.Prefs.Height = 8
	doc.Prefs.RenderFrames.Val = 4
	doc.Prefs.Duration.Val = 40
	doc.Prefs.Loop.Val = 0
	doc.Prefs.BiDir.Val = 1

	data, err := ExportAPNG(doc, nil)
	if err != nil {
		t.Fatalf("apng export failed: %v", err)
	}

	chunks, err := parsePNGChunks(data)
	if err != nil {
		t.Fatalf("parse apng failed: %v", err)
	}

	actl := firstChunkData(chunks, "acTL")
	if len(actl) != 8 {
		t.Fatalf("missing acTL")
	}
	// 4 frames with bidir => 0,1,2,3,2,1 => 6 APNG frames.
	if got, want := binary.BigEndian.Uint32(actl[0:4]), uint32(6); got != want {
		t.Fatalf("num_frames: got %d want %d", got, want)
	}
	// Loop=0 => infinite.
	if got, want := binary.BigEndian.Uint32(actl[4:8]), uint32(0); got != want {
		t.Fatalf("num_plays: got %d want %d", got, want)
	}
}
