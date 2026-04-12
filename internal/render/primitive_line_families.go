package render

import (
	"math"

	agg "github.com/cwbudde/agg_go"
	"knobman/internal/model"
)

func renderLine(dst *PixBuf, p *model.Primitive) {
	if canRenderLineAgg(p) && renderLineAgg(dst, p) {
		return
	}

	base := primitiveColor(p)
	rWidth := float64(dst.Width) * p.Width.Val / 400.0
	rLenY := float64(dst.Height)*p.Length.Val/100.0 - rWidth
	if rLenY < rWidth {
		rLenY = rWidth
	}
	rXC := float64(dst.Width) * 0.5
	rD := 1.0 - p.Diffuse.Val/100.0
	rWidth2 := (rWidth - 1.0) * rD

	for y := 0; y < dst.Height; y++ {
		rY := float64(y) + 0.5
		rYP := rY
		if rY < rWidth {
			rYP = rWidth
		} else if rY >= rLenY {
			rYP = rLenY
		}
		rDY := rY - rYP
		for x := 0; x < dst.Width; x++ {
			rX := float64(x) + 0.5
			rDX := rX - rXC
			rDistance := math.Hypot(rDX, rDY)
			if rDistance >= rWidth {
				continue
			}

			alpha := 255
			if rDistance >= rWidth2 {
				alpha = int(255.0 - 255.0*(rDistance-rWidth2)/(rWidth-rWidth2))
			}
			pix := changeBrightnessRGBA(base, clampInt(int((255.0-rDistance/rWidth*255.0)*p.Specular.Val/100.0), 0, 255))
			pix.A = uint8(clampInt(alpha, 0, 255))
			dst.Set(x, y, pix)
		}
	}
}

func canRenderLineAgg(p *model.Primitive) bool {
	return false
}

func renderLineAgg(dst *PixBuf, p *model.Primitive) bool {
	ctx := AggContextForPixBuf(dst)
	if ctx == nil {
		return false
	}

	rWidth := float64(dst.Width) * p.Width.Val / 400.0
	rLenY := float64(dst.Height)*p.Length.Val/100.0 - rWidth
	if rLenY < rWidth {
		rLenY = rWidth
	}
	if rWidth <= 0 {
		return false
	}

	base := primitiveColor(p)
	ctx.SetColor(agg.Color{R: base.R, G: base.G, B: base.B, A: base.A})
	ctx.SetLineWidth(rWidth * 2.0)
	ctx.GetAgg2D().LineCap(agg.CapRound)
	ctx.DrawLine(float64(dst.Width)*0.5, rWidth, float64(dst.Width)*0.5, rLenY)
	return true
}

