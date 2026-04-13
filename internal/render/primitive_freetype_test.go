//go:build freetype

package render

import (
	"image/color"
	"testing"

	agg "github.com/cwbudde/agg_go"
	"knobman/internal/model"
)

func TestRenderTextFreeTypeStaysRoughlyCenteredVertically(t *testing.T) {
	buf := NewPixBuf(64, 64)
	p := basePrim(model.PrimText)
	p.Color.Val = color.RGBA{R: 28, G: 28, B: 28, A: 255}
	p.Text.Val = "TX"
	p.FontSize.Val = 62
	p.TextAlign.Val = 1
	p.FontName = "SansSerif"

	ctx := agg.NewContext(buf.Width, buf.Height)
	if ctx == nil {
		t.Fatal("nil ctx")
	}

	fontCfg := configureAggTextFont(ctx, &p, p.FontSize.Val*0.01*float64(buf.Height))
	defer fontCfg.Close()
	if fontCfg.backend != aggTextBackendTrueType {
		t.Skip("freetype tag enabled, but TrueType backend is unavailable on this machine")
	}

	RenderPrimitive(buf, &p, nil, 0, 1)

	minY, maxY, ok := nonTransparentBoundsY(buf)
	if !ok {
		t.Fatal("expected rendered text coverage")
	}

	midY := float64(minY+maxY) * 0.5
	if delta := midY - 32; delta < -4 || delta > 4 {
		t.Fatalf("text vertical midpoint = %.1f (bounds [%d,%d]), want near canvas center", midY, minY, maxY)
	}
}

func nonTransparentBoundsY(buf *PixBuf) (minY, maxY int, ok bool) {
	if buf == nil || buf.Width == 0 || buf.Height == 0 {
		return 0, 0, false
	}

	minY = buf.Height
	maxY = -1

	for y := range buf.Height {
		for x := range buf.Width {
			if buf.At(x, y).A == 0 {
				continue
			}

			if y < minY {
				minY = y
			}

			if y > maxY {
				maxY = y
			}

			ok = true
		}
	}

	if !ok {
		return 0, 0, false
	}

	return minY, maxY, true
}
