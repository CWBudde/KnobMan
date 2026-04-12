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
