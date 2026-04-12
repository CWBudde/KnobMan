package render

import (
	"image"
	"image/color"
	"testing"

	agg "github.com/cwbudde/agg_go"
)

func TestImageToPixBufRoundTripPreservesStraightAlpha(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	src.SetNRGBA(0, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 40})
	src.SetNRGBA(1, 0, color.NRGBA{R: 200, G: 10, B: 60, A: 128})
	src.SetNRGBA(0, 1, color.NRGBA{R: 0, G: 0, B: 0, A: 0})
	src.SetNRGBA(1, 1, color.NRGBA{R: 255, G: 240, B: 16, A: 255})

	pb := ImageToPixBuf(src)
	if pb == nil {
		t.Fatal("ImageToPixBuf returned nil")
	}

	got := PixBufToNRGBA(pb)
	if got == nil {
		t.Fatal("PixBufToNRGBA returned nil")
	}

	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			if got.At(x, y) != src.At(x, y) {
				t.Fatalf("round-trip mismatch at (%d,%d): got %+v want %+v", x, y, got.At(x, y), src.At(x, y))
			}
		}
	}
}

func TestImageToRGBAUnpremultipliesRGBA(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1, 1))
	src.Pix[0] = 64
	src.Pix[1] = 32
	src.Pix[2] = 16
	src.Pix[3] = 128

	got := ImageToRGBA(src)
	if got == nil {
		t.Fatal("ImageToRGBA returned nil")
	}

	want := color.RGBA{R: 128, G: 64, B: 32, A: 128}
	if px := got.RGBAAt(0, 0); px != want {
		t.Fatalf("unexpected unpremultiplied pixel: got %+v want %+v", px, want)
	}
}

func TestAggContextForPixBufSharesBackingBuffer(t *testing.T) {
	pb := NewPixBuf(2, 2)

	ctx := AggContextForPixBuf(pb)
	if ctx == nil {
		t.Fatal("AggContextForPixBuf returned nil")
	}

	ctx.Clear(agg.Color{R: 7, G: 8, B: 9, A: 10})
	if got := pb.At(1, 1); got != (color.RGBA{R: 7, G: 8, B: 9, A: 10}) {
		t.Fatalf("agg context did not update pixbuf backing store: %+v", got)
	}
}
