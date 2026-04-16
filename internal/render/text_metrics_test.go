package render

import (
	"testing"

	agg "github.com/cwbudde/agg_go"
	"knobman/internal/model"
)

func TestConfiguredTextFontProducesPositiveMetrics(t *testing.T) {
	ctx := agg.NewContext(64, 64)
	if ctx == nil {
		t.Fatal("nil ctx")
	}

	p := model.NewPrimitive()
	p.Type.Val = int(model.PrimText)
	p.FontSize.Val = 62
	p.Text.Val = "TX"
	p.TextAlign.Val = 1
	p.FontName = ""

	size := p.FontSize.Val * 0.01 * float64(ctx.Height())

	fontCfg := configureAggTextFont(ctx, &p, size)
	defer fontCfg.Close()

	if fontCfg.size <= 0 {
		t.Fatalf("configured text size must be positive, got %v", fontCfg.size)
	}

	var width, space float64

	switch fontCfg.backend {
	case aggTextBackendGSV:
		width = measureLocalGSVTextWidth("TX", fontCfg.size)
		space = measureLocalGSVTextWidth(" ", fontCfg.size)
	case aggTextBackendTrueType:
		if fontCfg.trueType == nil {
			t.Fatal("truetype backend selected without configured font source")
		}

		width = fontCfg.trueType.MeasureText("TX")

		space = fontCfg.trueType.MeasureText(" ")
		if asc := fontCfg.trueType.GetAscender(); asc <= 0 {
			t.Fatalf("expected positive ascender for truetype backend, got %v", asc)
		}

		if desc := fontCfg.trueType.GetDescender(); desc >= 0 {
			t.Fatalf("expected negative descender for truetype backend, got %v", desc)
		}
	default:
		t.Fatalf("unexpected text backend %v", fontCfg.backend)
	}

	if width <= 0 {
		t.Fatalf("expected positive text width, got %v", width)
	}

	if space < 0 {
		t.Fatalf("expected non-negative space width, got %v", space)
	}

	if width <= space {
		t.Fatalf("expected \"TX\" width (%v) to exceed space width (%v)", width, space)
	}
}
