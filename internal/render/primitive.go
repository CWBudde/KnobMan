package render

import (
	"bytes"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"strconv"
	"strings"

	agg "github.com/cwbudde/agg_go"
	"knobman/internal/model"
)

// RenderPrimitive draws a primitive into dst.
// This is the Phase 2 baseline implementation; complex Java parity details
// (advanced lighting/masks/anti-aliasing) are layered in later phases.
func RenderPrimitive(dst *PixBuf, p *model.Primitive, textures []*Texture, frame, totalFrames int) {
	if dst == nil || p == nil || dst.Width == 0 || dst.Height == 0 {
		return
	}

	switch model.PrimitiveType(p.Type.Val) {
	case model.PrimNone:
		return
	case model.PrimImage:
		renderImage(dst, p, frame, totalFrames)
	case model.PrimCircle:
		renderCircle(dst, p, true, textures)
	case model.PrimCircleFill:
		renderCircle(dst, p, false, textures)
	case model.PrimMetalCircle:
		renderCircle(dst, p, false, textures)
	case model.PrimWaveCircle:
		renderWaveCircle(dst, p, textures)
	case model.PrimSphere:
		renderCircle(dst, p, false, textures)
	case model.PrimRect:
		renderRect(dst, p, true, textures)
	case model.PrimRectFill:
		renderRect(dst, p, false, textures)
	case model.PrimTriangle:
		renderTriangle(dst, p, textures)
	case model.PrimLine:
		renderLine(dst, p)
	case model.PrimRadiateLine:
		renderRadiateLines(dst, p)
	case model.PrimHLines:
		renderParallelLines(dst, p, true)
	case model.PrimVLines:
		renderParallelLines(dst, p, false)
	case model.PrimText:
		renderText(dst, p, frame, totalFrames)
	case model.PrimShape:
		renderShape(dst, p, textures)
	}
}

func primitiveColor(p *model.Primitive) color.RGBA {
	c := p.Color.Val
	if c.A == 0 {
		c.A = 255
	}

	return c
}

func renderImage(dst *PixBuf, p *model.Primitive, frame, totalFrames int) {
	if len(p.EmbeddedImage) == 0 {
		return
	}

	img, _, err := image.Decode(bytes.NewReader(p.EmbeddedImage))
	if err != nil {
		return
	}

	src := pixBufFromImage(img)
	if src == nil || src.Width <= 0 || src.Height <= 0 {
		return
	}

	nf := p.NumFrame.Val
	if nf < 1 {
		nf = 1
	}

	frameBuf := ExtractFrameAligned(src, nf, frame, totalFrames, p.FrameAlign.Val)
	if frameBuf == nil || frameBuf.Width <= 0 || frameBuf.Height <= 0 {
		return
	}

	def := frameBuf.At(0, 0)
	if p.AutoFit.Val != 0 {
		drawPixBufToRect(dst, frameBuf, 0, 0, dst.Width, dst.Height, p, def)
		return
	}
	// Java behavior: when AutoFit is off, the source frame is copied at native size
	// into the top-left corner (clipped by canvas bounds).
	drawPixBufToRect(dst, frameBuf, 0, 0, frameBuf.Width, frameBuf.Height, p, def)
}

func pixBufFromImage(img image.Image) *PixBuf {
	if img == nil {
		return nil
	}

	b := img.Bounds()

	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return nil
	}

	pb := NewPixBuf(w, h)
	for y := range h {
		for x := range w {
			pb.Set(x, y, rgbaFromImage(img, b.Min.X+x, b.Min.Y+y))
		}
	}

	return pb
}

func drawPixBufToRect(dst, src *PixBuf, dx0, dy0, dw, dh int, p *model.Primitive, def color.RGBA) {
	if src == nil || src.Width <= 0 || src.Height <= 0 || dw <= 0 || dh <= 0 {
		return
	}

	maxX := min(dst.Width, dx0+dw)

	maxY := min(dst.Height, dy0+dh)
	for y := max(0, dy0); y < maxY; y++ {
		sy := (y - dy0) * src.Height / dh
		for x := max(0, dx0); x < maxX; x++ {
			sx := (x - dx0) * src.Width / dw
			px := src.At(sx, sy)

			px = applyImageTransparency(px, def, p.Transparent.Val, p.IntelliAlpha.Val)
			if px.A == 0 {
				continue
			}

			dst.BlendOver(x, y, px)
		}
	}
}

