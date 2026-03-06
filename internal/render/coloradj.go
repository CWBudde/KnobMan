package render

import (
	"image/color"
	"math"
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
	bright := brightness / 100.0
	contrastFactor := 1.0 + contrast/100.0
	satFactor := 1.0 + saturation/100.0

	for y := range h {
		for x := range w {
			c := src.At(x, y)
			if c.A == 0 {
				dst.Set(x, y, c)
				continue
			}

			r := float64(c.R) / 255.0
			g := float64(c.G) / 255.0
			b := float64(c.B) / 255.0

			// Brightness and contrast in linear RGB.
			r = (r+bright-0.5)*contrastFactor + 0.5
			g = (g+bright-0.5)*contrastFactor + 0.5
			b = (b+bright-0.5)*contrastFactor + 0.5

			// Saturation via grayscale interpolation.
			gray := 0.299*r + 0.587*g + 0.114*b
			r = gray + (r-gray)*satFactor
			g = gray + (g-gray)*satFactor
			b = gray + (b-gray)*satFactor

			if hue != 0 {
				hh, ss, vv := rgbToHSV(clamp01(r), clamp01(g), clamp01(b))

				hh += hue
				for hh < 0 {
					hh += 360
				}

				for hh >= 360 {
					hh -= 360
				}

				r, g, b = hsvToRGB(hh, ss, vv)
			}

			a := clamp01(float64(c.A) / 255.0 * alphaScale)
			dst.Set(x, y, rgbaFromFloat(r, g, b, a))
		}
	}
}

func rgbToHSV(r, g, b float64) (h, s, v float64) {
	mx := math.Max(r, math.Max(g, b))
	mn := math.Min(r, math.Min(g, b))
	d := mx - mn

	v = mx
	if mx == 0 {
		s = 0
	} else {
		s = d / mx
	}

	if d == 0 {
		h = 0
		return h, s, v
	}

	switch mx {
	case r:
		h = 60 * math.Mod((g-b)/d, 6)
	case g:
		h = 60 * ((b-r)/d + 2)
	default:
		h = 60 * ((r-g)/d + 4)
	}

	if h < 0 {
		h += 360
	}

	return h, s, v
}

func hsvToRGB(h, s, v float64) (r, g, b float64) {
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60.0, 2)-1))
	m := v - c

	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}

	return r + m, g + m, b + m
}

func rgbaFromFloat(r, g, b, a float64) color.RGBA {
	return color.RGBA{
		R: uint8(clamp01(r)*255 + 0.5),
		G: uint8(clamp01(g)*255 + 0.5),
		B: uint8(clamp01(b)*255 + 0.5),
		A: uint8(clamp01(a)*255 + 0.5),
	}
}
