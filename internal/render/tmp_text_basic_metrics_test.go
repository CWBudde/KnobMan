package render

import (
	"testing"

	"knobman/internal/model"
)

func TestTmpTextBasicMetrics(t *testing.T) {
	buf := NewPixBuf(64, 64)
	ctx := AggContextForPixBuf(buf)
	if ctx == nil {
		t.Fatal("nil ctx")
	}

	p := model.NewPrimitive()
	p.Type.Val = int(model.PrimText)
	p.FontSize.Val = 62
	p.Text.Val = "TX"
	p.TextAlign.Val = 1
	p.FontName = ""

	size := p.FontSize.Val * 0.01 * float64(buf.Height)
	backend, textSize := configureAggTextFont(ctx, &p, size)
	width := ctx.GetAgg2D().TextWidth("TX")
	space := ctx.GetAgg2D().TextWidth(" ")
	if backend == aggTextBackendGSV {
		width = measureLocalGSVTextWidth("TX", textSize)
		space = measureLocalGSVTextWidth(" ", textSize)
	}
	t.Logf("family=%q path=%q backend=%v inputSize=%v configuredSize=%v width=%v space=%v asc=%v desc=%v",
		primitiveFontFamily(&p),
		resolveFontPath(primitiveFontFamily(&p), false, false),
		backend, size, textSize, width, space,
		ctx.GetAscender(), ctx.GetDescender(),
	)
}
