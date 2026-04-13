package render

import (
	"image/color"
	"math"
	"testing"

	"knobman/internal/model"
)

func TestTransformBilinearTranslate(t *testing.T) {
	src := NewPixBuf(6, 6)
	src.Set(2, 2, color.RGBA{255, 0, 0, 255})

	dst := NewPixBuf(6, 6)
	m := BuildMatrix(6, 6, 100, 100, 0, 1, 0, 0, 0, false)
	TransformBilinear(dst, src, m)

	if got := dst.At(3, 2); got.A == 0 {
		t.Fatalf("expected translated non-zero pixel at (3,2), got %+v", got)
	}
}

func TestTransformBilinearIdentityKeepsSemiTransparentPixelWithinOneLSB(t *testing.T) {
	src := NewPixBuf(4, 4)
	src.Set(1, 1, color.RGBA{R: 200, G: 10, B: 20, A: 128})

	dst := NewPixBuf(4, 4)
	m := [6]float64{1, 0, 0, 1, 0, 0}
	TransformBilinear(dst, src, m)

	got := dst.At(1, 1)
	want := src.At(1, 1)
	if deltaRGBA(got, want) > 1 {
		t.Fatalf("identity transform drift too large: got %+v want %+v", got, want)
	}
}

func TestBuildMatrixRotationAroundEffectCenterKeepsCenterFixed(t *testing.T) {
	const (
		w       = 100
		h       = 80
		centerX = 20.0
		centerY = -10.0
	)

	m := BuildMatrix(w, h, 100, 100, 33, 0, 0, centerX, centerY, false)
	cx := (centerX + 50.0) * 0.01 * float64(w)
	cy := (50.0 - centerY) * 0.01 * float64(h)
	tx, ty := applyAffine(m, cx, cy)

	if !nearlyEqual(tx, cx, 1e-6) || !nearlyEqual(ty, cy, 1e-6) {
		t.Fatalf("rotation center moved: got (%f,%f) want (%f,%f)", tx, ty, cx, cy)
	}
}

func TestBuildMatrixNonUniformScale(t *testing.T) {
	const (
		w  = 100
		h  = 80
		cx = 50.0
		cy = 40.0
	)

	m := BuildMatrix(w, h, 50, 200, 0, 0, 0, 0, 0, false)

	tx, ty := applyAffine(m, cx+10, cy+10)
	if !nearlyEqual(tx-cx, 20, 1e-6) {
		t.Fatalf("unexpected x scale delta: got %f want 20", tx-cx)
	}

	if !nearlyEqual(ty-cy, 5, 1e-6) {
		t.Fatalf("unexpected y scale delta: got %f want 5", ty-cy)
	}
}

func TestBuildMatrixKeepDirCancelsRotationAtImageCenter(t *testing.T) {
	const (
		w  = 100
		h  = 80
		px = 70.0
		py = 40.0
	)

	mNoKeep := BuildMatrix(w, h, 100, 100, 37, 0, 0, 0, 0, false)
	mKeep := BuildMatrix(w, h, 100, 100, 37, 0, 0, 0, 0, true)

	txNoKeep, tyNoKeep := applyAffine(mNoKeep, px, py)
	if nearlyEqual(txNoKeep, px, 1e-6) && nearlyEqual(tyNoKeep, py, 1e-6) {
		t.Fatalf("rotation without keepdir unexpectedly preserved point (%f,%f)", px, py)
	}

	txKeep, tyKeep := applyAffine(mKeep, px, py)
	if !nearlyEqual(txKeep, px, 1e-6) || !nearlyEqual(tyKeep, py, 1e-6) {
		t.Fatalf("keepdir should preserve point at image-centered rotation: got (%f,%f) want (%f,%f)", txKeep, tyKeep, px, py)
	}
}

