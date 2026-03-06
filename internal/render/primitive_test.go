package render

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"knobman/internal/model"
)

func TestRenderPrimitiveCoverage(t *testing.T) {
	textures := []*Texture{checkerTexture()}

	cases := []struct {
		name string
		prim func() model.Primitive
		want bool
	}{
		{"none", func() model.Primitive { p := model.NewPrimitive(); p.Type.Val = int(model.PrimNone); return p }, false},
		{"image", func() model.Primitive { p := basePrim(model.PrimImage); p.EmbeddedImage = tinyPNG(); return p }, true},
		{"circle", func() model.Primitive { p := basePrim(model.PrimCircle); return p }, true},
		{"circlefill", func() model.Primitive { p := basePrim(model.PrimCircleFill); return p }, true},
		{"metalcircle", func() model.Primitive { p := basePrim(model.PrimMetalCircle); return p }, true},
		{"wavecircle", func() model.Primitive {
			p := basePrim(model.PrimWaveCircle)
			p.Step.Val = 30
			p.Length.Val = 40
			return p
		}, true},
		{"sphere", func() model.Primitive {
			p := basePrim(model.PrimSphere)
			p.TextureFile.Val = 1
			p.TextureDepth.Val = 35
			return p
		}, true},
		{"rect", func() model.Primitive { p := basePrim(model.PrimRect); p.Length.Val = 80; p.Aspect.Val = 65; return p }, true},
		{"rectfill", func() model.Primitive {
			p := basePrim(model.PrimRectFill)
			p.Length.Val = 80
			p.Aspect.Val = 65
			return p
		}, true},
		{"triangle", func() model.Primitive { p := basePrim(model.PrimTriangle); p.Length.Val = 80; p.Fill.Val = 1; return p }, true},
		{"line", func() model.Primitive {
			p := basePrim(model.PrimLine)
			p.Width.Val = 12
			p.Length.Val = 90
			p.LightDir.Val = 30
			return p
		}, true},
		{"radiateline", func() model.Primitive {
			p := basePrim(model.PrimRadiateLine)
			p.Width.Val = 6
			p.Length.Val = 45
			p.AngleStep.Val = 30
			return p
		}, true},
		{"hlines", func() model.Primitive { p := basePrim(model.PrimHLines); p.Step.Val = 20; p.Width.Val = 8; return p }, true},
		{"vlines", func() model.Primitive { p := basePrim(model.PrimVLines); p.Step.Val = 20; p.Width.Val = 8; return p }, true},
		{"text", func() model.Primitive { p := basePrim(model.PrimText); p.Text.Val = "AB"; return p }, true},
		{"shape", func() model.Primitive {
			p := basePrim(model.PrimShape)
			p.Shape.Val = "M 10 10 L 90 10 L 50 90"
			p.Fill.Val = 1
			return p
		}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf := NewPixBuf(64, 64)
			p := tc.prim()
			RenderPrimitive(buf, &p, textures, 0, 31)
			opaque := countOpaque(buf)
			if tc.want && opaque == 0 {
				t.Fatalf("expected painted output")
			}
			if !tc.want && opaque != 0 {
				t.Fatalf("expected empty output, got %d pixels", opaque)
			}
		})
	}
}

func TestSphereNormalOutside(t *testing.T) {
	if _, _, _, ok := SphereNormal(100, 100, 0, 0, 10, 10); ok {
		t.Fatal("expected outside point to be rejected")
	}
}

func TestTextureBlendDepth(t *testing.T) {
	base := color.RGBA{10, 20, 30, 255}
	tex := color.RGBA{200, 210, 220, 255}
	full := TextureBlend(base, tex, 100)
	if full != tex {
		t.Fatalf("depth 100 should pick texture, got %+v", full)
	}
}

