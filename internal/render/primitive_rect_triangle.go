package render

import (
	"math"
	"strings"

	agg "github.com/cwbudde/agg_go"
	"knobman/internal/model"
)

func renderRect(dst *PixBuf, p *model.Primitive, outline bool, textures []*Texture) {
	if outline {
		renderRectOutline(dst, p)
		return
	}

	renderRectFill(dst, p, textures)
}

func renderRectOutline(dst *PixBuf, p *model.Primitive) {
	if canRenderRectOutlineAgg(p) && renderRectOutlineAgg(dst, p) {
		return
	}

	base := primitiveColor(p)
	rCX := float64(dst.Width) * 0.5
	rCY := float64(dst.Height) * 0.5
	rXRO := rCX + 0.5
	rYRO := rCY + 0.5

	if p.Aspect.Val > 0.0 {
		rXRO = rXRO * (100.0 - math.Min(p.Aspect.Val, 99.0)) / 100.0
	}

	if p.Aspect.Val < 0.0 {
		rYRO = rYRO * (100.0 + math.Max(p.Aspect.Val, -99.0)) / 100.0
	}

	rWidth := math.Min(rCX, rCY)*p.Width.Val/100.0 + 1.0
	rWidth2 := rWidth * 0.5
	rD := rWidth2 * p.Diffuse.Val / 100.0
	rXRC := rXRO - rWidth2
	rYRC := rYRO - rWidth2
	rXRI := rXRO - rWidth
	rYRI := rYRO - rWidth
	rRoundO := p.Round.Val * math.Min(rCX, rCY) / 100.0
	rRoundI := rRoundO - rWidth
	rXCC := rXRO - rRoundO
	rYCC := rYRO - rRoundO

	for y := range dst.Height {
		rY := -((float64(y) + 0.5) - rCY)
		rYA := math.Abs(rY)

		for x := range dst.Width {
			rX := -((float64(x) + 0.5) - rCX)
			rXA := math.Abs(rX)
			rAlpha := 1.0
			rSpec := rWidth

			if rXA > rXRO {
				rAlpha = 0.0
			} else if rXA > rXRO-1.0-rD {
				rAlpha = (rXRO - rXA) / (1.0 + rD)
			}

			if rYA > rYRO {
				rAlpha = 0.0
			} else if rYA > rYRO-1.0-rD {
				rAlpha *= (rYRO - rYA) / (1.0 + rD)
			}

			if rXA >= rXCC && rYA >= rYCC {
				r := math.Hypot(rXA-rXCC, rYA-rYCC)
				if r > rRoundO {
					rAlpha = 0.0
				}

				if r > rRoundO-1.0-rD {
					rAlpha *= (rRoundO - r) / (1.0 + rD)
				}

				if r < rRoundI {
					rAlpha = 0.0
				}

				if r < rRoundI+1.0 {
					rAlpha *= r - rRoundI
				}
			} else if rXA < rXRI && rYA < rYRI {
				rAlpha = 0.0
			} else if rXA < rXRI+1.0+rD && rYA < rYRI+1.0+rD {
				rAlpha *= math.Max(rYA-rYRI, rXA-rXRI) / (1.0 + rD)
			}

			if rYA < math.Min(rYRI, rYCC) {
				rSpec = math.Max(0.0, math.Abs(rXA-rXRC))
			} else if rXA < math.Min(rXRI, rXCC) {
				rSpec = math.Max(0.0, math.Abs(rYA-rYRC))
			} else if rXA >= rXCC && rYA >= rYCC {
				r := math.Hypot(rXA-rXCC, rYA-rYCC)
				rSpec = math.Abs(r + (rWidth2 - rRoundO))
			} else if rYA-rYRI > rXA-rXRI {
				rSpec = math.Abs(rYA - rYRC)
			} else {
				rSpec = math.Abs(rXA - rXRC)
			}

			alpha := clampInt(int(rAlpha*255.0), 0, 255)
			if alpha == 0 {
				continue
			}

			iSpec := int((1.0 - rSpec/rWidth2) * 255.0 * p.Specular.Val / 100.0)
			pix := changeBrightnessRGBA(base, iSpec)
			pix.A = uint8(alpha)
			dst.Set(x, y, pix)
		}
	}
}

