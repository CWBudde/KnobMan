package render

import (
	"image"
	"image/color"
	"path/filepath"
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

func TestShapeOutlinePlainMatchesJavaEndpointAlphaProfile(t *testing.T) {
	root := testRepoRoot(t)
	samplePath := filepath.Join(root, "tests", "parity", "primitives", "inputs", "shape_outline_plain.knob")

	doc, textures, err := LoadParityDocument(samplePath, root)
	if err != nil {
		t.Fatalf("LoadParityDocument: %v", err)
	}

	want, err := ReadPNGAsRGBA(filepath.Join(root, "tests", "parity", "primitives", "baseline-java", "shape_outline_plain.png"))
	if err != nil {
		t.Fatalf("ReadPNGAsRGBA: %v", err)
	}

	out := NewPixBuf(doc.Prefs.PWidth.Val, doc.Prefs.PHeight.Val)
	RenderFrame(out, doc, 0, textures)

	gotMinX, gotMaxX := visibleSpanX(out, 0, parityTolerance)
	wantMinX, wantMaxX := visibleSpanRGBA_X(want, 0, parityTolerance)

	if gotMinX != wantMinX || gotMaxX != wantMaxX {
		t.Fatalf("top-edge visible span mismatch: got [%d,%d] want [%d,%d]", gotMinX, gotMaxX, wantMinX, wantMaxX)
	}
}

func visibleSpanX(buf *PixBuf, y int, tol uint8) (minX, maxX int) {
	minX, maxX = -1, -1

	for x := range buf.Width {
		if buf.At(x, y).A <= tol {
			continue
		}

		if minX < 0 {
			minX = x
		}

		maxX = x
	}

	return minX, maxX
}

func visibleSpanRGBA_X(img *image.RGBA, y int, tol uint8) (minX, maxX int) {
	minX, maxX = -1, -1

	bounds := img.Bounds()
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		if img.RGBAAt(x, y).A <= tol {
			continue
		}

		if minX < 0 {
			minX = x
		}

		maxX = x
	}

	return minX, maxX
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