func TestTransformBilinearOutOfBoundsTransparentClipping(t *testing.T) {
	src := NewPixBuf(6, 6)
	src.FillRect(1, 1, 4, 4, color.RGBA{255, 0, 0, 255})

	dst := NewPixBuf(6, 6)
	m := BuildMatrix(6, 6, 100, 100, 0, 20, 0, 0, 0, false)
	TransformBilinear(dst, src, m)

	for y := range dst.Height {
		for x := range dst.Width {
			if got := dst.At(x, y); got.A != 0 {
				t.Fatalf("expected fully transparent output after out-of-bounds transform, got pixel (%d,%d)=%+v", x, y, got)
			}
		}
	}
}

func TestTransformBilinearPartialOutOfBoundsStaysTransparentAtEdge(t *testing.T) {
	src := NewPixBuf(4, 4)
	src.FillRect(0, 0, 4, 4, color.RGBA{255, 0, 0, 255})

	dst := NewPixBuf(4, 4)
	m := BuildMatrix(4, 4, 100, 100, 0, -1, 0, 0, 0, false)
	TransformBilinear(dst, src, m)

	if got := dst.At(3, 1); got.A != 0 {
		t.Fatalf("expected uncovered right edge transparent, got %+v", got)
	}

	if got := dst.At(2, 1); got.A == 0 {
		t.Fatalf("expected shifted interior pixel visible, got %+v", got)
	}
}

func TestApplyColorAdjustAlpha(t *testing.T) {
	src := NewPixBuf(1, 1)
	src.Set(0, 0, color.RGBA{100, 120, 140, 200})

	dst := NewPixBuf(1, 1)
	ApplyColorAdjust(dst, src, 50, 0, 0, 0, 0)

	got := dst.At(0, 0)
	if got.A < 95 || got.A > 105 {
		t.Fatalf("expected alpha around 100, got %d", got.A)
	}
}

func TestBuildMaskAndApply(t *testing.T) {
	buf := NewPixBuf(8, 8)
	buf.Clear(color.RGBA{255, 255, 255, 255})

	mask := BuildMask(8, 8, 2, -20, 20, 0, 0)
	ApplyMask(buf, mask)
	left := buf.At(0, 4).A

	center := buf.At(4, 4).A
	if center <= left {
		t.Fatalf("expected center alpha > edge alpha, got center=%d left=%d", center, left)
	}
}

func TestBuildMaskWithCenterShiftsMask(t *testing.T) {
	plain := BuildMask(10, 10, 2, -20, 20, 0, 0)
	shifted := BuildMaskWithCenter(10, 10, 2, -20, 20, 0, 0, 50, 0)

	centerIdx := 5*10 + 5
	if shifted[centerIdx] >= plain[centerIdx] {
		t.Fatalf("expected center-shifted mask to reduce center value, got shifted=%f plain=%f", shifted[centerIdx], plain[centerIdx])
	}
}

func TestMaskLegacyCenterAndCombineSemantics(t *testing.T) {
	leftHalf := BuildMask(10, 10, 2, -100, 0, 0, 0)
	shiftedRight := BuildMaskWithCenter(10, 10, 2, -100, 0, 0, 0, 50, 0)
	rightHalf := BuildMask(10, 10, 2, 0, 100, 0, 0)

	if got := leftHalf[5*10+1]; got < 0.99 {
		t.Fatalf("left-half mask should fully cover the left side, got %.4f", got)
	}

	if got := leftHalf[5*10+8]; got != 0 {
		t.Fatalf("left-half mask should clear the right side, got %.4f", got)
	}

	if got := shiftedRight[5*10+1]; got != 0 {
		t.Fatalf("center shift should move coverage off the old left edge, got %.4f", got)
	}

	if got := shiftedRight[5*10+6]; got < 0.99 {
		t.Fatalf("center shift should move coverage rightward, got %.4f", got)
	}

	andMask := CombineMasks(leftHalf, rightHalf, 0)
	orMask := CombineMasks(leftHalf, rightHalf, 1)

	if got := andMask[5*10+1]; got != 0 {
		t.Fatalf("AND-combined mask should reject left-only coverage, got %.4f", got)
	}

	if got := orMask[5*10+1]; got < 0.99 {
		t.Fatalf("OR-combined mask should keep left-half coverage, got %.4f", got)
	}

	if got := orMask[5*10+8]; got < 0.99 {
		t.Fatalf("OR-combined mask should keep right-half coverage, got %.4f", got)
	}
}

