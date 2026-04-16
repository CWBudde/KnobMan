//go:build js && wasm

package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"syscall/js"

	exportpkg "knobman/internal/export"
	"knobman/internal/fileio"
	"knobman/internal/model"
	"knobman/internal/render"
)

var (
	logicalW      = 64
	logicalH      = 64
	zoomFactor    = 8
	doc           *model.Document
	history       *model.History
	textures      []*render.Texture
	textureNames  []string
	textureRaw    [][]byte
	previewFrame  int
	selectedLayer int
	renderBuf     *render.PixBuf
	displayBuf    []byte
)

func pushHistory() {
	if history == nil {
		history = model.NewHistory()
	}
	history.Push(doc)
}

func main() {
	newDocument()
	syncDisplayBuffer()

	js.Global().Set("knobman_init", js.FuncOf(jsInit))
	js.Global().Set("knobman_render", js.FuncOf(jsRender))
	js.Global().Set("knobman_renderFrameRaw", js.FuncOf(jsRenderFrameRaw))
	js.Global().Set("knobman_getDimensions", js.FuncOf(jsGetDimensions))
	js.Global().Set("knobman_setZoom", js.FuncOf(jsSetZoom))

	js.Global().Set("knobman_newDocument", js.FuncOf(jsNewDocument))
	js.Global().Set("knobman_getLayerList", js.FuncOf(jsGetLayerList))
	js.Global().Set("knobman_getLayerPreview", js.FuncOf(jsGetLayerPreview))
	js.Global().Set("knobman_selectLayer", js.FuncOf(jsSelectLayer))
	js.Global().Set("knobman_addLayer", js.FuncOf(jsAddLayer))
	js.Global().Set("knobman_deleteLayer", js.FuncOf(jsDeleteLayer))
	js.Global().Set("knobman_moveLayer", js.FuncOf(jsMoveLayer))
	js.Global().Set("knobman_duplicateLayer", js.FuncOf(jsDuplicateLayer))
	js.Global().Set("knobman_setLayerVisible", js.FuncOf(jsSetLayerVisible))
	js.Global().Set("knobman_setLayerSolo", js.FuncOf(jsSetLayerSolo))

	js.Global().Set("knobman_setPreviewFrame", js.FuncOf(jsSetPreviewFrame))
	js.Global().Set("knobman_setPrefs", js.FuncOf(jsSetPrefs))
	js.Global().Set("knobman_getPrefs", js.FuncOf(jsGetPrefs))
	js.Global().Set("knobman_setParam", js.FuncOf(jsSetParam))
	js.Global().Set("knobman_getParam", js.FuncOf(jsGetParam))
	js.Global().Set("knobman_setEffectParam", js.FuncOf(jsSetEffectParam))
	js.Global().Set("knobman_getEffectParam", js.FuncOf(jsGetEffectParam))
	js.Global().Set("knobman_getCurve", js.FuncOf(jsGetCurve))
	js.Global().Set("knobman_setCurve", js.FuncOf(jsSetCurve))
	js.Global().Set("knobman_evalCurve", js.FuncOf(jsEvalCurve))
	js.Global().Set("knobman_getTextureList", js.FuncOf(jsGetTextureList))
	js.Global().Set("knobman_addTexture", js.FuncOf(jsAddTexture))
	js.Global().Set("knobman_getTextureData", js.FuncOf(jsGetTextureData))
	js.Global().Set("knobman_exportPNGStrip", js.FuncOf(jsExportPNGStrip))
	js.Global().Set("knobman_exportPNGFramesZip", js.FuncOf(jsExportPNGFramesZip))
	js.Global().Set("knobman_exportGIF", js.FuncOf(jsExportGIF))
	js.Global().Set("knobman_exportAPNG", js.FuncOf(jsExportAPNG))

	js.Global().Set("knobman_loadFile", js.FuncOf(jsLoadFile))
	js.Global().Set("knobman_saveFile", js.FuncOf(jsSaveFile))

	js.Global().Set("knobman_undo", js.FuncOf(jsUndo))
	js.Global().Set("knobman_redo", js.FuncOf(jsRedo))
	js.Global().Set("knobman_canUndo", js.FuncOf(jsCanUndo))
	js.Global().Set("knobman_canRedo", js.FuncOf(jsCanRedo))

	select {}
}

func jsInit(this js.Value, args []js.Value) any {
	if len(args) >= 2 {
		logicalW = maxInt(1, args[0].Int())
		logicalH = maxInt(1, args[1].Int())
	}
	if len(args) >= 3 {
		zoomFactor = maxInt(1, args[2].Int())
	}
	if doc == nil {
		newDocument()
	}
	doc.Prefs.PWidth.Val = logicalW
	doc.Prefs.PHeight.Val = logicalH
	doc.Prefs.Width = logicalW
	doc.Prefs.Height = logicalH
	syncDisplayBuffer()
	return nil
}

func jsRender(this js.Value, args []js.Value) any {
	renderScene()
	if len(args) >= 1 {
		js.CopyBytesToJS(args[0], displayBuf)
	}
	return nil
}

func jsRenderFrameRaw(this js.Value, args []js.Value) any {
	if doc == nil {
		return js.Null()
	}
	frame := 0
	if len(args) >= 1 {
		frame = args[0].Int()
	}
	total := maxInt(1, doc.Prefs.RenderFrames.Val)
	frame = clampInt(frame, 0, total-1)
	buf := render.NewPixBuf(logicalW, logicalH)
	render.RenderFrame(buf, doc, frame, textures)
	arr := js.Global().Get("Uint8Array").New(len(buf.Data))
	js.CopyBytesToJS(arr, buf.Data)
	return map[string]any{
		"width":  logicalW,
		"height": logicalH,
		"frame":  frame,
		"data":   arr,
	}
}

func jsGetDimensions(this js.Value, args []js.Value) any {
	return map[string]any{
		"width":  logicalW * zoomFactor,
		"height": logicalH * zoomFactor,
	}
}

func jsSetZoom(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return nil
	}
	z := maxInt(1, args[0].Int())
	if z == zoomFactor {
		return nil
	}
	zoomFactor = z
	syncDisplayBuffer()
	return nil
}

func jsNewDocument(this js.Value, args []js.Value) any {
	newDocument()
	syncDisplayBuffer()
	return true
}

func jsSetPreviewFrame(this js.Value, args []js.Value) any {
	if len(args) >= 1 {
		previewFrame = args[0].Int()
	}
	return nil
}

