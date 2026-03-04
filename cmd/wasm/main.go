//go:build js && wasm

package main

import (
	"syscall/js"

	agg "agg_go"
)

var (
	canvasWidth  = 64
	canvasHeight = 64
	zoomFactor   = 8 // display at 8× for a 64×64 canvas → 512×512 display
	ctx          *agg.Context
	canvasBuf    []uint8
)

func main() {
	ctx = agg.NewContext(canvasWidth*zoomFactor, canvasHeight*zoomFactor)
	canvasBuf = ctx.GetImage().Data

	js.Global().Set("knobman_init", js.FuncOf(jsInit))
	js.Global().Set("knobman_render", js.FuncOf(jsRender))
	js.Global().Set("knobman_getDimensions", js.FuncOf(jsGetDimensions))

	select {}
}

// jsInit initialises the canvas dimensions and resets the context.
func jsInit(this js.Value, args []js.Value) any {
	if len(args) >= 2 {
		canvasWidth = args[0].Int()
		canvasHeight = args[1].Int()
		if len(args) >= 3 {
			zoomFactor = args[2].Int()
		}
	}
	ctx = agg.NewContext(canvasWidth*zoomFactor, canvasHeight*zoomFactor)
	canvasBuf = ctx.GetImage().Data
	return nil
}

// jsRender renders the current document and copies pixels to the JS buffer.
func jsRender(this js.Value, args []js.Value) any {
	renderScene()

	if len(args) >= 1 {
		jsBuf := args[0]
		js.CopyBytesToJS(jsBuf, canvasBuf)
	}
	return nil
}

// jsGetDimensions returns the display canvas width and height (after zoom).
func jsGetDimensions(this js.Value, args []js.Value) any {
	return map[string]any{
		"width":  canvasWidth * zoomFactor,
		"height": canvasHeight * zoomFactor,
	}
}

// renderScene draws the current state of the document onto ctx.
// Phase 0: placeholder — clears to a checkerboard to prove agg_go works.
func renderScene() {
	dispW := canvasWidth * zoomFactor
	dispH := canvasHeight * zoomFactor

	ctx.Clear(agg.White)

	// Draw a checkerboard background to show transparency
	a2 := ctx.GetAgg2D()
	tileSize := float64(zoomFactor)
	for row := 0; row < canvasHeight; row++ {
		for col := 0; col < canvasWidth; col++ {
			if (row+col)%2 == 0 {
				a2.FillColor(agg.RGBA(0.8, 0.8, 0.8, 1))
			} else {
				a2.FillColor(agg.White)
			}
			a2.NoLine()
			x := float64(col) * tileSize
			y := float64(row) * tileSize
			a2.Rectangle(x, y, x+tileSize, y+tileSize)
			a2.DrawPath(agg.FillOnly)
		}
	}

	// Draw a label in the center
	a2.FillColor(agg.RGBA(0.2, 0.2, 0.8, 1))
	a2.NoLine()
	a2.FontGSV(float64(dispH) / 10)
	a2.TextAlignment(agg.AlignCenter, agg.AlignCenter)
	a2.Text(float64(dispW)/2, float64(dispH)/2, "KnobMan", false, 0, 0)
}
