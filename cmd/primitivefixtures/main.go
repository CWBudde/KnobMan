package main

import (
	"bytes"
	"errors"
	"flag"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"knobman/internal/fileio"
	"knobman/internal/model"
)

func main() {
	err := runPrimitiveFixtures(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
}

func runPrimitiveFixtures(args []string) error {
	fs := flag.NewFlagSet("primitivefixtures", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	suite := fs.String("suite", "primitives", "Fixture suite to generate: primitives or animated")
	outDir := fs.String("out", "", "Directory to write fixture .knob files (defaults to the suite's input directory)")
	overwrite := fs.Bool("overwrite", false, "Overwrite existing fixture files")

	if err := fs.Parse(args); err != nil {
		return err
	}

	fixtures, defaultOutDir, err := fixtureSet(*suite)
	if err != nil {
		return err
	}
	if *outDir == "" {
		*outDir = defaultOutDir
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		return errFixtureMkdir(*outDir, err)
	}

	for _, fixture := range fixtures {
		path := filepath.Join(*outDir, fixture.Name+".knob")
		if !*overwrite {
			_, err = os.Stat(path)
			if err == nil {
				continue
			}
		}

		doc := fixture.Build()

		data, err := fileio.Save(doc)
		if err != nil {
			return errFixtureSave(fixture.Name, err)
		}

		if err := os.WriteFile(path, data, 0o600); err != nil {
			return errFixtureWrite(path, err)
		}

		log.Println(path)
	}

	return nil
}

func fixtureSet(name string) ([]fixtureDef, string, error) {
	switch name {
	case "primitives":
		return primitiveFixtures(), filepath.Join("tests", "parity", "primitives", "inputs"), nil
	case "animated":
		return animatedFixtures(), filepath.Join("tests", "parity", "animated", "inputs"), nil
	default:
		return nil, "", errors.New("unknown fixture suite " + `"` + name + `"`)
	}
}

func errFixtureMkdir(path string, err error) error {
	return errors.New("mkdir " + path + ": " + err.Error())
}

func errFixtureSave(name string, err error) error {
	return errors.New("save " + name + ": " + err.Error())
}

func errFixtureWrite(path string, err error) error {
	return errors.New("write " + path + ": " + err.Error())
}

type fixtureDef struct {
	Name  string
	Build func() *model.Document
}

const (
	shapeTriangleLoop  = "/128,24,128,24,128,24:232,128,232,128,232,128:128,232,128,232,128,232:24,128,24,128,24,128"
	embeddedCheckerPNG = "embedded-checker.png"
	embeddedSeamPNG    = "embedded-seam.png"
)

func primitiveFixtures() []fixtureDef {
	return []fixtureDef{
		{Name: "circle_outline_basic", Build: func() *model.Document {
			doc := newPrimitiveDoc("Circle")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimCircle)
			ly.Prim.Color.Val = rgb(192, 32, 32)
			ly.Prim.Width.Val = 14
			ly.Prim.Aspect.Val = 10

			return doc
		}},
		{Name: "circle_outline_shell", Build: func() *model.Document {
			doc := newPrimitiveDoc("Tier3CircleOutlineShell")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimCircle)
			ly.Prim.Color.Val = rgb(192, 32, 32)
			ly.Prim.Width.Val = 14
			ly.Prim.Aspect.Val = 10

			return doc
		}},
		{Name: "circle_fill_basic", Build: func() *model.Document {
			doc := newPrimitiveDoc("CircleFill")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimCircleFill)
			ly.Prim.Color.Val = rgb(48, 96, 208)

			return doc
		}},
		{Name: "circle_fill_greenab_ring", Build: func() *model.Document {
			doc := newPrimitiveDoc("CircleFillGreenAbRing")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimCircleFill)
			ly.Prim.Color.Val = rgb(255, 0, 0)
			ly.Prim.Emboss.Val = -12
			ly.Prim.Specular.Val = 50
			ly.Eff.BrightF.Val = -26
			ly.Eff.BrightT.Val = -26
			ly.Eff.ContrastF.Val = -55
			ly.Eff.ContrastT.Val = -55
			ly.Eff.SaturationF.Val = -69
			ly.Eff.SaturationT.Val = -69
			ly.Eff.HueF.Val = 141
			ly.Eff.HueT.Val = 141

			return doc
		}},
		{Name: "circle_fill_shell", Build: func() *model.Document {
			doc := newPrimitiveDoc("Tier3CircleFillShell")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimCircleFill)
			ly.Prim.Color.Val = rgb(48, 96, 208)

			return doc
		}},
		{Name: "circle_fill_lit", Build: func() *model.Document {
			doc := newPrimitiveDoc("Tier3CircleFillLit")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimCircleFill)
			ly.Prim.Color.Val = rgb(48, 96, 208)
			ly.Prim.Specular.Val = 40
			ly.Prim.Diffuse.Val = 25
			ly.Prim.Emboss.Val = 20

			return doc
		}},
		{Name: "circle_fill_texture", Build: func() *model.Document {
			doc := newPrimitiveDoc("Tier3CircleFillTexture")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimCircleFill)
			ly.Prim.Color.Val = rgb(48, 96, 208)
			ly.Prim.TextureDepth.Val = 35
			ly.Prim.EmbeddedTexture = checkerTexturePNG()

			return doc
		}},
		{Name: "metal_circle_basic", Build: func() *model.Document {
			doc := newPrimitiveDoc("MetalCircle")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimMetalCircle)
			ly.Prim.Color.Val = rgb(100, 140, 180)
			ly.Prim.Ambient.Val = 40
			ly.Prim.Specular.Val = 32
			ly.Prim.SpecularWidth.Val = 35

			return doc
		}},
		{Name: "wave_circle_basic", Build: func() *model.Document {
			doc := newPrimitiveDoc("WaveCircle")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimWaveCircle)
			ly.Prim.Color.Val = rgb(24, 160, 180)
			ly.Prim.Width.Val = 12
			ly.Prim.Length.Val = 34
			ly.Prim.Step.Val = 9

			return doc
		}},
		{Name: "sphere_basic", Build: func() *model.Document {
			doc := newPrimitiveDoc("Sphere")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimSphere)
			ly.Prim.Color.Val = rgb(222, 132, 48)
			ly.Prim.Ambient.Val = 38
			ly.Prim.Diffuse.Val = 18
			ly.Prim.Specular.Val = 28
			ly.Prim.SpecularWidth.Val = 42
			ly.Prim.LightDir.Val = -30

			return doc
		}},
		{Name: "rect_outline_basic", Build: func() *model.Document {
			doc := newPrimitiveDoc("Rect")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimRect)
			ly.Prim.Color.Val = rgb(24, 128, 72)
			ly.Prim.Width.Val = 12
			ly.Prim.Length.Val = 76

			return doc
		}},
		{Name: "rect_fill_basic", Build: func() *model.Document {
			doc := newPrimitiveDoc("RectFill")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimRectFill)
			ly.Prim.Color.Val = rgb(72, 172, 112)
			ly.Prim.Length.Val = 72

			return doc
		}},
		{Name: "rect_fill_plain", Build: func() *model.Document {
			doc := newPrimitiveDoc("Tier1RectFillPlain")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimRectFill)
			ly.Prim.Color.Val = rgb(72, 172, 112)

			return doc
		}},
		{Name: "rect_fill_aspect", Build: func() *model.Document {
			doc := newPrimitiveDoc("Tier1RectFillAspect")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimRectFill)
			ly.Prim.Color.Val = rgb(72, 172, 112)
			ly.Prim.Aspect.Val = 50

			return doc
		}},
		{Name: "rect_outline_plain", Build: func() *model.Document {
			doc := newPrimitiveDoc("Tier1RectOutlinePlain")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimRect)
			ly.Prim.Color.Val = rgb(24, 128, 72)
			ly.Prim.Width.Val = 12

			return doc
		}},
		{Name: "triangle_plain", Build: func() *model.Document {
			doc := newPrimitiveDoc("Tier1TrianglePlain")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimTriangle)
			ly.Prim.Color.Val = rgb(220, 96, 40)
			ly.Prim.Width.Val = 62
			ly.Prim.Length.Val = 74

			return doc
		}},
		{Name: "shape_fill_plain", Build: func() *model.Document {
			doc := newPrimitiveDoc("Tier0ShapeFillPlain")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimShape)
			ly.Prim.Color.Val = rgb(56, 144, 88)
			ly.Prim.Fill.Val = 1
			ly.Prim.Shape.Val = shapeTriangleLoop

			return doc
		}},
		{Name: "shape_outline_plain", Build: func() *model.Document {
			doc := newPrimitiveDoc("Tier0ShapeOutlinePlain")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimShape)
			ly.Prim.Color.Val = rgb(56, 144, 88)
			ly.Prim.Fill.Val = 0
			ly.Prim.Shape.Val = shapeTriangleLoop

			return doc
		}},
		{Name: "line_basic", Build: func() *model.Document {
			doc := newPrimitiveDoc("Line")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimLine)
			ly.Prim.Color.Val = rgb(32, 32, 32)
			ly.Prim.Width.Val = 12
			ly.Prim.Length.Val = 82
			ly.Prim.LightDir.Val = 30

			return doc
		}},
		{Name: "line_plain", Build: func() *model.Document {
			doc := newPrimitiveDoc("Tier2LinePlain")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimLine)
			ly.Prim.Color.Val = rgb(32, 32, 32)
			ly.Prim.Width.Val = 40
			ly.Prim.Length.Val = 90

			return doc
		}},
		{Name: "radiate_line_basic", Build: func() *model.Document {
			doc := newPrimitiveDoc("RadiateLine")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimRadiateLine)
			ly.Prim.Color.Val = rgb(40, 92, 160)
			ly.Prim.Width.Val = 6
			ly.Prim.Length.Val = 34
			ly.Prim.AngleStep.Val = 30

			return doc
		}},
		{Name: "radiate_line_plain", Build: func() *model.Document {
			doc := newPrimitiveDoc("Tier2RadiateLinePlain")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimRadiateLine)
			ly.Prim.Color.Val = rgb(40, 92, 160)
			ly.Prim.Width.Val = 20
			ly.Prim.Length.Val = 90
			ly.Prim.AngleStep.Val = 90

			return doc
		}},
		{Name: "hlines_basic", Build: func() *model.Document {
			doc := newPrimitiveDoc("HLines")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimHLines)
			ly.Prim.Color.Val = rgb(60, 60, 60)
			ly.Prim.Width.Val = 8
			ly.Prim.Step.Val = 18

			return doc
		}},
		{Name: "hlines_plain", Build: func() *model.Document {
			doc := newPrimitiveDoc("Tier2HLinesPlain")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimHLines)
			ly.Prim.Color.Val = rgb(60, 60, 60)
			ly.Prim.Width.Val = 8
			ly.Prim.Length.Val = 50
			ly.Prim.Step.Val = 50

			return doc
		}},
		{Name: "vlines_basic", Build: func() *model.Document {
			doc := newPrimitiveDoc("VLines")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimVLines)
			ly.Prim.Color.Val = rgb(60, 60, 60)
			ly.Prim.Width.Val = 8
			ly.Prim.Step.Val = 18

			return doc
		}},
		{Name: "vlines_plain", Build: func() *model.Document {
			doc := newPrimitiveDoc("Tier2VLinesPlain")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimVLines)
			ly.Prim.Color.Val = rgb(60, 60, 60)
			ly.Prim.Width.Val = 8
			ly.Prim.Length.Val = 50
			ly.Prim.Step.Val = 50

			return doc
		}},
		{Name: "shape_fill_basic", Build: func() *model.Document {
			doc := newPrimitiveDoc("ShapeFill")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimShape)
			ly.Prim.Color.Val = rgb(224, 80, 48)
			ly.Prim.Fill.Val = 1
			ly.Prim.Shape.Val = shapeTriangleLoop

			return doc
		}},
		{Name: "shape_outline_basic", Build: func() *model.Document {
			doc := newPrimitiveDoc("ShapeOutline")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimShape)
			ly.Prim.Color.Val = rgb(192, 40, 40)
			ly.Prim.Fill.Val = 0
			ly.Prim.Width.Val = 8
			ly.Prim.Shape.Val = "/128,24,128,24,128,24:232,224,232,224,232,224:24,224,24,224,24,224"

			return doc
		}},
		{Name: "text_basic_center", Build: func() *model.Document {
			doc := newPrimitiveDoc("TextCenter")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimText)
			ly.Prim.Color.Val = rgb(28, 28, 28)
			ly.Prim.FontSize.Val = 62
			ly.Prim.TextAlign.Val = 1
			ly.Prim.Text.Val = "TX"

			return doc
		}},
		{Name: "rect_fill_texture_basic", Build: func() *model.Document {
			doc := newPrimitiveDoc("RectFillTexture")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimRectFill)
			ly.Prim.Color.Val = rgb(96, 120, 168)
			ly.Prim.TextureName = embeddedCheckerPNG
			ly.Prim.EmbeddedTexture = checkerTexturePNG()
			ly.Prim.TextureDepth.Val = 70
			ly.Prim.TextureZoom.Val = 100

			return doc
		}},
		{Name: "texture_wrap_rect_fill", Build: func() *model.Document {
			doc := newPrimitiveDoc("TextureWrapRectFill")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimRectFill)
			ly.Prim.Color.Val = rgb(112, 112, 112)
			ly.Prim.TextureName = embeddedSeamPNG
			ly.Prim.EmbeddedTexture = seamTexturePNG()
			ly.Prim.TextureDepth.Val = 80
			ly.Prim.TextureZoom.Val = 100

			return doc
		}},
		{Name: "texture_zoom_in_rect_fill", Build: func() *model.Document {
			doc := newPrimitiveDoc("TextureZoomInRectFill")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimRectFill)
			ly.Prim.Color.Val = rgb(112, 112, 112)
			ly.Prim.TextureName = embeddedSeamPNG
			ly.Prim.EmbeddedTexture = seamTexturePNG()
			ly.Prim.TextureDepth.Val = 80
			ly.Prim.TextureZoom.Val = 220

			return doc
		}},
		{Name: "texture_zoom_out_rect_fill", Build: func() *model.Document {
			doc := newPrimitiveDoc("TextureZoomOutRectFill")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimRectFill)
			ly.Prim.Color.Val = rgb(112, 112, 112)
			ly.Prim.TextureName = embeddedSeamPNG
			ly.Prim.EmbeddedTexture = seamTexturePNG()
			ly.Prim.TextureDepth.Val = 80
			ly.Prim.TextureZoom.Val = 40

			return doc
		}},
		{Name: "texture_tiling_seam_circle_fill", Build: func() *model.Document {
			doc := newPrimitiveDoc("TextureTilingSeamCircleFill")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimCircleFill)
			ly.Prim.Color.Val = rgb(96, 120, 168)
			ly.Prim.TextureName = embeddedSeamPNG
			ly.Prim.EmbeddedTexture = seamTexturePNG()
			ly.Prim.TextureDepth.Val = 70
			ly.Prim.TextureZoom.Val = 100

			return doc
		}},
		{Name: "circle_fill_texture_basic", Build: func() *model.Document {
			doc := newPrimitiveDoc("CircleFillTexture")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimCircleFill)
			ly.Prim.Color.Val = rgb(80, 150, 96)
			ly.Prim.TextureName = embeddedCheckerPNG
			ly.Prim.EmbeddedTexture = checkerTexturePNG()
			ly.Prim.TextureDepth.Val = 65
			ly.Prim.TextureZoom.Val = 140
			ly.Prim.Specular.Val = 18

			return doc
		}},
	}
}

