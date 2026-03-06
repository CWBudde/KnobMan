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

	for y := range src.Height {
		for x := range src.Width {
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

	for y := range src.Height {
		for x := range src.Width {
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
// This keeps the original helper semantics used by existing tests.
func MakeShadow(src *PixBuf, offset, density, diffuse float64, lightDir float64, shadowColor color.RGBA) *PixBuf {
	if src == nil {
		return nil
	}

	alphaScale := clamp01(density / 100.0)
	if alphaScale <= 0 {
		return NewPixBuf(src.Width, src.Height)
	}

	base := NewPixBuf(src.Width, src.Height)
	for y := range src.Height {
		for x := range src.Width {
			a := src.At(x, y).A
			if a == 0 {
				continue
			}

			base.Set(x, y, color.RGBA{
				R: shadowColor.R,
				G: shadowColor.G,
				B: shadowColor.B,
				A: uint8(float64(a)*alphaScale + 0.5),
			})
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

	for y := range src.Height {
		for x := range src.Width {
			c := base.At(x, y)
			if c.A == 0 {
				continue
			}

			out.BlendOver(x+dx, y+dy, c)
		}
	}

	return out
}

// MakeShadowLegacy mirrors legacy Eff.MakeShadow + Eff.Diffuse behavior.
// shadowType: 0=single sample, 1=multi-step directional sweep.
func MakeShadowLegacy(src *PixBuf, inside bool, shadowType int, shadowGradate, offset, lightDir, density, diffuse float64, shadowColor color.RGBA) *PixBuf {
	if src == nil {
		return nil
	}

	out := NewPixBuf(src.Width, src.Height)

	iSD := int(math.Abs(density))
	if iSD <= 0 {
		return out
	}

	rad := lightDir * math.Pi / 180.0
	offX := -math.Sin(rad) * offset
	offY := math.Cos(rad) * offset

	iLoop := 1
	if shadowType != 0 {
		iLoop = int(math.Abs(offset))
		if iLoop == 0 {
			iLoop = 1
		}
	}

	for y := range src.Height {
		for x := range src.Width {
			fValMax := 0.0

			for i := 1; i <= iLoop; i++ {
				fy := float64(y) - offY*float64(i)/float64(iLoop)
				fx := float64(x) - offX*float64(i)/float64(iLoop)
				fVal := sampleAlphaBilinear(src, fx, fy)
				gradMul := (shadowGradate*float64(iLoop-i+1)/float64(iLoop) + (100.0 - shadowGradate)) * 0.01

				fVal *= gradMul
				if fVal > fValMax {
					fValMax = fVal
				}
			}

			if inside {
				fValMax = 255.0 - fValMax
			}

			a := int(fValMax) * iSD / 100
			if a <= 0 {
				continue
			}

			out.Set(x, y, color.RGBA{
				R: shadowColor.R,
				G: shadowColor.G,
				B: shadowColor.B,
				A: uint8(clampInt(a, 0, 255)),
			})
		}
	}

	if diffuse > 0 {
		out = DiffuseLegacy(out, diffuse)
	}

	return out
}

// DiffuseLegacy reproduces the two-pass box blur used by legacy Eff.Diffuse.
func DiffuseLegacy(src *PixBuf, diff float64) *PixBuf {
	if src == nil {
		return nil
	}

	wx := int(float64(src.Width/8) * diff / 100.0)
	if wx <= 0 {
		return src.Clone()
	}

	tmp := NewPixBuf(src.Width, src.Height)
	out := NewPixBuf(src.Width, src.Height)

	for y := range src.Height {
		for x := range src.Width {
			x0 := max(0, x-wx)
			x1 := min(src.Width-1, x+wx)
			n := x1 - x0 + 1
			var sr, sg, sb, sa int

			for xx := x0; xx <= x1; xx++ {
				c := src.At(xx, y)
				sr += int(c.R)
				sg += int(c.G)
				sb += int(c.B)
				sa += int(c.A)
			}

			tmp.Set(x, y, color.RGBA{
				R: uint8(sr / n),
				G: uint8(sg / n),
				B: uint8(sb / n),
				A: uint8(sa / n),
			})
		}
	}

	for x := range src.Width {
		for y := range src.Height {
			y0 := max(0, y-wx)
			y1 := min(src.Height-1, y+wx)
			n := y1 - y0 + 1
			var sr, sg, sb, sa int

			for yy := y0; yy <= y1; yy++ {
				c := tmp.At(x, yy)
				sr += int(c.R)
				sg += int(c.G)
				sb += int(c.B)
				sa += int(c.A)
			}

			out.Set(x, y, color.RGBA{
				R: uint8(sr / n),
				G: uint8(sg / n),
				B: uint8(sb / n),
				A: uint8(sa / n),
			})
		}
	}

	return out
}

// MultiplyAlphaByMask scales src alpha by mask alpha in-place.
func MultiplyAlphaByMask(src, mask *PixBuf) {
	if src == nil || mask == nil {
		return
	}

	n := min(len(src.Data), len(mask.Data))
	for i := 3; i < n; i += 4 {
		src.Data[i] = uint8((int(src.Data[i])*int(mask.Data[i]) + 127) / 255)
	}
}

func sampleAlphaBilinear(src *PixBuf, fx, fy float64) float64 {
	x0 := int(math.Floor(fx))
	y0 := int(math.Floor(fy))
	tx := fx - float64(x0)
	ty := fy - float64(y0)

	a00 := sampleAlphaAt(src, x0, y0)
	a01 := sampleAlphaAt(src, x0+1, y0)
	a10 := sampleAlphaAt(src, x0, y0+1)
	a11 := sampleAlphaAt(src, x0+1, y0+1)
	v0 := a00 + (a01-a00)*tx
	v1 := a10 + (a11-a10)*tx

	return v0 + (v1-v0)*ty
}

func sampleAlphaAt(src *PixBuf, x, y int) float64 {
	if src == nil || x < 0 || y < 0 || x >= src.Width || y >= src.Height {
		return 0
	}

	return float64(src.At(x, y).A)
}

// HilightLegacy mirrors legacy Eff.Hilight in-place brightness shaping.
func HilightLegacy(img *PixBuf, sdir, sden, edir, offset, eden float64) {
	if img == nil || img.Width <= 0 || img.Height <= 0 {
		return
	}

	w, h := img.Width, img.Height

	rLX := -math.Sin(sdir * math.Pi / 180.0)
	rLY := math.Cos(sdir * math.Pi / 180.0)
	edirx := -math.Sin(edir * math.Pi / 180.0)
	ediry := math.Cos(edir * math.Pi / 180.0)

	iTemp := make([]int, w*h)
	iTemp2 := make([]int, w*h)
	table := make([]float64, 64)

	sigma := offset

	xMax := int(math.Min(32.0, sigma*3.0+1.0))
	if xMax >= len(table) {
		xMax = len(table) - 1
	}

	if xMax < 0 {
		xMax = 0
	}

	if sigma != 0 {
		for x := range xMax {
			table[x] = math.Exp(float64(-x*x) / (2.0 * sigma * sigma))
		}
	}

	for y := range h {
		for x := range w {
			ax := int(img.At(x, y).A)
			ax0, ax1 := 0, 0
			ay0, ay1 := 0, 0

			if x >= 1 {
				ax0 = int(img.At(x-1, y).A)
			}

			if x < w-1 {
				ax1 = int(img.At(x+1, y).A)
			}

			if y >= 1 {
				ay0 = int(img.At(x, y-1).A)
			}

			if y < h-1 {
				ay1 = int(img.At(x, y+1).A)
			}

			iTemp[y*w+x] = int((float64(ay1-ay0)*ediry + float64(ax1-ax0)*edirx) * float64(ax))
		}
	}

	for y := range h {
		row := y * w
		for x3 := range w {
			acc := 0

			for i := -xMax; i <= xMax; i++ {
				xx := x3 + i
				if xx < 0 || xx >= w {
					continue
				}

				ai := i
				if ai < 0 {
					ai = -ai
				}

				acc += int(table[ai] * float64(iTemp[row+xx]))
			}

			iTemp2[row+x3] = acc
		}
	}

	for i := range iTemp {
		iTemp[i] = 0
	}

	for x := range w {
		for y3 := range h {
			acc := 0

			for i := -xMax; i <= xMax; i++ {
				yy := y3 + i
				if yy < 0 || yy >= h {
					continue
				}

				ai := i
				if ai < 0 {
					ai = -ai
				}

				acc += int(table[ai] * float64(iTemp2[yy*w+x]))
			}

			iTemp[y3*w+x] = acc
		}
	}

	for y := range h {
		rY := (float64(y)+0.5)*2.0/float64(h) - 1.0

		for x := range w {
			rX := (float64(x)+0.5)*2.0/float64(w) - 1.0
			br := -(rLX*rX + rLY*rY) * sden * 2.55
			delta := int(br + float64(iTemp[y*w+x])*eden)
			c := img.At(x, y)
			r := clampInt(int(c.R)+delta, 0, 255)
			g := clampInt(int(c.G)+delta, 0, 255)
			b := clampInt(int(c.B)+delta, 0, 255)
			img.Set(x, y, color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: c.A})
		}
	}
}

// MakeHighlight creates a blurred white highlight from src alpha.
func MakeHighlight(src *PixBuf, offset, density, diffuse float64, lightDir float64) *PixBuf {
	return MakeShadow(src, offset, density, diffuse, lightDir, color.RGBA{255, 255, 255, 255})
}
