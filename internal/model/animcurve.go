package model

import "math"

const animCurvePoints = 12

// AnimCurve is a piecewise-linear animation curve with up to 12 keypoints.
// Points are stored as integer time (0–100) and level (0–100).
// Inactive points are marked with tm=-1 / lv=-1.
// The first point is always (tm=0, lv=0..100) and the last is (tm=100, lv=0..100).
type AnimCurve struct {
	Tm       [animCurvePoints]int // time values (0–100), -1 = inactive
	Lv       [animCurvePoints]int // level values (0–100), -1 = inactive
	StepReso IntParam             // 0=smooth, 1=freeze, ≥2=quantize to N steps
}

// NewAnimCurve returns an AnimCurve initialised to a linear 0→100 ramp.
func NewAnimCurve() AnimCurve {
	c := AnimCurve{}
	c.Reset()

	return c
}

// Reset initialises the curve to a simple 0→100 linear ramp.
func (c *AnimCurve) Reset() {
	for i := range c.Tm {
		c.Tm[i] = -1
		c.Lv[i] = -1
	}

	c.Tm[0], c.Lv[0] = 0, 0
	c.Tm[11], c.Lv[11] = 100, 100
	c.StepReso.Val = 0
}

// Eval returns the interpolated level (0.0–1.0) for input ratio (0.0–1.0).
// Mirrors AnimCurve.GetVal in the Java original exactly.
func (c *AnimCurve) Eval(ratio float64) float64 {
	iT0 := 0

	iL0 := c.Lv[0]
	for i := 1; i < animCurvePoints; i++ {
		if c.Tm[i] < 0 {
			continue
		}

		iT1 := c.Tm[i]
		iL1 := c.Lv[i]

		if ratio*100.0 <= float64(iT1) {
			if iT1 != iT0 {
				ratio = (float64(iL0) + float64(iL1-iL0)*(ratio*100.0-float64(iT0))/float64(iT1-iT0)) / 100.0
			} else {
				ratio = float64(iL0) / 100.0
			}

			break
		}

		iT0 = iT1
		iL0 = iL1
	}

	switch {
	case c.StepReso.Val == 1:
		return 0
	case c.StepReso.Val >= 2:
		steps := c.StepReso.Val

		i := int(math.Floor(ratio * float64(steps)))
		if i >= steps {
			i = steps - 1
		}

		return float64(i) / float64(steps-1)
	default:
		return ratio
	}
}

// EvalParam returns the interpolated value between from and to, driven by
// the given ratio and using curveIdx to select how animation is applied.
// Legacy mapping:
//
//	curveIdx == 0 => animation off (returns from)
//	curveIdx == 1 => linear interpolation
//	curveIdx >= 2 => anim curves 1..8 (curves[0..7])
func EvalParam(from, to float64, curveIdx int, curves *[8]AnimCurve, ratio float64) float64 {
	if curveIdx <= 0 {
		return from
	}

	t := ratio
	if curveIdx >= 2 && curves != nil {
		idx := curveIdx - 2
		if idx >= 8 {
			idx = 7
		}

		t = curves[idx].Eval(ratio)
	}

	return from + (to-from)*t
}
