package render

import (
	"image/color"
	"math"
	"testing"

	"knobman/internal/model"
)

func TestDynamicTextEvalSupportsMathFunctionsAndPrecedence(t *testing.T) {
	dt := newDynamicText("pow(x,2)+sqrt(9)+log(exp(1))+log10(100)-5/2")

	got := dt.eval(0, 4)

	want := 19.5
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("eval mismatch: got %v want %v", got, want)
	}
}

func TestResolveDynamicTextExpressionFormatting(t *testing.T) {
	got := ResolveDynamicText("(1:3:%+04d:x*2)", 1, 3)
	if got != "+0004" {
		t.Fatalf("unexpected formatted dynamic text: got %q want %q", got, "+0004")
	}
}

func TestDynamicTextWSprintfSupportsNumericFormats(t *testing.T) {
	cases := []struct {
		fmt  string
		val  float64
		want string
	}{
		{fmt: "%04x", val: 15, want: "000f"},
		{fmt: "%04X", val: 15, want: "000F"},
		{fmt: "%.2f", val: 15.5, want: "15.50"},
	}

	for _, tc := range cases {
		dt := dynamicText{fmt: tc.fmt}
		if got := dt.wsprintf(tc.val); got != tc.want {
			t.Fatalf("wsprintf(%q, %v) = %q, want %q", tc.fmt, tc.val, got, tc.want)
		}
	}
}

func TestDynamicTextGetANumAndEval0HandleWhitespaceAndUnary(t *testing.T) {
	dt := newDynamicText("   -12.5")
	if got := dt.getANum(0); math.Abs(got+12.5) > 1e-9 {
		t.Fatalf("unexpected parsed number: %v", got)
	}

	plus := newDynamicText("+(x)")
	if got := plus.eval0(0, 7); got != 7 {
		t.Fatalf("unexpected unary plus result: %v", got)
	}

	minus := newDynamicText("-(x)")
	if got := minus.eval0(0, 7); got != -7 {
		t.Fatalf("unexpected unary minus result: %v", got)
	}
}

func TestGaussian1DNormalizedAndSymmetric(t *testing.T) {
	kernel := Gaussian1D(2)
	if len(kernel) != 5 {
		t.Fatalf("unexpected kernel size: %d", len(kernel))
	}

	sum := 0.0
	for i := range kernel {
		sum += kernel[i]
		if math.Abs(kernel[i]-kernel[len(kernel)-1-i]) > 1e-9 {
			t.Fatalf("kernel not symmetric at %d: %v vs %v", i, kernel[i], kernel[len(kernel)-1-i])
		}
	}

	if math.Abs(sum-1) > 1e-9 {
		t.Fatalf("kernel should sum to 1, got %v", sum)
	}

	if got := Gaussian1D(0); len(got) != 1 || got[0] != 1 {
		t.Fatalf("zero-radius kernel mismatch: %v", got)
	}
}

func TestPixBufDegenerateAllocationAndClippedRectOps(t *testing.T) {
	empty := NewPixBuf(0, 0)
	if empty.Width != 0 || empty.Height != 0 || len(empty.Data) != 0 {
		t.Fatalf("unexpected degenerate pixbuf: %#v", empty)
	}

	buf := NewPixBuf(4, 4)
	buf.FillRect(3, 3, 1, 1, color.RGBA{R: 20, G: 40, B: 60, A: 255})

	if got := buf.At(2, 2); got.A == 0 {
		t.Fatalf("expected clipped fill to cover interior, got %+v", got)
	}

	if got := buf.At(-1, 0); got != (color.RGBA{}) {
		t.Fatalf("expected out-of-bounds read to return zero, got %+v", got)
	}

	buf.Set(-1, 0, color.RGBA{R: 255, A: 255})

	if got := buf.At(0, 0); got.A != 0 {
		t.Fatalf("expected out-of-bounds write to be ignored, got %+v", got)
	}
}

func TestBlurHAndBlurVSpreadCoverage(t *testing.T) {
	src := NewPixBuf(5, 5)
	src.Set(2, 2, color.RGBA{R: 200, G: 50, B: 25, A: 255})

	kernel := Gaussian1D(1)
	blurred := BlurV(BlurH(src, kernel), kernel)

	center := blurred.At(2, 2)
	if center.A == 0 || center.A >= 255 {
		t.Fatalf("expected blurred center alpha in (0,255), got %+v", center)
	}

	if got := blurred.At(1, 2); got.A == 0 || got.R == 0 {
		t.Fatalf("expected horizontal blur spill, got %+v", got)
	}

	if got := blurred.At(2, 1); got.A == 0 || got.R == 0 {
		t.Fatalf("expected vertical blur spill, got %+v", got)
	}

	if got := blurred.At(0, 0); got.A != 0 {
		t.Fatalf("expected far corner to remain transparent, got %+v", got)
	}
}

