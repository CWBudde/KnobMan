package render

import (
	"strings"

	agg "github.com/cwbudde/agg_go"
	"knobman/internal/model"
)

func configureAggTextFont(ctx *agg.Context, p *model.Primitive, size float64) {
	if !loadAggTrueTypeFont(ctx, p, size) {
		ctx.GetAgg2D().FontGSV(size)
	}
}

func primitiveFontFamily(p *model.Primitive) string {
	if p == nil {
		return "SansSerif"
	}

	name := strings.TrimSpace(p.FontName)
	if name == "" {
		return "SansSerif"
	}

	return name
}