func canRenderRectOutlineAgg(p *model.Primitive) bool {
	if p == nil {
		return false
	}

	if p.Diffuse.Val != 0 || p.Specular.Val != 0 || p.Round.Val != 0 {
		return false
	}

	return true
}

func renderRectOutlineAgg(dst *PixBuf, p *model.Primitive) bool {
	ctx := AggContextForPixBuf(dst)
	if ctx == nil {
		return false
	}

	base := primitiveColor(p)
	rCX := float64(dst.Width) * 0.5
	rCY := float64(dst.Height) * 0.5
	rXRO := rCX + 0.5
	rYRO := rCY + 0.5

	if p.Aspect.Val > 0.0 {
		rXRO = rXRO * (100.0 - math.Min(p.Aspect.Val, 99.0)) / 100.0
	}

	if p.Aspect.Val < 0.0 {
		rYRO = rYRO * (100.0 + math.Max(p.Aspect.Val, -99.0)) / 100.0
	}

	strokeWidth := math.Min(rCX, rCY)*p.Width.Val/100.0 + 1.0
	if strokeWidth <= 0 {
		return false
	}

	x0 := rCX - rXRO + strokeWidth*0.5
	y0 := rCY - rYRO + strokeWidth*0.5
	w := rXRO*2.0 - strokeWidth

	h := rYRO*2.0 - strokeWidth
	if w <= 0 || h <= 0 {
		return false
	}

	ctx.SetColor(agg.Color{R: base.R, G: base.G, B: base.B, A: base.A})
	ctx.SetLineWidth(strokeWidth)
	ctx.DrawRectangle(x0, y0, w, h)

	return true
}

