package main

import (
	"bytes"
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
	outDir := flag.String("out", filepath.Join("tests", "parity", "primitives", "inputs"), "Directory to write primitive .knob fixtures")
	overwrite := flag.Bool("overwrite", false, "Overwrite existing fixture files")

	flag.Parse()

	err := os.MkdirAll(*outDir, 0o755)
	if err != nil {
		log.Fatalf("mkdir %s: %v", *outDir, err)
	}

	for _, fixture := range primitiveFixtures() {
		path := filepath.Join(*outDir, fixture.Name+".knob")
		if !*overwrite {
			if _, err := os.Stat(path); err == nil {
				continue
			}
		}

		doc := fixture.Build()

		data, err := fileio.Save(doc)
		if err != nil {
			log.Fatalf("save %s: %v", fixture.Name, err)
		}

		if err := os.WriteFile(path, data, 0o644); err != nil {
			log.Fatalf("write %s: %v", path, err)
		}

		log.Println(path)
	}
}

type fixtureDef struct {
	Name  string
	Build func() *model.Document
}

func primitiveFixtures() []fixtureDef {
	return []fixtureDef{
		{Name: "vu3_circle_mask_black", Build: func() *model.Document {
			doc := newPrimitiveDocWithSize("Vu3CircleMaskBlack", 120, 80)
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimCircle)
			ly.Prim.Fill.Val = 1
			ly.Prim.Color.Val = rgb(255, 255, 255)
			ly.Prim.Width.Val = 8
			ly.Prim.Length.Val = 100
			ly.Eff.CenterY.Val = -60
			ly.Eff.OffYF.Val = -38
			ly.Eff.Mask1Ena.Val = 1
			ly.Eff.Mask1Type.Val = 0
			ly.Eff.Mask1StartF.Val = -47
			ly.Eff.Mask1StopF.Val = 20

			return doc
		}},
		{Name: "vu3_circle_nomask", Build: func() *model.Document {
			doc := newPrimitiveDocWithSize("Vu3CircleNomask", 120, 80)
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimCircle)
			ly.Prim.Fill.Val = 1
			ly.Prim.Color.Val = rgb(255, 255, 255)
			ly.Prim.Width.Val = 8
			ly.Prim.Length.Val = 100
			ly.Eff.CenterY.Val = -60
			ly.Eff.OffYF.Val = -38

			return doc
		}},
		{Name: "vu3_circle_nomask_yflip_check", Build: func() *model.Document {
			doc := newPrimitiveDocWithSize("Vu3CircleNomaskYFlipCheck", 120, 80)
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimCircle)
			ly.Prim.Fill.Val = 1
			ly.Prim.Color.Val = rgb(255, 255, 255)
			ly.Prim.Width.Val = 8
			ly.Prim.Length.Val = 100
			ly.Eff.CenterY.Val = -60
			ly.Eff.OffYF.Val = 38

			return doc
		}},
		{Name: "vu3_circle_mask_red", Build: func() *model.Document {
			doc := newPrimitiveDocWithSize("Vu3CircleMaskRed", 120, 80)
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimCircle)
			ly.Prim.Fill.Val = 1
			ly.Prim.Color.Val = rgb(255, 32, 32)
			ly.Prim.Width.Val = 8
			ly.Prim.Length.Val = 100
			ly.Eff.CenterY.Val = -60
			ly.Eff.OffYF.Val = -38
			ly.Eff.Mask1Ena.Val = 1
			ly.Eff.Mask1Type.Val = 0
			ly.Eff.Mask1StartF.Val = 20
			ly.Eff.Mask1StopF.Val = 47

			return doc
		}},
		{Name: "circle_outline_basic", Build: func() *model.Document {
			doc := newPrimitiveDoc("Circle")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimCircle)
			ly.Prim.Color.Val = rgb(192, 32, 32)
			ly.Prim.Width.Val = 14
			ly.Prim.Aspect.Val = 10

			return doc
		}},
		{Name: "tier3_circle_outline_shell", Build: func() *model.Document {
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
		{Name: "tier3_circle_fill_shell", Build: func() *model.Document {
			doc := newPrimitiveDoc("Tier3CircleFillShell")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimCircleFill)
			ly.Prim.Color.Val = rgb(48, 96, 208)

			return doc
		}},
		{Name: "tier3_circle_fill_lit", Build: func() *model.Document {
			doc := newPrimitiveDoc("Tier3CircleFillLit")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimCircleFill)
			ly.Prim.Color.Val = rgb(48, 96, 208)
			ly.Prim.Specular.Val = 40
			ly.Prim.Diffuse.Val = 25
			ly.Prim.Emboss.Val = 20

			return doc
		}},
		{Name: "tier3_circle_fill_texture", Build: func() *model.Document {
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
		{Name: "tier1_rect_fill_plain", Build: func() *model.Document {
			doc := newPrimitiveDoc("Tier1RectFillPlain")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimRectFill)
			ly.Prim.Color.Val = rgb(72, 172, 112)

			return doc
		}},
		{Name: "tier1_rect_fill_aspect", Build: func() *model.Document {
			doc := newPrimitiveDoc("Tier1RectFillAspect")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimRectFill)
			ly.Prim.Color.Val = rgb(72, 172, 112)
			ly.Prim.Aspect.Val = 50

			return doc
		}},
		{Name: "tier1_rect_outline_plain", Build: func() *model.Document {
			doc := newPrimitiveDoc("Tier1RectOutlinePlain")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimRect)
			ly.Prim.Color.Val = rgb(24, 128, 72)
			ly.Prim.Width.Val = 12

			return doc
		}},
		{Name: "triangle_basic", Build: func() *model.Document {
			doc := newPrimitiveDoc("Triangle")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimTriangle)
			ly.Prim.Color.Val = rgb(220, 96, 40)
			ly.Prim.Width.Val = 62
			ly.Prim.Length.Val = 74

			return doc
		}},
		{Name: "tier1_triangle_plain", Build: func() *model.Document {
			doc := newPrimitiveDoc("Tier1TrianglePlain")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimTriangle)
			ly.Prim.Color.Val = rgb(220, 96, 40)
			ly.Prim.Width.Val = 62
			ly.Prim.Length.Val = 74

			return doc
		}},
		{Name: "tier0_shape_fill_plain", Build: func() *model.Document {
			doc := newPrimitiveDoc("Tier0ShapeFillPlain")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimShape)
			ly.Prim.Color.Val = rgb(56, 144, 88)
			ly.Prim.Fill.Val = 1
			ly.Prim.Shape.Val = "/128,24,128,24,128,24:232,128,232,128,232,128:128,232,128,232,128,232:24,128,24,128,24,128"

			return doc
		}},
		{Name: "tier0_shape_outline_plain", Build: func() *model.Document {
			doc := newPrimitiveDoc("Tier0ShapeOutlinePlain")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimShape)
			ly.Prim.Color.Val = rgb(56, 144, 88)
			ly.Prim.Fill.Val = 0
			ly.Prim.Shape.Val = "/128,24,128,24,128,24:232,128,232,128,232,128:128,232,128,232,128,232:24,128,24,128,24,128"

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
		{Name: "tier2_line_plain", Build: func() *model.Document {
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
		{Name: "tier2_radiate_line_plain", Build: func() *model.Document {
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
		{Name: "tier2_hlines_plain", Build: func() *model.Document {
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
		{Name: "tier2_vlines_plain", Build: func() *model.Document {
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
			ly.Prim.Shape.Val = "/128,24,128,24,128,24:232,128,232,128,232,128:128,232,128,232,128,232:24,128,24,128,24,128"

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
			ly.Prim.TextureName = "embedded-checker.png"
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
			ly.Prim.TextureName = "embedded-seam.png"
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
			ly.Prim.TextureName = "embedded-seam.png"
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
			ly.Prim.TextureName = "embedded-seam.png"
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
			ly.Prim.TextureName = "embedded-seam.png"
			ly.Prim.EmbeddedTexture = seamTexturePNG()
			ly.Prim.TextureDepth.Val = 70
			ly.Prim.TextureZoom.Val = 100

			return doc
		}},
		{Name: "vu3_line_transform_flip_probe", Build: func() *model.Document {
			doc := newPrimitiveDocWithSize("Vu3LineTransformFlipProbe", 120, 80)
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimLine)
			ly.Prim.Color.Val = rgb(255, 255, 255)
			ly.Prim.Width.Val = 10
			ly.Prim.Length.Val = 84
			ly.Eff.AngleF.Val = 45
			ly.Eff.AngleT.Val = 45
			ly.Eff.CenterY.Val = -60
			ly.Eff.OffYF.Val = -38

			return doc
		}},
		{Name: "vu3_line_transform_flip_probe_yflip", Build: func() *model.Document {
			doc := newPrimitiveDocWithSize("Vu3LineTransformFlipProbeYFlip", 120, 80)
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimLine)
			ly.Prim.Color.Val = rgb(255, 255, 255)
			ly.Prim.Width.Val = 10
			ly.Prim.Length.Val = 84
			ly.Eff.AngleF.Val = 45
			ly.Eff.AngleT.Val = 45
			ly.Eff.CenterY.Val = -60
			ly.Eff.OffYF.Val = 38

			return doc
		}},
		{Name: "vu3_radiate_mask_black", Build: func() *model.Document {
			doc := newPrimitiveDocWithSize("Vu3RadiateMaskBlack", 120, 80)
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimRadiateLine)
			ly.Prim.Color.Val = rgb(255, 255, 255)
			ly.Prim.Width.Val = 0
			ly.Prim.Length.Val = 11
			ly.Prim.AngleStep.Val = 8
			ly.Prim.Aspect.Val = 17
			ly.Eff.ZoomXF.Val = 157
			ly.Eff.ZoomXT.Val = 157
			ly.Eff.CenterY.Val = -60
			ly.Eff.OffYF.Val = -60
			ly.Eff.Mask1Ena.Val = 1
			ly.Eff.Mask1Type.Val = 0
			ly.Eff.Mask1StartF.Val = -50
			ly.Eff.Mask1StopF.Val = 16

			return doc
		}},
		{Name: "vu3_radiate_nomask", Build: func() *model.Document {
			doc := newPrimitiveDocWithSize("Vu3RadiateNomask", 120, 80)
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimRadiateLine)
			ly.Prim.Color.Val = rgb(255, 255, 255)
			ly.Prim.Width.Val = 0
			ly.Prim.Length.Val = 11
			ly.Prim.AngleStep.Val = 8
			ly.Prim.Aspect.Val = 17
			ly.Eff.ZoomXF.Val = 157
			ly.Eff.ZoomXT.Val = 157
			ly.Eff.CenterY.Val = -60
			ly.Eff.OffYF.Val = -60

			return doc
		}},
		{Name: "vu3_radiate_nomask_yflip", Build: func() *model.Document {
			doc := newPrimitiveDocWithSize("Vu3RadiateNomaskYFlip", 120, 80)
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimRadiateLine)
			ly.Prim.Color.Val = rgb(255, 255, 255)
			ly.Prim.Width.Val = 0
			ly.Prim.Length.Val = 11
			ly.Prim.AngleStep.Val = 8
			ly.Prim.Aspect.Val = 17
			ly.Eff.ZoomXF.Val = 157
			ly.Eff.ZoomXT.Val = 157
			ly.Eff.CenterY.Val = -60
			ly.Eff.OffYF.Val = 60

			return doc
		}},
		{Name: "vu3_radiate_mask_red", Build: func() *model.Document {
			doc := newPrimitiveDocWithSize("Vu3RadiateMaskRed", 120, 80)
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimRadiateLine)
			ly.Prim.Color.Val = rgb(255, 64, 64)
			ly.Prim.Width.Val = 0
			ly.Prim.Length.Val = 11
			ly.Prim.AngleStep.Val = 8
			ly.Prim.Aspect.Val = 17
			ly.Eff.ZoomXF.Val = 157
			ly.Eff.ZoomXT.Val = 157
			ly.Eff.CenterY.Val = -60
			ly.Eff.OffYF.Val = -60
			ly.Eff.Mask1Ena.Val = 1
			ly.Eff.Mask1Type.Val = 0
			ly.Eff.Mask1StartF.Val = 15
			ly.Eff.Mask1StopF.Val = 47

			return doc
		}},
		{Name: "circle_fill_texture_basic", Build: func() *model.Document {
			doc := newPrimitiveDoc("CircleFillTexture")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimCircleFill)
			ly.Prim.Color.Val = rgb(80, 150, 96)
			ly.Prim.TextureName = "embedded-checker.png"
			ly.Prim.EmbeddedTexture = checkerTexturePNG()
			ly.Prim.TextureDepth.Val = 65
			ly.Prim.TextureZoom.Val = 140
			ly.Prim.Specular.Val = 18

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

	var buf bytes.Buffer

	err := png.Encode(&buf, img)
	if err != nil {
		log.Fatalf("encode checker texture: %v", err)
	}

	return buf.Bytes()
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

	var buf bytes.Buffer

	err := png.Encode(&buf, img)
	if err != nil {
		log.Fatalf("encode seam texture: %v", err)
	}

	return buf.Bytes()
}