func jsGetLayerList(this js.Value, args []js.Value) any {
	if doc == nil {
		return []any{}
	}
	out := make([]any, 0, len(doc.Layers))
	for i := range doc.Layers {
		ly := &doc.Layers[i]
		out = append(out, map[string]any{
			"index":    i,
			"name":     ly.Name,
			"visible":  ly.Visible.Val != 0,
			"solo":     ly.Solo.Val != 0,
			"primType": ly.Prim.Type.Val,
			"selected": i == selectedLayer,
		})
	}
	return out
}

func jsGetLayerPreview(this js.Value, args []js.Value) any {
	if doc == nil || len(doc.Layers) == 0 || len(args) < 2 {
		return js.Null()
	}
	idx := args[0].Int()
	if idx < 0 || idx >= len(doc.Layers) {
		return js.Null()
	}
	frame := args[1].Int()
	size := 40
	if len(args) >= 3 {
		size = clampInt(args[2].Int(), 16, 128)
	}
	total := maxInt(1, doc.Prefs.RenderFrames.Val)
	frame = clampInt(frame, 0, total-1)

	buf := render.NewPixBuf(size, size)
	render.RenderPrimitive(buf, &doc.Layers[idx].Prim, textures, frame, total)
	arr := js.Global().Get("Uint8Array").New(len(buf.Data))
	js.CopyBytesToJS(arr, buf.Data)
	return map[string]any{
		"width":  size,
		"height": size,
		"data":   arr,
	}
}

func jsSelectLayer(this js.Value, args []js.Value) any {
	if len(args) >= 1 {
		selectedLayer = clampLayer(args[0].Int())
	}
	return selectedLayer
}

func jsAddLayer(this js.Value, args []js.Value) any {
	if doc == nil {
		return -1
	}
	pushHistory()
	idx := selectedLayer + 1
	if idx < 0 || idx > len(doc.Layers) {
		idx = len(doc.Layers)
	}
	ly := model.NewLayer()
	ly.Name = "Layer"
	doc.Layers = append(doc.Layers, model.Layer{})
	copy(doc.Layers[idx+1:], doc.Layers[idx:])
	doc.Layers[idx] = ly
	selectedLayer = idx
	return selectedLayer
}

func jsDeleteLayer(this js.Value, args []js.Value) any {
	if doc == nil || len(doc.Layers) <= 1 {
		return false
	}
	pushHistory()
	idx := selectedLayer
	if len(args) >= 1 {
		idx = clampLayer(args[0].Int())
	}
	doc.Layers = append(doc.Layers[:idx], doc.Layers[idx+1:]...)
	if selectedLayer >= len(doc.Layers) {
		selectedLayer = len(doc.Layers) - 1
	}
	if selectedLayer < 0 {
		selectedLayer = 0
	}
	return true
}

func jsMoveLayer(this js.Value, args []js.Value) any {
	if doc == nil || len(doc.Layers) <= 1 {
		return selectedLayer
	}
	delta := 0
	if len(args) >= 1 {
		delta = args[0].Int()
	}
	target := selectedLayer + delta
	if target < 0 || target >= len(doc.Layers) {
		return selectedLayer
	}
	pushHistory()
	doc.Layers[selectedLayer], doc.Layers[target] = doc.Layers[target], doc.Layers[selectedLayer]
	selectedLayer = target
	return selectedLayer
}

func jsDuplicateLayer(this js.Value, args []js.Value) any {
	if doc == nil || selectedLayer < 0 || selectedLayer >= len(doc.Layers) {
		return -1
	}
	pushHistory()
	cp := doc.Layers[selectedLayer].Clone()
	cp.Name = cp.Name + " Copy"
	idx := selectedLayer + 1
	doc.Layers = append(doc.Layers, model.Layer{})
	copy(doc.Layers[idx+1:], doc.Layers[idx:])
	doc.Layers[idx] = cp
	selectedLayer = idx
	return selectedLayer
}

func jsSetLayerVisible(this js.Value, args []js.Value) any {
	if doc == nil || len(args) < 2 {
		return false
	}
	pushHistory()
	idx := clampLayer(args[0].Int())
	doc.Layers[idx].Visible.Val = boolToInt(args[1].Bool())
	return true
}

func jsSetLayerSolo(this js.Value, args []js.Value) any {
	if doc == nil || len(args) < 2 {
		return false
	}
	pushHistory()
	idx := clampLayer(args[0].Int())
	doc.Layers[idx].Solo.Val = boolToInt(args[1].Bool())
	return true
}

func jsSetPrefs(this js.Value, args []js.Value) any {
	if doc == nil || len(args) < 1 || args[0].Type() != js.TypeObject {
		return false
	}
	pushHistory()
	obj := args[0]
	if v := obj.Get("width"); v.Type() != js.TypeUndefined && v.Type() != js.TypeNull {
		logicalW = maxInt(1, v.Int())
		doc.Prefs.PWidth.Val = logicalW
	}
	if v := obj.Get("height"); v.Type() != js.TypeUndefined && v.Type() != js.TypeNull {
		logicalH = maxInt(1, v.Int())
		doc.Prefs.PHeight.Val = logicalH
	}
	if v := obj.Get("frames"); v.Type() != js.TypeUndefined && v.Type() != js.TypeNull {
		doc.Prefs.RenderFrames.Val = maxInt(1, v.Int())
	}
	if v := obj.Get("renderFrames"); v.Type() != js.TypeUndefined && v.Type() != js.TypeNull {
		doc.Prefs.RenderFrames.Val = maxInt(1, v.Int())
	}
	if v := obj.Get("previewFrames"); v.Type() != js.TypeUndefined && v.Type() != js.TypeNull {
		doc.Prefs.PreviewFrames.Val = maxInt(1, v.Int())
	}
	if v := obj.Get("oversampling"); v.Type() != js.TypeUndefined && v.Type() != js.TypeNull {
		doc.Prefs.Oversampling.Val = clampInt(v.Int(), 0, 3)
	}
	if v := obj.Get("alignHorizontal"); v.Type() != js.TypeUndefined && v.Type() != js.TypeNull {
		doc.Prefs.AlignHorz.Val = clampInt(v.Int(), 0, 1)
	}
	if v := obj.Get("exportOption"); v.Type() != js.TypeUndefined && v.Type() != js.TypeNull {
		doc.Prefs.ExportOption.Val = v.Int()
	}
	if v := obj.Get("duration"); v.Type() != js.TypeUndefined && v.Type() != js.TypeNull {
		doc.Prefs.Duration.Val = maxInt(1, v.Int())
	}
	if v := obj.Get("loop"); v.Type() != js.TypeUndefined && v.Type() != js.TypeNull {
		doc.Prefs.Loop.Val = maxInt(0, v.Int())
	}
	if v := obj.Get("biDir"); v.Type() != js.TypeUndefined && v.Type() != js.TypeNull {
		doc.Prefs.BiDir.Val = boolOrInt(v)
	}
	if v := obj.Get("bgColor"); v.Type() == js.TypeString {
		if c, ok := parseHexColor(v.String()); ok {
			doc.Prefs.BkColor.Val = c
		}
	}
	if v := obj.Get("bgAlpha"); v.Type() != js.TypeUndefined && v.Type() != js.TypeNull {
		a := clampInt(v.Int(), 0, 255)
		c := doc.Prefs.BkColor.Val
		c.A = uint8(a)
		doc.Prefs.BkColor.Val = c
	}
	doc.Prefs.Width = doc.Prefs.EffectiveWidth()
	doc.Prefs.Height = doc.Prefs.EffectiveHeight()
	if previewFrame >= doc.Prefs.RenderFrames.Val {
		previewFrame = doc.Prefs.RenderFrames.Val - 1
	}
	syncDisplayBuffer()
	return true
}