func animatedFixtures() []fixtureDef {
	return []fixtureDef{
		{Name: "anim_primitive_image_strip", Build: func() *model.Document {
			doc := newAnimatedDoc("AnimPrimitiveImageStrip", 16, 16, 5)
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimImage)
			ly.Prim.AutoFit.Val = 0
			ly.Prim.NumFrame.Val = 3
			ly.Prim.FrameAlign.Val = 1
			ly.Prim.EmbeddedImage = colorStripPNG(16, 16,
				color.NRGBA{R: 224, G: 48, B: 48, A: 255},
				color.NRGBA{R: 48, G: 200, B: 72, A: 255},
				color.NRGBA{R: 48, G: 96, B: 224, A: 255},
			)

			return doc
		}},
		{Name: "anim_effect_offset_image", Build: func() *model.Document {
			doc := newAnimatedDoc("AnimEffectOffsetImage", 32, 16, 5)
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimRectFill)
			ly.Prim.Color.Val = rgb(255, 180, 32)
			ly.Prim.Length.Val = 50
			ly.Prim.Aspect.Val = 0
			ly.Eff.OffXF.Val = -25
			ly.Eff.OffXT.Val = 25
			ly.Eff.OffXAnim.Val = 1

			return doc
		}},
		{Name: "anim_layer_animstep_strip", Build: func() *model.Document {
			doc := newAnimatedDoc("AnimLayerAnimStepStrip", 16, 16, 7)
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimImage)
			ly.Prim.AutoFit.Val = 0
			ly.Prim.NumFrame.Val = 4
			ly.Prim.FrameAlign.Val = 1
			ly.Prim.EmbeddedImage = colorStripPNG(16, 16,
				color.NRGBA{R: 220, G: 48, B: 48, A: 255},
				color.NRGBA{R: 220, G: 180, B: 48, A: 255},
				color.NRGBA{R: 48, G: 180, B: 220, A: 255},
				color.NRGBA{R: 96, G: 48, B: 220, A: 255},
			)
			ly.Eff.AnimStep.Val = 4

			return doc
		}},
		{Name: "anim_frame_mask_ranges", Build: func() *model.Document {
			doc := newAnimatedDoc("AnimFrameMaskRanges", 16, 16, 5)

			doc.Layers = []model.Layer{model.NewLayer(), model.NewLayer(), model.NewLayer()}
			for i := range doc.Layers {
				doc.Layers[i].Name = doc.Layers[0].Name
				doc.Layers[i].Visible.Val = 1
				doc.Layers[i].Prim.Type.Val = int(model.PrimImage)
				doc.Layers[i].Prim.AutoFit.Val = 0
			}

			doc.Layers[0].Name = "FirstFrameRed"
			doc.Layers[0].Prim.EmbeddedImage = solidColorPNG(16, 16, color.NRGBA{R: 220, G: 48, B: 48, A: 255})
			doc.Layers[0].Eff.FMaskEna.Val = 1
			doc.Layers[0].Eff.FMaskStart.Val = 0
			doc.Layers[0].Eff.FMaskStop.Val = 20

			doc.Layers[1].Name = "MidFrameGreen"
			doc.Layers[1].Prim.EmbeddedImage = solidColorPNG(16, 16, color.NRGBA{R: 48, G: 200, B: 72, A: 255})
			doc.Layers[1].Eff.FMaskEna.Val = 1
			doc.Layers[1].Eff.FMaskStart.Val = 40
			doc.Layers[1].Eff.FMaskStop.Val = 60

			doc.Layers[2].Name = "LastFrameBlue"
			doc.Layers[2].Prim.EmbeddedImage = solidColorPNG(16, 16, color.NRGBA{R: 48, G: 96, B: 224, A: 255})
			doc.Layers[2].Eff.FMaskEna.Val = 1
			doc.Layers[2].Eff.FMaskStart.Val = 80
			doc.Layers[2].Eff.FMaskStop.Val = 100

			return doc
		}},
		{Name: "anim_effect_transform_mask", Build: func() *model.Document {
			doc := newAnimatedDoc("AnimEffectTransformMask", 96, 96, 5)
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimCircleFill)
			ly.Prim.Color.Val = rgb(58, 136, 214)
			ly.Prim.Specular.Val = 18
			ly.Eff.ZoomXYSepa.Val = 1
			ly.Eff.ZoomXF.Val = 82
			ly.Eff.ZoomXT.Val = 126
			ly.Eff.ZoomXAnim.Val = 1
			ly.Eff.ZoomYF.Val = 128
			ly.Eff.ZoomYT.Val = 72
			ly.Eff.ZoomYAnim.Val = 1
			ly.Eff.OffXF.Val = -20
			ly.Eff.OffXT.Val = 18
			ly.Eff.OffXAnim.Val = 1
			ly.Eff.OffYF.Val = 8
			ly.Eff.OffYT.Val = -10
			ly.Eff.OffYAnim.Val = 1
			ly.Eff.AngleF.Val = 0
			ly.Eff.AngleT.Val = 162
			ly.Eff.AngleAnim.Val = 1
			ly.Eff.CenterX.Val = 10
			ly.Eff.CenterY.Val = -6
			ly.Eff.Mask1Ena.Val = 1
			ly.Eff.Mask1Type.Val = 2
			ly.Eff.Mask1StartF.Val = -120
			ly.Eff.Mask1StartT.Val = -18
			ly.Eff.Mask1StartAnim.Val = 1
			ly.Eff.Mask1StopF.Val = 12
			ly.Eff.Mask1StopT.Val = 118
			ly.Eff.Mask1StopAnim.Val = 1
			ly.Eff.Mask1Grad.Val = 18

			return doc
		}},
		{Name: "anim_effect_shadow_color", Build: func() *model.Document {
			doc := newAnimatedDoc("AnimEffectShadowColor", 96, 96, 5)
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimRectFill)
			ly.Prim.Color.Val = rgb(120, 136, 164)
			ly.Prim.Length.Val = 80
			ly.Prim.Aspect.Val = 30
			ly.Prim.TextureName = embeddedSeamPNG
			ly.Prim.EmbeddedTexture = seamTexturePNG()
			ly.Prim.TextureDepth.Val = 54
			ly.Prim.TextureZoom.Val = 120
			ly.Eff.AlphaF.Val = 100
			ly.Eff.AlphaT.Val = 68
			ly.Eff.AlphaAnim.Val = 1
			ly.Eff.BrightF.Val = -18
			ly.Eff.BrightT.Val = 16
			ly.Eff.BrightAnim.Val = 1
			ly.Eff.SaturationF.Val = -44
			ly.Eff.SaturationT.Val = 22
			ly.Eff.SaturationAnim.Val = 1
			ly.Eff.HueF.Val = 0
			ly.Eff.HueT.Val = 138
			ly.Eff.HueAnim.Val = 1
			ly.Eff.DLightDirEna.Val = 1
			ly.Eff.DLightDirF.Val = 295
			ly.Eff.DOffsetF.Val = 4
			ly.Eff.DOffsetT.Val = 16
			ly.Eff.DOffsetAnim.Val = 1
			ly.Eff.DDensityF.Val = 0
			ly.Eff.DDensityT.Val = 72
			ly.Eff.DDensityAnim.Val = 1
			ly.Eff.DDiffuseF.Val = 0
			ly.Eff.DDiffuseT.Val = 42
			ly.Eff.DDiffuseAnim.Val = 1
			ly.Eff.DSType.Val = 1
			ly.Eff.DSGrad.Val = 72

			return doc
		}},
		{Name: "anim_effect_combo_stack", Build: func() *model.Document {
			doc := newAnimatedDoc("AnimEffectComboStack", 96, 96, 5)

			doc.Layers = []model.Layer{model.NewLayer(), model.NewLayer(), model.NewLayer()}
			for i := range doc.Layers {
				doc.Layers[i].Visible.Val = 1
			}

			background := &doc.Layers[0]
			background.Name = "Backdrop"
			background.Prim.Type.Val = int(model.PrimRectFill)
			background.Prim.Color.Val = rgb(34, 38, 46)
			background.Prim.Length.Val = 100
			background.Prim.Aspect.Val = 100
			background.Prim.TextureName = embeddedSeamPNG
			background.Prim.EmbeddedTexture = seamTexturePNG()
			background.Prim.TextureDepth.Val = 18
			background.Prim.TextureZoom.Val = 85

			main := &doc.Layers[1]
			main.Name = "Main"
			main.Prim.Type.Val = int(model.PrimCircleFill)
			main.Prim.Color.Val = rgb(214, 88, 66)
			main.Prim.TextureName = embeddedCheckerPNG
			main.Prim.EmbeddedTexture = checkerTexturePNG()
			main.Prim.TextureDepth.Val = 34
			main.Prim.TextureZoom.Val = 150
			main.Prim.Specular.Val = 22
			main.Eff.AngleF.Val = 0
			main.Eff.AngleT.Val = 140
			main.Eff.AngleAnim.Val = 1
			main.Eff.Mask1Ena.Val = 1
			main.Eff.Mask1Type.Val = 1
			main.Eff.Mask1StartF.Val = -78
			main.Eff.Mask1StartT.Val = -10
			main.Eff.Mask1StartAnim.Val = 1
			main.Eff.Mask1StopF.Val = 56
			main.Eff.Mask1StopT.Val = 108
			main.Eff.Mask1StopAnim.Val = 1
			main.Eff.Mask1Grad.Val = 16
			main.Eff.CenterX.Val = 8
			main.Eff.CenterY.Val = -8
			main.Eff.BrightF.Val = -10
			main.Eff.BrightT.Val = 18
			main.Eff.BrightAnim.Val = 1
			main.Eff.ContrastF.Val = 20
			main.Eff.ContrastT.Val = -12
			main.Eff.ContrastAnim.Val = 1
			main.Eff.SaturationF.Val = -18
			main.Eff.SaturationT.Val = 10
			main.Eff.SaturationAnim.Val = 1
			main.Eff.DLightDirEna.Val = 1
			main.Eff.DLightDirF.Val = 315
			main.Eff.DOffsetF.Val = 8
			main.Eff.DOffsetT.Val = 16
			main.Eff.DOffsetAnim.Val = 1
			main.Eff.DDensityF.Val = 30
			main.Eff.DDensityT.Val = 58
			main.Eff.DDensityAnim.Val = 1
			main.Eff.DDiffuseF.Val = 18
			main.Eff.DDiffuseT.Val = 32
			main.Eff.DDiffuseAnim.Val = 1
			main.Eff.DSType.Val = 1
			main.Eff.DSGrad.Val = 68
			main.Eff.ILightDirEna.Val = 1
			main.Eff.ILightDirF.Val = 220
			main.Eff.IOffsetF.Val = 8
			main.Eff.IDensityF.Val = 36
			main.Eff.IDensityT.Val = 64
			main.Eff.IDensityAnim.Val = 1
			main.Eff.IDiffuseF.Val = 26
			main.Eff.IDiffuseT.Val = 34
			main.Eff.IDiffuseAnim.Val = 1

			accent := &doc.Layers[2]
			accent.Name = "Accent"
			accent.Prim.Type.Val = int(model.PrimLine)
			accent.Prim.Color.Val = rgb(244, 232, 208)
			accent.Prim.Width.Val = 16
			accent.Prim.Length.Val = 84
			accent.Prim.LightDir.Val = 18
			accent.Eff.AlphaF.Val = 72
			accent.Eff.AlphaT.Val = 92
			accent.Eff.AlphaAnim.Val = 1
			accent.Eff.AngleF.Val = 16
			accent.Eff.AngleT.Val = 52
			accent.Eff.AngleAnim.Val = 1
			accent.Eff.OffXF.Val = 6
			accent.Eff.OffXT.Val = -12
			accent.Eff.OffXAnim.Val = 1
			accent.Eff.OffYF.Val = -8
			accent.Eff.OffYT.Val = 10
			accent.Eff.OffYAnim.Val = 1

			return doc
		}},
	}
}

