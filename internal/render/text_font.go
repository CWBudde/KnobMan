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

type configuredAggTextFont struct {
	backend  aggTextBackend
	size     float64
	trueType *agg.FreeTypeOutlineText
}

func (f configuredAggTextFont) Close() {
	if f.trueType != nil {
		_ = f.trueType.Close()
	}
}

func configureAggTextFont(_ *agg.Context, p *model.Primitive, size float64) configuredAggTextFont {
	if tt := loadAggTrueTypeFont(p, size); tt != nil {
		return configuredAggTextFont{
			backend:  aggTextBackendTrueType,
			size:     size,
			trueType: tt,
		}
	}

	gsvSize := size * 0.65
	if gsvSize < 6 {
		gsvSize = 6
	}

	return configuredAggTextFont{
		backend: aggTextBackendGSV,
		size:    gsvSize,
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
