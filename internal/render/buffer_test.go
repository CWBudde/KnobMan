package render

import (
	"image/color"
	"testing"
)

func TestPixBufBasics(t *testing.T) {
	b := NewPixBuf(4, 3)
	if b.Width != 4 || b.Height != 3 || b.Stride != 16 {
		t.Fatalf("unexpected geometry: %+v", b)
	}

	b.Clear(color.RGBA{R: 10, G: 20, B: 30, A: 40})
	if got := b.At(2, 1); got != (color.RGBA{10, 20, 30, 40}) {
		t.Fatalf("clear mismatch: %+v", got)
	}

	b.Set(1, 1, color.RGBA{255, 0, 0, 255})
	if got := b.At(1, 1); got != (color.RGBA{255, 0, 0, 255}) {
		t.Fatalf("set mismatch: %+v", got)
	}

	b2 := NewPixBuf(4, 3)
	b2.CopyFrom(b)
	if got := b2.At(1, 1); got != (color.RGBA{255, 0, 0, 255}) {
		t.Fatalf("copy mismatch: %+v", got)
	}

	b3 := NewPixBuf(1, 1)
	b3.Set(0, 0, color.RGBA{0, 0, 0, 255})
	b3.BlendOver(0, 0, color.RGBA{255, 255, 255, 128})
	got := b3.At(0, 0)
	if got.R == 0 || got.A == 0 {
		t.Fatalf("blend did not update pixel: %+v", got)
	}
}
