package render

import (
	"strings"

	agg "github.com/cwbudde/agg_go"
	"knobman/internal/model"
)

type aggTextBackend int

const (
	aggTextBackendTrueType aggTextBackend = iota
	aggTextBackendGSV
)

func configureAggTextFont(ctx *agg.Context, p *model.Primitive, size float64) (aggTextBackend, float64) {
	if loadAggTrueTypeFont(ctx, p, size) {
		return aggTextBackendTrueType, size
	}

	gsvSize := size * 0.65
	if gsvSize < 6 {
		gsvSize = 6
	}

	ctx.GetAgg2D().FontGSV(gsvSize)

	return aggTextBackendGSV, gsvSize
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
