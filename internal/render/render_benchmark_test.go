package render

import (
	"path/filepath"
	"runtime"
	"testing"
)

func BenchmarkRenderFrameVU3Frame0(b *testing.B) {
	root := benchmarkRepoRoot(b)
	samplePath := filepath.Join(root, "assets", "samples", "vu3.knob")

	doc, textures, err := LoadParityDocument(samplePath, root)
	if err != nil {
		b.Fatalf("load sample: %v", err)
	}

	out := NewPixBuf(doc.Prefs.PWidth.Val, doc.Prefs.PHeight.Val)
	if out == nil {
		b.Fatal("allocate output buffer")
	}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		RenderFrame(out, doc, 0, textures)
	}
}

func benchmarkRepoRoot(b *testing.B) string {
	b.Helper()

	_, f, _, ok := runtime.Caller(0)
	if !ok {
		b.Fatal("runtime.Caller failed")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(f), "..", ".."))
}