func jsGetPrefs(this js.Value, args []js.Value) any {
	if doc == nil {
		return map[string]any{}
	}
	c := doc.Prefs.BkColor.Val
	return map[string]any{
		"width":           logicalW,
		"height":          logicalH,
		"frames":          doc.Prefs.RenderFrames.Val,
		"renderFrames":    doc.Prefs.RenderFrames.Val,
		"previewFrames":   doc.Prefs.PreviewFrames.Val,
		"oversampling":    doc.Prefs.Oversampling.Val,
		"alignHorizontal": doc.Prefs.AlignHorz.Val,
		"exportOption":    doc.Prefs.ExportOption.Val,
		"duration":        doc.Prefs.Duration.Val,
		"loop":            doc.Prefs.Loop.Val,
		"biDir":           doc.Prefs.BiDir.Val != 0,
		"bgColor":         colorToHex(c),
		"bgAlpha":         int(c.A),
	}
}

func jsSetParam(this js.Value, args []js.Value) any {
	if doc == nil || len(args) < 3 {
		return false
	}
	pushHistory()
	idx := clampLayer(args[0].Int())
	key := args[1].String()
	v := args[2]
	ly := &doc.Layers[idx]

	switch key {
	case "name":
		ly.Name = v.String()
	case "primType":
		ly.Prim.Type.Val = v.Int()
	case "color":
		if c, ok := parseHexColor(v.String()); ok {
			c.A = ly.Prim.Color.Val.A
			if c.A == 0 {
				c.A = 255
			}
			ly.Prim.Color.Val = c
		}
	case "colorAlpha":
		c := ly.Prim.Color.Val
		c.A = uint8(clampInt(v.Int(), 0, 255))
		ly.Prim.Color.Val = c
	case "file":
		ly.Prim.File.Val = v.String()
	case "text":
		ly.Prim.Text.Val = v.String()
	case "shape":
		ly.Prim.Shape.Val = v.String()
	case "fill":
		ly.Prim.Fill.Val = boolOrInt(v)
	case "bold":
		ly.Prim.Bold.Val = boolOrInt(v)
	case "italic":
		ly.Prim.Italic.Val = boolOrInt(v)
	case "font":
		ly.Prim.Font.Val = maxInt(0, v.Int())
	case "textureFile":
		idx := v.Int()
		if idx < 0 {
			idx = 0
		}
		if idx > len(textures) {
			idx = len(textures)
		}
		ly.Prim.TextureFile.Val = idx
		if idx > 0 {
			applyTextureSlotToLayer(ly, idx)
		} else {
			ly.Prim.TextureName = ""
			ly.Prim.EmbeddedTexture = nil
		}
	case "textureName":
		ly.Prim.TextureName = v.String()
	case "width":
		ly.Prim.Width.Val = v.Float()
	case "length":
		ly.Prim.Length.Val = v.Float()
	case "aspect":
		ly.Prim.Aspect.Val = v.Float()
	case "round":
		ly.Prim.Round.Val = v.Float()
	case "step":
		ly.Prim.Step.Val = v.Float()
	case "angleStep":
		ly.Prim.AngleStep.Val = v.Float()
	case "emboss":
		ly.Prim.Emboss.Val = v.Float()
	case "embossDiffuse":
		ly.Prim.EmbossDiffuse.Val = v.Float()
	case "ambient":
		ly.Prim.Ambient.Val = v.Float()
	case "lightDir":
		ly.Prim.LightDir.Val = v.Float()
	case "specular":
		ly.Prim.Specular.Val = v.Float()
	case "specularWidth":
		ly.Prim.SpecularWidth.Val = v.Float()
	case "textureDepth":
		ly.Prim.TextureDepth.Val = v.Float()
	case "textureZoom":
		ly.Prim.TextureZoom.Val = v.Float()
	case "diffuse":
		ly.Prim.Diffuse.Val = v.Float()
	case "fontSize":
		ly.Prim.FontSize.Val = v.Float()
	case "textAlign":
		ly.Prim.TextAlign.Val = v.Int()
	case "frameAlign":
		ly.Prim.FrameAlign.Val = v.Int()
	case "numFrame":
		ly.Prim.NumFrame.Val = maxInt(1, v.Int())
	case "autoFit":
		ly.Prim.AutoFit.Val = boolOrInt(v)
	case "transparent":
		ly.Prim.Transparent.Val = v.Int()
	case "intelliAlpha":
		ly.Prim.IntelliAlpha.Val = v.Int()
	case "embeddedImage":
		if v.Type() != js.TypeObject {
			ly.Prim.EmbeddedImage = nil
			return true
		}
		nv := v.Get("length")
		if nv.Type() == js.TypeUndefined || nv.Type() == js.TypeNull {
			ly.Prim.EmbeddedImage = nil
			return true
		}
		n := nv.Int()
		if n <= 0 {
			ly.Prim.EmbeddedImage = nil
			return true
		}
		buf := make([]byte, n)
		js.CopyBytesToGo(buf, v)
		ly.Prim.EmbeddedImage = buf
	case "embeddedTexture":
		if v.Type() != js.TypeObject {
			ly.Prim.EmbeddedTexture = nil
			return true
		}
		nv := v.Get("length")
		if nv.Type() == js.TypeUndefined || nv.Type() == js.TypeNull {
			ly.Prim.EmbeddedTexture = nil
			return true
		}
		n := nv.Int()
		if n <= 0 {
			ly.Prim.EmbeddedTexture = nil
			return true
		}
		buf := make([]byte, n)
		js.CopyBytesToGo(buf, v)
		ly.Prim.EmbeddedTexture = buf
	case "clearEmbeddedImage":
		ly.Prim.EmbeddedImage = nil
	case "clearEmbeddedTexture":
		ly.Prim.EmbeddedTexture = nil
	default:
		return false
	}
	return true
}

