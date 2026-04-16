package render

import (
	"image/color"
	"math"

	"knobman/internal/model"
)

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

	for y := range dst.Height {
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

		for x := range dst.Width {
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
			dst.BlendOver(x, y, pix)
		}
	}
}

type circleFillGeometry struct {
	cx, cy      float64
	ax, ay      float64
	rThick      float64
	rEmbossEdge float64
	rD2         float64
	rMin        float64
}

type circleFillSample struct {
	px, py float64
	rx, ry float64
	rxy    float64
	rxym   float64
	rxye   float64
	rxyem  float64
}

type circleFillLighting struct {
	rLX, rLY, rLZ       float64
	rTZ                 float64
	iMetalAmbient       int
	iMetalSpecular      int
	dMetalSpecularWidth float64
	rSX, rSY, rSZ       float64
}

func newCircleFillGeometry(dst *PixBuf, p *model.Primitive) circleFillGeometry {
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

	rD := 1.0 - p.Diffuse.Val/100.0

	return circleFillGeometry{
		cx:          cx,
		cy:          cy,
		ax:          ax,
		ay:          ay,
		rThick:      rThick,
		rEmbossEdge: (100.0 - p.EmbossDiffuse.Val) / 100.0,
		rD2:         rD * rD,
		rMin:        math.Min(ax, ay),
	}
}

func (g circleFillGeometry) sample(x, y int) circleFillSample {
	py := float64(y) - g.cy + 0.5
	px := float64(x) - g.cx + 0.5
	ry := py / (g.ay + 0.5)
	rym := py / (g.ay - 0.5)
	rye := py / (g.ay + 0.5 - g.rMin*g.rThick*g.rEmbossEdge)
	ryem := py / (g.ay - g.rMin*g.rThick)
	rx := px / (g.ax + 0.5)
	rxm := px / (g.ax - 0.5)
	rxe := px / (g.ax + 0.5 - g.rMin*g.rThick*g.rEmbossEdge)
	rxem := px / (g.ax - g.rMin*g.rThick)

	return circleFillSample{
		px:    px,
		py:    py,
		rx:    rx,
		ry:    ry,
		rxy:   rx*rx + ry*ry,
		rxym:  rxm*rxm + rym*rym,
		rxye:  rxe*rxe + rye*rye,
		rxyem: rxem*rxem + ryem*ryem,
	}
}

func (g circleFillGeometry) applyAlpha(alpha int, s circleFillSample) int {
	if s.rxy > 1.0 {
		alpha = 0
	}

	if g.rD2 < 1.0 && s.rxy > g.rD2 {
		alpha = int(float64(alpha) * (1.0 - (s.rxy-g.rD2)/(1.0-g.rD2)))
	}

	if s.rxym >= 1.0 {
		alpha = int((1.0 - s.rxy) / (s.rxym - s.rxy) * float64(alpha))
	}

	return alpha
}

func newCircleFillLighting(p *model.Primitive) circleFillLighting {
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

	return circleFillLighting{
		rLX: rLX, rLY: rLY, rLZ: rLZ,
		rTZ:                 rTZ,
		iMetalAmbient:       iMetalAmbient,
		iMetalSpecular:      iMetalSpecular,
		dMetalSpecularWidth: dMetalSpecularWidth,
		rSX:                 rSX, rSY: rSY, rSZ: rSZ,
	}
}

