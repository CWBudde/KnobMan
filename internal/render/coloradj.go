package render

import (
	"image/color"
)

// ApplyColorAdjust applies layer-wide alpha and color adjustments.
// alpha is percent (100 = unchanged), brightness/contrast/saturation are
// percent deltas, hue is a degree shift.
func ApplyColorAdjust(dst, src *PixBuf, alpha, brightness, contrast, saturation, hue float64) {
	if dst == nil || src == nil || dst.Width == 0 || dst.Height == 0 || src.Width == 0 || src.Height == 0 {
		return
	}

	w := min(dst.Width, src.Width)
	h := min(dst.Height, src.Height)

	alphaScale := clamp01(alpha / 100.0)

	for y := range h {
		for x := range w {
			c := src.At(x, y)
			if c.A == 0 {
				dst.Set(x, y, c)
				continue
			}

			r, g, b := applyLegacyHLSAdjust(c.R, c.G, c.B, brightness, contrast, saturation, hue)
			a := clamp01(float64(c.A) / 255.0 * alphaScale)
			dst.Set(x, y, color.RGBA{
				R: r,
				G: g,
				B: b,
				A: uint8(a*255 + 0.5),
			})
		}
	}
}

func applyLegacyHLSAdjust(r, g, b uint8, brightness, contrast, saturation, hue float64) (uint8, uint8, uint8) {
	h, l, s := rgbToLegacyHLS(int(r), int(g), int(b))

	l = int((float64(l-120)*(contrast+100.0))/100.0 + 120.0)

	l = int(float64(l) + brightness*240.0/100.0)
	if l >= 240 {
		l = 239
	}

	if l < 0 {
		l = 0
	}

	s = int(float64(s) * (saturation + 100.0) / 100.0)
	h += int(hue * (2.0 / 3.0))

	r8, g8, b8 := legacyHLSToRGB(h, l, s)

	return uint8(r8), uint8(g8), uint8(b8)
}

func rgbToLegacyHLS(r, g, b int) (h, l, s int) {
	cmax := max(r, max(g, b))
	cmin := min(r, min(g, b))

	l = ((cmax+cmin)*240 + 255) / 510
	if cmax == cmin {
		return 0, l, 0
	}

	if l < 120 {
		s = ((cmax-cmin)*240 + (cmax+cmin)/2) / (cmax + cmin)
	} else {
		s = ((cmax-cmin)*240 + (510-cmax-cmin)/2) / (510 - cmax - cmin)
	}

	rr := ((cmax-r)*40 + (cmax-cmin)/2) / (cmax - cmin)
	gg := ((cmax-g)*40 + (cmax-cmin)/2) / (cmax - cmin)

	bb := ((cmax-b)*40 + (cmax-cmin)/2) / (cmax - cmin)
	switch {
	case r == cmax:
		h = bb - gg
	case g == cmax:
		h = 80 + rr - bb
	default:
		h = 160 + gg - rr
	}

	if h < 0 {
		h += 240
	}

	if h > 240 {
		h -= 240
	}

	return h, l, s
}

func legacyHLSToRGB(h, l, s int) (r, g, b int) {
	if l > 240 {
		l = 240
	}

	if l < 0 {
		l = 0
	}

	if s > 240 {
		s = 240
	}

	if s <= 0 {
		v := l * 255 / 240
		return v, v, v
	}

	var tmp2 int
	if l <= 120 {
		tmp2 = (l*(240+s) + 120) / 240
	} else {
		tmp2 = l + s - (l*s+120)/240
	}

	tmp1 := 2*l - tmp2

	r = (legacyHueToRGB(tmp1, tmp2, h+80)*255 + 120) / 240
	g = (legacyHueToRGB(tmp1, tmp2, h)*255 + 120) / 240
	b = (legacyHueToRGB(tmp1, tmp2, h-80)*255 + 120) / 240

	return r, g, b
}

func legacyHueToRGB(n1, n2, h int) int {
	for h < 0 {
		h += 240
	}

	for h > 240 {
		h -= 240
	}

	switch {
	case h < 40:
		return n1 + ((n2-n1)*h+20)/40
	case h < 120:
		return n2
	case h < 160:
		return n1 + ((n2-n1)*(160-h)+20)/40
	default:
		return n1
	}
}
