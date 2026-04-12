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

// TransformBilinear applies matrix m from src to dst with agg_go affine image
// rendering configured for bilinear sampling.
func TransformBilinear(dst, src *PixBuf, m [6]float64) {
	if dst == nil || src == nil || dst.Width == 0 || dst.Height == 0 || src.Width == 0 || src.Height == 0 {
		return
	}

	dst.Clear(color.RGBA{})

	invTr := agg.Transformations{AffineMatrix: m}
	if !invTr.Invert() {
		return
	}

	srcImg := AggImageForPixBuf(src)
	a := Agg2DForPixBuf(dst)
	if srcImg == nil || a == nil {
		return
	}
	a.ImageFilter(agg.FilterBilinear)
	a.ImageResample(agg.NoResample)
	a.AffineImageResamplePolicy(agg.AffineImageResamplePreferFiltered)
	a.SetTransformations(&invTr)

	if err := a.TransformImageSimple(srcImg, 0, 0, float64(src.Width), float64(src.Height)); err != nil {
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
			clip := clamp01(sampleAlphaBilinear(src, sx, sy) / 255.0)
			if clip >= 1.0 {
				continue
			}

			i := y*dst.Stride + x*4
			dst.Data[i+0] = uint8(float64(dst.Data[i+0])*clip + 0.5)
			dst.Data[i+1] = uint8(float64(dst.Data[i+1])*clip + 0.5)
			dst.Data[i+2] = uint8(float64(dst.Data[i+2])*clip + 0.5)
			dst.Data[i+3] = uint8(float64(dst.Data[i+3])*clip + 0.5)
		}
	}
}
