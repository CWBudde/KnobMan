package render

import (
	"image/color"
	"testing"

	"knobman/internal/model"
)

func TestTransformBilinearTranslate(t *testing.T) {
	src := NewPixBuf(6, 6)
	src.Set(2, 2, color.RGBA{255, 0, 0, 255})

	dst := NewPixBuf(6, 6)
	m := BuildMatrix(100, 100, 0, 1, 0, 0, 0)
	TransformBilinear(dst, src, m)

	if got := dst.At(3, 2); got.A == 0 {
		t.Fatalf("expected translated non-zero pixel at (3,2), got %+v", got)
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

func TestMakeShadowOffset(t *testing.T) {
	src := NewPixBuf(8, 8)
	src.Set(2, 2, color.RGBA{255, 255, 255, 255})

	shadow := MakeShadow(src, 2, 100, 0, 0, color.RGBA{0, 0, 0, 255})
	if got := shadow.At(4, 2); got.A == 0 {
		t.Fatalf("expected shifted shadow pixel, got %+v", got)
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

	if got := buf.At(8, 8); got != (color.RGBA{0, 0, 0, 0}) {
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
