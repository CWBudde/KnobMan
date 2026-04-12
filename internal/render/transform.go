package render

import (
	"image/color"
	"math"

	agg "github.com/cwbudde/agg_go"
)

// BuildMatrix builds a destination-to-source affine transform matrix:
// sx = a*dx + c*dy + e
// sy = b*dx + d*dy + f
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

	angleRad := -angle * math.Pi / 180.0

	m := agg.Translation(0, 0)
	m.Multiply(
		rotateAround(angleRad, cx, cy),
	)
	// Legacy JKnobMan applies X offset with opposite sign relative to the
	// matrix translation and Y offset with the same sign.
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

	a := Agg2DForPixBuf(dst)

	srcImg := AggImageForPixBuf(src)
	if a == nil || srcImg == nil {
		return
	}

	tr := agg.NewTransformationsFromValues(m[0], m[1], m[2], m[3], m[4], m[5])
	if tr == nil {
		return
	}

	if !tr.Invert() {
		return
	}

	a.SetTransformations(tr)
	a.ImageFilter(agg.Bilinear)
	a.ImageResample(agg.NoResample)
	_ = a.TransformImageSimple(srcImg, 0, 0, float64(src.Width), float64(src.Height))
}