func jsGetParam(this js.Value, args []js.Value) any {
	if doc == nil || len(args) < 2 {
		return nil
	}
	idx := clampLayer(args[0].Int())
	key := args[1].String()
	ly := &doc.Layers[idx]
	switch key {
	case "name":
		return ly.Name
	case "primType":
		return ly.Prim.Type.Val
	case "color":
		return colorToHex(ly.Prim.Color.Val)
	case "colorAlpha":
		if ly.Prim.Color.Val.A == 0 {
			return 255
		}
		return int(ly.Prim.Color.Val.A)
	case "file":
		return ly.Prim.File.Val
	case "text":
		return ly.Prim.Text.Val
	case "shape":
		return ly.Prim.Shape.Val
	case "fill":
		return ly.Prim.Fill.Val != 0
	case "bold":
		return ly.Prim.Bold.Val != 0
	case "italic":
		return ly.Prim.Italic.Val != 0
	case "font":
		return ly.Prim.Font.Val
	case "textureFile":
		return ly.Prim.TextureFile.Val
	case "textureName":
		return ly.Prim.TextureName
	case "width":
		return ly.Prim.Width.Val
	case "length":
		return ly.Prim.Length.Val
	case "aspect":
		return ly.Prim.Aspect.Val
	case "round":
		return ly.Prim.Round.Val
	case "step":
		return ly.Prim.Step.Val
	case "angleStep":
		return ly.Prim.AngleStep.Val
	case "emboss":
		return ly.Prim.Emboss.Val
	case "embossDiffuse":
		return ly.Prim.EmbossDiffuse.Val
	case "ambient":
		return ly.Prim.Ambient.Val
	case "lightDir":
		return ly.Prim.LightDir.Val
	case "specular":
		return ly.Prim.Specular.Val
	case "specularWidth":
		return ly.Prim.SpecularWidth.Val
	case "textureDepth":
		return ly.Prim.TextureDepth.Val
	case "textureZoom":
		return ly.Prim.TextureZoom.Val
	case "diffuse":
		return ly.Prim.Diffuse.Val
	case "fontSize":
		return ly.Prim.FontSize.Val
	case "textAlign":
		return ly.Prim.TextAlign.Val
	case "frameAlign":
		return ly.Prim.FrameAlign.Val
	case "numFrame":
		return ly.Prim.NumFrame.Val
	case "autoFit":
		return ly.Prim.AutoFit.Val != 0
	case "transparent":
		return ly.Prim.Transparent.Val
	case "intelliAlpha":
		return ly.Prim.IntelliAlpha.Val
	case "hasEmbeddedImage":
		return len(ly.Prim.EmbeddedImage) > 0
	case "hasEmbeddedTexture":
		return len(ly.Prim.EmbeddedTexture) > 0
	default:
		return nil
	}
}

