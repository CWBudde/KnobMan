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

func TestMapFrameToAnimStepMatchesLegacyBuckets(t *testing.T) {
	cases := []struct {
		frame       int
		totalFrames int
		animTotal   int
		want        int
	}{
		{frame: 0, totalFrames: 7, animTotal: 4, want: 0},
		{frame: 1, totalFrames: 7, animTotal: 4, want: 0},
		{frame: 2, totalFrames: 7, animTotal: 4, want: 1},
		{frame: 3, totalFrames: 7, animTotal: 4, want: 1},
		{frame: 4, totalFrames: 7, animTotal: 4, want: 2},
		{frame: 5, totalFrames: 7, animTotal: 4, want: 2},
		{frame: 6, totalFrames: 7, animTotal: 4, want: 3},
	}

	for _, tc := range cases {
		if got := mapFrameToAnimStep(tc.frame, tc.totalFrames, tc.animTotal); got != tc.want {
			t.Fatalf("mapFrameToAnimStep(frame=%d,total=%d,anim=%d) = %d, want %d", tc.frame, tc.totalFrames, tc.animTotal, got, tc.want)
		}
	}
}

func TestLayerRenderSpanHonorsAnimStepAndUnfold(t *testing.T) {
	eff := model.NewEffect()
	eff.AnimStep.Val = 4

	animTotal, startFrame, endFrame := layerRenderSpan(&eff, 3, 7)
	if animTotal != 4 || startFrame != 1 || endFrame != 1 {
		t.Fatalf("animstep span = (%d,%d,%d), want (4,1,1)", animTotal, startFrame, endFrame)
	}

	eff.Unfold.Val = 1

	animTotal, startFrame, endFrame = layerRenderSpan(&eff, 3, 7)
	if animTotal != 4 || startFrame != 0 || endFrame != 3 {
		t.Fatalf("unfold span = (%d,%d,%d), want (4,0,3)", animTotal, startFrame, endFrame)
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