func TestRenderImageFrameAlignHorizontal(t *testing.T) {
	// 2 frames side-by-side: frame0 red, frame1 green.
	img := image.NewRGBA(image.Rect(0, 0, 4, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			img.SetRGBA(x, y, color.RGBA{255, 0, 0, 255})
			img.SetRGBA(x+2, y, color.RGBA{0, 255, 0, 255})
		}
	}
	var enc bytes.Buffer
	_ = png.Encode(&enc, img)

	p := basePrim(model.PrimImage)
	p.EmbeddedImage = enc.Bytes()
	p.NumFrame.Val = 2
	p.FrameAlign.Val = 1 // horizontal strip
	p.AutoFit.Val = 1
	p.Transparent.Val = 1 // force opaque

	buf0 := NewPixBuf(8, 8)
	RenderPrimitive(buf0, &p, nil, 0, 2)
	c0 := buf0.At(4, 4)
	if c0.R < 200 || c0.G > 30 {
		t.Fatalf("frame0 expected red-ish center, got %+v", c0)
	}

	buf1 := NewPixBuf(8, 8)
	RenderPrimitive(buf1, &p, nil, 1, 2)
	c1 := buf1.At(4, 4)
	if c1.G < 200 || c1.R > 30 {
		t.Fatalf("frame1 expected green-ish center, got %+v", c1)
	}
}

func TestRenderImageTransparentKeyColor(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.SetRGBA(0, 0, color.RGBA{10, 20, 30, 255})  // key color (first pixel)
	img.SetRGBA(1, 0, color.RGBA{200, 10, 10, 255}) // visible pixel
	var enc bytes.Buffer
	_ = png.Encode(&enc, img)

	p := basePrim(model.PrimImage)
	p.EmbeddedImage = enc.Bytes()
	p.AutoFit.Val = 0
	p.Transparent.Val = 2 // key on first pixel
	p.IntelliAlpha.Val = 0

	buf := NewPixBuf(4, 2)
	RenderPrimitive(buf, &p, nil, 0, 1)
	if got := buf.At(0, 0); got.A != 0 {
		t.Fatalf("expected key pixel transparent, got %+v", got)
	}
	if got := buf.At(1, 0); got.A == 0 {
		t.Fatalf("expected non-key pixel visible, got %+v", got)
	}
}

func TestSubstituteFrameCounters(t *testing.T) {
	got := SubstituteFrameCounters("F(1:9)", 4, 9)
	// 4/8 => midpoint => 5
	if got != "F5" {
		t.Fatalf("unexpected substitution: %q", got)
	}
	got2 := SubstituteFrameCounters("N(01:99)", 0, 10)
	if got2 != "N01" {
		t.Fatalf("unexpected zero-padded substitution: %q", got2)
	}
}

func TestParseKnobShapePolylines(t *testing.T) {
	s := "/192,256,192,256,192,256:320,256,320,256,320,256"
	polys := parseKnobShapePolylines(s, 64, 64)
	if len(polys) == 0 {
		t.Fatal("expected parsed polylines")
	}
	if len(polys[0]) < 2 {
		t.Fatalf("expected polyline points, got %d", len(polys[0]))
	}
}

func TestParseSimpleShapePointsSVGCurves(t *testing.T) {
	s := "M 10 10 C 20 80 80 20 90 90 Q 60 60 20 90 Z"
	pts := parseSimpleShapePoints(s, 64, 64)
	if len(pts) < 10 {
		t.Fatalf("expected flattened curve points, got %d", len(pts))
	}
}

func basePrim(t model.PrimitiveType) model.Primitive {
	p := model.NewPrimitive()
	p.Type.Val = int(t)
	p.Color.Val = color.RGBA{R: 220, G: 140, B: 80, A: 255}
	return p
}

func checkerTexture() *Texture {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if (x+y)%2 == 0 {
				img.SetRGBA(x, y, color.RGBA{255, 255, 255, 255})
			} else {
				img.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
			}
		}
	}
	return NewTextureFromImage(img)
}

func tinyPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.SetRGBA(0, 0, color.RGBA{255, 0, 0, 255})
	img.SetRGBA(1, 0, color.RGBA{0, 255, 0, 255})
	img.SetRGBA(0, 1, color.RGBA{0, 0, 255, 255})
	img.SetRGBA(1, 1, color.RGBA{255, 255, 255, 255})
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func countOpaque(b *PixBuf) int {
	c := 0
	for i := 3; i < len(b.Data); i += 4 {
		if b.Data[i] != 0 {
			c++
		}
	}
	return c
}
