package render

import (
	"strings"

	"knobman/internal/model"

	agg "github.com/cwbudde/agg_go"
)

type aggTextBackend int

const (
	aggTextBackendTrueType aggTextBackend = iota
	aggTextBackendGSV
)

type configuredAggTextFont struct {
	backend  aggTextBackend
	size     float64
	italic   bool
	trueType *agg.FreeTypeOutlineText
}

type loadedTrueTypeFont struct {
	face            *agg.FreeTypeOutlineText
	syntheticItalic bool
}

const fontFamilySansSerif = "SansSerif"

func (f configuredAggTextFont) Close() {
	if f.trueType != nil {
		_ = f.trueType.Close()
	}
}

func configureAggTextFont(_ *agg.Context, p *model.Primitive, size float64) configuredAggTextFont {
	if tt := loadAggTrueTypeFont(p, size); tt.face != nil {
		return configuredAggTextFont{
			backend:  aggTextBackendTrueType,
			size:     size,
			italic:   tt.syntheticItalic,
			trueType: tt.face,
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
		return fontFamilySansSerif
	}

	name := strings.TrimSpace(p.FontName)
	if name == "" {
		return fontFamilySansSerif
	}

	return name
}
