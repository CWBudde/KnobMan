package render

import "math"

// BuildMask creates a per-pixel mask in [0,1].
// maskType: 0=rotation, 1=radial, 2=horizontal, 3=vertical.
func BuildMask(w, h int, maskType int, start, stop, grad float64, gradDir int) []float64 {
	return BuildMaskWithCenter(w, h, maskType, start, stop, grad, gradDir, 0, 0)
}

// BuildMaskWithCenter creates a per-pixel mask in [0,1] using legacy MaskWipe
// semantics, including effect center offsets (in percent).
func BuildMaskWithCenter(w, h int, maskType int, start, stop, grad float64, gradDir int, centerX, centerY float64) []float64 {
	if w <= 0 || h <= 0 {
		return nil
	}

	mask := make([]float64, w*h)
	for y := range h {
		for x := range w {
			// Legacy MaskWipe samples at pixel center and applies effect center offset.
			dx := float64(x) + 0.5 - centerX*float64(w)/100.0
			dy := float64(y) + 0.5 + centerY*float64(h)/100.0
			mask[y*w+x] = clamp01(maskWipeLegacy(maskType, dx, dy, w, h, start, stop, grad, gradDir) / 255.0)
		}
	}

	return mask
}

func maskWipeLegacy(maskType int, x, y float64, xDest, yDest int, start, stop, grad float64, gradDir int) float64 {
	if start == stop {
		return 0
	}

	dir := false

	if start > stop {
		start, stop = stop, start
		dir = true
	}

	switch maskType {
	case 0:
		if stop-start >= 360.0 {
			return 255.0
		}

		xx := x - float64(xDest)*0.5
		yy := y - float64(yDest)*0.5

		c := math.Sqrt(xx*xx + yy*yy)
		if c == 0.0 {
			return 255.0
		}

		rc := 1.0 / c
		aStart := start * math.Pi / 180.0
		aStop := stop * math.Pi / 180.0
		aMin := math.Min(aStart, aStop)
		aMax := math.Max(aStart, aStop)

		a := math.Atan2(xx, -yy) - math.Pi*2.0
		for a < aMin-rc {
			a += math.Pi * 2.0
		}

		for a > aMax+rc {
			a -= math.Pi * 2.0
		}

		alpha := 255.0

		if a >= aMin-rc && a <= aMax+rc {
			if a < aMin {
				alpha = alpha * (a - (aMin - rc)) / rc
			}

			if a > aMax {
				alpha = alpha * (aMax + rc - a) / rc
			}
		} else {
			return 0.0
		}

		var rGrad float64
		if gradDir != 0 {
			rGrad = (a - aMin + rc) / ((aMax - aMin + 2.0*rc) * grad / 100.0)
			rGrad2 := (a - aMax - rc) / ((aMin - aMax) * grad / 100.0)
			rGrad = math.Min(rGrad, rGrad2)
		} else if !dir {
			rGrad = (a - aMin + rc) / ((aMax - aMin + 2.0*rc) * grad / 100.0)
		} else {
			rGrad = (a - aMax - rc) / ((aMin - aMax) * grad / 100.0)
		}

		if rGrad < 0.0 {
			rGrad = 0.0
		}

		if rGrad > 1.0 {
			rGrad = 1.0
		}

		return alpha * rGrad
	case 1:
		rStart := start * float64(min(yDest, xDest)) / 200.0
		rStop := stop * float64(min(yDest, xDest)) / 200.0
		xx := x - float64(xDest/2)
		yy := y - float64(yDest/2)
		c := math.Sqrt(xx*xx + yy*yy)

		alpha := 255.0
		if c < rStart {
			alpha = 0.0
		}

		if c < rStart+1.0 {
			alpha = (c - rStart) * alpha
		}

		if c > rStop {
			alpha = 0.0
		}

		if c > rStop-1.0 {
			alpha = (rStop - c) * alpha
		}

		if grad == 0.0 {
			return alpha
		}

		var rGrad float64
		if gradDir != 0 {
			rGrad = (c - rStart) / ((rStop - rStart) * grad / 100.0)
			rGrad2 := (c - rStop) / ((rStart - rStop) * grad / 100.0)
			rGrad = math.Min(rGrad, rGrad2)
		} else if !dir {
			rGrad = (c - rStart) / ((rStop - rStart) * grad / 100.0)
		} else {
			rGrad = (c - rStop) / ((rStart - rStop) * grad / 100.0)
		}

		if rGrad < 0.0 {
			rGrad = 0.0
		}

		if rGrad > 1.0 {
			rGrad = 1.0
		}

		return alpha * rGrad
	case 2:
		rStart := (start + 100.0) * float64(xDest) / 200.0
		rStop := (stop + 100.0) * float64(xDest) / 200.0
		alpha := 255.0

		if x < rStart-1.0 {
			return 0.0
		}

		if x < rStart {
			alpha *= x - (rStart - 1.0)
		}

		if x > rStop+1.0 {
			return 0.0
		}

		if x > rStop {
			alpha *= rStop + 1.0 - x
		}

		if grad == 0.0 {
			return alpha
		}

		var rGrad float64
		if gradDir != 0 {
			rGrad = (x - rStart) / ((rStop - rStart) * grad / 100.0)
			rGrad2 := (x - rStop) / ((rStart - rStop) * grad / 100.0)
			rGrad = math.Min(rGrad, rGrad2)
		} else if !dir {
			rGrad = (x - rStart) / ((rStop - rStart) * grad / 100.0)
		} else {
			rGrad = (x - rStop) / ((rStart - rStop) * grad / 100.0)
		}

		if rGrad < 0.0 {
			rGrad = 0.0
		}

		if rGrad > 1.0 {
			rGrad = 1.0
		}

		return alpha * rGrad
	case 3:
		rStop := (-start + 100.0) * float64(yDest) / 200.0
		rStart := (-stop + 100.0) * float64(yDest) / 200.0
		alpha := 255.0

		if y < rStart-1.0 {
			return 0.0
		}

		if y < rStart {
			alpha *= y - (rStart - 1.0)
		}

		if y > rStop+1.0 {
			return 0.0
		}

		if y > rStop {
			alpha *= rStop + 1.0 - y
		}

		if grad == 0.0 {
			return alpha
		}

		var rGrad float64
		if gradDir != 0 {
			rGrad = (y - rStart) / ((rStop - rStart) * grad / 100.0)
			rGrad2 := (y - rStop) / ((rStart - rStop) * grad / 100.0)
			rGrad = math.Min(rGrad, rGrad2)
		} else if dir {
			rGrad = (y - rStart) / ((rStop - rStart) * grad / 100.0)
		} else {
			rGrad = (y - rStop) / ((rStart - rStop) * grad / 100.0)
		}

		if rGrad < 0.0 {
			rGrad = 0.0
		}

		if rGrad > 1.0 {
			rGrad = 1.0
		}

		return alpha * rGrad
	default:
		return 255.0
	}
}

// CombineMasks combines two masks with AND (op=0) or OR (op!=0).
func CombineMasks(m1, m2 []float64, op int) []float64 {
	if len(m1) == 0 {
		return append([]float64(nil), m2...)
	}

	if len(m2) == 0 {
		return append([]float64(nil), m1...)
	}

	n := min(len(m2), len(m1))

	out := make([]float64, n)
	for i := range n {
		if op == 0 {
			out[i] = math.Min(m1[i], m2[i])
		} else {
			out[i] = math.Max(m1[i], m2[i])
		}
	}

	return out
}

// ApplyMask multiplies src alpha by the given mask values.
func ApplyMask(src *PixBuf, mask []float64) {
	if src == nil || len(mask) == 0 {
		return
	}

	n := min(len(mask), src.Width*src.Height)

	for i := range n {
		ai := i*4 + 3
		a := float64(src.Data[ai]) / 255.0
		a *= clamp01(mask[i])
		src.Data[ai] = uint8(clamp01(a)*255 + 0.5)
	}
}
