package fileio

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadSamples verifies that every sample .knob file can be loaded
// without error and produces a document with at least one layer.
func TestLoadSamples(t *testing.T) {
	samples, err := filepath.Glob("../../assets/samples/*.knob")
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) == 0 {
		t.Fatal("no sample .knob files found in assets/samples/")
	}
	for _, path := range samples {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			doc, err := Load(data)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			if len(doc.Layers) == 0 {
				t.Error("document has zero layers")
			}
			if doc.Prefs.PWidth.Val <= 0 || doc.Prefs.PHeight.Val <= 0 {
				t.Errorf("invalid canvas size %dx%d", doc.Prefs.PWidth.Val, doc.Prefs.PHeight.Val)
			}
		})
	}
}

// TestRoundTrip loads each sample, saves it, and re-loads to verify no data
// is lost in the key structural fields.
func TestRoundTrip(t *testing.T) {
	samples, err := filepath.Glob("../../assets/samples/*.knob")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range samples {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			doc1, err := Load(data)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			saved, err := Save(doc1)
			if err != nil {
				t.Fatalf("save: %v", err)
			}
			doc2, err := Load(saved)
			if err != nil {
				t.Fatalf("reload: %v", err)
			}

			// Structural checks
			if len(doc1.Layers) != len(doc2.Layers) {
				t.Errorf("layer count: want %d got %d", len(doc1.Layers), len(doc2.Layers))
			}
			if doc1.Prefs.PWidth.Val != doc2.Prefs.PWidth.Val {
				t.Errorf("PWidth: want %d got %d", doc1.Prefs.PWidth.Val, doc2.Prefs.PWidth.Val)
			}
			if doc1.Prefs.PHeight.Val != doc2.Prefs.PHeight.Val {
				t.Errorf("PHeight: want %d got %d", doc1.Prefs.PHeight.Val, doc2.Prefs.PHeight.Val)
			}
			if doc1.Prefs.RenderFrames.Val != doc2.Prefs.RenderFrames.Val {
				t.Errorf("RenderFrames: want %d got %d", doc1.Prefs.RenderFrames.Val, doc2.Prefs.RenderFrames.Val)
			}
			for i := range doc1.Layers {
				if i >= len(doc2.Layers) {
					break
				}
				l1, l2 := doc1.Layers[i], doc2.Layers[i]
				if l1.Prim.Type.Val != l2.Prim.Type.Val {
					t.Errorf("layer[%d] PrimType: want %d got %d", i, l1.Prim.Type.Val, l2.Prim.Type.Val)
				}
				if l1.Eff.AngleF.Val != l2.Eff.AngleF.Val {
					t.Errorf("layer[%d] AngleF: want %v got %v", i, l1.Eff.AngleF.Val, l2.Eff.AngleF.Val)
				}
				if l1.Eff.AlphaF.Val != l2.Eff.AlphaF.Val {
					t.Errorf("layer[%d] AlphaF: want %v got %v", i, l1.Eff.AlphaF.Val, l2.Eff.AlphaF.Val)
				}
			}
		})
	}
}

