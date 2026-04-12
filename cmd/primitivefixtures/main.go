package main

import (
	"flag"
	"image/color"
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

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
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
		{Name: "circle_outline_basic", Build: func() *model.Document {
			doc := newPrimitiveDoc("Circle")
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
		{Name: "triangle_basic", Build: func() *model.Document {
			doc := newPrimitiveDoc("Triangle")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimTriangle)
			ly.Prim.Color.Val = rgb(220, 96, 40)
			ly.Prim.Width.Val = 62
			ly.Prim.Length.Val = 74
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
		{Name: "hlines_basic", Build: func() *model.Document {
			doc := newPrimitiveDoc("HLines")
			ly := &doc.Layers[0]
			ly.Prim.Type.Val = int(model.PrimHLines)
			ly.Prim.Color.Val = rgb(60, 60, 60)
			ly.Prim.Width.Val = 8
			ly.Prim.Step.Val = 18
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
	doc.Prefs.BkColor.Val = rgb(255, 255, 255)

	ly := &doc.Layers[0]
	ly.Name = name
	ly.Visible.Val = 1
	ly.Prim.Color.Val = rgb(255, 0, 0)
	ly.Prim.Transparent.Val = 0
	ly.Eff.AntiAlias.Val = 1

	return doc
}

func rgb(r, g, b uint8) color.RGBA {
	return color.RGBA{R: r, G: g, B: b, A: 255}
}
