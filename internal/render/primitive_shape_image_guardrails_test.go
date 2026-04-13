package render

import (
	"image/color"
	"testing"

	model "knobman/internal/model"
)

func TestShapeFillAggInteriorAndClip(t *testing.T) {
	p := basePrim(model.PrimShape)
	p.Fill.Val = 1
	p.Shape.Val = "M 10 10 L 90 10 L 50 90 Z"
	p.Color.Val = color.RGBA{R: 56, G: 144, B: 88, A: 255}

	buf := NewPixBuf(32, 32)
	RenderPrimitive(buf, &p, nil, 0, 1)

	if got := buf.At(16, 12); got.A == 0 {
		t.Fatalf("expected shape fill interior visible, got %+v", got)
	}

	if got := buf.At(0, 0); got.A != 0 {
		t.Fatalf("expected outer corner transparent, got %+v", got)
	}
}

func TestShapeFillAggCubicPathInteriorVisible(t *testing.T) {
	p := basePrim(model.PrimShape)
	p.Fill.Val = 1
	p.Shape.Val = "M 20 20 C 20 80 80 20 80 80 L 20 80 Z"
	p.Color.Val = color.RGBA{R: 96, G: 120, B: 200, A: 255}

	buf := NewPixBuf(40, 40)
	RenderPrimitive(buf, &p, nil, 0, 1)

	if got := buf.At(20, 28); got.A == 0 {
		t.Fatalf("expected cubic shape interior visible, got %+v", got)
	}

	if got := buf.At(0, 0); got.A != 0 {
		t.Fatalf("expected outer corner transparent, got %+v", got)
	}
}

func TestShapeOutlineKnobCurveUsesBezierControls(t *testing.T) {
	p := basePrim(model.PrimShape)
	p.Fill.Val = 0
	p.Width.Val = 12
	p.Shape.Val = "/192,256,192,256,192,256:192,208,224,208,256,208:256,304,288,304,320,304:320,256,320,256,320,256"
	p.Color.Val = color.RGBA{R: 255, G: 48, B: 48, A: 255}

	buf := NewPixBuf(64, 64)
	RenderPrimitive(buf, &p, nil, 0, 1)

	minY := buf.Height
	for y := range buf.Height {
		for x := 20; x <= 44; x++ {
			if buf.At(x, y).A != 0 {
				if y < minY {
					minY = y
				}
			}
		}
	}

	if minY >= 28 {
		t.Fatalf("expected bezier outline to rise above anchor polyline, top pixel row=%d", minY)
	}
}

func TestShapeFillBlendsOverExistingPixels(t *testing.T) {
	p := basePrim(model.PrimShape)
	p.Fill.Val = 1
	p.Shape.Val = "M 10 10 L 90 10 L 50 90 Z"
	p.Color.Val = color.RGBA{R: 200, G: 20, B: 40, A: 128}

	bg := color.RGBA{R: 10, G: 30, B: 160, A: 255}
	src := renderPrimitiveTransparent(&p, 32, 32).At(16, 12)
	want := blendOverColor(bg, src)

	buf := NewPixBuf(32, 32)
	buf.Clear(bg)
	RenderPrimitive(buf, &p, nil, 0, 1)

	if got := buf.At(16, 12); got != want {
		t.Fatalf("blended shape fill mismatch at center: got %+v want %+v", got, want)
	}

	if got := buf.At(0, 0); got != bg {
		t.Fatalf("expected outer corner to preserve background, got %+v want %+v", got, bg)
	}
}

func TestImageTransparentKeyPreserved(t *testing.T) {
	key := color.RGBA{R: 255, G: 0, B: 255, A: 255}
	fill := color.RGBA{R: 32, G: 160, B: 96, A: 255}

	if got := applyImageTransparency(key, key, 2, 0); got.A != 0 {
		t.Fatalf("expected keyed first pixel transparent, got %+v", got)
	}

	if got := applyImageTransparency(fill, key, 2, 0); got.A == 0 {
		t.Fatalf("expected non-key pixel visible, got %+v", got)
	}
}
