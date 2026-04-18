package render

import (
	"image/color"
	"testing"
)

func TestParseLooseShapePairsSkipsInvalidTokens(t *testing.T) {
	pts := parseLooseShapePairs("M 0,0 junk 50 L 50,50 100,100 z", 101, 101)
	if len(pts) != 3 {
		t.Fatalf("expected 3 parsed points, got %d", len(pts))
	}

	if pts[0] != (point{x: 0, y: 0}) {
		t.Fatalf("first point = %+v, want origin", pts[0])
	}

	if pts[1] != (point{x: 50, y: 50}) {
		t.Fatalf("second point = %+v, want midpoint", pts[1])
	}

	if pts[2] != (point{x: 100, y: 100}) {
		t.Fatalf("third point = %+v, want bottom-right", pts[2])
	}
}

func TestPointInSquareCappedSegment(t *testing.T) {
	a := fpoint{x: 1, y: 1}
	b := fpoint{x: 5, y: 1}

	if !pointInSquareCappedSegment(3, 1.4, a, b, 0.5) {
		t.Fatal("expected point near horizontal segment to be inside stroke")
	}

	if pointInSquareCappedSegment(3, 2.0, a, b, 0.5) {
		t.Fatal("expected distant point to be outside stroke")
	}

	if !pointInSquareCappedSegment(1.2, 1.2, a, a, 0.5) {
		t.Fatal("expected zero-length segment to behave like a capped point")
	}
}

func TestPointOnStrokePolys(t *testing.T) {
	polys := [][]fpoint{{
		{x: 1, y: 1},
		{x: 5, y: 1},
		{x: 5, y: 5},
	}}

	if !pointOnStrokePolys(3, 1.2, polys, 0.4) {
		t.Fatal("expected point on first segment to hit stroke")
	}

	if pointOnStrokePolys(2, 4, polys, 0.4) {
		t.Fatal("expected point away from both segments to miss stroke")
	}
}

func TestRenderTriangleAggMask(t *testing.T) {
	if renderTriangleAggMask(nil, 8, 8, 50, 50) {
		t.Fatal("expected nil mask to fail")
	}

	mask := NewPixBuf(8, 8)
	if !renderTriangleAggMask(mask, 8, 8, 60, 80) {
		t.Fatal("expected valid triangle mask to render")
	}

	covered := 0
	for y := range mask.Height {
		for x := range mask.Width {
			if px := mask.At(x, y); px.B != 0 || px.A != 0 {
				covered++
			}
		}
	}

	if covered == 0 {
		t.Fatal("expected triangle mask to paint coverage")
	}
}

func TestFillTrianglePaintsInterior(t *testing.T) {
	dst := NewPixBuf(8, 8)
	fillTriangle(dst, point{1, 1}, point{6, 1}, point{3, 6}, color.RGBA{R: 255, A: 255})

	if got := dst.At(3, 3); got.A == 0 {
		t.Fatalf("expected interior point to be painted, got %+v", got)
	}

	if got := dst.At(0, 0); got.A != 0 {
		t.Fatalf("expected outside point to remain empty, got %+v", got)
	}
}

func TestDrawLinePaintsConfiguredWidth(t *testing.T) {
	dst := NewPixBuf(9, 9)
	drawLine(dst, 1, 4, 7, 4, color.RGBA{G: 255, A: 255}, 3)

	for _, pt := range []point{{1, 4}, {4, 4}, {7, 4}, {4, 3}, {4, 5}} {
		if got := dst.At(pt.x, pt.y); got.A == 0 {
			t.Fatalf("expected line coverage at %+v, got %+v", pt, got)
		}
	}

	if got := dst.At(4, 0); got.A != 0 {
		t.Fatalf("expected far-away pixel to remain empty, got %+v", got)
	}
}