func renderRectFill(dst *PixBuf, p *model.Primitive, textures []*Texture) {
	if canRenderRectFillAgg(p, textures) && renderRectFillAgg(dst, p) {
		return
	}

	base := primitiveColor(p)
	rLY := math.Sqrt(1.0 / 3.0)
	rLX := rLY
	rLZ := rLY
	root2 := math.Sqrt(2.0)
	rMin := math.Min(float64(dst.Width), float64(dst.Height))
	rRound := p.Round.Val * rMin / 200.0
	iEm := int(math.Abs(p.Emboss.Val))
	rEmbossEdge := (100.0 - p.EmbossDiffuse.Val) / 100.0
	rCX := float64(dst.Width) * 0.5
	rCY := float64(dst.Height) * 0.5
	rWidth := rCX
	rHeight := rCY

	if p.Aspect.Val > 0.0 {
		rWidth = rWidth * (100.0 - math.Min(p.Aspect.Val, 99.0)) / 100.0
	}

	if p.Aspect.Val < 0.0 {
		rHeight = rHeight * (100.0 + math.Max(p.Aspect.Val, -99.0)) / 100.0
	}

	rXD := rWidth * (100.0 - p.Diffuse.Val) / 100.0
	rYD := rHeight * (100.0 - p.Diffuse.Val) / 100.0
	rXE := rWidth + 1.0 - rMin*float64(iEm)/200.0*rEmbossEdge
	rXEM := rWidth - rMin*float64(iEm)/200.0
	rYE := rHeight + 1.0 - rMin*float64(iEm)/200.0*rEmbossEdge
	rYEM := rHeight - rMin*float64(iEm)/200.0

	for y := range dst.Height {
		rY := -((float64(y) + 0.5) - rCY)
		rYN := rY / rCY
		rYA := math.Abs(rY)
		rPY := float64(y) - rCY + 0.5

		for x := range dst.Width {
			rPX := float64(x) - rCX + 0.5
			alpha := 255
			lumi := 0
			lumi, alpha = sampleTextureLumiAlpha(textures, p, rPX, rPY)

			rX := -((float64(x) + 0.5) - rCX)
			rXN := rX / rCX

			rXA := math.Abs(rX)
			if rYA > rHeight || rXA > rWidth {
				alpha = 0
			}

			if rXA > rXD && rWidth != rXD {
				alpha = int(math.Max(0.0, float64(alpha)*(1.0-(rXA-rXD)/(rWidth-rXD))))
			}

			if rYA > rYD && rHeight != rYD {
				alpha = int(math.Max(0.0, float64(alpha)*(1.0-(rYA-rYD)/(rHeight-rYD))))
			}

			if rXA > rWidth-rRound && rYA > rHeight-rRound {
				rXR := rXA - rWidth + rRound
				rYR := rYA - rHeight + rRound
				rXRM := rXR + 1.0
				rYRM := rYR + 1.0

				rR2 := rRound * rRound
				if rXRM*rXRM+rYRM*rYRM >= rR2 {
					if rXR*rXR+rYR*rYR >= rR2 {
						alpha = 0
					} else {
						b := (rXR + rYR) * 2.0
						c := rXR*rXR + rYR*rYR - rR2
						v := (math.Sqrt(b*b-8.0*c) - b) * 0.25
						alpha = int(float64(alpha) * v)
					}
				}
			}

			rRound2 := 0.0
			if min(rWidth, rHeight) != 0 {
				rRound2 = rRound * math.Min(rXE, rYE) / math.Min(rWidth, rHeight)
			}

			rXR := 0.0
			rYR := 0.0
			iEmbossMode := 0
			rXYR := 0.0

			if rXA >= rXEM {
				rXR = rX / (rXA * root2)
				rYR = 0.0
				iEmbossMode = 1
			}

			if rYA >= rYEM {
				rXR = 0.0
				rYR = rY / (rYA * root2)
				iEmbossMode = 2
			}

			if rXA >= rXEM-rRound2 && rYA >= rYEM-rRound2 {
				if rX > 0.0 {
					rXR = rXA - rXEM + rRound2
				} else {
					rXR = -(rXA - rXEM + rRound2)
				}

				if rY > 0.0 {
					rYR = rYA - rYEM + rRound2
				} else {
					rYR = -(rYA - rYEM + rRound2)
				}

				rXYR = rXR*rXR + rYR*rYR
				if r2 := rRound2 * rRound2; rXYR >= r2 {
					r := math.Sqrt(2.0 * rXYR)
					rXR /= r
					rYR /= r
					iEmbossMode = 3
				}
			}

			pix := changeBrightnessRGBA(base, int((rXN+rYN)*128.0*p.Specular.Val/100.0)+lumi)

			if iEmbossMode != 0 {
				rTZ := 1.0 / root2

				r := 0.0
				if p.Emboss.Val > 0.0 {
					r = rXR*rLX + rYR*rLY + rTZ*rLZ
				} else {
					r = -rXR*rLX - rYR*rLY + rTZ*rLZ
				}

				if r < 0.0 {
					r = 0.0
				}

				edgePix := changeBrightnessRGBA(base, int(r*255.0-128.0))

				a := 0.0

				switch iEmbossMode {
				case 1:
					a = 255.0 * (rXA - rXEM) / (rXE - rXEM)
				case 2:
					a = 255.0 * (rYA - rYEM) / (rYE - rYEM)
				case 3:
					a = 255.0 * (math.Sqrt(rXYR) - rRound2) / (rXE - rXEM)
				}

				a = math.Min(255.0, math.Max(0.0, math.Abs(a)))
				pix = blendToRGBA(pix, edgePix, int(a))
			}

			if alpha <= 0 {
				continue
			}

			pix.A = uint8(clampInt(alpha, 0, 255))
			dst.Set(x, y, pix)
		}
	}
}

func canRenderRectFillAgg(p *model.Primitive, textures []*Texture) bool {
	if p == nil {
		return false
	}

	if p.TextureDepth.Val != 0 || p.TextureFile.Val != 0 || strings.TrimSpace(p.TextureName) != "" || len(p.EmbeddedTexture) != 0 {
		return false
	}

	if p.Diffuse.Val != 0 || p.Specular.Val != 0 || p.Emboss.Val != 0 || p.Round.Val != 0 {
		return false
	}

	return true
}

