package fileio

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
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

func TestLoadInheritsSharedEmbeddedAssets(t *testing.T) {
	// 1x1 PNG (opaque black)
	png := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
		0x42, 0x60, 0x82,
	}
	hexPNG := hex.EncodeToString(png)
	profile := strings.Join([]string{
		"[Prefs]",
		"Version=1490",
		"Layers=2",
		"OutputSizeX=64",
		"OutputSizeY=64",
		"RenderFrames=31",
		"PreviewFrames=5",
		"BkColorR=0",
		"BkColorG=0",
		"BkColorB=0",
		"Visible1_0=1",
		"Visible1_1=1",
		"[Layer1]",
		"Primitive=Image",
		"PrimFile=shared-image.png",
		"PrimTextureFile=shared-tex.png",
		"TexBmp0=" + hexPNG,
		"ImgBmp0=" + hexPNG,
		"[Layer2]",
		"Primitive=Image",
		"PrimFile=shared-image.png",
		"PrimTextureFile=shared-tex.png",
		"[End]",
		"",
	}, "\n")

	doc, err := loadDocument(parseINI([]byte(profile)))
	if err != nil {
		t.Fatalf("loadDocument: %v", err)
	}

	if len(doc.Layers) != 2 {
		t.Fatalf("layer count: want 2 got %d", len(doc.Layers))
	}

	l1 := doc.Layers[0].Prim
	l2 := doc.Layers[1].Prim

	if len(l1.EmbeddedTexture) == 0 || len(l1.EmbeddedImage) == 0 {
		t.Fatalf("layer1 embedded data missing: tex=%d img=%d", len(l1.EmbeddedTexture), len(l1.EmbeddedImage))
	}

	if !bytes.Equal(l2.EmbeddedTexture, l1.EmbeddedTexture) {
		t.Fatalf("layer2 texture did not inherit from layer1")
	}

	if !bytes.Equal(l2.EmbeddedImage, l1.EmbeddedImage) {
		t.Fatalf("layer2 image did not inherit from layer1")
	}
}

func TestLoadSavePreservesPrimFont(t *testing.T) {
	profile := strings.Join([]string{
		"[Prefs]",
		"Version=1490",
		"Layers=1",
		"OutputSizeX=64",
		"OutputSizeY=64",
		"[Layer1]",
		"Primitive=Text",
		"PrimFont=DejaVu Sans",
		"PrimText=TX",
		"[End]",
		"",
	}, "\n")

	doc, err := Load([]byte(profile))
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if got := doc.Layers[0].Prim.FontName; got != "DejaVu Sans" {
		t.Fatalf("FontName after load = %q", got)
	}

	saved, err := Save(doc)
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	if !strings.Contains(string(saved), "PrimFont=DejaVu Sans") {
		t.Fatalf("saved knob missing PrimFont, got:\n%s", saved)
	}
}

func TestLoadUsesPerLayerVisibleWhenVisible1FlagsAbsent(t *testing.T) {
	input := strings.Join([]string{
		"[Prefs]",
		"Layers=2",
		"OutputSizeX=32",
		"OutputSizeY=32",
		"[Layer1]",
		"Visible=0",
		"Primitive=None",
		"[Layer2]",
		"Visible=1",
		"Primitive=None",
	}, "\n")

	doc, err := Load([]byte(input))
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if got := doc.Layers[0].Visible.Val; got != 0 {
		t.Fatalf("layer1 visible = %d, want 0", got)
	}

	if got := doc.Layers[1].Visible.Val; got != 1 {
		t.Fatalf("layer2 visible = %d, want 1", got)
	}
}

func TestLoadFMaskStopLegacyDefault(t *testing.T) {
	profile := strings.Join([]string{
		"[Prefs]",
		"Version=1490",
		"Layers=1",
		"OutputSizeX=64",
		"OutputSizeY=64",
		"[Layer1]",
		"Primitive=None",
		"UseFMask=1",
		"FMaskStart=10",
		// FMaskStop intentionally omitted: Java loader default is 0.
		"[End]",
		"",
	}, "\n")

	doc, err := loadDocument(parseINI([]byte(profile)))
	if err != nil {
		t.Fatalf("loadDocument: %v", err)
	}

	got := doc.Layers[0].Eff.FMaskStop.Val
	if got != 0 {
		t.Fatalf("FMaskStop default mismatch: got %v want 0", got)
	}
}
