package render

import "knobman/internal/model"

// EvalAnim evaluates an animatable from/to pair at frameFrac in [0,1].
// animCurveIdx: 0 => linear, 1..8 => curves[0..7].
func EvalAnim(from, to float64, animCurveIdx int, curves *[8]model.AnimCurve, frameFrac float64) float64 {
	if frameFrac < 0 {
		frameFrac = 0
	}

	if frameFrac > 1 {
		frameFrac = 1
	}

	return model.EvalParam(from, to, animCurveIdx, curves, frameFrac)
}

// FrameFrac returns animation progress in [0,1], honoring AnimStep.
// When animStep > 0, it overrides totalFrames for this layer.
func FrameFrac(frame, totalFrames, animStep int) float64 {
	if animStep > 0 {
		if animStep <= 1 {
			return 0
		}

		if frame < 0 {
			frame = 0
		}

		if frame >= animStep {
			frame = animStep - 1
		}

		return float64(frame) / float64(animStep-1)
	}

	if totalFrames <= 1 {
		return 0
	}

	if frame < 0 {
		frame = 0
	}

	if frame >= totalFrames {
		frame = totalFrames - 1
	}

	return float64(frame) / float64(totalFrames-1)
}