func renderRadiateLines(dst *PixBuf, p *model.Primitive) {
	if canRenderRadiateLinesAgg(p) && renderRadiateLinesAgg(dst, p) {
		return
	}

	if dst.Width == 0 || dst.Height == 0 || p.AngleStep.Val == 0.0 {
		return
	}

	base := primitiveColor(p)
	vCX := float64(dst.Width) * 0.5
	vCY := float64(dst.Height) * 0.5
	rWidth := vCX*p.Width.Val/200.0 + 1.0
	rD := 1.0 - p.Diffuse.Val/100.0
	rWidth2 := (rWidth - 1.0) * rD
	rLenY := float64(dst.Height) * p.Length.Val / 200.0
	if rLenY < rWidth {
		rLenY = rWidth
	}

	aspectX := 1.0
	aspectY := 1.0
	if p.Aspect.Val > 0.0 {
		aspectX = (100.0 - math.Min(p.Aspect.Val, 99.0)) / 100.0
	}
	if p.Aspect.Val < 0.0 {
		aspectY = (100.0 + math.Max(p.Aspect.Val, -99.0)) / 100.0
	}
	aspectX *= float64(dst.Width) / float64(dst.Height)

	p1x, p1y := 0.0, -vCY+rWidth
	p2x, p2y := 0.0, -vCY+rLenY
	tc1x, tc1y := p1x*aspectX+vCX, p1y*aspectY+vCY
	tc2x, tc2y := p2x*aspectX+vCX, p2y*aspectY+vCY

	for y := 0; y < dst.Height; y++ {
		py := float64(y) + 0.5
		for x := 0; x < dst.Width; x++ {
			px := float64(x) + 0.5
			d := linePointDistance(tc1x, tc1y, tc2x, tc2y, px, py)

			for rTh := p.AngleStep.Val; rTh <= 180.0; rTh += p.AngleStep.Val {
				rad := rTh * math.Pi / 180.0
				sinT, cosT := math.Sin(rad), math.Cos(rad)

				t1x, t1y := rotatePoint(p1x, p1y, sinT, cosT)
				t2x, t2y := rotatePoint(p2x, p2y, sinT, cosT)
				t1x, t1y = t1x*aspectX+vCX, t1y*aspectY+vCY
				t2x, t2y = t2x*aspectX+vCX, t2y*aspectY+vCY
				d = math.Min(d, linePointDistance(t1x, t1y, t2x, t2y, px, py))

				t1x, t1y = rotatePoint(p1x, p1y, -sinT, cosT)
				t2x, t2y = rotatePoint(p2x, p2y, -sinT, cosT)
				t1x, t1y = t1x*aspectX+vCX, t1y*aspectY+vCY
				t2x, t2y = t2x*aspectX+vCX, t2y*aspectY+vCY
				d = math.Min(d, linePointDistance(t1x, t1y, t2x, t2y, px, py))
			}

			if d >= rWidth {
				continue
			}

			alpha := 255
			if d >= rWidth2 {
				alpha = int(255.0 - (d-rWidth2)/(rWidth-rWidth2)*255.0)
			}
			pix := changeBrightnessRGBA(base, int((rWidth-d)/rWidth*255.0*p.Specular.Val/100.0))
			pix.A = uint8(clampInt(alpha, 0, 255))
			dst.Set(x, y, pix)
		}
	}
}

func canRenderRadiateLinesAgg(p *model.Primitive) bool {
	return false
}

func renderRadiateLinesAgg(dst *PixBuf, p *model.Primitive) bool {
	if dst.Width == 0 || dst.Height == 0 || p.AngleStep.Val == 0.0 {
		return false
	}

	ctx := AggContextForPixBuf(dst)
	if ctx == nil {
		return false
	}

	base := primitiveColor(p)
	vCX := float64(dst.Width) * 0.5
	vCY := float64(dst.Height) * 0.5
	rWidth := vCX*p.Width.Val/200.0 + 1.0
	rLenY := float64(dst.Height) * p.Length.Val / 200.0
	if rLenY < rWidth {
		rLenY = rWidth
	}
	if rWidth <= 0 {
		return false
	}

	aspectX := 1.0
	aspectY := 1.0
	if p.Aspect.Val > 0.0 {
		aspectX = (100.0 - math.Min(p.Aspect.Val, 99.0)) / 100.0
	}
	if p.Aspect.Val < 0.0 {
		aspectY = (100.0 + math.Max(p.Aspect.Val, -99.0)) / 100.0
	}
	aspectX *= float64(dst.Width) / float64(dst.Height)

	p1x, p1y := 0.0, -vCY+rWidth
	p2x, p2y := 0.0, -vCY+rLenY

	ctx.SetColor(agg.Color{R: base.R, G: base.G, B: base.B, A: base.A})
	ctx.SetLineWidth(rWidth * 2.0)
	ctx.GetAgg2D().LineCap(agg.CapRound)

	drawRadiateStroke := func(x1, y1, x2, y2 float64) {
		ctx.DrawLine(x1*aspectX+vCX, y1*aspectY+vCY, x2*aspectX+vCX, y2*aspectY+vCY)
	}

	drawRadiateStroke(p1x, p1y, p2x, p2y)
	for rTh := p.AngleStep.Val; rTh <= 180.0; rTh += p.AngleStep.Val {
		rad := rTh * math.Pi / 180.0
		sinT, cosT := math.Sin(rad), math.Cos(rad)

		t1x, t1y := rotatePoint(p1x, p1y, sinT, cosT)
		t2x, t2y := rotatePoint(p2x, p2y, sinT, cosT)
		drawRadiateStroke(t1x, t1y, t2x, t2y)

		t1x, t1y = rotatePoint(p1x, p1y, -sinT, cosT)
		t2x, t2y = rotatePoint(p2x, p2y, -sinT, cosT)
		drawRadiateStroke(t1x, t1y, t2x, t2y)
	}

	return true
}