func newPrimitiveDoc(name string) *model.Document {
	doc := model.NewDocument()
	doc.Layers = []model.Layer{model.NewLayer()}
	doc.Prefs.PWidth.Val = 64
	doc.Prefs.PHeight.Val = 64
	doc.Prefs.Width = 64
	doc.Prefs.Height = 64
	doc.Prefs.RenderFrames.Val = 1
	doc.Prefs.PreviewFrames.Val = 1
	doc.Prefs.BkColor.Val = color.RGBA{}

	ly := &doc.Layers[0]
	ly.Name = name
	ly.Visible.Val = 1
	ly.Prim.Color.Val = rgb(255, 0, 0)
	ly.Prim.Transparent.Val = 0
	ly.Eff.AntiAlias.Val = 1

	return doc
}

func newAnimatedDoc(name string, w, h, frames int) *model.Document {
	doc := newPrimitiveDocWithSize(name, w, h)
	doc.Prefs.RenderFrames.Val = frames
	doc.Prefs.PreviewFrames.Val = frames

	return doc
}

func newPrimitiveDocWithSize(name string, w, h int) *model.Document {
	doc := newPrimitiveDoc(name)
	doc.Prefs.Width = w
	doc.Prefs.Height = h
	doc.Prefs.PWidth.Val = w
	doc.Prefs.PHeight.Val = h

	return doc
}