func renderRectFillAgg(dst *PixBuf, p *model.Primitive) bool {
	ctx := AggContextForPixBuf(dst)
	if ctx == nil {
		return false
	}

	base := primitiveColor(p)
	rCX := float64(dst.Width) * 0.5
	rCY := float64(dst.Height) * 0.5
	rWidth := rCX
	rHeight := rCY

	if p.Aspect.Val > 0.0 {
		rWidth = rWidth * (100.0 - math.Min(p.Aspect.Val, 99.0)) / 100.0
	}

	if p.Aspect.Val < 0.0 {
		rHeight = rHeight * (100.0 + math.Max(p.Aspect.Val, -99.0)) / 100.0
	}

	x0 := rCX - rWidth
	y0 := rCY - rHeight
	w := rWidth * 2.0

	h := rHeight * 2.0
	if w <= 0 || h <= 0 {
		return false
	}

	ctx.SetColor(agg.Color{R: base.R, G: base.G, B: base.B, A: base.A})
	ctx.FillRectangle(x0, y0, w, h)

	return true
}

func renderTriangle(dst *PixBuf, p *model.Primitive, textures []*Texture) {
	if canRenderTriangleAgg(p, textures) && renderTriangleAgg(dst, p) {
		return
	}

	base := primitiveColor(p)
	rCX := float64(dst.Width) * 0.5
	rCY := float64(dst.Height) * 0.5
	rYLen := float64(dst.Height) * p.Length.Val * 0.01
	rWidth := float64(dst.Width) * p.Width.Val * 0.005
	rD := 1.0 - p.Diffuse.Val*0.01

	for y := range dst.Height {
		rPY := float64(y) - rCY + 0.5

		for x := range dst.Width {
			rPX := float64(x) - rCX + 0.5
			alpha := 255
			lumi := 0
			lumi, alpha = sampleTextureLumiAlpha(textures, p, rPX, rPY)

			pix := base

			if float64(y) > rYLen {
				alpha = 0
			} else {
				rX := math.Abs(float64(x) + 0.5 - rCX)
				rXLine := rWidth*float64(y)/rYLen + 1.0

				rXLine2 := (rXLine - 1.0) * rD
				if !(rX < rXLine2) {
					if rX < rXLine {
						alpha = int((1.0 - (rX-rXLine2)/(rXLine-rXLine2)) * float64(alpha))
					} else {
						alpha = 0
					}
				}

				iA := 255
				if rXLine != 0.0 {
					iA = int((255.0 - rX/rXLine*255.0) * p.Specular.Val * 0.01)
				}

				iA += lumi
				iA = clampInt(iA, 0, 254)
				pix = changeBrightnessRGBA(pix, iA)
			}

			if alpha <= 0 {
				continue
			}

			pix.A = uint8(clampInt(alpha, 0, 255))
			dst.Set(x, y, pix)
		}
	}
}

func canRenderTriangleAgg(p *model.Primitive, textures []*Texture) bool {
	if p == nil {
		return false
	}

	if p.TextureDepth.Val != 0 || p.TextureFile.Val != 0 || strings.TrimSpace(p.TextureName) != "" || len(p.EmbeddedTexture) != 0 {
		return false
	}

	if p.Diffuse.Val != 0 || p.Specular.Val != 0 {
		return false
	}

	return true
}

func renderTriangleAgg(dst *PixBuf, p *model.Primitive) bool {
	ctx := AggContextForPixBuf(dst)
	if ctx == nil {
		return false
	}

	base := primitiveColor(p)
	rCX := float64(dst.Width) * 0.5
	rYLen := float64(dst.Height) * p.Length.Val * 0.01

	rWidth := float64(dst.Width) * p.Width.Val * 0.005
	if rYLen <= 0 || rWidth <= 0 {
		return false
	}

	ctx.BeginPath()
	ctx.MoveTo(rCX, 0.0)
	ctx.LineTo(rCX-rWidth, rYLen)
	ctx.LineTo(rCX+rWidth, rYLen)
	ctx.ClosePath()
	ctx.SetColor(agg.Color{R: base.R, G: base.G, B: base.B, A: base.A})
	ctx.Fill()

	return true
}
