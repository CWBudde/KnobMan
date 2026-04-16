package export

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"testing"

	"knobman/internal/model"
)

func TestAnimatedExportsStayAlignedAcrossFramesGIFAndAPNG(t *testing.T) {
	doc := animatedExportDoc()

	pngFrames, err := ExportPNGFrames(doc, nil)
	if err != nil {
		t.Fatalf("png frames export failed: %v", err)
	}

	if got, want := len(pngFrames), 3; got != want {
		t.Fatalf("png frame count: got %d want %d", got, want)
	}

	wantColors := []color.RGBA{
		{R: 224, G: 48, B: 48, A: 255},
		{R: 48, G: 200, B: 72, A: 255},
		{R: 48, G: 96, B: 224, A: 255},
	}

	for i, want := range wantColors {
		img := decodePNGForTest(t, pngFrames[i])
		if got := rgbaAtCenter(img); !approxRGBA(got, want, 0) {
			t.Fatalf("png frame %d center: got %+v want %+v", i, got, want)
		}
	}

	horizontalStrip, err := ExportPNGStrip(doc, nil, true)
	if err != nil {
		t.Fatalf("horizontal strip export failed: %v", err)
	}

	strip := decodePNGForTest(t, horizontalStrip)

	for i, want := range wantColors {
		x := i*16 + 8

		got := rgbaAt(strip, x, 8)
		if !approxRGBA(got, want, 0) {
			t.Fatalf("strip frame %d center: got %+v want %+v", i, got, want)
		}
	}

	gifData, err := ExportGIF(doc, nil)
	if err != nil {
		t.Fatalf("gif export failed: %v", err)
	}

	g, err := gif.DecodeAll(bytes.NewReader(gifData))
	if err != nil {
		t.Fatalf("gif decode failed: %v", err)
	}

	wantSequence := []color.RGBA{
		wantColors[0],
		wantColors[1],
		wantColors[2],
		wantColors[1],
	}

	if got, want := len(g.Image), len(wantSequence); got != want {
		t.Fatalf("gif frame count: got %d want %d", got, want)
	}

	for i, want := range wantSequence {
		if got := rgbaAtCenter(g.Image[i]); !approxRGBA(got, want, 64) {
			t.Fatalf("gif frame %d center: got %+v want %+v", i, got, want)
		}

		if gotDelay, wantDelay := g.Delay[i], 3; gotDelay != wantDelay {
			t.Fatalf("gif delay[%d]: got %d want %d", i, gotDelay, wantDelay)
		}
	}

	if got, want := g.LoopCount, 2; got != want {
		t.Fatalf("gif loop count: got %d want %d", got, want)
	}

	apngData, err := ExportAPNG(doc, nil)
	if err != nil {
		t.Fatalf("apng export failed: %v", err)
	}

	chunks, err := parsePNGChunks(apngData)
	if err != nil {
		t.Fatalf("parse apng failed: %v", err)
	}

	actl := firstChunkData(chunks, "acTL")
	if got, want := binary.BigEndian.Uint32(actl[0:4]), uint32(len(wantSequence)); got != want {
		t.Fatalf("apng num_frames: got %d want %d", got, want)
	}

	if got, want := binary.BigEndian.Uint32(actl[4:8]), uint32(2); got != want {
		t.Fatalf("apng num_plays: got %d want %d", got, want)
	}

	fctlCount := 0

	for _, ch := range chunks {
		if ch.Type != "fcTL" {
			continue
		}

		fctlCount++

		if got, want := binary.BigEndian.Uint16(ch.Data[20:22]), uint16(25); got != want {
			t.Fatalf("apng delay numerator: got %d want %d", got, want)
		}

		if got, want := binary.BigEndian.Uint16(ch.Data[22:24]), uint16(1000); got != want {
			t.Fatalf("apng delay denominator: got %d want %d", got, want)
		}
	}

	if got, want := fctlCount, len(wantSequence); got != want {
		t.Fatalf("apng fcTL count: got %d want %d", got, want)
	}
}

func animatedExportDoc() *model.Document {
	doc := model.NewDocument()
	doc.Layers = []model.Layer{model.NewLayer()}
	doc.Prefs.Width = 16
	doc.Prefs.Height = 16
	doc.Prefs.PWidth.Val = 16
	doc.Prefs.PHeight.Val = 16
	doc.Prefs.RenderFrames.Val = 3
	doc.Prefs.Duration.Val = 25
	doc.Prefs.Loop.Val = 2
	doc.Prefs.BiDir.Val = 1
	doc.Prefs.BkColor.Val = color.RGBA{}

	ly := &doc.Layers[0]
	ly.Prim.Type.Val = int(model.PrimImage)
	ly.Prim.AutoFit.Val = 0
	ly.Prim.NumFrame.Val = 3
	ly.Prim.FrameAlign.Val = 1
	ly.Prim.EmbeddedImage = exportStripPNG(
		color.NRGBA{R: 224, G: 48, B: 48, A: 255},
		color.NRGBA{R: 48, G: 200, B: 72, A: 255},
		color.NRGBA{R: 48, G: 96, B: 224, A: 255},
	)

	return doc
}

func exportStripPNG(colors ...color.NRGBA) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 16*len(colors), 16))
	for i, fill := range colors {
		x0 := i * 16

		for y := range 16 {
			for x := range 16 {
				img.SetNRGBA(x0+x, y, fill)
			}
		}
	}

	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		panic(err)
	}

	return buf.Bytes()
}

func decodePNGForTest(t *testing.T, data []byte) image.Image {
	t.Helper()

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("png decode failed: %v", err)
	}

	return img
}

func rgbaAtCenter(img image.Image) color.RGBA {
	b := img.Bounds()
	return rgbaAt(img, b.Min.X+b.Dx()/2, b.Min.Y+b.Dy()/2)
}

func rgbaAt(img image.Image, x, y int) color.RGBA {
	return color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
}

func approxRGBA(got, want color.RGBA, tol uint8) bool {
	return channelDelta(got.R, want.R) <= tol &&
		channelDelta(got.G, want.G) <= tol &&
		channelDelta(got.B, want.B) <= tol &&
		channelDelta(got.A, want.A) <= tol
}

func channelDelta(a, b uint8) uint8 {
	if a > b {
		return a - b
	}

	return b - a
}