func TestMakeHighlightUsesWhiteShadowTint(t *testing.T) {
	src := NewPixBuf(6, 6)
	src.Set(1, 3, color.RGBA{A: 255})

	highlight := MakeHighlight(src, 2, 100, 0, 0)

	got := highlight.At(3, 3)
	if got.A == 0 {
		t.Fatalf("expected shifted highlight pixel, got %+v", got)
	}

	if got.R != 255 || got.G != 255 || got.B != 255 {
		t.Fatalf("expected white highlight tint, got %+v", got)
	}
}

func TestTextureSetAddGetAndDecodeTextureInvalid(t *testing.T) {
	set := NewTextureSet()
	tex := &Texture{Data: []uint8{1, 2, 3, 4}, W: 1, H: 1}

	if idx := set.Add(tex); idx != 1 {
		t.Fatalf("unexpected add index: %d", idx)
	}

	if got := set.Get(1); got != tex {
		t.Fatalf("unexpected texture lookup: got %p want %p", got, tex)
	}

	if got := set.Get(2); got != nil {
		t.Fatalf("expected nil out-of-range texture, got %p", got)
	}

	if idx := set.Add(nil); idx != 0 {
		t.Fatalf("expected nil texture add to return 0, got %d", idx)
	}

	var nilSet *TextureSet
	if idx := nilSet.Add(tex); idx != 0 {
		t.Fatalf("expected nil set add to return 0, got %d", idx)
	}

	if got := nilSet.Get(1); got != nil {
		t.Fatalf("expected nil set lookup to return nil, got %p", got)
	}

	tex, err := DecodeTexture([]byte("not an image"))
	if err == nil || tex != nil {
		t.Fatalf("expected invalid texture decode to fail, got tex=%v err=%v", tex, err)
	}
}

func TestExtractFrameAutoDetectAndClonePaths(t *testing.T) {
	wide := NewPixBuf(6, 2)

	for y := range 2 {
		for x := range 2 {
			wide.Set(x, y, color.RGBA{R: 255, A: 255})
			wide.Set(x+2, y, color.RGBA{G: 255, A: 255})
			wide.Set(x+4, y, color.RGBA{B: 255, A: 255})
		}
	}

	if got := ExtractFrame(wide, 3, 1, 3).At(1, 1); got.G != 255 {
		t.Fatalf("auto-detected horizontal frame mismatch: %+v", got)
	}

	tall := NewPixBuf(2, 6)

	for y := range 2 {
		for x := range 2 {
			tall.Set(x, y, color.RGBA{R: 255, A: 255})
			tall.Set(x, y+2, color.RGBA{G: 255, A: 255})
			tall.Set(x, y+4, color.RGBA{B: 255, A: 255})
		}
	}

	if got := ExtractFrame(tall, 3, 2, 3).At(1, 1); got.B != 255 {
		t.Fatalf("auto-detected vertical frame mismatch: %+v", got)
	}

	if got := frameIndex(-1, 5, 3); got != 0 {
		t.Fatalf("negative frame should clamp to first index, got %d", got)
	}

	if got := frameIndex(9, 5, 3); got != 2 {
		t.Fatalf("overflow frame should clamp to last index, got %d", got)
	}

	clone := ExtractFrameAligned(wide, 3, 1, 3, 2)
	clone.Set(0, 0, color.RGBA{})

	if got := wide.At(0, 0); got.A == 0 {
		t.Fatalf("align=2 should clone, original was mutated: %+v", got)
	}

	single := ExtractFrameAligned(wide, 1, 0, 1, 1)
	single.Set(1, 1, color.RGBA{})

	if got := wide.At(1, 1); got.A == 0 {
		t.Fatalf("numFrames<=1 should clone, original was mutated: %+v", got)
	}
}

func TestWrapIntHandlesNegativeAndDegenerateRanges(t *testing.T) {
	if got := wrapInt(-1, 4); got != 3 {
		t.Fatalf("unexpected negative wrap result: %d", got)
	}

	if got := wrapInt(5, 4); got != 1 {
		t.Fatalf("unexpected positive wrap result: %d", got)
	}

	if got := wrapInt(3, 0); got != 0 {
		t.Fatalf("unexpected zero-range wrap result: %d", got)
	}
}

