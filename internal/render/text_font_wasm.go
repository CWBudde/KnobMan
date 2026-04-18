//go:build js && wasm

package render

import (
	"knobman/internal/model"
)

func loadAggTrueTypeFont(_ *model.Primitive, _ float64) loadedTrueTypeFont {
	return loadedTrueTypeFont{}
}
