//go:build js && wasm

package render

import (
	agg "github.com/cwbudde/agg_go"
	"knobman/internal/model"
)

func loadAggTrueTypeFont(_ *model.Primitive, _ float64) *agg.FreeTypeOutlineText {
	return nil
}
