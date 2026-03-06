package render

import (
	"math"
	"testing"

	"knobman/internal/model"
)

func TestFrameFrac(t *testing.T) {
	if got := FrameFrac(0, 31, 0); got != 0 {
		t.Fatalf("frame 0 should map to 0, got %v", got)
	}

	if got := FrameFrac(30, 31, 0); math.Abs(got-1) > 1e-9 {
		t.Fatalf("last frame should map to 1, got %v", got)
	}

	if got := FrameFrac(10, 31, 5); got != 1 {
		t.Fatalf("animStep should clamp frame index, got %v", got)
	}
}

func TestEvalAnim(t *testing.T) {
	curves := [8]model.AnimCurve{}
	for i := range curves {
		curves[i] = model.NewAnimCurve()
	}

	if got := EvalAnim(10, 30, 0, &curves, 0.5); math.Abs(got-10) > 1e-9 {
		t.Fatalf("off eval expected 10, got %v", got)
	}

	if got := EvalAnim(10, 30, 1, &curves, 0.5); math.Abs(got-20) > 1e-9 {
		t.Fatalf("linear eval expected 20, got %v", got)
	}
}
