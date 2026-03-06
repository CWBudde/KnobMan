package render

import (
	"image/color"
	"math"
	"strings"

	"knobman/internal/model"
)

// CompositeOver alpha-blends src over dst.
func CompositeOver(dst, src *PixBuf) {
	CompositeOverAt(dst, src, 0, 0)
}

// CompositeOverAt alpha-blends src over dst with an integer offset.
func CompositeOverAt(dst, src *PixBuf, ox, oy int) {
	if dst == nil || src == nil {
		return
	}

	for y := range src.Height {
		dy := y + oy
		if dy < 0 || dy >= dst.Height {
			continue
		}

		for x := range src.Width {
			dx := x + ox
			if dx < 0 || dx >= dst.Width {
				continue
			}

			c := src.At(x, y)
			if c.A == 0 {
				continue
			}

			dst.BlendOver(dx, dy, c)
		}
	}
}

// ApplyEffect applies the effect stack to primBuf and composites it onto dst.
func ApplyEffect(dst *PixBuf, primBuf *PixBuf, eff *model.Effect, curves *[8]model.AnimCurve, frame, totalFrames int, textures []*Texture) {
	_ = textures // kept for API continuity; used by later effect extensions.

	if dst == nil || primBuf == nil || eff == nil {
		return
	}

	ratio := FrameFrac(frame, totalFrames, eff.AnimStep.Val)
	if !frameMaskVisible(eff, frame, ratio) {
		return
	}

	work := primBuf.Clone()

	zoomX := EvalAnim(eff.ZoomXF.Val, eff.ZoomXT.Val, eff.ZoomXAnim.Val, curves, ratio)

	zoomY := EvalAnim(eff.ZoomYF.Val, eff.ZoomYT.Val, eff.ZoomYAnim.Val, curves, ratio)
	if eff.ZoomXYSepa.Val == 0 {
		zoomY = zoomX
	}

	offX := EvalAnim(eff.OffXF.Val, eff.OffXT.Val, eff.OffXAnim.Val, curves, ratio)
	offY := EvalAnim(eff.OffYF.Val, eff.OffYT.Val, eff.OffYAnim.Val, curves, ratio)
	angle := EvalAnim(eff.AngleF.Val, eff.AngleT.Val, eff.AngleAnim.Val, curves, ratio)

	needTransform := math.Abs(zoomX-100.0) > 1e-6 || math.Abs(zoomY-100.0) > 1e-6 || math.Abs(angle) > 1e-6 || math.Abs(offX) > 1e-6 || math.Abs(offY) > 1e-6 || math.Abs(eff.CenterX.Val) > 1e-6 || math.Abs(eff.CenterY.Val) > 1e-6
	if needTransform {
		tmp := NewPixBuf(work.Width, work.Height)
		offXPx := offX * 0.01 * float64(work.Width)
		offYPx := offY * 0.01 * float64(work.Height)
		cx := float64(work.Width)*0.5 + eff.CenterX.Val*0.01*float64(work.Width)
		cy := float64(work.Height)*0.5 + eff.CenterY.Val*0.01*float64(work.Height)
		m := BuildMatrix(zoomX, zoomY, angle, offXPx, offYPx, cx, cy)
		TransformBilinear(tmp, work, m)
		work = tmp
	}

	alpha := EvalAnim(eff.AlphaF.Val, eff.AlphaT.Val, eff.AlphaAnim.Val, curves, ratio)
	bright := EvalAnim(eff.BrightF.Val, eff.BrightT.Val, eff.BrightAnim.Val, curves, ratio)
	contrast := EvalAnim(eff.ContrastF.Val, eff.ContrastT.Val, eff.ContrastAnim.Val, curves, ratio)
	sat := EvalAnim(eff.SaturationF.Val, eff.SaturationT.Val, eff.SaturationAnim.Val, curves, ratio)

	hue := EvalAnim(eff.HueF.Val, eff.HueT.Val, eff.HueAnim.Val, curves, ratio)
	if math.Abs(alpha-100.0) > 1e-6 || math.Abs(bright) > 1e-6 || math.Abs(contrast) > 1e-6 || math.Abs(sat) > 1e-6 || math.Abs(hue) > 1e-6 {
		tmp := NewPixBuf(work.Width, work.Height)
		ApplyColorAdjust(tmp, work, alpha, bright, contrast, sat, hue)
		work = tmp
	}

	var mask []float64

	if eff.Mask1Ena.Val != 0 {
		m1 := BuildMask(
			work.Width,
			work.Height,
			eff.Mask1Type.Val,
			EvalAnim(eff.Mask1StartF.Val, eff.Mask1StartT.Val, eff.Mask1StartAnim.Val, curves, ratio),
			EvalAnim(eff.Mask1StopF.Val, eff.Mask1StopT.Val, eff.Mask1StopAnim.Val, curves, ratio),
			eff.Mask1Grad.Val,
			eff.Mask1GradDir.Val,
		)
		mask = m1
	}

	if eff.Mask2Ena.Val != 0 {
		m2 := BuildMask(
			work.Width,
			work.Height,
			eff.Mask2Type.Val,
			EvalAnim(eff.Mask2StartF.Val, eff.Mask2StartT.Val, eff.Mask2StartAnim.Val, curves, ratio),
			EvalAnim(eff.Mask2StopF.Val, eff.Mask2StopT.Val, eff.Mask2StopAnim.Val, curves, ratio),
			eff.Mask2Grad.Val,
			eff.Mask2GradDir.Val,
		)
		if mask == nil {
			mask = m2
		} else {
			mask = CombineMasks(mask, m2, eff.Mask2Op.Val)
		}
	}

	if mask != nil {
		ApplyMask(work, mask)
	}

	slightDir := EvalAnim(eff.SLightDirF.Val, eff.SLightDirT.Val, eff.SLightDirAnim.Val, curves, ratio)
	sden := EvalAnim(eff.SDensityF.Val, eff.SDensityT.Val, eff.SDensityAnim.Val, curves, ratio)
	edens := EvalAnim(eff.EDensityF.Val, eff.EDensityT.Val, eff.EDensityAnim.Val, curves, ratio)
	eoff := EvalAnim(eff.EOffsetF.Val, eff.EOffsetT.Val, eff.EOffsetAnim.Val, curves, ratio)

	edir := slightDir
	if eff.ELightDirEna.Val != 0 {
		edir = EvalAnim(eff.ELightDirF.Val, eff.ELightDirT.Val, eff.ELightDirAnim.Val, curves, ratio)
	}

	few := float64(min(work.Width, work.Height)) * (eoff + 1.0) / 400.0
	if math.Abs(few) > 1e-12 {
		HilightLegacy(work, slightDir, sden, edir, few, edens/(40000.0*few))
	}

	ddens := EvalAnim(eff.DDensityF.Val, eff.DDensityT.Val, eff.DDensityAnim.Val, curves, ratio)
	if ddens != 0 {
		off := EvalAnim(eff.DOffsetF.Val, eff.DOffsetT.Val, eff.DOffsetAnim.Val, curves, ratio)
		diff := EvalAnim(eff.DDiffuseF.Val, eff.DDiffuseT.Val, eff.DDiffuseAnim.Val, curves, ratio)

		dir := slightDir
		if eff.DLightDirEna.Val != 0 {
			dir = EvalAnim(eff.DLightDirF.Val, eff.DLightDirT.Val, eff.DLightDirAnim.Val, curves, ratio)
		}

		shadowColor := color.RGBA{0, 0, 0, 255}
		if ddens < 0 {
			shadowColor = color.RGBA{255, 255, 255, 255}
		}

		shadow := MakeShadowLegacy(
			work,
			false,
			eff.DSType.Val,
			eff.DSGrad.Val,
			off*0.01*float64(min(work.Width, work.Height)),
			dir,
			math.Abs(ddens),
			diff,
			shadowColor,
		)
		CompositeOver(dst, shadow)
	}

	idens := EvalAnim(eff.IDensityF.Val, eff.IDensityT.Val, eff.IDensityAnim.Val, curves, ratio)
	if idens != 0 {
		ioff := EvalAnim(eff.IOffsetF.Val, eff.IOffsetT.Val, eff.IOffsetAnim.Val, curves, ratio)
		idiff := EvalAnim(eff.IDiffuseF.Val, eff.IDiffuseT.Val, eff.IDiffuseAnim.Val, curves, ratio)

		dir := slightDir
		if eff.ILightDirEna.Val != 0 {
			dir = EvalAnim(eff.ILightDirF.Val, eff.ILightDirT.Val, eff.ILightDirAnim.Val, curves, ratio)
		}

		innerColor := color.RGBA{0, 0, 0, 255}
		if idens < 0 {
			innerColor = color.RGBA{255, 255, 255, 255}
		}

		inner := MakeShadowLegacy(
			work,
			true,
			0,
			0,
			ioff*0.01*float64(min(work.Width, work.Height)),
			dir,
			math.Abs(idens),
			idiff,
			innerColor,
		)
		MultiplyAlphaByMask(inner, work)
		CompositeOver(work, inner)
	}

	CompositeOver(dst, work)
}

func frameMaskVisible(eff *model.Effect, frame int, ratio float64) bool {
	switch eff.FMaskEna.Val {
	case 0:
		return true
	case 1:
		v := ratio * 100.0

		lo, hi := eff.FMaskStart.Val, eff.FMaskStop.Val
		if lo > hi {
			lo, hi = hi, lo
		}

		return v >= lo && v <= hi
	case 2:
		bits := strings.TrimSpace(eff.FMaskBits.Val)
		if bits == "" {
			return true
		}

		idx := frame % len(bits)
		if idx < 0 {
			idx += len(bits)
		}

		c := bits[idx]

		return c == '1' || c == 'y' || c == 'Y' || c == '*'
	default:
		return true
	}
}
