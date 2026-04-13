package render

import (
	"image/color"
	"testing"

	model "knobman/internal/model"
)

func TestRenderLineAggDrawsCenteredVerticalStroke(t *testing.T) {
	p := basePrim(model.PrimLine)
	p.Color.Val = color.RGBA{R: 32, G: 32, B: 32, A: 255}
	p.Width.Val = 40
	p.Length.Val = 90

	buf := NewPixBuf(32, 32)
	RenderPrimitive(buf, &p, nil, 0, 1)

	if got := buf.At(16, 16); got.A == 0 {
		t.Fatalf("expected line center visible, got %+v", got)
	}

	if got := buf.At(16, 1); got.A == 0 {
		t.Fatalf("expected top cap visible, got %+v", got)
	}
}

func TestRenderLineAggKeepsOutsideTransparent(t *testing.T) {
	p := basePrim(model.PrimLine)
	p.Color.Val = color.RGBA{R: 32, G: 32, B: 32, A: 255}
	p.Width.Val = 40
	p.Length.Val = 90

	buf := NewPixBuf(32, 32)
	RenderPrimitive(buf, &p, nil, 0, 1)

	if got := buf.At(0, 16); got.A != 0 {
		t.Fatalf("expected far left transparent, got %+v", got)
	}

	if got := buf.At(31, 16); got.A != 0 {
		t.Fatalf("expected far right transparent, got %+v", got)
	}

	if got := buf.At(16, 31); got.A != 0 {
		t.Fatalf("expected area below line transparent, got %+v", got)
	}
}

func TestRenderHLinesAggDrawsRepeatedHorizontalStrokes(t *testing.T) {
	p := basePrim(model.PrimHLines)
	p.Color.Val = color.RGBA{R: 60, G: 60, B: 60, A: 255}
	p.Width.Val = 8
	p.Length.Val = 50
	p.Step.Val = 50

	buf := NewPixBuf(32, 32)
	RenderPrimitive(buf, &p, nil, 0, 1)

	if got := buf.At(16, 16); got.A == 0 {
		t.Fatalf("expected center horizontal line visible, got %+v", got)
	}

	if got := buf.At(16, 9); got.A == 0 {
		t.Fatalf("expected repeated upper horizontal line visible, got %+v", got)
	}

	if got := buf.At(0, 9); got.A != 0 {
		t.Fatalf("expected left outer area transparent, got %+v", got)
	}
}

func TestRenderVLinesAggDrawsRepeatedVerticalStrokes(t *testing.T) {
	p := basePrim(model.PrimVLines)
	p.Color.Val = color.RGBA{R: 60, G: 60, B: 60, A: 255}
	p.Width.Val = 8
	p.Length.Val = 50
	p.Step.Val = 50

	buf := NewPixBuf(32, 32)
	RenderPrimitive(buf, &p, nil, 0, 1)

	if got := buf.At(16, 16); got.A == 0 {
		t.Fatalf("expected center vertical line visible, got %+v", got)
	}

	if got := buf.At(9, 16); got.A == 0 {
		t.Fatalf("expected repeated left vertical line visible, got %+v", got)
	}

	if got := buf.At(9, 0); got.A != 0 {
		t.Fatalf("expected top outer area transparent, got %+v", got)
	}
}

func TestRenderRadiateLinesAggDrawsCardinalSpokes(t *testing.T) {
	p := basePrim(model.PrimRadiateLine)
	p.Color.Val = color.RGBA{R: 40, G: 92, B: 160, A: 255}
	p.Width.Val = 20
	p.Length.Val = 90
	p.AngleStep.Val = 90

	buf := NewPixBuf(32, 32)
	RenderPrimitive(buf, &p, nil, 0, 1)

	if got := buf.At(16, 3); got.A == 0 {
		t.Fatalf("expected top spoke visible, got %+v", got)
	}

	if got := buf.At(3, 16); got.A == 0 {
		t.Fatalf("expected left spoke visible, got %+v", got)
	}

	if got := buf.At(29, 16); got.A == 0 {
		t.Fatalf("expected right spoke visible, got %+v", got)
	}

	if got := buf.At(16, 29); got.A == 0 {
		t.Fatalf("expected bottom spoke visible, got %+v", got)
	}
}

