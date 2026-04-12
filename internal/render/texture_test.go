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

func TestTextureSampleNegativeWrapMatchesPositiveEquivalent(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.SetRGBA(0, 0, color.RGBA{255, 0, 0, 255})
	img.SetRGBA(1, 0, color.RGBA{0, 255, 0, 255})
	img.SetRGBA(0, 1, color.RGBA{0, 0, 255, 255})
	img.SetRGBA(1, 1, color.RGBA{255, 255, 0, 255})

	tex := NewTextureFromImage(img)
	if tex == nil {
		t.Fatal("nil texture")
	}

	got := tex.Sample(-0.25, 1.25, 1)

	want := tex.Sample(1.75, 1.25, 1)
	if got != want {
		t.Fatalf("wrapped samples differ: got %+v want %+v", got, want)
	}
}

func TestTextureSampleBlendsAcrossTilingSeam(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.SetRGBA(0, 0, color.RGBA{255, 0, 0, 255})
	img.SetRGBA(1, 0, color.RGBA{0, 255, 0, 255})

	tex := NewTextureFromImage(img)
	if tex == nil {
		t.Fatal("nil texture")
	}

	got := tex.Sample(1.5, 0, 1)

	want := color.RGBA{R: 128, G: 128, B: 0, A: 255}
	if got != want {
		t.Fatalf("unexpected seam blend: got %+v want %+v", got, want)
	}
}

func TestTextureSampleZoomChangesSamplingFrequency(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 1))
	img.SetRGBA(0, 0, color.RGBA{0, 0, 0, 255})
	img.SetRGBA(1, 0, color.RGBA{100, 0, 0, 255})
	img.SetRGBA(2, 0, color.RGBA{0, 100, 0, 255})
	img.SetRGBA(3, 0, color.RGBA{0, 0, 100, 255})

	tex := NewTextureFromImage(img)
	if tex == nil {
		t.Fatal("nil texture")
	}

	gotZoom100 := tex.Sample(3, 0, 1)
	if gotZoom100 != (color.RGBA{0, 0, 100, 255}) {
		t.Fatalf("unexpected zoom=100 sample: %+v", gotZoom100)
	}

	gotZoom200 := tex.Sample(3, 0, 2)

	wantZoom200 := color.RGBA{R: 50, G: 50, B: 0, A: 255}
	if gotZoom200 != wantZoom200 {
		t.Fatalf("unexpected zoom=200 sample: got %+v want %+v", gotZoom200, wantZoom200)
	}
}

func TestTextureSampleHeightAlphaExactTexel(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.SetRGBA(0, 0, color.RGBA{10, 20, 30, 40})
	img.SetRGBA(1, 0, color.RGBA{60, 70, 80, 90})
	img.SetRGBA(0, 1, color.RGBA{100, 110, 120, 130})
	img.SetRGBA(1, 1, color.RGBA{140, 150, 160, 170})

	tex := NewTextureFromImage(img)
	if tex == nil {
		t.Fatal("nil texture")
	}

	luma, alpha := tex.SampleHeightAlpha(-1.5, -1.5, 100)
	if luma != 18 || alpha != 40 {
		t.Fatalf("unexpected sample: luma=%d alpha=%d", luma, alpha)
	}
}

func TestTextureSampleHeightAlphaHalfResolution(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	block := []color.RGBA{
		{10, 10, 10, 20},
		{20, 20, 20, 40},
		{30, 30, 30, 60},
		{40, 40, 40, 80},
	}
	img.SetRGBA(0, 0, block[0])
	img.SetRGBA(1, 0, block[1])
	img.SetRGBA(0, 1, block[2])
	img.SetRGBA(1, 1, block[3])

	tex := NewTextureFromImage(img)
	if tex == nil {
		t.Fatal("nil texture")
	}

	luma, alpha := tex.SampleHeightAlpha(-1.5, -1.5, 50)
	if luma != 25 || alpha != 50 {
		t.Fatalf("unexpected half-res sample: luma=%d alpha=%d", luma, alpha)
	}
}

func TestTextureSampleHeightAlphaLowZoomWrapMatchesEquivalentPeriod(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	fillBlock := func(x0, y0 int, c color.RGBA) {
		for y := y0; y < y0+2; y++ {
			for x := x0; x < x0+2; x++ {
				img.SetRGBA(x, y, c)
			}
		}
	}
	fillBlock(0, 0, color.RGBA{20, 20, 20, 40})
	fillBlock(2, 0, color.RGBA{80, 80, 80, 90})
	fillBlock(0, 2, color.RGBA{140, 140, 140, 160})
	fillBlock(2, 2, color.RGBA{200, 200, 200, 220})

	tex := NewTextureFromImage(img)
	if tex == nil {
		t.Fatal("nil texture")
	}

	lumaA, alphaA := tex.SampleHeightAlpha(-1.5, -1.5, 50)
	lumaB, alphaB := tex.SampleHeightAlpha(0.5, -1.5, 50)

	if lumaA != 20 || alphaA != 40 {
		t.Fatalf("unexpected low-zoom wrapped sample: luma=%d alpha=%d", lumaA, alphaA)
	}

	if lumaA != lumaB || alphaA != alphaB {
		t.Fatalf("low-zoom wrapped samples differ: (%d,%d) vs (%d,%d)", lumaA, alphaA, lumaB, alphaB)
	}
}