func renderParallelLines(dst *PixBuf, p *model.Primitive, horizontal bool) {
	if canRenderParallelLinesAgg(p) && renderParallelLinesAgg(dst, p, horizontal) {
		return
	}

	base := primitiveColor(p)
	rCX := float64(dst.Width) * 0.5
	rCY := float64(dst.Height) * 0.5
	rAY := 1.0
	rAX := 1.0
	if p.Aspect.Val > 0.0 {
		rAX = 100.0 / (100.0 - math.Min(p.Aspect.Val, 99.0))
	}
	if p.Aspect.Val < 0.0 {
		rAY = 100.0 / (100.0 + math.Max(p.Aspect.Val, -99.0))
	}

	if horizontal {
		rLen := rCX / rAX * p.Length.Val / 100.0
		rYArea := rCY / rAY
		rWidth := rYArea*p.Width.Val/200.0 + 1.0
		rWidth2 := (rWidth - 1.0) * (1.0 - p.Diffuse.Val/100.0)
		if dst.Width == 0 || dst.Height == 0 {
			return
		}

		for y := 0; y < dst.Height; y++ {
			rY := -((float64(y) + 0.5) - rCY)
			rYA := math.Abs(rY)
			for x := 0; x < dst.Width; x++ {
				rX := float64(x) + 0.5 - rCX
				rXA := math.Abs(rX)
				rMin := 1.0
				alpha := 0.0
				if rYA < rYArea {
					rMin = 100000.0
					yy := 0.0
					for yy <= 100.0 {
						r := 0.0
						if rXA < rLen {
							r = yy * (rYArea - rWidth) / 100.0
							r = math.Abs(r - rYA)
						} else {
							yyy := yy * (rYArea - rWidth) / 100.0
							r = math.Hypot(rXA-rLen, rYA-yyy)
						}
						if r < rMin {
							rMin = r
						}
						if p.Step.Val == 0.0 {
							break
						}
						yy += p.Step.Val
					}
					if rMin < rWidth {
						if rMin < rWidth2 {
							alpha = 255.0
						} else {
							alpha = 255.0 - (rMin-rWidth2)/(rWidth-rWidth2)*255.0
						}
					}
				}
				if alpha == 0.0 {
					continue
				}
				pix := changeBrightnessRGBA(base, int((1.0-rMin/rWidth)*255.0*p.Specular.Val/100.0))
				pix.A = uint8(clampInt(int(alpha), 0, 255))
				dst.Set(x, y, pix)
			}
		}
		return
	}

	rLen := rCY / rAY * p.Length.Val / 100.0
	rXArea := rCX / rAX
	rWidth := rXArea*p.Width.Val/200.0 + 1.0
	rWidth2 := (rWidth - 1.0) * (1.0 - p.Diffuse.Val/100.0)
	if dst.Width == 0 || dst.Height == 0 {
		return
	}
	for y := 0; y < dst.Height; y++ {
		rY := -((float64(y) + 0.5) - rCY)
		rYA := math.Abs(rY)
		for x := 0; x < dst.Width; x++ {
			rX := float64(x) + 0.5 - rCX
			rXA := math.Abs(rX)
			rMin := 1.0
			alpha := 0.0
			if rXA < rXArea {
				rMin = 100000.0
				xx := 0.0
				for xx <= 100.0 {
					r := 0.0
					if rYA < rLen {
						r = xx * (rXArea - rWidth) / 100.0
						r = math.Abs(r - rXA)
					} else {
						xxx := xx * (rXArea - rWidth) / 100.0
						r = math.Hypot(rYA-rLen, rXA-xxx)
					}
					if r < rMin {
						rMin = r
					}
					if p.Step.Val == 0.0 {
						break
					}
					xx += p.Step.Val
				}
				if rMin < rWidth {
					if rMin < rWidth2 {
						alpha = 255.0
					} else {
						alpha = 255.0 - (rMin-rWidth2)/(rWidth-rWidth2)*255.0
					}
				}
			}
			if alpha == 0.0 {
				continue
			}
			pix := changeBrightnessRGBA(base, int((1.0-rMin/rWidth)*255.0*p.Specular.Val/100.0))
			pix.A = uint8(clampInt(int(alpha), 0, 255))
			dst.Set(x, y, pix)
		}
	}
}