func rgbaFromImage(img image.Image, x, y int) color.RGBA {
	r, g, b, a := img.At(x, y).RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

func applyImageTransparency(px, def color.RGBA, transparentMode, intelliAlpha int) color.RGBA {
	switch transparentMode {
	case 0: // file alpha
		return px
	case 1: // force opaque
		px.A = 255
		return px
	default: // first-pixel key
		if intelliAlpha > 0 {
			return intelliAlphaPix(def, px, intelliAlpha)
		}

		if px.R == def.R && px.G == def.G && px.B == def.B {
			return color.RGBA{}
		}

		px.A = 255

		return px
	}
}

func intelliAlphaPix(def, target color.RGBA, intelliAlpha int) color.RGBA {
	if def.R == target.R && def.G == target.G && def.B == target.B {
		return color.RGBA{}
	}

	alphaStep := 255 - intelliAlpha*254/100
	if alphaStep < 16 {
		alphaStep = 16
	}

	piAlpha := 0
	var r, g, b int

	for {
		piAlpha += alphaStep
		a := 1.0

		if piAlpha >= 255 {
			piAlpha = 255
		} else {
			a = float64(piAlpha) / 255.0
		}

		r = int((float64(target.R) - float64(def.R)*(1.0-a)) / a)
		g = int((float64(target.G) - float64(def.G)*(1.0-a)) / a)

		b = int((float64(target.B) - float64(def.B)*(1.0-a)) / a)
		if a >= 1.0 || (r >= 0 && r <= 255 && g >= 0 && g <= 255 && b >= 0 && b <= 255) {
			break
		}
	}

	return color.RGBA{R: uint8(clampInt(r, 0, 255)), G: uint8(clampInt(g, 0, 255)), B: uint8(clampInt(b, 0, 255)), A: uint8(piAlpha)}
}

func renderCircle(dst *PixBuf, p *model.Primitive, outline bool, textures []*Texture) {
	if outline {
		renderCircleOutline(dst, p, textures)
		return
	}

	renderCircleFill(dst, p, textures)
}

func renderCircleOutline(dst *PixBuf, p *model.Primitive, textures []*Texture) {
	base := primitiveColor(p)
	rCX := float64(dst.Width) * 0.5
	rCY := float64(dst.Height) * 0.5
	rMinSize := math.Min(rCX, rCY) * p.Width.Val * 0.01

	rAY := 1.0
	rAX := 1.0
	if p.Aspect.Val > 0 {
		rAX = 100.0 / (100.0 - math.Min(p.Aspect.Val, 99.0))
	}
	if p.Aspect.Val < 0 {
		rAY = 100.0 / (100.0 + math.Max(p.Aspect.Val, -99.0))
	}

	for y := 0; y < dst.Height; y++ {
		rPY := -((float64(y) + 0.5) - rCY) * rAY
		rY := rPY / rCY
		rY2 := rY * rY
		rYM := rPY / (rCY - rAY)
		rYM2 := rYM * rYM
		rYC := rPY / math.Max(0.1, rCY-rMinSize*0.5*rAY)
		rYC2 := rYC * rYC
		rYEM := rPY / math.Max(0.1, rCY-rMinSize*rAY+rAY)
		rYEM2 := rYEM * rYEM
		rYD1 := rPY / math.Max(0.1, rCY-rMinSize*0.5*p.Diffuse.Val/100.0*rAY)
		rYD12 := rYD1 * rYD1
		rYD2 := rPY / math.Max(0.1, rCY-rMinSize*rAY+rMinSize*0.5*p.Diffuse.Val/100.0*rAY)
		rYD22 := rYD2 * rYD2
		rYE := rPY / math.Max(0.1, rCY-rMinSize*rAY)
		rYE2 := rYE * rYE

		for x := 0; x < dst.Width; x++ {
			rPX := -((float64(x) + 0.5) - rCX) * rAX
			rX := rPX / rCX
			rXM := rPX / (rCX - rAX)
			rXC := rPX / math.Max(0.1, rCX-rMinSize*0.5*rAX)
			rXEM := rPX / math.Max(0.1, rCX-rMinSize*rAX+rAX)
			rXE := rPX / math.Max(0.1, rCX-rMinSize*rAX)
			rXD1 := rPX / math.Max(0.1, rCX-rMinSize*0.5*p.Diffuse.Val*0.01*rAX)
			rXD2 := rPX / math.Max(0.1, rCX-rMinSize*rAX+rMinSize*0.5*p.Diffuse.Val*0.01*rAX)

			r := rX*rX + rY2
			rM := rXM*rXM + rYM2
			rE := rXE*rXE + rYE2
			rD1 := rXD1*rXD1 + rYD12
			rD2 := rXD2*rXD2 + rYD22
			rC := rXC*rXC + rYC2
			rEM := rXEM*rXEM + rYEM2
			_, _ = rC, rEM

			alpha := 255.0
			if rE < 1.0 || r > 1.0 {
				alpha = 0.0
			} else {
				if rE >= 1.0 && rEM <= 1.0 {
					alpha *= (1.0 - rE) / (rEM - rE)
				}
				if r < 1.0 && rM >= 1.0 {
					alpha *= (1.0 - r) / (rM - r)
				}
				if rD1 >= 1.0 && r < 1.0 {
					alpha *= (1.0 - r) / (rD1 - r)
				} else if rE >= 1.0 && rD2 < 1.0 {
					v := (rE - 1.0) / (rE - rD2)
					alpha *= v * v
				}
			}
			if alpha <= 0.0 {
				continue
			}

			pix := base
			if p.Specular.Val != 0.0 && r > 0 && rE > 0 && alpha > 0.0 {
				v1 := 1.0 / math.Sqrt(r)
				v2 := 1.0 / math.Sqrt(rE)
				v := 2.0 * (1.0 - v2) / (v1 - v2)
				if v > 1.0 {
					v = 2.0 - v
				}
				pix = changeBrightnessRGBA(pix, int(v*p.Specular.Val*2.55))
			}
			pix = applyTextureOverlay(pix, textures, p, x, y)
			pix.A = uint8(clampInt(int(alpha), 0, 255))
			dst.Set(x, y, pix)
		}
	}
}

func renderCircleFill(dst *PixBuf, p *model.Primitive, textures []*Texture) {
	base := primitiveColor(p)
	cx := float64(dst.Width) * 0.5
	cy := float64(dst.Height) * 0.5
	ax := cx
	ay := cy
	if p.Aspect.Val > 0.0 {
		ax = (100.0 - math.Min(p.Aspect.Val, 99.0)) / 100.0 * cx
	}
	if p.Aspect.Val < 0.0 {
		ay = (100.0 + math.Max(p.Aspect.Val, -99.0)) / 100.0 * cy
	}

	rThick := math.Abs(p.Emboss.Val) / 100.0
	if rThick == 1.0 {
		rThick = 0.99
	}
	rEmbossEdge := (100.0 - p.EmbossDiffuse.Val) / 100.0
	rD := 1.0 - p.Diffuse.Val/100.0
	rD2 := rD * rD
	rMin := math.Min(ax, ay)

	rLY := math.Sqrt(1.0 / 3.0)
	rLX := rLY
	rLZ := rLY
	rTZ := 1.0 / math.Sqrt(2.0)

	iMetalAmbient := int(p.Ambient.Val * 255.0 / 100.0)
	iMetalSpecular := int(p.Specular.Val * 255.0 / 100.0)
	dMetalSpecularWidth := 0.0
	if p.SpecularWidth.Val == 0.0 {
		iMetalSpecular = 0
	} else {
		dMetalSpecularWidth = math.Pow(1.0/(p.SpecularWidth.Val*0.01), 3.0)
	}

	rSX := -p.LightDir.Val
	rSY := -p.LightDir.Val
	rSZ := 50.0
	a := math.Sqrt(rSX*rSX + rSY*rSY + rSZ*rSZ)
	rSX /= a
	rSY /= a
	rSZ /= a

	for y := 0; y < dst.Height; y++ {
		py := float64(y) - cy + 0.5
		ry := py / (ay + 0.5)
		rym := py / (ay - 0.5)
		rye := py / (ay + 0.5 - rMin*rThick*rEmbossEdge)
		ryem := py / (ay - rMin*rThick)

		for x := 0; x < dst.Width; x++ {
			px := float64(x) - cx + 0.5
			rx := px / (ax + 0.5)
			rxy := rx*rx + ry*ry
			rxm := px / (ax - 0.5)
			rxym := rxm*rxm + rym*rym
			rxe := px / (ax + 0.5 - rMin*rThick*rEmbossEdge)
			rxem := px / (ax - rMin*rThick)
			rxye := rxe*rxe + rye*rye
			rxyem := rxem*rxem + ryem*ryem

			pix := base
			alpha := 255
			lumi := 0
			lumi, alpha = sampleTextureLumiAlpha(textures, p, px, py)

			if rxye < 1.0 {
				switch model.PrimitiveType(p.Type.Val) {
				case model.PrimCircleFill:
					pix = changeBrightnessRGBA(pix, int((-rx-ry)*128.0*p.Specular.Val/100.0)+lumi)
				case model.PrimMetalCircle:
					d := math.Sin(math.Atan2(ry, rx) * 2.0)
					d2 := math.Pow((d+1.0)*0.5, dMetalSpecularWidth)
					a := (d+1.0)*0.5*float64(255-iMetalAmbient) + float64(iMetalAmbient) + d2*float64(iMetalSpecular) + float64(lumi)
					pix = changeBrightnessRGBA(pix, int(a-256.0))
				case model.PrimSphere:
					rZ := math.Sqrt(math.Max(0.0, 1.0-rxy))
					d := -rx*rSX - ry*rSY + rZ*rSZ
					a := float64(iMetalAmbient) + float64(255-iMetalAmbient)*d
					d = 2.0*rZ*d - rSZ
					if d <= 0.0 {
						d = 0.0
					} else {
						d = math.Exp(math.Log(d) * (110.0 - p.SpecularWidth.Val) / 10.0)
					}
					pix = brightRGBA(pix, int(a)+lumi)
					pix = changeBrightnessRGBA(pix, int(float64(iMetalSpecular)*d))
				}
			}

			if rxyem >= 1.0 {
				rR2 := math.Sqrt(math.Max(1e-12, 2.0*rxy))
				rTX := -rx / rR2
				rTY := -ry / rR2
				edgePix := base
				if p.Emboss.Val > 0.0 {
					r := rTX*rLX + rTY*rLY + rTZ*rLZ
					edgePix = changeBrightnessRGBA(edgePix, int(r*255.0-128.0))
				} else if p.Emboss.Val < 0.0 {
					r := -rTX*rLX - rTY*rLY + rTZ*rLZ
					edgePix = changeBrightnessRGBA(edgePix, int(r*255.0-128.0))
				}
				a := math.Min(255.0, math.Max(0.0, 255.0*math.Abs((1.0-rxyem)/(rxye-rxyem))))
				if rxye < 1.0 {
					pix = blendToRGBA(pix, edgePix, int(a))
				} else {
					pix = edgePix
				}
			}

			if rxy > 1.0 {
				alpha = 0
			}
			if rD2 < 1.0 && rxy > rD2 {
				alpha = int(float64(alpha) * (1.0 - (rxy-rD2)/(1.0-rD2)))
			}
			if rxym >= 1.0 {
				alpha = int((1.0 - rxy) / (rxym - rxy) * float64(alpha))
			}
			if alpha <= 0 {
				continue
			}

			pix.A = uint8(clampInt(alpha, 0, 255))
			dst.Set(x, y, pix)
		}
	}
}

func renderMetalCircle(dst *PixBuf, p *model.Primitive, textures []*Texture) {
	base := primitiveColor(p)
	cx := float64(dst.Width-1) * 0.5
	cy := float64(dst.Height-1) * 0.5

	r := float64(min(dst.Width, dst.Height)) * 0.46
	for y := 0; y < dst.Height; y++ {
		for x := 0; x < dst.Width; x++ {
			dx := (float64(x) - cx) / r
			dy := (float64(y) - cy) / r

			d2 := dx*dx + dy*dy
			if d2 > 1 {
				continue
			}

			band := 0.5 + 0.5*math.Cos((dx+0.3)*math.Pi*2.0)
			shade := 0.35 + 0.65*band
			pix := color.RGBA{
				R: uint8(clamp01(float64(base.R)/255.0*shade) * 255),
				G: uint8(clamp01(float64(base.G)/255.0*shade) * 255),
				B: uint8(clamp01(float64(base.B)/255.0*shade) * 255),
				A: base.A,
			}
			pix = applyTextureOverlay(pix, textures, p, x, y)
			dst.BlendOver(x, y, pix)
		}
	}
}

func renderWaveCircle(dst *PixBuf, p *model.Primitive, textures []*Texture) {
	base := primitiveColor(p)
	lumi := 0.0
	rStep := 180.0 / p.AngleStep.Val
	rDepth := p.Width.Val * 0.01
	rCX := float64(dst.Width) * 0.5
	rCY := float64(dst.Height) * 0.5
	rAY := 1.0
	rAX := 1.0
	if p.Aspect.Val > 0.0 {
		rAX = (100.0 - math.Min(p.Aspect.Val, 99.0)) * 0.01
	}
	if p.Aspect.Val < 0.0 {
		rAY = (100.0 + math.Max(p.Aspect.Val, -99.0)) * 0.01
	}
	rMax := math.Sqrt(math.Min(rCX, rCY) * math.Min(rCX, rCY))
	rD := 1.0 - p.Diffuse.Val*0.01
	for y := 0; y < dst.Height; y++ {
		rPY := float64(y) - rCY + 0.5
		rY := rPY / rAY
		for x := 0; x < dst.Width; x++ {
			rPX := float64(x) - rCX + 0.5
			lumiInt, alpha := sampleTextureLumiAlpha(textures, p, rPX, rPY)
			lumi = float64(lumiInt)

			rX := rPX / rAX
			rTh := math.Abs(math.Atan2(rX, rY))
			rCos := math.Abs(math.Sin(rTh * rStep))
			rR := math.Hypot(rX, rY)
			rMax2 := rMax * (1.0 - rCos*rDepth)
			pix := changeBrightnessRGBA(base, int((-rX/rCX-rY/rCY)*128.0*p.Specular.Val*0.01+lumi))
			if rR < rMax2 {
				rMax2M := (rMax2 - 1.0) * rD
				if rR >= rMax2M {
					alpha = int(float64(alpha) - (rR-rMax2M)*255.0/(rMax2-rMax2M))
				}
			} else {
				alpha = 0
			}
			if alpha <= 0 {
				continue
			}
			pix.A = uint8(clampInt(alpha, 0, 255))
			dst.Set(x, y, pix)
		}
	}
}

func renderSphere(dst *PixBuf, p *model.Primitive, textures []*Texture) {
	base := primitiveColor(p)
	cx := float64(dst.Width-1) * 0.5
	cy := float64(dst.Height-1) * 0.5
	rx := float64(dst.Width) * 0.45

	ry := float64(dst.Height) * 0.45
	for y := 0; y < dst.Height; y++ {
		for x := 0; x < dst.Width; x++ {
			nx, ny, nz, ok := SphereNormal(float64(x), float64(y), cx, cy, rx, ry)
			if !ok {
				continue
			}

			pix := PhongLighting(
				[3]float64{nx, ny, nz},
				p.LightDir.Val,
				p.Ambient.Val,
				p.Diffuse.Val+60,
				p.Specular.Val,
				p.SpecularWidth.Val,
				base,
			)
			pix = applyTextureOverlay(pix, textures, p, x, y)
			dst.BlendOver(x, y, pix)
		}
	}
}

func renderRect(dst *PixBuf, p *model.Primitive, outline bool, textures []*Texture) {
	if outline {
		renderRectOutline(dst, p)
		return
	}

	renderRectFill(dst, p, textures)
}

func renderRectOutline(dst *PixBuf, p *model.Primitive) {
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

	for y := 0; y < dst.Height; y++ {
		rY := -((float64(y) + 0.5) - rCY)
		rYA := math.Abs(rY)

		for x := 0; x < dst.Width; x++ {
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

func renderRectFill(dst *PixBuf, p *model.Primitive, textures []*Texture) {
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
	for y := 0; y < dst.Height; y++ {
		rY := -((float64(y) + 0.5) - rCY)
		rYN := rY / rCY
		rYA := math.Abs(rY)
		rPY := float64(y) - rCY + 0.5

		for x := 0; x < dst.Width; x++ {
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

func renderTriangle(dst *PixBuf, p *model.Primitive, textures []*Texture) {
	base := primitiveColor(p)
	rCX := float64(dst.Width) * 0.5
	rCY := float64(dst.Height) * 0.5
	rYLen := float64(dst.Height) * p.Length.Val * 0.01
	rWidth := float64(dst.Width) * p.Width.Val * 0.005
	rD := 1.0 - p.Diffuse.Val*0.01
	for y := 0; y < dst.Height; y++ {
		rPY := float64(y) - rCY + 0.5
		for x := 0; x < dst.Width; x++ {
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

func renderLine(dst *PixBuf, p *model.Primitive) {
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

func renderRadiateLines(dst *PixBuf, p *model.Primitive) {
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

func renderParallelLines(dst *PixBuf, p *model.Primitive, horizontal bool) {
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

func renderText(dst *PixBuf, p *model.Primitive, frame, total int) {
	c := primitiveColor(p)

	txt := strings.TrimSpace(SubstituteFrameCounters(p.Text.Val, frame, total))
	if txt == "" {
		txt = "TEXT"
	}

	size := p.FontSize.Val * 0.01
	if size <= 0 {
		size = 0.5
	}

	charH := int(math.Max(6, float64(min(dst.Width, dst.Height))*size))
	charW := int(math.Max(4, float64(charH)*0.62))
	spacing := int(math.Max(1, float64(charW)/4.0))
	runes := []rune(txt)

	totalW := len(runes) * charW
	if len(runes) > 1 {
		totalW += (len(runes) - 1) * spacing
	}

	x := (dst.Width - totalW) / 2

	switch p.TextAlign.Val {
	case 1: // left
		x = spacing / 2
	case 2: // right
		x = dst.Width - totalW - spacing/2
	}

	y := (dst.Height - charH) / 2

	for _, r := range runes {
		if r != ' ' {
			dst.FillRect(x, y, x+charW, y+charH, c)
			dst.FillRect(x+1, y+1, x+charW-1, y+charH-1, color.RGBA{0, 0, 0, 0})
		}

		x += charW + spacing
	}
}

func renderShape(dst *PixBuf, p *model.Primitive, textures []*Texture) {
	base := primitiveColor(p)
	s := strings.TrimSpace(p.Shape.Val)
	if s == "" {
		return
	}

	mask := NewPixBuf(dst.Width, dst.Height)
	mask.Clear(color.RGBA{A: 255})
	maskImg := agg.NewImage(mask.Data, mask.Width, mask.Height, mask.Stride)
	maskCtx := agg.NewContextForImage(maskImg)
	if maskCtx == nil {
		return
	}

	maskCtx.Clear(agg.Color{A: 255})
	a := maskCtx.GetAgg2D()
	a.FillEvenOdd(true)
	maskCtx.BeginPath()
	if !appendShapePath(maskCtx, s, dst.Width, dst.Height, p.Fill.Val != 0) {
		return
	}

	blue := agg.Color{B: 255, A: 255}
	if p.Fill.Val != 0 {
		maskCtx.SetColor(blue)
		a.NoLine()
		maskCtx.Fill()
	} else {
		renderShapeOutlineMask(mask, s, dst.Width, dst.Height, p.Width.Val*float64(dst.Width)*0.004)
	}

	rCX := float64(dst.Width) * 0.5
	rCY := float64(dst.Height) * 0.5
	for y := 0; y < dst.Height; y++ {
		rY := -((float64(y) + 0.5) - rCY)
		rYN := rY / rCY
		rPY := float64(y) - rCY + 0.5
		for x := 0; x < dst.Width; x++ {
			i := y*mask.Stride + x*4
			coverage := int(mask.Data[i+2])
			if coverage == 0 {
				continue
			}

			lumi := 0
			alpha := 255
			rPX := float64(x) - rCX + 0.5
			rX := -((float64(x) + 0.5) - rCX)
			rXN := rX / rCX
			lumi, alpha = sampleTextureLumiAlpha(textures, p, rPX, rPY)

			col := base
			col = changeBrightnessRGBA(col, int((rXN+rYN)*128.0*p.Specular.Val*0.01)+lumi)
			col.A = uint8(clampInt(coverage*alpha/255, 0, 255))
			dst.Set(x, y, col)
		}
	}
}

func renderShapeOutlineMask(mask *PixBuf, s string, w, h int, strokeWidth float64) {
	if mask == nil || strokeWidth <= 0 {
		return
	}

	polys := parseKnobShapeAnchorPolylines(s, w, h)
	if len(polys) == 0 {
		pts := parseSimpleShapePoints(s, w, h)
		if len(pts) < 2 {
			return
		}
		poly := make([]fpoint, len(pts))
		for i := range pts {
			poly[i] = fpoint{x: float64(pts[i].x), y: float64(pts[i].y)}
		}
		polys = [][]fpoint{poly}
	}

	const samples = 16
	half := strokeWidth * 0.5
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			hits := 0
			for sy := 0; sy < samples; sy++ {
				py := float64(y) + (float64(sy)+0.5)/samples
				for sx := 0; sx < samples; sx++ {
					px := float64(x) + (float64(sx)+0.5)/samples
					if pointOnStrokePolys(px, py, polys, half) {
						hits++
					}
				}
			}
			if hits == 0 {
				continue
			}
			cov := uint8(clampInt(int(float64(hits)*255.0/float64(samples*samples)+0.5), 0, 255))
			mask.Set(x, y, color.RGBA{B: cov, A: 255})
		}
	}
}

func pointOnStrokePolys(px, py float64, polys [][]fpoint, halfWidth float64) bool {
	for _, poly := range polys {
		for i := 0; i+1 < len(poly); i++ {
			if pointInSquareCappedSegment(px, py, poly[i], poly[i+1], halfWidth) {
				return true
			}
		}
	}
	return false
}

func pointInSquareCappedSegment(px, py float64, a, b fpoint, halfWidth float64) bool {
	dx := b.x - a.x
	dy := b.y - a.y
	l := math.Hypot(dx, dy)
	if l == 0 {
		return math.Hypot(px-a.x, py-a.y) <= halfWidth
	}

	ux := dx / l
	uy := dy / l
	vx := px - a.x
	vy := py - a.y
	t := vx*ux + vy*uy
	n := -vx*uy + vy*ux
	return t >= -halfWidth && t <= l+halfWidth && math.Abs(n) <= halfWidth
}

func appendShapePath(ctx *agg.Context, s string, w, h int, closePath bool) bool {
	if polys := parseKnobShapeKnots(s); len(polys) > 0 {
		for _, knots := range polys {
			if len(knots) < 2 {
				continue
			}

			ctx.MoveTo(shapeScaleX(knots[0].pX, w), shapeScaleY(knots[0].pY, h))
			for i := 1; i < len(knots); i++ {
				prev := knots[i-1]
				cur := knots[i]
				ctx.GetAgg2D().CubicCurveTo(
					shapeScaleX(prev.outX, w), shapeScaleY(prev.outY, h),
					shapeScaleX(cur.inX, w), shapeScaleY(cur.inY, h),
					shapeScaleX(cur.pX, w), shapeScaleY(cur.pY, h),
				)
			}
			if closePath {
				last := knots[len(knots)-1]
				first := knots[0]
				ctx.GetAgg2D().CubicCurveTo(
					shapeScaleX(last.outX, w), shapeScaleY(last.outY, h),
					shapeScaleX(first.inX, w), shapeScaleY(first.inY, h),
					shapeScaleX(first.pX, w), shapeScaleY(first.pY, h),
				)
			}
		}
		return true
	}

	pts := parseSimpleShapePoints(s, w, h)
	if len(pts) < 2 {
		return false
	}

	ctx.MoveTo(float64(pts[0].x), float64(pts[0].y))
	for i := 1; i < len(pts); i++ {
		ctx.LineTo(float64(pts[i].x), float64(pts[i].y))
	}
	if closePath {
		ctx.ClosePath()
	}
	return true
}

type svgPathCmd struct {
	op   byte
	vals []float64
}

func parseSimpleShapePoints(s string, w, h int) []point {
	cmds := parseSVGPathCommands(s)
	if len(cmds) == 0 {
		return parseLooseShapePairs(s, w, h)
	}

	toPix := func(x, y float64) point {
		px := int(clamp01(x/100.0)*float64(w-1) + 0.5)
		py := int(clamp01(y/100.0)*float64(h-1) + 0.5)

		return point{px, py}
	}

	pts := make([]point, 0, 64)
	var curX, curY float64
	var startX, startY float64
	hasCur := false

	for _, c := range cmds {
		switch c.op {
		case 'M':
			curX, curY = c.vals[0], c.vals[1]
			startX, startY = curX, curY
			hasCur = true

			pts = append(pts, toPix(curX, curY))
		case 'L':
			if !hasCur {
				continue
			}

			curX, curY = c.vals[0], c.vals[1]
			pts = append(pts, toPix(curX, curY))
		case 'Q':
			if !hasCur {
				continue
			}

			poly := flattenQuadratic(
				fpoint{x: curX, y: curY},
				fpoint{x: c.vals[0], y: c.vals[1]},
				fpoint{x: c.vals[2], y: c.vals[3]},
				12,
			)
			for i := 1; i < len(poly); i++ {
				pts = append(pts, toPix(poly[i].x, poly[i].y))
			}

			curX, curY = c.vals[2], c.vals[3]
		case 'C':
			if !hasCur {
				continue
			}

			poly := flattenCubic(
				fpoint{x: curX, y: curY},
				fpoint{x: c.vals[0], y: c.vals[1]},
				fpoint{x: c.vals[2], y: c.vals[3]},
				fpoint{x: c.vals[4], y: c.vals[5]},
				12,
			)
			for i := 1; i < len(poly); i++ {
				pts = append(pts, toPix(poly[i].x, poly[i].y))
			}

			curX, curY = c.vals[4], c.vals[5]
		case 'Z':
			if !hasCur {
				continue
			}

			curX, curY = startX, startY
			pts = append(pts, toPix(curX, curY))
		}
	}

	return pts
}

func parseLooseShapePairs(s string, w, h int) []point {
	fields := strings.Fields(strings.NewReplacer(",", " ", "M", " ", "L", " ", "m", " ", "l", " ", "Z", " ", "z", " ").Replace(s))

	pts := make([]point, 0, len(fields)/2)
	for i := 0; i+1 < len(fields); i += 2 {
		x, okx := parseFloat(fields[i])

		y, oky := parseFloat(fields[i+1])
		if !okx || !oky {
			continue
		}

		px := int(clamp01(x/100.0)*float64(w-1) + 0.5)
		py := int(clamp01(y/100.0)*float64(h-1) + 0.5)
		pts = append(pts, point{px, py})
	}

	return pts
}

func parseSVGPathCommands(s string) []svgPathCmd {
	tokens := tokenizeSVGPath(s)
	out := make([]svgPathCmd, 0, 32)
	var cur byte

	for i := 0; i < len(tokens); {
		tk := tokens[i]
		if len(tk) == 1 && isAlphaASCII(tk[0]) {
			cur = byte(strings.ToUpper(tk)[0])
			i++

			if cur == 'Z' {
				out = append(out, svgPathCmd{op: 'Z'})
			}

			continue
		}

		if cur == 0 {
			i++
			continue
		}

		need := svgPathCmdArity(cur)
		if need <= 0 {
			i++
			continue
		}

		vals := make([]float64, 0, need)
		for i < len(tokens) && len(vals) < need {
			if len(tokens[i]) == 1 && isAlphaASCII(tokens[i][0]) {
				break
			}

			v, err := strconv.ParseFloat(tokens[i], 64)
			if err != nil {
				break
			}

			vals = append(vals, v)
			i++
		}

		if len(vals) != need {
			break
		}

		out = append(out, svgPathCmd{op: cur, vals: vals})
		if cur == 'M' {
			cur = 'L'
		}
	}

	return out
}

func tokenizeSVGPath(s string) []string {
	tokens := make([]string, 0, len(s)/2)
	for i := 0; i < len(s); {
		c := s[i]
		if isAlphaASCII(c) {
			tokens = append(tokens, s[i:i+1])
			i++

			continue
		}

		if c == ',' || c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			i++
			continue
		}

		j := i
		if s[j] == '+' || s[j] == '-' {
			j++
		}

		sawDigit := false
		sawDot := false

		for j < len(s) {
			ch := s[j]
			if ch >= '0' && ch <= '9' {
				sawDigit = true
				j++

				continue
			}

			if ch == '.' && !sawDot {
				sawDot = true
				j++

				continue
			}

			break
		}

		if !sawDigit {
			i++
			continue
		}

		tokens = append(tokens, s[i:j])
		i = j
	}

	return tokens
}

func svgPathCmdArity(op byte) int {
	switch op {
	case 'M', 'L':
		return 2
	case 'Q':
		return 4
	case 'C':
		return 6
	case 'Z':
		return 0
	default:
		return -1
	}
}

func isAlphaASCII(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

type fpoint struct{ x, y float64 }

type shapeKnot struct {
	inX, inY, pX, pY, outX, outY float64
}

func parseKnobShapeKnots(s string) [][]shapeKnot {
	if !strings.Contains(s, "/") {
		return nil
	}

	parts := strings.Split(s, "/")
	var polys [][]shapeKnot
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		chunks := strings.Split(part, ":")
		knots := make([]shapeKnot, 0, len(chunks))
		for _, ch := range chunks {
			vals := strings.Split(ch, ",")
			if len(vals) != 6 {
				continue
			}
			var k shapeKnot
			ok := true
			nums := []*float64{&k.inX, &k.inY, &k.pX, &k.pY, &k.outX, &k.outY}
			for i := range 6 {
				v, err := strconv.ParseFloat(strings.TrimSpace(vals[i]), 64)
				if err != nil {
					ok = false
					break
				}
				*nums[i] = v
			}
			if ok {
				knots = append(knots, k)
			}
		}
		if len(knots) >= 2 {
			polys = append(polys, knots)
		}
	}
	return polys
}

func parseKnobShapePolylines(s string, w, h int) [][]fpoint {
	return parseKnobShapePolylinesWithClosure(s, w, h, true)
}

func parseKnobShapePolylinesOpen(s string, w, h int) [][]fpoint {
	return parseKnobShapePolylinesWithClosure(s, w, h, false)
}

func parseKnobShapeAnchorPolylines(s string, w, h int) [][]fpoint {
	knotPolys := parseKnobShapeKnots(s)
	if len(knotPolys) == 0 {
		return nil
	}

	polys := make([][]fpoint, 0, len(knotPolys))
	for _, knots := range knotPolys {
		poly := make([]fpoint, 0, len(knots))
		for _, k := range knots {
			poly = append(poly, fpoint{x: shapeScaleX(k.pX, w), y: shapeScaleY(k.pY, h)})
		}
		if len(poly) >= 2 {
			polys = append(polys, poly)
		}
	}
	return polys
}

func parseKnobShapePolylinesWithClosure(s string, w, h int, closePath bool) [][]fpoint {
	knotPolys := parseKnobShapeKnots(s)
	if len(knotPolys) == 0 {
		return nil
	}

	polys := make([][]fpoint, 0, len(knotPolys))
	for _, knots := range knotPolys {
		poly := make([]fpoint, 0, len(knots)*12)
		start := fpoint{x: shapeScaleX(knots[0].pX, w), y: shapeScaleY(knots[0].pY, h)}
		poly = append(poly, start)
		for i := 1; i < len(knots); i++ {
			prev := knots[i-1]
			cur := knots[i]
			cubic := flattenCubic(
				fpoint{x: shapeScaleX(prev.pX, w), y: shapeScaleY(prev.pY, h)},
				fpoint{x: shapeScaleX(prev.outX, w), y: shapeScaleY(prev.outY, h)},
				fpoint{x: shapeScaleX(cur.inX, w), y: shapeScaleY(cur.inY, h)},
				fpoint{x: shapeScaleX(cur.pX, w), y: shapeScaleY(cur.pY, h)},
				12,
			)
			poly = append(poly, cubic[1:]...)
		}
		if closePath {
			last := knots[len(knots)-1]
			first := knots[0]
			cubic := flattenCubic(
				fpoint{x: shapeScaleX(last.pX, w), y: shapeScaleY(last.pY, h)},
				fpoint{x: shapeScaleX(last.outX, w), y: shapeScaleY(last.outY, h)},
				fpoint{x: shapeScaleX(first.inX, w), y: shapeScaleY(first.inY, h)},
				fpoint{x: shapeScaleX(first.pX, w), y: shapeScaleY(first.pY, h)},
				12,
			)
			poly = append(poly, cubic[1:]...)
		}
		polys = append(polys, poly)
	}
	return polys
}

func shapeScaleX(v float64, w int) float64 { return (v - 128.0) / 256.0 * float64(w) }
func shapeScaleY(v float64, h int) float64 { return (v - 128.0) / 256.0 * float64(h) }

func flattenQuadratic(p0, c, p1 fpoint, steps int) []fpoint {
	if steps < 2 {
		steps = 2
	}

	out := make([]fpoint, 0, steps+1)
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		u := 1.0 - t
		x := u*u*p0.x + 2*u*t*c.x + t*t*p1.x
		y := u*u*p0.y + 2*u*t*c.y + t*t*p1.y
		out = append(out, fpoint{x: x, y: y})
	}

	return out
}

func flattenCubic(p0, c1, c2, p1 fpoint, steps int) []fpoint {
	if steps < 2 {
		steps = 2
	}

	out := make([]fpoint, 0, steps+1)
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		u := 1.0 - t
		x := u*u*u*p0.x + 3*u*u*t*c1.x + 3*u*t*t*c2.x + t*t*t*p1.x
		y := u*u*u*p0.y + 3*u*u*t*c1.y + 3*u*t*t*c2.y + t*t*t*p1.y
		out = append(out, fpoint{x: x, y: y})
	}

	return out
}

func pointInPolysEvenOdd(x, y float64, polys [][]fpoint) bool {
	inside := false

	for _, poly := range polys {
		if len(poly) < 3 {
			continue
		}

		j := len(poly) - 1
		for i := range poly {
			xi, yi := poly[i].x, poly[i].y
			xj, yj := poly[j].x, poly[j].y

			intersect := ((yi > y) != (yj > y)) && (x < (xj-xi)*(y-yi)/(yj-yi+1e-12)+xi)
			if intersect {
				inside = !inside
			}

			j = i
		}
	}

	return inside
}

func parseFloat(s string) (float64, bool) {
	v, err := strconvParseFloat(s)
	if err != nil {
		return 0, false
	}

	return v, true
}

// strconvParseFloat is split for testability and to keep parse helpers close.
var strconvParseFloat = func(s string) (float64, error) { return strconv.ParseFloat(s, 64) }

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}

	if v > hi {
		return hi
	}

	return v
}

func linePointDistance(x0, y0, x1, y1, px, py float64) float64 {
	dx := x1 - x0
	dy := y1 - y0
	a := dx*dx + dy*dy
	if a == 0.0 {
		return math.Hypot(x0-px, y0-py)
	}

	b := dx*(x0-px) + dy*(y0-py)
	t := -(b / a)
	if t < 0.0 {
		t = 0.0
	}
	if t >= 1.0 {
		t = 1.0
	}

	x := t*dx + x0
	y := t*dy + y0
	return math.Hypot(x-px, y-py)
}

func rotatePoint(x, y, sinT, cosT float64) (float64, float64) {
	return x*cosT - y*sinT, x*sinT + y*cosT
}

func changeBrightnessRGBA(c color.RGBA, delta int) color.RGBA {
	c.R = uint8(clampInt(int(c.R)+delta, 0, 255))
	c.G = uint8(clampInt(int(c.G)+delta, 0, 255))
	c.B = uint8(clampInt(int(c.B)+delta, 0, 255))
	return c
}

func brightRGBA(c color.RGBA, scale int) color.RGBA {
	c.R = uint8(clampInt(int(c.R)*scale/255, 0, 255))
	c.G = uint8(clampInt(int(c.G)*scale/255, 0, 255))
	c.B = uint8(clampInt(int(c.B)*scale/255, 0, 255))
	return c
}

func blendToRGBA(base, target color.RGBA, alpha int) color.RGBA {
	base.R = uint8((int(base.R)*(256-alpha) + int(target.R)*alpha) / 256)
	base.G = uint8((int(base.G)*(256-alpha) + int(target.G)*alpha) / 256)
	base.B = uint8((int(base.B)*(256-alpha) + int(target.B)*alpha) / 256)
	return base
}

func applyTextureOverlay(base color.RGBA, textures []*Texture, p *model.Primitive, x, y int) color.RGBA {
	idx := p.TextureFile.Val
	if idx <= 0 || idx > len(textures) {
		return base
	}

	tex := textures[idx-1]
	if tex == nil {
		return base
	}

	zoom := p.TextureZoom.Val / 100.0
	if zoom <= 0 {
		zoom = 1
	}

	tx := float64(x)
	ty := float64(y)
	tc := tex.Sample(tx, ty, zoom)

	return TextureBlend(base, tc, p.TextureDepth.Val)
}

func sampleTextureLumiAlpha(textures []*Texture, p *model.Primitive, x, y float64) (lumi, alpha int) {
	if p == nil || p.TextureDepth.Val == 0 {
		return 0, 255
	}

	idx := p.TextureFile.Val
	if idx <= 0 || idx > len(textures) {
		return 0, 255
	}

	tex := textures[idx-1]
	if tex == nil {
		return 0, 255
	}

	luma, texAlpha := tex.SampleHeightAlpha(x, y, p.TextureZoom.Val)
	txd := p.TextureDepth.Val * 0.01
	lumi = int(float64(luma-128) * txd)
	alpha = 255 - int(float64(255-texAlpha)*txd)
	return lumi, alpha
}

type point struct{ x, y int }

func fillTriangle(dst *PixBuf, p0, p1, p2 point, c color.RGBA) {
	minX := min(p0.x, min(p1.x, p2.x))
	maxX := max(p0.x, max(p1.x, p2.x))
	minY := min(p0.y, min(p1.y, p2.y))
	maxY := max(p0.y, max(p1.y, p2.y))

	if minX < 0 {
		minX = 0
	}

	if minY < 0 {
		minY = 0
	}

	if maxX >= dst.Width {
		maxX = dst.Width - 1
	}

	if maxY >= dst.Height {
		maxY = dst.Height - 1
	}

	area := edge(p0, p1, p2)
	if area == 0 {
		return
	}

	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			p := point{x, y}
			w0 := edge(p1, p2, p)
			w1 := edge(p2, p0, p)

			w2 := edge(p0, p1, p)
			if (w0 >= 0 && w1 >= 0 && w2 >= 0) || (w0 <= 0 && w1 <= 0 && w2 <= 0) {
				dst.BlendOver(x, y, c)
			}
		}
	}
}

func edge(a, b, c point) int {
	return (c.x-a.x)*(b.y-a.y) - (c.y-a.y)*(b.x-a.x)
}

func drawLine(dst *PixBuf, x0, y0, x1, y1 int, c color.RGBA, width int) {
	if width < 1 {
		width = 1
	}

	dx := int(math.Abs(float64(x1 - x0)))
	dy := -int(math.Abs(float64(y1 - y0)))

	sx := -1
	if x0 < x1 {
		sx = 1
	}

	sy := -1
	if y0 < y1 {
		sy = 1
	}

	err := dx + dy

	for {
		hw := width / 2
		for oy := -hw; oy <= hw; oy++ {
			for ox := -hw; ox <= hw; ox++ {
				dst.BlendOver(x0+ox, y0+oy, c)
			}
		}

		if x0 == x1 && y0 == y1 {
			break
		}

		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}

		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}