func TestMakeShadowOffset(t *testing.T) {
	src := NewPixBuf(8, 8)
	src.Set(2, 2, color.RGBA{255, 255, 255, 255})

	shadow := MakeShadow(src, 2, 100, 0, 0, color.RGBA{0, 0, 0, 255})
	if got := shadow.At(4, 2); got.A == 0 {
		t.Fatalf("expected shifted shadow pixel, got %+v", got)
	}
}

func TestShadowLegacyDirectionalSweepAndDiffuse(t *testing.T) {
	src := NewPixBuf(40, 40)
	src.Set(10, 20, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	swept := MakeShadowLegacy(src, false, 1, 100, 4, 270, 100, 0, color.RGBA{A: 255})
	for x := 11; x <= 14; x++ {
		if got := swept.At(x, 20); got.A == 0 {
			t.Fatalf("directional sweep should cover trail pixel (%d,20), got %+v", x, got)
		}
	}

	if got := swept.At(12, 19); got.A != 0 {
		t.Fatalf("non-diffused sweep should stay on-axis, got %+v", got)
	}

	diffused := MakeShadowLegacy(src, false, 1, 100, 4, 270, 100, 40, color.RGBA{A: 255})
	if got := diffused.At(12, 19); got.A == 0 {
		t.Fatalf("diffuse blur should spread alpha off-axis, got %+v", got)
	}
}

func TestApplyEffectFrameMaskBits(t *testing.T) {
	prim := NewPixBuf(4, 4)
	prim.Clear(color.RGBA{255, 0, 0, 255})

	eff := model.NewEffect()
	eff.FMaskEna.Val = 2
	eff.FMaskBits.Val = "10"
	curves := [8]model.AnimCurve{}

	dst0 := NewPixBuf(4, 4)
	ApplyEffect(dst0, prim, &eff, &curves, 0, 2, nil)

	if dst0.At(1, 1).A == 0 {
		t.Fatal("frame 0 should be visible for bitmask 10")
	}

	dst1 := NewPixBuf(4, 4)
	ApplyEffect(dst1, prim, &eff, &curves, 1, 2, nil)

	if dst1.At(1, 1).A != 0 {
		t.Fatal("frame 1 should be hidden for bitmask 10")
	}
}

func TestColorAdjustKeepsLocalHueSaturationAlphaSemantics(t *testing.T) {
	src := NewPixBuf(1, 1)
	src.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 200})

	gray := NewPixBuf(1, 1)
	ApplyColorAdjust(gray, src, 100, 0, 0, -100, 0)

	gotGray := gray.At(0, 0)
	if gotGray.R != gotGray.G || gotGray.G != gotGray.B {
		t.Fatalf("desaturation should produce grayscale output, got %+v", gotGray)
	}

	rotated := NewPixBuf(1, 1)
	ApplyColorAdjust(rotated, src, 100, 0, 0, 0, 120)

	gotRotated := rotated.At(0, 0)
	if gotRotated.A != 200 {
		t.Fatalf("hue rotation should preserve alpha when alpha scale is 100, got %+v", gotRotated)
	}

	if gotRotated.G <= gotRotated.R || gotRotated.G <= gotRotated.B {
		t.Fatalf("120-degree hue shift should move red toward green, got %+v", gotRotated)
	}
}