func jsSetEffectParam(this js.Value, args []js.Value) any {
	if doc == nil || len(args) < 3 {
		return false
	}
	pushHistory()
	idx := clampLayer(args[0].Int())
	key := args[1].String()
	v := args[2]
	eff := &doc.Layers[idx].Eff

	switch key {
	case "antiAlias":
		eff.AntiAlias.Val = boolOrInt(v)
	case "unfold":
		eff.Unfold.Val = boolOrInt(v)
	case "animStep":
		eff.AnimStep.Val = maxInt(0, v.Int())
	case "zoomXYSepa":
		eff.ZoomXYSepa.Val = boolOrInt(v)
	case "zoomXF":
		eff.ZoomXF.Val = v.Float()
	case "zoomXT":
		eff.ZoomXT.Val = v.Float()
	case "zoomXAnim":
		eff.ZoomXAnim.Val = clampInt(v.Int(), 0, 8)
	case "zoomYF":
		eff.ZoomYF.Val = v.Float()
	case "zoomYT":
		eff.ZoomYT.Val = v.Float()
	case "zoomYAnim":
		eff.ZoomYAnim.Val = clampInt(v.Int(), 0, 8)
	case "offXF":
		eff.OffXF.Val = v.Float()
	case "offXT":
		eff.OffXT.Val = v.Float()
	case "offXAnim":
		eff.OffXAnim.Val = clampInt(v.Int(), 0, 8)
	case "offYF":
		eff.OffYF.Val = v.Float()
	case "offYT":
		eff.OffYT.Val = v.Float()
	case "offYAnim":
		eff.OffYAnim.Val = clampInt(v.Int(), 0, 8)
	case "keepDir":
		eff.KeepDir.Val = boolOrInt(v)
	case "centerX":
		eff.CenterX.Val = v.Float()
	case "centerY":
		eff.CenterY.Val = v.Float()
	case "angleF":
		eff.AngleF.Val = v.Float()
	case "angleT":
		eff.AngleT.Val = v.Float()
	case "angleAnim":
		eff.AngleAnim.Val = clampInt(v.Int(), 0, 8)

	case "alphaF":
		eff.AlphaF.Val = v.Float()
	case "alphaT":
		eff.AlphaT.Val = v.Float()
	case "alphaAnim":
		eff.AlphaAnim.Val = clampInt(v.Int(), 0, 8)
	case "brightF":
		eff.BrightF.Val = v.Float()
	case "brightT":
		eff.BrightT.Val = v.Float()
	case "brightAnim":
		eff.BrightAnim.Val = clampInt(v.Int(), 0, 8)
	case "contrastF":
		eff.ContrastF.Val = v.Float()
	case "contrastT":
		eff.ContrastT.Val = v.Float()
	case "contrastAnim":
		eff.ContrastAnim.Val = clampInt(v.Int(), 0, 8)
	case "saturationF":
		eff.SaturationF.Val = v.Float()
	case "saturationT":
		eff.SaturationT.Val = v.Float()
	case "saturationAnim":
		eff.SaturationAnim.Val = clampInt(v.Int(), 0, 8)
	case "hueF":
		eff.HueF.Val = v.Float()
	case "hueT":
		eff.HueT.Val = v.Float()
	case "hueAnim":
		eff.HueAnim.Val = clampInt(v.Int(), 0, 8)

	case "mask1Ena":
		eff.Mask1Ena.Val = boolOrInt(v)
	case "mask1Type":
		eff.Mask1Type.Val = v.Int()
	case "mask1Grad":
		eff.Mask1Grad.Val = v.Float()
	case "mask1GradDir":
		eff.Mask1GradDir.Val = v.Int()
	case "mask1StartF":
		eff.Mask1StartF.Val = v.Float()
	case "mask1StartT":
		eff.Mask1StartT.Val = v.Float()
	case "mask1StartAnim":
		eff.Mask1StartAnim.Val = clampInt(v.Int(), 0, 8)
	case "mask1StopF":
		eff.Mask1StopF.Val = v.Float()
	case "mask1StopT":
		eff.Mask1StopT.Val = v.Float()
	case "mask1StopAnim":
		eff.Mask1StopAnim.Val = clampInt(v.Int(), 0, 8)

	case "mask2Ena":
		eff.Mask2Ena.Val = boolOrInt(v)
	case "mask2Op":
		eff.Mask2Op.Val = v.Int()
	case "mask2Type":
		eff.Mask2Type.Val = v.Int()
	case "mask2Grad":
		eff.Mask2Grad.Val = v.Float()
	case "mask2GradDir":
		eff.Mask2GradDir.Val = v.Int()
	case "mask2StartF":
		eff.Mask2StartF.Val = v.Float()
	case "mask2StartT":
		eff.Mask2StartT.Val = v.Float()
	case "mask2StartAnim":
		eff.Mask2StartAnim.Val = clampInt(v.Int(), 0, 8)
	case "mask2StopF":
		eff.Mask2StopF.Val = v.Float()
	case "mask2StopT":
		eff.Mask2StopT.Val = v.Float()
	case "mask2StopAnim":
		eff.Mask2StopAnim.Val = clampInt(v.Int(), 0, 8)

	case "fMaskEna":
		eff.FMaskEna.Val = v.Int()
	case "fMaskStart":
		eff.FMaskStart.Val = v.Float()
	case "fMaskStop":
		eff.FMaskStop.Val = v.Float()
	case "fMaskBits":
		eff.FMaskBits.Val = v.String()

	case "sLightDirF":
		eff.SLightDirF.Val = v.Float()
	case "sLightDirT":
		eff.SLightDirT.Val = v.Float()
	case "sLightDirAnim":
		eff.SLightDirAnim.Val = clampInt(v.Int(), 0, 8)
	case "sDensityF":
		eff.SDensityF.Val = v.Float()
	case "sDensityT":
		eff.SDensityT.Val = v.Float()
	case "sDensityAnim":
		eff.SDensityAnim.Val = clampInt(v.Int(), 0, 8)

	case "dLightDirEna":
		eff.DLightDirEna.Val = boolOrInt(v)
	case "dLightDirF":
		eff.DLightDirF.Val = v.Float()
	case "dLightDirT":
		eff.DLightDirT.Val = v.Float()
	case "dLightDirAnim":
		eff.DLightDirAnim.Val = clampInt(v.Int(), 0, 8)
	case "dOffsetF":
		eff.DOffsetF.Val = v.Float()
	case "dOffsetT":
		eff.DOffsetT.Val = v.Float()
	case "dOffsetAnim":
		eff.DOffsetAnim.Val = clampInt(v.Int(), 0, 8)
	case "dDensityF":
		eff.DDensityF.Val = v.Float()
	case "dDensityT":
		eff.DDensityT.Val = v.Float()
	case "dDensityAnim":
		eff.DDensityAnim.Val = clampInt(v.Int(), 0, 8)
	case "dDiffuseF":
		eff.DDiffuseF.Val = v.Float()
	case "dDiffuseT":
		eff.DDiffuseT.Val = v.Float()
	case "dDiffuseAnim":
		eff.DDiffuseAnim.Val = clampInt(v.Int(), 0, 8)
	case "dsType":
		eff.DSType.Val = v.Int()
	case "dsGrad":
		eff.DSGrad.Val = v.Float()

	case "iLightDirEna":
		eff.ILightDirEna.Val = boolOrInt(v)
	case "iLightDirF":
		eff.ILightDirF.Val = v.Float()
	case "iLightDirT":
		eff.ILightDirT.Val = v.Float()
	case "iLightDirAnim":
		eff.ILightDirAnim.Val = clampInt(v.Int(), 0, 8)
	case "iOffsetF":
		eff.IOffsetF.Val = v.Float()
	case "iOffsetT":
		eff.IOffsetT.Val = v.Float()
	case "iOffsetAnim":
		eff.IOffsetAnim.Val = clampInt(v.Int(), 0, 8)
	case "iDensityF":
		eff.IDensityF.Val = v.Float()
	case "iDensityT":
		eff.IDensityT.Val = v.Float()
	case "iDensityAnim":
		eff.IDensityAnim.Val = clampInt(v.Int(), 0, 8)
	case "iDiffuseF":
		eff.IDiffuseF.Val = v.Float()
	case "iDiffuseT":
		eff.IDiffuseT.Val = v.Float()
	case "iDiffuseAnim":
		eff.IDiffuseAnim.Val = clampInt(v.Int(), 0, 8)

	case "eLightDirEna":
		eff.ELightDirEna.Val = boolOrInt(v)
	case "eLightDirF":
		eff.ELightDirF.Val = v.Float()
	case "eLightDirT":
		eff.ELightDirT.Val = v.Float()
	case "eLightDirAnim":
		eff.ELightDirAnim.Val = clampInt(v.Int(), 0, 8)
	case "eOffsetF":
		eff.EOffsetF.Val = v.Float()
	case "eOffsetT":
		eff.EOffsetT.Val = v.Float()
	case "eOffsetAnim":
		eff.EOffsetAnim.Val = clampInt(v.Int(), 0, 8)
	case "eDensityF":
		eff.EDensityF.Val = v.Float()
	case "eDensityT":
		eff.EDensityT.Val = v.Float()
	case "eDensityAnim":
		eff.EDensityAnim.Val = clampInt(v.Int(), 0, 8)
	default:
		return false
	}
	return true
}