func TestSphereNormalAndPhongLighting(t *testing.T) {
	nx, ny, nz, ok := SphereNormal(5, 5, 5, 5, 4, 4)
	if !ok {
		t.Fatal("expected center point to lie on the sphere")
	}

	if math.Abs(nx) > 1e-9 || math.Abs(ny) > 1e-9 || math.Abs(nz-1) > 1e-9 {
		t.Fatalf("unexpected center normal: (%v,%v,%v)", nx, ny, nz)
	}

	if _, _, _, ok := SphereNormal(10, 10, 5, 5, 4, 4); ok {
		t.Fatal("expected out-of-bounds point to be rejected")
	}

	base := color.RGBA{R: 100, G: 120, B: 140, A: 200}
	lit := PhongLighting([3]float64{0, 0, 1}, 0, 10, 100, 40, 20, base)
	dim := PhongLighting([3]float64{0, 0, -1}, 0, 10, 100, 40, 20, base)

	if lit.R <= dim.R || lit.G <= dim.G || lit.B <= dim.B {
		t.Fatalf("expected front-facing lighting to be brighter: lit=%+v dim=%+v", lit, dim)
	}

	if lit.A != base.A || dim.A != base.A {
		t.Fatalf("lighting should preserve alpha: lit=%+v dim=%+v", lit, dim)
	}
}

func TestRenderAllUsesFallbackDocumentSize(t *testing.T) {
	doc := model.NewDocument()
	doc.Prefs.Width = 10
	doc.Prefs.Height = 12
	doc.Prefs.PWidth.Val = 0
	doc.Prefs.PHeight.Val = 0
	doc.Prefs.RenderFrames.Val = 3
	doc.Prefs.BkColor.Val = color.RGBA{}

	for i := range doc.Layers {
		doc.Layers[i].Visible.Val = 0
	}

	ly := &doc.Layers[0]
	ly.Visible.Val = 1
	ly.Prim.Type.Val = int(model.PrimRectFill)
	ly.Prim.Color.Val = color.RGBA{R: 10, G: 20, B: 30, A: 200}
	ly.Prim.Aspect.Val = 0

	frames := RenderAll(doc, nil)
	if len(frames) != 3 {
		t.Fatalf("unexpected frame count: %d", len(frames))
	}

	for i, frame := range frames {
		if frame == nil || frame.Width != 10 || frame.Height != 12 {
			t.Fatalf("frame %d has unexpected size: %#v", i, frame)
		}

		if got := frame.At(5, 6); got.A == 0 {
			t.Fatalf("frame %d center should be visible, got %+v", i, got)
		}
	}
}

func TestRenderFrameOversamplingPreservesTransparentBackground(t *testing.T) {
	doc := model.NewDocument()
	doc.Prefs.PWidth.Val = 8
	doc.Prefs.PHeight.Val = 8
	doc.Prefs.RenderFrames.Val = 1
	doc.Prefs.Oversampling.Val = 1
	doc.Prefs.BkColor.Val = color.RGBA{}

	for i := range doc.Layers {
		doc.Layers[i].Visible.Val = 0
	}

	ly := &doc.Layers[0]
	ly.Visible.Val = 1
	ly.Prim.Type.Val = int(model.PrimRectFill)
	ly.Prim.Color.Val = color.RGBA{R: 200, G: 60, B: 30, A: 128}
	ly.Prim.Aspect.Val = 0

	dst := NewPixBuf(4, 4)
	RenderFrame(dst, doc, 0, nil)

	got := dst.At(2, 2)
	if got != (color.RGBA{R: 200, G: 60, B: 30, A: 255}) {
		t.Fatalf("unexpected oversampled output pixel: %+v", got)
	}
}

func TestRenderMetalCircleAndSphereProduceVisibleInterior(t *testing.T) {
	metal := basePrim(model.PrimMetalCircle)
	metal.Color.Val = color.RGBA{R: 180, G: 180, B: 200, A: 255}

	metalBuf := NewPixBuf(32, 32)
	renderMetalCircle(metalBuf, &metal, nil)

	if got := metalBuf.At(16, 16); got.A == 0 {
		t.Fatalf("expected metal circle center visible, got %+v", got)
	}

	if got := metalBuf.At(0, 0); got.A != 0 {
		t.Fatalf("expected metal circle corner transparent, got %+v", got)
	}

	if left, right := metalBuf.At(8, 16), metalBuf.At(24, 16); left == right {
		t.Fatalf("expected metal circle shading variation, got left=%+v right=%+v", left, right)
	}

	sphere := basePrim(model.PrimSphere)
	sphere.Color.Val = color.RGBA{R: 90, G: 120, B: 220, A: 255}
	sphere.Ambient.Val = 20
	sphere.Diffuse.Val = 40
	sphere.Specular.Val = 50
	sphere.SpecularWidth.Val = 30
	sphere.LightDir.Val = 0

	sphereBuf := NewPixBuf(32, 32)
	renderSphere(sphereBuf, &sphere, nil)

	if got := sphereBuf.At(16, 16); got.A == 0 {
		t.Fatalf("expected sphere center visible, got %+v", got)
	}

	if got := sphereBuf.At(0, 0); got.A != 0 {
		t.Fatalf("expected sphere corner transparent, got %+v", got)
	}

	if left, right := sphereBuf.At(8, 16), sphereBuf.At(24, 16); left == right {
		t.Fatalf("expected sphere lighting variation, got left=%+v right=%+v", left, right)
	}
}
