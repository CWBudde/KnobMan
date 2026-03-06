//go:build js && wasm

package main

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"syscall/js"

	"knobman/internal/fileio"
	"knobman/internal/model"
	"knobman/internal/render"
)

var (
	logicalW     = 64
	logicalH     = 64
	zoomFactor   = 8
	doc          *model.Document
	textures     []*render.Texture
	previewFrame int
	selectedLayer int
	renderBuf    *render.PixBuf
	displayBuf   []byte
)

func main() {
	newDocument()
	syncDisplayBuffer()

	js.Global().Set("knobman_init", js.FuncOf(jsInit))
	js.Global().Set("knobman_render", js.FuncOf(jsRender))
	js.Global().Set("knobman_getDimensions", js.FuncOf(jsGetDimensions))

	js.Global().Set("knobman_newDocument", js.FuncOf(jsNewDocument))
	js.Global().Set("knobman_getLayerList", js.FuncOf(jsGetLayerList))
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

	js.Global().Set("knobman_loadFile", js.FuncOf(jsLoadFile))
	js.Global().Set("knobman_saveFile", js.FuncOf(jsSaveFile))

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

func jsGetDimensions(this js.Value, args []js.Value) any {
	return map[string]any{
		"width":  logicalW * zoomFactor,
		"height": logicalH * zoomFactor,
	}
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
	doc.Layers[selectedLayer], doc.Layers[target] = doc.Layers[target], doc.Layers[selectedLayer]
	selectedLayer = target
	return selectedLayer
}

func jsDuplicateLayer(this js.Value, args []js.Value) any {
	if doc == nil || selectedLayer < 0 || selectedLayer >= len(doc.Layers) {
		return -1
	}
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
	idx := clampLayer(args[0].Int())
	doc.Layers[idx].Visible.Val = boolToInt(args[1].Bool())
	return true
}

func jsSetLayerSolo(this js.Value, args []js.Value) any {
	if doc == nil || len(args) < 2 {
		return false
	}
	idx := clampLayer(args[0].Int())
	doc.Layers[idx].Solo.Val = boolToInt(args[1].Bool())
	return true
}

func jsSetPrefs(this js.Value, args []js.Value) any {
	if doc == nil || len(args) < 1 || args[0].Type() != js.TypeObject {
		return false
	}
	obj := args[0]
	if v := obj.Get("width"); v.Truthy() {
		logicalW = maxInt(1, v.Int())
		doc.Prefs.PWidth.Val = logicalW
		doc.Prefs.Width = logicalW
	}
	if v := obj.Get("height"); v.Truthy() {
		logicalH = maxInt(1, v.Int())
		doc.Prefs.PHeight.Val = logicalH
		doc.Prefs.Height = logicalH
	}
	if v := obj.Get("frames"); v.Truthy() {
		doc.Prefs.RenderFrames.Val = maxInt(1, v.Int())
	}
	if v := obj.Get("oversampling"); v.Truthy() {
		doc.Prefs.Oversampling.Val = clampInt(v.Int(), 0, 3)
	}
	if v := obj.Get("exportOption"); v.Truthy() {
		doc.Prefs.ExportOption.Val = v.Int()
	}
	if v := obj.Get("bgColor"); v.Truthy() && v.Type() == js.TypeString {
		if c, ok := parseHexColor(v.String()); ok {
			doc.Prefs.BkColor.Val = c
		}
	}
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
		"width":       logicalW,
		"height":      logicalH,
		"frames":      doc.Prefs.RenderFrames.Val,
		"oversampling": doc.Prefs.Oversampling.Val,
		"exportOption": doc.Prefs.ExportOption.Val,
		"bgColor":     colorToHex(c),
	}
}

func jsSetParam(this js.Value, args []js.Value) any {
	if doc == nil || len(args) < 3 {
		return false
	}
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
			ly.Prim.Color.Val = c
		}
	case "text":
		ly.Prim.Text.Val = v.String()
	case "shape":
		ly.Prim.Shape.Val = v.String()
	case "fill":
		ly.Prim.Fill.Val = boolOrInt(v)
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
	case "text":
		return ly.Prim.Text.Val
	case "shape":
		return ly.Prim.Shape.Val
	case "fill":
		return ly.Prim.Fill.Val != 0
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
	syncDisplayBuffer()
	return true
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
	selectedLayer = 0
	previewFrame = 0
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
