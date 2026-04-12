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