func jsGetEffectParam(this js.Value, args []js.Value) any {
	if doc == nil || len(args) < 2 {
		return nil
	}
	idx := clampLayer(args[0].Int())
	key := args[1].String()
	eff := &doc.Layers[idx].Eff

	switch key {
	case "antiAlias":
		return eff.AntiAlias.Val != 0
	case "unfold":
		return eff.Unfold.Val != 0
	case "animStep":
		return eff.AnimStep.Val
	case "zoomXYSepa":
		return eff.ZoomXYSepa.Val != 0
	case "zoomXF":
		return eff.ZoomXF.Val
	case "zoomXT":
		return eff.ZoomXT.Val
	case "zoomXAnim":
		return eff.ZoomXAnim.Val
	case "zoomYF":
		return eff.ZoomYF.Val
	case "zoomYT":
		return eff.ZoomYT.Val
	case "zoomYAnim":
		return eff.ZoomYAnim.Val
	case "offXF":
		return eff.OffXF.Val
	case "offXT":
		return eff.OffXT.Val
	case "offXAnim":
		return eff.OffXAnim.Val
	case "offYF":
		return eff.OffYF.Val
	case "offYT":
		return eff.OffYT.Val
	case "offYAnim":
		return eff.OffYAnim.Val
	case "keepDir":
		return eff.KeepDir.Val != 0
	case "centerX":
		return eff.CenterX.Val
	case "centerY":
		return eff.CenterY.Val
	case "angleF":
		return eff.AngleF.Val
	case "angleT":
		return eff.AngleT.Val
	case "angleAnim":
		return eff.AngleAnim.Val

	case "alphaF":
		return eff.AlphaF.Val
	case "alphaT":
		return eff.AlphaT.Val
	case "alphaAnim":
		return eff.AlphaAnim.Val
	case "brightF":
		return eff.BrightF.Val
	case "brightT":
		return eff.BrightT.Val
	case "brightAnim":
		return eff.BrightAnim.Val
	case "contrastF":
		return eff.ContrastF.Val
	case "contrastT":
		return eff.ContrastT.Val
	case "contrastAnim":
		return eff.ContrastAnim.Val
	case "saturationF":
		return eff.SaturationF.Val
	case "saturationT":
		return eff.SaturationT.Val
	case "saturationAnim":
		return eff.SaturationAnim.Val
	case "hueF":
		return eff.HueF.Val
	case "hueT":
		return eff.HueT.Val
	case "hueAnim":
		return eff.HueAnim.Val

	case "mask1Ena":
		return eff.Mask1Ena.Val != 0
	case "mask1Type":
		return eff.Mask1Type.Val
	case "mask1Grad":
		return eff.Mask1Grad.Val
	case "mask1GradDir":
		return eff.Mask1GradDir.Val
	case "mask1StartF":
		return eff.Mask1StartF.Val
	case "mask1StartT":
		return eff.Mask1StartT.Val
	case "mask1StartAnim":
		return eff.Mask1StartAnim.Val
	case "mask1StopF":
		return eff.Mask1StopF.Val
	case "mask1StopT":
		return eff.Mask1StopT.Val
	case "mask1StopAnim":
		return eff.Mask1StopAnim.Val

	case "mask2Ena":
		return eff.Mask2Ena.Val != 0
	case "mask2Op":
		return eff.Mask2Op.Val
	case "mask2Type":
		return eff.Mask2Type.Val
	case "mask2Grad":
		return eff.Mask2Grad.Val
	case "mask2GradDir":
		return eff.Mask2GradDir.Val
	case "mask2StartF":
		return eff.Mask2StartF.Val
	case "mask2StartT":
		return eff.Mask2StartT.Val
	case "mask2StartAnim":
		return eff.Mask2StartAnim.Val
	case "mask2StopF":
		return eff.Mask2StopF.Val
	case "mask2StopT":
		return eff.Mask2StopT.Val
	case "mask2StopAnim":
		return eff.Mask2StopAnim.Val

	case "fMaskEna":
		return eff.FMaskEna.Val
	case "fMaskStart":
		return eff.FMaskStart.Val
	case "fMaskStop":
		return eff.FMaskStop.Val
	case "fMaskBits":
		return eff.FMaskBits.Val

	case "sLightDirF":
		return eff.SLightDirF.Val
	case "sLightDirT":
		return eff.SLightDirT.Val
	case "sLightDirAnim":
		return eff.SLightDirAnim.Val
	case "sDensityF":
		return eff.SDensityF.Val
	case "sDensityT":
		return eff.SDensityT.Val
	case "sDensityAnim":
		return eff.SDensityAnim.Val

	case "dLightDirEna":
		return eff.DLightDirEna.Val != 0
	case "dLightDirF":
		return eff.DLightDirF.Val
	case "dLightDirT":
		return eff.DLightDirT.Val
	case "dLightDirAnim":
		return eff.DLightDirAnim.Val
	case "dOffsetF":
		return eff.DOffsetF.Val
	case "dOffsetT":
		return eff.DOffsetT.Val
	case "dOffsetAnim":
		return eff.DOffsetAnim.Val
	case "dDensityF":
		return eff.DDensityF.Val
	case "dDensityT":
		return eff.DDensityT.Val
	case "dDensityAnim":
		return eff.DDensityAnim.Val
	case "dDiffuseF":
		return eff.DDiffuseF.Val
	case "dDiffuseT":
		return eff.DDiffuseT.Val
	case "dDiffuseAnim":
		return eff.DDiffuseAnim.Val
	case "dsType":
		return eff.DSType.Val
	case "dsGrad":
		return eff.DSGrad.Val

	case "iLightDirEna":
		return eff.ILightDirEna.Val != 0
	case "iLightDirF":
		return eff.ILightDirF.Val
	case "iLightDirT":
		return eff.ILightDirT.Val
	case "iLightDirAnim":
		return eff.ILightDirAnim.Val
	case "iOffsetF":
		return eff.IOffsetF.Val
	case "iOffsetT":
		return eff.IOffsetT.Val
	case "iOffsetAnim":
		return eff.IOffsetAnim.Val
	case "iDensityF":
		return eff.IDensityF.Val
	case "iDensityT":
		return eff.IDensityT.Val
	case "iDensityAnim":
		return eff.IDensityAnim.Val
	case "iDiffuseF":
		return eff.IDiffuseF.Val
	case "iDiffuseT":
		return eff.IDiffuseT.Val
	case "iDiffuseAnim":
		return eff.IDiffuseAnim.Val

	case "eLightDirEna":
		return eff.ELightDirEna.Val != 0
	case "eLightDirF":
		return eff.ELightDirF.Val
	case "eLightDirT":
		return eff.ELightDirT.Val
	case "eLightDirAnim":
		return eff.ELightDirAnim.Val
	case "eOffsetF":
		return eff.EOffsetF.Val
	case "eOffsetT":
		return eff.EOffsetT.Val
	case "eOffsetAnim":
		return eff.EOffsetAnim.Val
	case "eDensityF":
		return eff.EDensityF.Val
	case "eDensityT":
		return eff.EDensityT.Val
	case "eDensityAnim":
		return eff.EDensityAnim.Val
	default:
		return nil
	}
}

func jsLoadFile(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return false
	}
	in := args[0]
	if in.Type() != js.TypeObject {
		return false
	}
	buf := make([]byte, in.Get("length").Int())
	js.CopyBytesToGo(buf, in)
	loaded, err := fileio.Load(buf)
	if err != nil {
		return false
	}
	doc = loaded
	logicalW = maxInt(1, doc.Prefs.PWidth.Val)
	logicalH = maxInt(1, doc.Prefs.PHeight.Val)
	if selectedLayer >= len(doc.Layers) {
		selectedLayer = len(doc.Layers) - 1
	}
	if selectedLayer < 0 {
		selectedLayer = 0
	}
	rebuildTextureSlotsFromDoc()
	syncDisplayBuffer()
	history = model.NewHistory()
	return true
}

