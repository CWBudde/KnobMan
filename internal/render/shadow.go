package render

import (
	"image/color"
	"math"
)

// Gaussian1D builds a normalized 1D Gaussian kernel.
func Gaussian1D(radius float64) []float64 {
	if radius <= 0 {
		return []float64{1}
	}
	sigma := math.Max(0.5, radius*0.5)
	r := int(math.Ceil(radius))
	k := make([]float64, 2*r+1)
	sum := 0.0
	for i := -r; i <= r; i++ {
		v := math.Exp(-(float64(i * i)) / (2 * sigma * sigma))
		k[i+r] = v
		sum += v
	}
	if sum == 0 {
		return []float64{1}
	}
	for i := range k {
		k[i] /= sum
	}
	return k
}

// BlurH applies horizontal convolution.
func BlurH(src *PixBuf, kernel []float64) *PixBuf {
	if src == nil || len(kernel) == 0 {
		return src.Clone()
	}
	dst := NewPixBuf(src.Width, src.Height)
	r := len(kernel) / 2
	for y := 0; y < src.Height; y++ {
		for x := 0; x < src.Width; x++ {
			var rr, gg, bb, aa float64
			for k := -r; k <= r; k++ {
				sx := x + k
				if sx < 0 || sx >= src.Width {
					continue
				}
				c := src.At(sx, y)
				w := kernel[k+r]
				rr += float64(c.R) * w
				gg += float64(c.G) * w
				bb += float64(c.B) * w
				aa += float64(c.A) * w
			}
			dst.Set(x, y, color.RGBA{R: uint8(clamp01(rr/255.0)*255 + 0.5), G: uint8(clamp01(gg/255.0)*255 + 0.5), B: uint8(clamp01(bb/255.0)*255 + 0.5), A: uint8(clamp01(aa/255.0)*255 + 0.5)})
		}
	}
	return dst
}

// BlurV applies vertical convolution.
func BlurV(src *PixBuf, kernel []float64) *PixBuf {
	if src == nil || len(kernel) == 0 {
		return src.Clone()
	}
	dst := NewPixBuf(src.Width, src.Height)
	r := len(kernel) / 2
	for y := 0; y < src.Height; y++ {
		for x := 0; x < src.Width; x++ {
			var rr, gg, bb, aa float64
			for k := -r; k <= r; k++ {
				sy := y + k
				if sy < 0 || sy >= src.Height {
					continue
				}
				c := src.At(x, sy)
				w := kernel[k+r]
				rr += float64(c.R) * w
				gg += float64(c.G) * w
				bb += float64(c.B) * w
				aa += float64(c.A) * w
			}
			dst.Set(x, y, color.RGBA{R: uint8(clamp01(rr/255.0)*255 + 0.5), G: uint8(clamp01(gg/255.0)*255 + 0.5), B: uint8(clamp01(bb/255.0)*255 + 0.5), A: uint8(clamp01(aa/255.0)*255 + 0.5)})
		}
	}
	return dst
}

// MakeShadow creates a blurred, offset alpha shadow from src.
func MakeShadow(src *PixBuf, offset, density, diffuse float64, lightDir float64, shadowColor color.RGBA) *PixBuf {
	if src == nil {
		return nil
	}
	alphaScale := clamp01(density / 100.0)
	if alphaScale <= 0 {
		return NewPixBuf(src.Width, src.Height)
	}
	base := NewPixBuf(src.Width, src.Height)
	for y := 0; y < src.Height; y++ {
		for x := 0; x < src.Width; x++ {
			a := src.At(x, y).A
			if a == 0 {
				continue
			}
			base.Set(x, y, color.RGBA{R: shadowColor.R, G: shadowColor.G, B: shadowColor.B, A: uint8(float64(a)*alphaScale + 0.5)})
		}
	}
	if diffuse > 0 {
		k := Gaussian1D(diffuse)
		base = BlurV(BlurH(base, k), k)
	}

	out := NewPixBuf(src.Width, src.Height)
	rad := lightDir * math.Pi / 180.0
	dx := int(math.Round(math.Cos(rad) * offset))
	dy := int(math.Round(-math.Sin(rad) * offset))
	for y := 0; y < src.Height; y++ {
		for x := 0; x < src.Width; x++ {
			c := base.At(x, y)
			if c.A == 0 {
				continue
			}
			out.BlendOver(x+dx, y+dy, c)
		}
	}
	return out
}

// MakeHighlight creates a blurred white highlight from src alpha.
func MakeHighlight(src *PixBuf, offset, density, diffuse float64, lightDir float64) *PixBuf {
	return MakeShadow(src, offset, density, diffuse, lightDir, color.RGBA{255, 255, 255, 255})
}
