package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunPrimitiveFixturesWritesPrimitiveSuite(t *testing.T) {
	t.Parallel()

	outDir := t.TempDir()

	if err := runPrimitiveFixtures([]string{"-suite", "primitives", "-out", outDir, "-overwrite"}); err != nil {
		t.Fatalf("runPrimitiveFixtures: %v", err)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("ReadDir(%q): %v", outDir, err)
	}

	if len(entries) < 30 {
		t.Fatalf("expected primitive fixture run to create many .knob files, got %d", len(entries))
	}

	for _, name := range []string{"circle_fill_basic.knob", "text_basic_center.knob", "shape_outline_basic.knob"} {
		info, err := os.Stat(filepath.Join(outDir, name))
		if err != nil {
			t.Fatalf("expected output file %q: %v", name, err)
		}

		if info.Size() == 0 {
			t.Fatalf("expected output file %q to be non-empty", name)
		}
	}
}

func TestRunPrimitiveFixturesDoesNotOverwriteExistingFilesByDefault(t *testing.T) {
	t.Parallel()

	outDir := t.TempDir()
	target := filepath.Join(outDir, "circle_fill_basic.knob")
	original := []byte("keep me")
	if err := os.WriteFile(target, original, 0o600); err != nil {
		t.Fatalf("WriteFile(%q): %v", target, err)
	}

	if err := runPrimitiveFixtures([]string{"-suite", "primitives", "-out", outDir}); err != nil {
		t.Fatalf("runPrimitiveFixtures: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", target, err)
	}

	if string(got) != string(original) {
		t.Fatalf("expected existing fixture to remain unchanged, got %q want %q", got, original)
	}
}

func TestRunPrimitiveFixturesRejectsUnknownSuite(t *testing.T) {
	t.Parallel()

	if err := runPrimitiveFixtures([]string{"-suite", "unknown", "-out", t.TempDir()}); err == nil {
		t.Fatal("expected unknown suite to fail")
	}
}
