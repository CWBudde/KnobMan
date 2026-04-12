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

	angleRad := -angle * math.Pi / 180.0

	m := agg.Translation(0, 0)
	m.Multiply(
		rotateAround(angleRad, cx, cy),
	)
	m.Multiply(agg.Translation(offX, offY))
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
	for y := range dst.Height {
		fy := float64(y) + 0.5
		for x := range dst.Width {
			fx := float64(x) + 0.5
			sx := m[0]*fx + m[2]*fy + m[4] - 0.5
			sy := m[1]*fx + m[3]*fy + m[5] - 0.5
			dst.Set(x, y, sampleColorBilinear(src, sx, sy))
		}
	}
}

func sampleColorBilinear(src *PixBuf, fx, fy float64) color.RGBA {
	x0 := int(math.Floor(fx))
	y0 := int(math.Floor(fy))
	tx := fx - float64(x0)
	ty := fy - float64(y0)

	c00 := sampleColorAt(src, x0, y0)
	c01 := sampleColorAt(src, x0+1, y0)
	c10 := sampleColorAt(src, x0, y0+1)
	c11 := sampleColorAt(src, x0+1, y0+1)

	xw0 := 1.0 - tx
	yw0 := 1.0 - ty

	xy00 := xw0 * yw0 * float64(c00.A)
	xy01 := tx * yw0 * float64(c01.A)
	xy10 := xw0 * ty * float64(c10.A)
	xy11 := tx * ty * float64(c11.A)
	at := xy00 + xy01 + xy10 + xy11
	if at != 0 {
		xy00 /= at
		xy01 /= at
		xy10 /= at
		xy11 /= at
	} else {
		xy00, xy01, xy10, xy11 = 0, 0, 0, 0
	}

	rr := float64(c00.R)*xy00 + float64(c01.R)*xy01 + float64(c10.R)*xy10 + float64(c11.R)*xy11
	gg := float64(c00.G)*xy00 + float64(c01.G)*xy01 + float64(c10.G)*xy10 + float64(c11.G)*xy11
	bb := float64(c00.B)*xy00 + float64(c01.B)*xy01 + float64(c10.B)*xy10 + float64(c11.B)*xy11
	aa := float64(c00.A)*xw0*yw0 + float64(c01.A)*tx*yw0 + float64(c10.A)*xw0*ty + float64(c11.A)*tx*ty

	return color.RGBA{
		R: uint8(clampInt(int(rr), 0, 255)),
		G: uint8(clampInt(int(gg), 0, 255)),
		B: uint8(clampInt(int(bb), 0, 255)),
		A: uint8(clampInt(int(aa), 0, 255)),
	}
}

func sampleColorAt(src *PixBuf, x, y int) color.RGBA {
	if src == nil || x < 0 || y < 0 || x >= src.Width || y >= src.Height {
		return color.RGBA{}
	}

	return src.At(x, y)
}