func TestRenderRadiateLinesAggKeepsCornersTransparent(t *testing.T) {
	p := basePrim(model.PrimRadiateLine)
	p.Color.Val = color.RGBA{R: 40, G: 92, B: 160, A: 255}
	p.Width.Val = 20
	p.Length.Val = 90
	p.AngleStep.Val = 90

	buf := NewPixBuf(32, 32)
	RenderPrimitive(buf, &p, nil, 0, 1)

	if got := buf.At(0, 0); got.A != 0 {
		t.Fatalf("expected top-left corner transparent, got %+v", got)
	}

	if got := buf.At(31, 0); got.A != 0 {
		t.Fatalf("expected top-right corner transparent, got %+v", got)
	}

	if got := buf.At(0, 31); got.A != 0 {
		t.Fatalf("expected bottom-left corner transparent, got %+v", got)
	}

	if got := buf.At(31, 31); got.A != 0 {
		t.Fatalf("expected bottom-right corner transparent, got %+v", got)
	}
}

func TestRenderLineAggBlendsOverExistingPixels(t *testing.T) {
	p := basePrim(model.PrimLine)
	p.Color.Val = color.RGBA{R: 200, G: 20, B: 40, A: 128}
	p.Width.Val = 40
	p.Length.Val = 90

	bg := color.RGBA{R: 10, G: 30, B: 160, A: 255}
	src := renderPrimitiveTransparent(&p, 32, 32).At(16, 16)
	want := blendOverColor(bg, src)

	buf := NewPixBuf(32, 32)
	buf.Clear(bg)
	RenderPrimitive(buf, &p, nil, 0, 1)

	if got := buf.At(16, 16); got != want {
		t.Fatalf("blended line mismatch at center: got %+v want %+v", got, want)
	}

	if got := buf.At(0, 16); got != bg {
		t.Fatalf("expected outside line to preserve background, got %+v want %+v", got, bg)
	}
}

func TestRenderRadiateLinesAggBlendsOverExistingPixels(t *testing.T) {
	p := basePrim(model.PrimRadiateLine)
	p.Color.Val = color.RGBA{R: 200, G: 20, B: 40, A: 128}
	p.Width.Val = 20
	p.Length.Val = 90
	p.AngleStep.Val = 90

	bg := color.RGBA{R: 10, G: 30, B: 160, A: 255}
	src := renderPrimitiveTransparent(&p, 32, 32).At(16, 3)
	want := blendOverColor(bg, src)

	buf := NewPixBuf(32, 32)
	buf.Clear(bg)
	RenderPrimitive(buf, &p, nil, 0, 1)

	if got := buf.At(16, 3); got != want {
		t.Fatalf("blended radiate line mismatch at top spoke: got %+v want %+v", got, want)
	}

	if got := buf.At(0, 0); got != bg {
		t.Fatalf("expected corner to preserve background, got %+v want %+v", got, bg)
	}
}

func TestRenderParallelLinesAggBlendsOverExistingPixels(t *testing.T) {
	p := basePrim(model.PrimHLines)
	p.Color.Val = color.RGBA{R: 200, G: 20, B: 40, A: 128}
	p.Width.Val = 8
	p.Length.Val = 50
	p.Step.Val = 50

	bg := color.RGBA{R: 10, G: 30, B: 160, A: 255}
	src := renderPrimitiveTransparent(&p, 32, 32).At(16, 16)
	want := blendOverColor(bg, src)

	buf := NewPixBuf(32, 32)
	buf.Clear(bg)
	RenderPrimitive(buf, &p, nil, 0, 1)

	if got := buf.At(16, 16); got != want {
		t.Fatalf("blended parallel line mismatch at center: got %+v want %+v", got, want)
	}

	if got := buf.At(0, 9); got != bg {
		t.Fatalf("expected outside parallel lines to preserve background, got %+v want %+v", got, bg)
	}
}