func shadeCircleFillInterior(base color.RGBA, p *model.Primitive, s circleFillSample, lumi int, l circleFillLighting) color.RGBA {
	pix := base
	if s.rxye >= 1.0 {
		return pix
	}

	switch model.PrimitiveType(p.Type.Val) {
	case model.PrimCircleFill:
		pix = changeBrightnessRGBA(pix, int((-s.rx-s.ry)*128.0*p.Specular.Val/100.0)+lumi)
	case model.PrimMetalCircle:
		d := math.Sin(math.Atan2(s.ry, s.rx) * 2.0)
		d2 := math.Pow((d+1.0)*0.5, l.dMetalSpecularWidth)
		a := (d+1.0)*0.5*float64(255-l.iMetalAmbient) + float64(l.iMetalAmbient) + d2*float64(l.iMetalSpecular) + float64(lumi)
		pix = changeBrightnessRGBA(pix, int(a-256.0))
	case model.PrimSphere:
		rZ := math.Sqrt(math.Max(0.0, 1.0-s.rxy))
		d := -s.rx*l.rSX - s.ry*l.rSY + rZ*l.rSZ
		a := float64(l.iMetalAmbient) + float64(255-l.iMetalAmbient)*d

		d = 2.0*rZ*d - l.rSZ
		if d <= 0.0 {
			d = 0.0
		} else {
			d = math.Exp(math.Log(d) * (110.0 - p.SpecularWidth.Val) / 10.0)
		}

		pix = brightRGBA(pix, int(a)+lumi)
		pix = changeBrightnessRGBA(pix, int(float64(l.iMetalSpecular)*d))
	}

	return pix
}

func applyCircleFillEmboss(base, pix color.RGBA, p *model.Primitive, s circleFillSample, l circleFillLighting) color.RGBA {
	if s.rxyem < 1.0 {
		return pix
	}

	rR2 := math.Sqrt(math.Max(1e-12, 2.0*s.rxy))
	rTX := -s.rx / rR2
	rTY := -s.ry / rR2
	edgePix := base

	if p.Emboss.Val > 0.0 {
		r := rTX*l.rLX + rTY*l.rLY + l.rTZ*l.rLZ
		edgePix = changeBrightnessRGBA(edgePix, int(r*255.0-128.0))
	} else if p.Emboss.Val < 0.0 {
		r := -rTX*l.rLX - rTY*l.rLY + l.rTZ*l.rLZ
		edgePix = changeBrightnessRGBA(edgePix, int(r*255.0-128.0))
	}

	a := math.Min(255.0, math.Max(0.0, 255.0*math.Abs((1.0-s.rxyem)/(s.rxye-s.rxyem))))
	if s.rxye < 1.0 {
		return blendToRGBA(pix, edgePix, int(a))
	}

	return edgePix
}

func renderCircleFill(dst *PixBuf, p *model.Primitive, textures []*Texture) {
	base := primitiveColor(p)
	geom := newCircleFillGeometry(dst, p)
	lighting := newCircleFillLighting(p)

	for y := range dst.Height {
		for x := range dst.Width {
			s := geom.sample(x, y)

			pix := base
			lumi, alpha := sampleTextureLumiAlpha(textures, p, s.px, s.py)

			pix = shadeCircleFillInterior(pix, p, s, lumi, lighting)
			pix = applyCircleFillEmboss(base, pix, p, s, lighting)

			alpha = geom.applyAlpha(alpha, s)
			if alpha <= 0 {
				continue
			}

			pix.A = uint8(clampInt(alpha, 0, 255))
			dst.BlendOver(x, y, pix)
		}
	}
}

func renderMetalCircle(dst *PixBuf, p *model.Primitive, textures []*Texture) {
	base := primitiveColor(p)
	cx := float64(dst.Width-1) * 0.5
	cy := float64(dst.Height-1) * 0.5

	r := float64(min(dst.Width, dst.Height)) * 0.46
	for y := range dst.Height {
		for x := range dst.Width {
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
	var lumi float64
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

	for y := range dst.Height {
		rPY := float64(y) - rCY + 0.5
		rY := rPY / rAY

		for x := range dst.Width {
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
			dst.BlendOver(x, y, pix)
		}
	}
}

func renderSphere(dst *PixBuf, p *model.Primitive, textures []*Texture) {
	base := primitiveColor(p)
	cx := float64(dst.Width-1) * 0.5
	cy := float64(dst.Height-1) * 0.5
	rx := float64(dst.Width) * 0.45

	ry := float64(dst.Height) * 0.45
	for y := range dst.Height {
		for x := range dst.Width {
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
