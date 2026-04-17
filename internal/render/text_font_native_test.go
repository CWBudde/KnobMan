//go:build !js || !wasm

package render

import "testing"

func TestFamilyListMatchesRequested(t *testing.T) {
	t.Parallel()

	if !familyListMatchesRequested([]string{"Georgia", "Serif"}, "Georgia") {
		t.Fatal("expected exact family match to succeed")
	}

	if familyListMatchesRequested([]string{"Noto Sans"}, "WST_Engl") {
		t.Fatal("unexpected substitute family match")
	}
}

func TestParseFCMatchOutput(t *testing.T) {
	t.Parallel()

	families, path := parseFCMatchOutput("Georgia,Serif\n/usr/share/fonts/truetype/msttcorefonts/Georgia.ttf\n")
	if path != "/usr/share/fonts/truetype/msttcorefonts/Georgia.ttf" {
		t.Fatalf("path = %q", path)
	}

	if len(families) != 2 || families[0] != "Georgia" || families[1] != "Serif" {
		t.Fatalf("families = %#v", families)
	}
}

func TestFontconfigPatternsPreferRequestedStyles(t *testing.T) {
	t.Parallel()

	patterns := fontconfigPatterns("Georgia", true, true)
	if len(patterns) < 2 {
		t.Fatalf("expected multiple patterns, got %d", len(patterns))
	}

	if patterns[0].value != "Georgia:style=Bold Italic" {
		t.Fatalf("first pattern = %q, want bold italic style first", patterns[0].value)
	}

	if !patterns[0].realItalic {
		t.Fatal("first pattern should be marked as a real italic request")
	}

	if tail := patterns[len(patterns)-1].value; tail != "Georgia" {
		t.Fatalf("fallback family pattern = %q, want bare family last", tail)
	}
}

func TestIsJavaGenericFontFamily(t *testing.T) {
	t.Parallel()

	cases := map[string]bool{
		"SansSerif":  true,
		"Dialog":     true,
		"Serif":      true,
		"Monospaced": true,
		"WST_Engl":   false,
		"Georgia":    false,
	}

	for family, want := range cases {
		if got := isJavaGenericFontFamily(family); got != want {
			t.Fatalf("isJavaGenericFontFamily(%q) = %v, want %v", family, got, want)
		}
	}
}

func TestClassifyFamilyFallback(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"":                "SansSerif",
		"   ":             "SansSerif",
		"Arial":           "SansSerif",
		"Helvetica":       "SansSerif",
		"Verdana":         "SansSerif",
		"WST_Engl":        "Serif",
		"wst_germ":        "Serif",
		"Times New Roman": "Serif",
		"Georgia":         "Serif",
		"Palatino Linotype": "Serif",
		"Garamond":        "Serif",
		"Century Schoolbook": "Serif",
		"Courier New":     "Monospaced",
		"Consolas":        "Monospaced",
		"DejaVu Sans Mono": "Monospaced",
		"Source Code Pro": "Monospaced",
	}

	for family, want := range cases {
		if got := classifyFamilyFallback(family); got != want {
			t.Fatalf("classifyFamilyFallback(%q) = %q, want %q", family, got, want)
		}
	}
}