func jsGetTextureList(this js.Value, args []js.Value) any {
	out := make([]any, 0, len(textures))
	for i, t := range textures {
		name := fmt.Sprintf("Texture %d", i+1)
		if i < len(textureNames) && strings.TrimSpace(textureNames[i]) != "" {
			name = textureNames[i]
		}
		w, h := 0, 0
		if t != nil {
			w, h = t.W, t.H
		}
		out = append(out, map[string]any{
			"index":  i + 1,
			"name":   name,
			"width":  w,
			"height": h,
		})
	}
	return out
}

func jsAddTexture(this js.Value, args []js.Value) any {
	if len(args) < 2 {
		return 0
	}
	name := strings.TrimSpace(args[0].String())
	in := args[1]
	if in.Type() != js.TypeObject {
		return 0
	}
	nv := in.Get("length")
	if nv.Type() == js.TypeUndefined || nv.Type() == js.TypeNull {
		return 0
	}
	n := nv.Int()
	if n <= 0 {
		return 0
	}
	buf := make([]byte, n)
	js.CopyBytesToGo(buf, in)
	idx, ok := addTextureSlot(name, buf)
	if !ok {
		return 0
	}
	return idx
}

func jsGetTextureData(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return js.Null()
	}
	idx := args[0].Int()
	if idx <= 0 || idx > len(textureRaw) {
		return js.Null()
	}
	data := textureRaw[idx-1]
	if len(data) == 0 {
		return js.Null()
	}
	arr := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(arr, data)
	return arr
}

func jsGetCurve(this js.Value, args []js.Value) any {
	if doc == nil {
		return nil
	}
	idx := 0
	if len(args) >= 1 {
		idx = clampInt(args[0].Int()-1, 0, 7)
	}
	c := &doc.Curves[idx]
	tm := make([]any, len(c.Tm))
	lv := make([]any, len(c.Lv))
	for i := range c.Tm {
		tm[i] = c.Tm[i]
		lv[i] = c.Lv[i]
	}
	return map[string]any{
		"tm":       tm,
		"lv":       lv,
		"stepReso": c.StepReso.Val,
	}
}

func jsSetCurve(this js.Value, args []js.Value) any {
	if doc == nil || len(args) < 2 {
		return false
	}
	pushHistory()
	idx := clampInt(args[0].Int()-1, 0, 7)
	obj := args[1]
	if obj.Type() != js.TypeObject {
		return false
	}
	tmArr := obj.Get("tm")
	lvArr := obj.Get("lv")
	if tmArr.Type() != js.TypeObject || lvArr.Type() != js.TypeObject {
		return false
	}
	c := doc.Curves[idx]
	for i := 0; i < len(c.Tm); i++ {
		t := tmArr.Index(i)
		l := lvArr.Index(i)
		if (t.Type() == js.TypeUndefined || t.Type() == js.TypeNull) ||
			(l.Type() == js.TypeUndefined || l.Type() == js.TypeNull) {
			continue
		}
		tv := t.Int()
		lv := l.Int()
		if i == 0 {
			tv = 0
			lv = clampInt(lv, 0, 100)
		} else if i == len(c.Tm)-1 {
			tv = 100
			lv = clampInt(lv, 0, 100)
		} else if tv < 0 || lv < 0 {
			tv, lv = -1, -1
		} else {
			tv = clampInt(tv, 0, 100)
			lv = clampInt(lv, 0, 100)
		}
		c.Tm[i] = tv
		c.Lv[i] = lv
	}
	if step := obj.Get("stepReso"); step.Type() != js.TypeUndefined && step.Type() != js.TypeNull {
		c.StepReso.Val = clampInt(step.Int(), 0, 64)
	}
	doc.Curves[idx] = c
	return true
}

func jsEvalCurve(this js.Value, args []js.Value) any {
	if doc == nil {
		return 0
	}
	idx := 0
	if len(args) >= 1 {
		idx = clampInt(args[0].Int()-1, 0, 7)
	}
	ratio := 0.0
	if len(args) >= 2 {
		ratio = args[1].Float()
	}
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	return doc.Curves[idx].Eval(ratio) * 100.0
}

func jsUndo(this js.Value, args []js.Value) any {
	if history == nil || !history.CanUndo() {
		return false
	}
	prev := history.Undo(doc)
	if prev == nil {
		return false
	}
	doc = prev
	logicalW = maxInt(1, doc.Prefs.PWidth.Val)
	logicalH = maxInt(1, doc.Prefs.PHeight.Val)
	if selectedLayer >= len(doc.Layers) {
		selectedLayer = len(doc.Layers) - 1
	}
	if selectedLayer < 0 {
		selectedLayer = 0
	}
	rebuildTextureSlotsFromDoc()
	syncDisplayBuffer()
	return true
}

func jsRedo(this js.Value, args []js.Value) any {
	if history == nil || !history.CanRedo() {
		return false
	}
	next := history.Redo(doc)
	if next == nil {
		return false
	}
	doc = next
	logicalW = maxInt(1, doc.Prefs.PWidth.Val)
	logicalH = maxInt(1, doc.Prefs.PHeight.Val)
	if selectedLayer >= len(doc.Layers) {
		selectedLayer = len(doc.Layers) - 1
	}
	if selectedLayer < 0 {
		selectedLayer = 0
	}
	rebuildTextureSlotsFromDoc()
	syncDisplayBuffer()
	return true
}

func jsCanUndo(this js.Value, args []js.Value) any {
	return history != nil && history.CanUndo()
}

func jsCanRedo(this js.Value, args []js.Value) any {
	return history != nil && history.CanRedo()
}

func jsSaveFile(this js.Value, args []js.Value) any {
	if doc == nil {
		return js.Null()
	}
	b, err := fileio.Save(doc)
	if err != nil {
		return js.Null()
	}
	arr := js.Global().Get("Uint8Array").New(len(b))
	js.CopyBytesToJS(arr, b)
	return arr
}

func jsExportPNGStrip(this js.Value, args []js.Value) any {
	if doc == nil {
		return js.Null()
	}
	horizontal := doc.Prefs.AlignHorz.Val != 0
	if len(args) >= 1 {
		horizontal = args[0].Bool()
	}
	out, err := exportpkg.ExportPNGStrip(doc, textures, horizontal)
	if err != nil || len(out) == 0 {
		return js.Null()
	}
	arr := js.Global().Get("Uint8Array").New(len(out))
	js.CopyBytesToJS(arr, out)
	return arr
}

