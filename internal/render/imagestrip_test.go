package render

import (
	"image/color"
	"testing"
)

func TestExtractFrameAlignedHorizontal(t *testing.T) {
	strip := NewPixBuf(6, 2)
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			strip.Set(x, y, color.RGBA{255, 0, 0, 255})
			strip.Set(x+2, y, color.RGBA{0, 255, 0, 255})
			strip.Set(x+4, y, color.RGBA{0, 0, 255, 255})
		}
	}
	f := ExtractFrameAligned(strip, 3, 2, 3, 1)
	if f == nil {
		t.Fatal("nil frame")
	}
	if got := f.At(1, 1); got.B < 200 {
		t.Fatalf("expected blue frame, got %+v", got)
	}
}

func TestExtractFrameAlignedVertical(t *testing.T) {
	strip := NewPixBuf(2, 6)
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			strip.Set(x, y, color.RGBA{255, 0, 0, 255})
			strip.Set(x, y+2, color.RGBA{0, 255, 0, 255})
			strip.Set(x, y+4, color.RGBA{0, 0, 255, 255})
		}
	}
	f := ExtractFrameAligned(strip, 3, 1, 3, 0)
	if f == nil {
		t.Fatal("nil frame")
	}
	if got := f.At(1, 1); got.G < 200 {
		t.Fatalf("expected green frame, got %+v", got)
	}
}