func canRenderParallelLinesAgg(p *model.Primitive) bool {
	return false
}

func renderParallelLinesAgg(dst *PixBuf, p *model.Primitive, horizontal bool) bool {
	if dst.Width == 0 || dst.Height == 0 {
		return false
	}

	ctx := AggContextForPixBuf(dst)
	if ctx == nil {
		return false
	}

	base := primitiveColor(p)
	rCX := float64(dst.Width) * 0.5
	rCY := float64(dst.Height) * 0.5
	rAY := 1.0
	rAX := 1.0
	if p.Aspect.Val > 0.0 {
		rAX = 100.0 / (100.0 - math.Min(p.Aspect.Val, 99.0))
	}
	if p.Aspect.Val < 0.0 {
		rAY = 100.0 / (100.0 + math.Max(p.Aspect.Val, -99.0))
	}

	ctx.SetColor(agg.Color{R: base.R, G: base.G, B: base.B, A: base.A})
	ctx.SetLineWidth(1.0)
	ctx.GetAgg2D().LineCap(agg.CapRound)

	if horizontal {
		rLen := rCX / rAX * p.Length.Val / 100.0
		rYArea := rCY / rAY
		rWidth := rYArea*p.Width.Val/200.0 + 1.0
		if rLen <= 0 || rWidth <= 0 {
			return false
		}
		ctx.SetLineWidth(rWidth * 2.0)
		for _, off := range parallelLineOffsets(rYArea-rWidth, p.Step.Val) {
			y0 := rCY - off
			ctx.DrawLine(rCX-rLen, y0, rCX+rLen, y0)
			if off != 0 {
				y1 := rCY + off
				ctx.DrawLine(rCX-rLen, y1, rCX+rLen, y1)
			}
		}
		return true
	}

	rLen := rCY / rAY * p.Length.Val / 100.0
	rXArea := rCX / rAX
	rWidth := rXArea*p.Width.Val/200.0 + 1.0
	if rLen <= 0 || rWidth <= 0 {
		return false
	}
	ctx.SetLineWidth(rWidth * 2.0)
	for _, off := range parallelLineOffsets(rXArea-rWidth, p.Step.Val) {
		x0 := rCX - off
		ctx.DrawLine(x0, rCY-rLen, x0, rCY+rLen)
		if off != 0 {
			x1 := rCX + off
			ctx.DrawLine(x1, rCY-rLen, x1, rCY+rLen)
		}
	}
	return true
}

func parallelLineOffsets(maxOffset, step float64) []float64 {
	if maxOffset < 0 {
		maxOffset = 0
	}

	offsets := make([]float64, 0, 8)
	for pos := 0.0; pos <= 100.0; {
		offsets = append(offsets, pos*maxOffset/100.0)
		if step == 0.0 {
			break
		}
		pos += step
	}
	return offsets
}