func jsExportPNGFramesZip(this js.Value, args []js.Value) any {
	if doc == nil {
		return js.Null()
	}
	frames, err := exportpkg.ExportPNGFrames(doc, textures)
	if err != nil || len(frames) == 0 {
		return js.Null()
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	pad := len(strconv.Itoa(maxInt(0, len(frames)-1)))
	if pad < 3 {
		pad = 3
	}
	for i, framePNG := range frames {
		name := fmt.Sprintf("frame_%0*d.png", pad, i)
		fw, err := zw.Create(name)
		if err != nil {
			_ = zw.Close()
			return js.Null()
		}
		if _, err := fw.Write(framePNG); err != nil {
			_ = zw.Close()
			return js.Null()
		}
	}
	if err := zw.Close(); err != nil {
		return js.Null()
	}
	out := buf.Bytes()
	arr := js.Global().Get("Uint8Array").New(len(out))
	js.CopyBytesToJS(arr, out)
	return arr
}

func jsExportGIF(this js.Value, args []js.Value) any {
	if doc == nil {
		return js.Null()
	}
	out, err := exportpkg.ExportGIF(doc, textures)
	if err != nil || len(out) == 0 {
		return js.Null()
	}
	arr := js.Global().Get("Uint8Array").New(len(out))
	js.CopyBytesToJS(arr, out)
	return arr
}

func jsExportAPNG(this js.Value, args []js.Value) any {
	if doc == nil {
		return js.Null()
	}
	out, err := exportpkg.ExportAPNG(doc, textures)
	if err != nil || len(out) == 0 {
		return js.Null()
	}
	arr := js.Global().Get("Uint8Array").New(len(out))
	js.CopyBytesToJS(arr, out)
	return arr
}

func renderScene() {
	if doc == nil {
		return
	}
	if renderBuf == nil || renderBuf.Width != logicalW || renderBuf.Height != logicalH {
		renderBuf = render.NewPixBuf(logicalW, logicalH)
	}
	if previewFrame < 0 {
		previewFrame = 0
	}
	if doc.Prefs.RenderFrames.Val > 0 && previewFrame >= doc.Prefs.RenderFrames.Val {
		previewFrame = doc.Prefs.RenderFrames.Val - 1
	}
	render.RenderFrame(renderBuf, doc, previewFrame, textures)
	upscaleNearest(displayBuf, logicalW, logicalH, zoomFactor, renderBuf.Data)
}

func syncDisplayBuffer() {
	renderBuf = render.NewPixBuf(logicalW, logicalH)
	displayBuf = make([]byte, logicalW*logicalH*zoomFactor*zoomFactor*4)
}

func newDocument() {
	doc = model.NewDocument()
	doc.Prefs.PWidth.Val = logicalW
	doc.Prefs.PHeight.Val = logicalH
	doc.Prefs.Width = logicalW
	doc.Prefs.Height = logicalH
	resetTextureSlots()
	selectedLayer = 0
	previewFrame = 0
	history = model.NewHistory()
}

func resetTextureSlots() {
	textures = nil
	textureNames = nil
	textureRaw = nil
}

func addTextureSlot(name string, data []byte) (int, bool) {
	if len(data) == 0 {
		return 0, false
	}
	tex, err := render.DecodeTexture(data)
	if err != nil || tex == nil {
		return 0, false
	}
	textures = append(textures, tex)
	textureRaw = append(textureRaw, append([]byte(nil), data...))
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("Texture %d", len(textures))
	}
	textureNames = append(textureNames, name)
	return len(textures), true
}

func applyTextureSlotToLayer(ly *model.Layer, idx int) {
	if ly == nil || idx <= 0 || idx > len(textureRaw) {
		return
	}
	ly.Prim.TextureFile.Val = idx
	if idx-1 < len(textureNames) {
		ly.Prim.TextureName = textureNames[idx-1]
	}
	ly.Prim.EmbeddedTexture = append([]byte(nil), textureRaw[idx-1]...)
}

func rebuildTextureSlotsFromDoc() {
	resetTextureSlots()
	if doc == nil {
		return
	}
	for i := range doc.Layers {
		ly := &doc.Layers[i]
		if len(ly.Prim.EmbeddedTexture) == 0 {
			continue
		}
		idx, ok := addTextureSlot(ly.Prim.TextureName, ly.Prim.EmbeddedTexture)
		if !ok {
			continue
		}
		ly.Prim.TextureFile.Val = idx
		if strings.TrimSpace(ly.Prim.TextureName) == "" && idx-1 < len(textureNames) {
			ly.Prim.TextureName = textureNames[idx-1]
		}
	}
}

func upscaleNearest(dst []byte, srcW, srcH, zoom int, src []byte) {
	if zoom <= 1 {
		copy(dst, src)
		return
	}
	dstW := srcW * zoom
	for y := 0; y < srcH; y++ {
		for x := 0; x < srcW; x++ {
			si := (y*srcW + x) * 4
			r, g, b, a := src[si], src[si+1], src[si+2], src[si+3]
			for oy := 0; oy < zoom; oy++ {
				dy := y*zoom + oy
				for ox := 0; ox < zoom; ox++ {
					dx := x*zoom + ox
					di := (dy*dstW + dx) * 4
					dst[di], dst[di+1], dst[di+2], dst[di+3] = r, g, b, a
				}
			}
		}
	}
}

func clampLayer(i int) int {
	if doc == nil || len(doc.Layers) == 0 {
		return 0
	}
	if i < 0 {
		return 0
	}
	if i >= len(doc.Layers) {
		return len(doc.Layers) - 1
	}
	return i
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func boolOrInt(v js.Value) int {
	switch v.Type() {
	case js.TypeBoolean:
		if v.Bool() {
			return 1
		}
		return 0
	default:
		if v.Int() != 0 {
			return 1
		}
		return 0
	}
}

func parseHexColor(s string) (color.RGBA, bool) {
	s = strings.TrimSpace(strings.TrimPrefix(s, "#"))
	if len(s) != 6 && len(s) != 8 {
		return color.RGBA{}, false
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return color.RGBA{}, false
	}
	if len(s) == 6 {
		return color.RGBA{
			R: uint8((v >> 16) & 0xFF),
			G: uint8((v >> 8) & 0xFF),
			B: uint8(v & 0xFF),
			A: 255,
		}, true
	}
	return color.RGBA{
		R: uint8((v >> 24) & 0xFF),
		G: uint8((v >> 16) & 0xFF),
		B: uint8((v >> 8) & 0xFF),
		A: uint8(v & 0xFF),
	}, true
}

func colorToHex(c color.RGBA) string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
