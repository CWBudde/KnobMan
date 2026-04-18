//go:build js && wasm

package render

import (
	"testing"

	agg "github.com/cwbudde/agg_go"
	"knobman/internal/model"
)

func TestConfigureAggTextFontFallsBackToGSVOnWasm(t *testing.T) {
	ctx := agg.NewContext(64, 64)
	if ctx == nil {
		t.Fatal("nil ctx")
	}

	p := model.NewPrimitive()
	p.Type.Val = int(model.PrimText)
	p.FontName = "SansSerif"
	p.Italic.Val = 1

	fontCfg := configureAggTextFont(ctx, &p, 24)
	defer fontCfg.Close()

	if fontCfg.backend != aggTextBackendGSV {
		t.Fatalf("backend = %v, want %v", fontCfg.backend, aggTextBackendGSV)
	}

	if fontCfg.trueType != nil {
		t.Fatal("expected wasm text backend to avoid TrueType face loading")
	}

	if fontCfg.italic {
		t.Fatal("expected synthetic italic flag to stay disabled for GSV fallback")
	}
}