func TestEffectMaskPipelineAppliesCombinedMasks(t *testing.T) {
	prim := NewPixBuf(12, 12)
	prim.Clear(color.RGBA{R: 40, G: 80, B: 120, A: 255})

	eff := model.NewEffect()
	eff.Mask1Ena.Val = 1
	eff.Mask1Type.Val = 2
	eff.Mask1StartF.Val = -100
	eff.Mask1StopF.Val = 0
	eff.Mask2Ena.Val = 1
	eff.Mask2Type.Val = 2
	eff.Mask2StartF.Val = 0
	eff.Mask2StopF.Val = 100
	eff.Mask2Op.Val = 0

	dst := NewPixBuf(12, 12)
	curves := [8]model.AnimCurve{}
	ApplyEffect(dst, prim, &eff, &curves, 0, 1, nil)

	if got := dst.At(2, 6); got.A != 0 {
		t.Fatalf("non-overlapping AND masks should clear left side, got %+v", got)
	}

	if got := dst.At(9, 6); got.A != 0 {
		t.Fatalf("non-overlapping AND masks should clear right side, got %+v", got)
	}
}

func TestRenderFrameSoloSelection(t *testing.T) {
	doc := model.NewDocument()
	doc.Prefs.PWidth.Val = 16
	doc.Prefs.PHeight.Val = 16
	doc.Prefs.RenderFrames.Val = 1
	doc.Prefs.Oversampling.Val = 0
	doc.Prefs.BkColor.Val = color.RGBA{255, 255, 255, 255}

	if len(doc.Layers) < 2 {
		t.Fatal("expected at least 2 layers")
	}

	doc.Layers[0].Visible.Val = 1
	doc.Layers[0].Solo.Val = 0
	doc.Layers[0].Prim.Type.Val = int(model.PrimRectFill)
	doc.Layers[0].Prim.Color.Val = color.RGBA{255, 0, 0, 255}
	doc.Layers[0].Prim.Length.Val = 100
	doc.Layers[0].Prim.Aspect.Val = 100

	doc.Layers[1].Visible.Val = 1
	doc.Layers[1].Solo.Val = 1
	doc.Layers[1].Prim.Type.Val = int(model.PrimNone)

	buf := NewPixBuf(16, 16)
	RenderFrame(buf, doc, 0, nil)

	if got := buf.At(8, 8); got != doc.Prefs.BkColor.Val {
		t.Fatalf("solo layer should suppress non-solo layers; got %+v", got)
	}

	doc.Layers[1].Prim.Type.Val = int(model.PrimRectFill)
	doc.Layers[1].Prim.Color.Val = color.RGBA{0, 255, 0, 255}
	doc.Layers[1].Prim.Length.Val = 100
	doc.Layers[1].Prim.Aspect.Val = 100
	RenderFrame(buf, doc, 0, nil)

	if got := buf.At(8, 8); got.G < 150 {
		t.Fatalf("solo layer draw expected green-ish center, got %+v", got)
	}
}

func TestDownsampleBoxPreservesUniformColorAndAveragesAlpha(t *testing.T) {
	src := NewPixBuf(2, 2)
	src.Set(0, 0, color.RGBA{R: 80, G: 120, B: 160, A: 0})
	src.Set(1, 0, color.RGBA{R: 80, G: 120, B: 160, A: 64})
	src.Set(0, 1, color.RGBA{R: 80, G: 120, B: 160, A: 128})
	src.Set(1, 1, color.RGBA{R: 80, G: 120, B: 160, A: 255})

	dst := NewPixBuf(1, 1)
	downsampleBox(dst, src, 2)

	got := dst.At(0, 0)
	if got.R != 80 || got.G != 120 || got.B != 160 {
		t.Fatalf("expected uniform color to stay unchanged, got %+v", got)
	}

	if got.A < 111 || got.A > 112 {
		t.Fatalf("expected alpha average near 112, got %+v", got)
	}
}

func deltaRGBA(a, b color.RGBA) int {
	d := absInt(int(a.R) - int(b.R))
	d = max(d, absInt(int(a.G)-int(b.G)))
	d = max(d, absInt(int(a.B)-int(b.B)))
	d = max(d, absInt(int(a.A)-int(b.A)))

	return d
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}

	return v
}

func nearlyEqual(a, b, eps float64) bool {
	return math.Abs(a-b) <= eps
}
