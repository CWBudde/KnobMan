package render

import (
	"image/color"
	"testing"

	model "knobman/internal/model"
)

func TestRenderCircleOutlineLeavesCenterTransparent(t *testing.T) {
	p := basePrim(model.PrimCircle)
	p.Color.Val = color.RGBA{R: 192, G: 32, B: 32, A: 255}
	p.Width.Val = 14
	p.Aspect.Val = 10

	buf := NewPixBuf(32, 32)
	RenderPrimitive(buf, &p, nil, 0, 1)

	if got := buf.At(16, 16); got.A != 0 {
		t.Fatalf("expected circle outline center transparent, got %+v", got)
	}

	visible := 0

	for y := range 32 {
		for x := range 32 {
			if buf.At(x, y).A != 0 {
				visible++
			}
		}
	}

	if visible == 0 {
		t.Fatal("expected circle outline to draw visible shell pixels")
	}
}

func TestRenderCircleFillCoversCenter(t *testing.T) {
	p := basePrim(model.PrimCircleFill)
	p.Color.Val = color.RGBA{R: 48, G: 96, B: 208, A: 255}

	buf := NewPixBuf(32, 32)
	RenderPrimitive(buf, &p, nil, 0, 1)

	if got := buf.At(16, 16); got.A == 0 {
		t.Fatalf("expected circle fill center visible, got %+v", got)
	}

	if got := buf.At(0, 0); got.A != 0 {
		t.Fatalf("expected outer corner transparent, got %+v", got)
	}
}
