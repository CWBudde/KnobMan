package render

import (
	"image/color"
	"math"

	agg "github.com/cwbudde/agg_go"
)

// BuildMatrix builds a forward affine transform matrix:
// x' = a*x + c*y + e
// y' = b*x + d*y + f
// Zoom is interpreted as percentage (100 = identity).
//
// The implementation mirrors legacy JKnobMan behavior using agg_go transform builders.
func BuildMatrix(srcWidth, srcHeight int, zoomX, zoomY, angle, offX, offY, centerX, centerY float64, keepDir bool) [6]float64 {
	if srcWidth <= 0 || srcHeight <= 0 {
		return [6]float64{1, 0, 0, 1, 0, 0}
	}

	if zoomX == 0 {
		zoomX = 100
	}

	if zoomY == 0 {
		zoomY = 100
	}

	w := float64(srcWidth)
	h := float64(srcHeight)
	cx := (centerX + 50.0) * 0.01 * w
	cy := (50.0 - centerY) * 0.01 * h

	angleRad := angle * math.Pi / 180.0

	m := agg.Translation(0, 0)
	m.Multiply(
		rotateAround(angleRad, cx, cy),
	)
	m.Multiply(agg.Translation(-offX, offY))
	if keepDir {
		m.Multiply(rotateAround(-angleRad, w*0.5, h*0.5))
	}
	m.Multiply(scaleAround(100.0/zoomX, 100.0/zoomY, w*0.5, h*0.5))

	return m.AffineMatrix
}

func rotateAround(angle, x, y float64) *agg.Transformations {
	m := agg.Translation(-x, -y)
	m.Multiply(agg.Rotation(angle))
	m.Multiply(agg.Translation(x, y))
	return m
}

func scaleAround(sx, sy, x, y float64) *agg.Transformations {
	if sx == 0 {
		sx = 1
	}

	if sy == 0 {
		sy = 1
	}

	m := agg.Translation(-x, -y)
	m.Multiply(agg.Scaling(sx, sy))
	m.Multiply(agg.Translation(x, y))
	return m
}

// TransformBilinear applies matrix m from src to dst with bilinear resampling.
func TransformBilinear(dst, src *PixBuf, m [6]float64) {
	if dst == nil || src == nil || dst.Width == 0 || dst.Height == 0 || src.Width == 0 || src.Height == 0 {
		return
	}

	invTr := agg.Transformations{AffineMatrix: m}
	ok := invTr.Invert()
	if !ok {
		dst.Clear(color.RGBA{})
		return
	}
	inv := invTr.AffineMatrix

	for y := range dst.Height {
		fy := float64(y) + 0.5

		for x := range dst.Width {
			fx := float64(x) + 0.5
			sx := inv[0]*fx + inv[2]*fy + inv[4] - 0.5
			sy := inv[1]*fx + inv[3]*fy + inv[5] - 0.5
			dst.Set(x, y, sampleBilinear(src, sx, sy))
		}
	}
}

func sampleBilinear(src *PixBuf, x, y float64) color.RGBA {
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	x1 := x0 + 1
	y1 := y0 + 1
	fx := x - float64(x0)
	fy := y - float64(y0)

	c00 := src.At(x0, y0)
	c10 := src.At(x1, y0)
	c01 := src.At(x0, y1)
	c11 := src.At(x1, y1)

	lerp := func(a, b uint8, t float64) float64 {
		return float64(a) + (float64(b)-float64(a))*t
	}

	r0 := lerp(c00.R, c10.R, fx)
	r1 := lerp(c01.R, c11.R, fx)
	g0 := lerp(c00.G, c10.G, fx)
	g1 := lerp(c01.G, c11.G, fx)
	b0 := lerp(c00.B, c10.B, fx)
	b1 := lerp(c01.B, c11.B, fx)
	a0 := lerp(c00.A, c10.A, fx)
	a1 := lerp(c01.A, c11.A, fx)

	return color.RGBA{
		R: uint8(clamp01((r0+(r1-r0)*fy)/255.0)*255 + 0.5),
		G: uint8(clamp01((g0+(g1-g0)*fy)/255.0)*255 + 0.5),
		B: uint8(clamp01((b0+(b1-b0)*fy)/255.0)*255 + 0.5),
		A: uint8(clamp01((a0+(a1-a0)*fy)/255.0)*255 + 0.5),
	}
}
