package render

import (
	"path/filepath"
	"testing"
)

// TestResolveTexturesForParityAnonymousEmbedded exercises the circle_fill_texture fixture
// where PrimTextureFile is empty but TexBmp0 carries embedded PNG bytes. The
// loader must still decode the embedded texture and wire TextureFile.Val so
// the renderer can sample it.
func TestResolveTexturesForParityAnonymousEmbedded(t *testing.T) {
	root := testRepoRoot(t)
	samplePath := filepath.Join(root, "tests", "parity", "primitives", "inputs", "circle_fill_texture.knob")

	doc, textures, err := LoadParityDocument(samplePath, root)
	if err != nil {
		t.Fatalf("LoadParityDocument: %v", err)
	}

	if len(textures) != 1 {
		t.Fatalf("expected 1 resolved texture, got %d", len(textures))
	}

	if doc == nil || len(doc.Layers) == 0 {
		t.Fatalf("expected at least one layer in loaded document")
	}

	ly := &doc.Layers[0]
	if ly.Prim.TextureFile.Val != 1 {
		t.Fatalf("expected layer TextureFile.Val=1, got %d", ly.Prim.TextureFile.Val)
	}

	if ly.Prim.TextureDepth.Val == 0 {
		t.Fatalf("expected non-zero TextureDepth for textured fixture")
	}
}