func rgb(r, g, b uint8) color.RGBA {
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

func solidColorPNG(w, h int, fill color.NRGBA) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, fill)
		}
	}

	return encodePNG(img, "solid color")
}

func centeredBlockPNG(w, h, blockW, blockH int, fill color.NRGBA) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	startX := (w - blockW) / 2

	startY := (h - blockH) / 2
	for y := startY; y < startY+blockH; y++ {
		for x := startX; x < startX+blockW; x++ {
			img.SetNRGBA(x, y, fill)
		}
	}

	return encodePNG(img, "centered block")
}

func colorStripPNG(frameW, frameH int, fills ...color.NRGBA) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, frameW*len(fills), frameH))
	for i, fill := range fills {
		x0 := i * frameW
		for y := range frameH {
			for x := range frameW {
				img.SetNRGBA(x0+x, y, fill)
			}
		}
	}

	return encodePNG(img, "color strip")
}

func encodePNG(img image.Image, label string) []byte {
	var buf bytes.Buffer

	err := png.Encode(&buf, img)
	if err != nil {
		log.Fatalf("encode %s: %v", label, err)
	}

	return buf.Bytes()
}

func checkerTexturePNG() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	light := color.NRGBA{R: 240, G: 240, B: 240, A: 255}
	dark := color.NRGBA{R: 48, G: 48, B: 48, A: 255}

	for y := range 8 {
		for x := range 8 {
			c := light
			if (x/2+y/2)%2 == 1 {
				c = dark
			}

			img.Set(x, y, c)
		}
	}

	return encodePNG(img, "checker texture")
}

func seamTexturePNG() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	palette := [4]color.NRGBA{
		{R: 240, G: 32, B: 32, A: 255},
		{R: 32, G: 32, B: 32, A: 255},
		{R: 240, G: 240, B: 240, A: 255},
		{R: 32, G: 120, B: 240, A: 255},
	}

	for y := range 4 {
		for x := range 4 {
			img.Set(x, y, palette[(x+y)%len(palette)])
		}
	}

	return encodePNG(img, "seam texture")
}
