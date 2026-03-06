package render

import "math"

// BuildMask creates a per-pixel mask in [0,1].
// maskType: 0=rotation, 1=radial, 2=horizontal, 3=vertical.
func BuildMask(w, h int, maskType int, start, stop, grad float64, gradDir int) []float64 {
	if w <= 0 || h <= 0 {
		return nil
	}
	mask := make([]float64, w*h)
	cx := float64(w-1) * 0.5
	cy := float64(h-1) * 0.5
	hw := math.Max(1, float64(w)*0.5)
	hh := math.Max(1, float64(h)*0.5)
	maxR := math.Max(1, math.Hypot(hw, hh))

	lo, hi := start, stop
	if lo > hi {
		lo, hi = hi, lo
	}
	span := math.Max(1e-6, hi-lo)
	edge := span * clamp01(grad/100.0) * 0.5

	for y := 0; y < h; y++ {
		fy := float64(y) - cy
		for x := 0; x < w; x++ {
			fx := float64(x) - cx
			var v float64
			switch maskType {
			case 0: // rotation angle mapped to [-180, 180]
				v = math.Atan2(-fy, fx) * 180.0 / math.Pi
			case 1: // radial distance mapped to [-140, 140]
				v = math.Hypot(fx, fy)/maxR*280.0 - 140.0
			case 2: // horizontal mapped to [-140, 140]
				v = fx / hw * 140.0
			case 3: // vertical mapped to [-140, 140]
				v = fy / hh * 140.0
			default:
				v = fx / hw * 140.0
			}

			m := 0.0
			if v >= lo && v <= hi {
				m = 1.0
			}
			if edge > 0 {
				if v > lo-edge && v < lo {
					m = math.Max(m, (v-(lo-edge))/edge)
				}
				if v > hi && v < hi+edge {
					m = math.Max(m, 1.0-(v-hi)/edge)
				}
			}
			m = clamp01(m)
			if gradDir != 0 {
				m = 1.0 - m
			}
			mask[y*w+x] = m
		}
	}
	return mask
}

// CombineMasks combines two masks with AND (op=0) or OR (op!=0).
func CombineMasks(m1, m2 []float64, op int) []float64 {
	if len(m1) == 0 {
		return append([]float64(nil), m2...)
	}
	if len(m2) == 0 {
		return append([]float64(nil), m1...)
	}
	n := len(m1)
	if len(m2) < n {
		n = len(m2)
	}
	out := make([]float64, n)
	for i := 0; i < n; i++ {
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
	n := src.Width * src.Height
	if len(mask) < n {
		n = len(mask)
	}
	for i := 0; i < n; i++ {
		ai := i*4 + 3
		a := float64(src.Data[ai]) / 255.0
		a *= clamp01(mask[i])
		src.Data[ai] = uint8(clamp01(a)*255 + 0.5)
	}
}
