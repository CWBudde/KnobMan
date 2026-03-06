package render

import (
	"image"
	"image/color"
	"testing"
)

func TestTextureSampleWrap(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.SetRGBA(0, 0, color.RGBA{255, 0, 0, 255})
	img.SetRGBA(1, 0, color.RGBA{0, 255, 0, 255})
	img.SetRGBA(0, 1, color.RGBA{0, 0, 255, 255})
	img.SetRGBA(1, 1, color.RGBA{255, 255, 0, 255})

	tex := NewTextureFromImage(img)
	if tex == nil {
		t.Fatal("nil texture")
	}

	c := tex.Sample(2, 0, 1) // wraps to x=0
	if c.R < 200 {
		t.Fatalf("expected wrapped sample near red, got %+v", c)
	}

	c2 := tex.Sample(0.5, 0.5, 1) // bilinear mix of all 4 texels
	if c2.A != 255 {
		t.Fatalf("unexpected alpha: %+v", c2)
	}

	if c2.R == 0 && c2.G == 0 && c2.B == 0 {
		t.Fatalf("unexpected black bilinear sample: %+v", c2)
	}
}
