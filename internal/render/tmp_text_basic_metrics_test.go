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
	a := ctx.GetAgg2D()
	t.Logf("family=%q path=%q backend=%v inputSize=%v configuredSize=%v width=%v space=%v asc=%v desc=%v",
		primitiveFontFamily(&p),
		resolveFontPath(primitiveFontFamily(&p), false, false),
		backend, size, textSize, a.TextWidth("TX"), a.TextWidth(" "),
		ctx.GetAscender(), ctx.GetDescender(),
	)
}
