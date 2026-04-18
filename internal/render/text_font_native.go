//go:build !js || !wasm

package render

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"knobman/internal/model"

	agg "github.com/cwbudde/agg_go"
)

var fontPathCache sync.Map

const (
	fontFamilyDialog       = "dialog"
	fontFamilyDialogInput  = "dialoginput"
	fontFamilySansSerifKey = "sansserif"
	fontFamilySerif        = "serif"
	fontFamilyMonospaced   = "monospaced"
	fontFamilyMonospace    = "monospace"
)

type resolvedFontFace struct {
	path            string
	syntheticItalic bool
}

func loadAggTrueTypeFont(p *model.Primitive, size float64) loadedTrueTypeFont {
	if p == nil || size <= 0 {
		return loadedTrueTypeFont{}
	}

	resolved := resolveFontPath(primitiveFontFamily(p), p.Bold.Val != 0, p.Italic.Val != 0)
	if resolved.path == "" {
		return loadedTrueTypeFont{}
	}

	txt, err := agg.NewFreeTypeOutlineText()
	if err != nil {
		return loadedTrueTypeFont{}
	}

	txt.SetHinting(true)
	txt.SetFlip(true)

	err = txt.SetTrueTypeInterpreterVersion(35)
	if err != nil {
		_ = txt.Close()
		return loadedTrueTypeFont{}
	}

	txt.SetSize(size, 0)

	err = txt.LoadFont(resolved.path)
	if err != nil {
		_ = txt.Close()
		return loadedTrueTypeFont{}
	}

	return loadedTrueTypeFont{
		face:            txt,
		syntheticItalic: resolved.syntheticItalic,
	}
}

func resolveFontPath(family string, bold, italic bool) resolvedFontFace {
	key := strings.ToLower(strings.TrimSpace(family))

	key += "|"
	if bold {
		key += "b"
	}

	if italic {
		key += "i"
	}

	if cached, ok := fontPathCache.Load(key); ok {
		if v, ok := cached.(resolvedFontFace); ok {
			return v
		}
	}

	resolved := findFontPath(family, bold, italic)
	fontPathCache.Store(key, resolved)

	return resolved
}

func findFontPath(family string, bold, italic bool) resolvedFontFace {
	family = strings.TrimSpace(family)
	if family == "" {
		family = fontFamilySansSerif
	}

	if filepath.IsAbs(family) {
		_, err := os.Stat(family)
		if err == nil {
			return resolvedFontFace{
				path:            family,
				syntheticItalic: italic,
			}
		}
	}

	if isJavaGenericFontFamily(family) {
		for _, name := range candidateFontFamilies(family) {
			if resolved := resolveFontWithFCMatchExact(name, name, bold, italic); resolved.path != "" {
				return resolved
			}
		}

		for _, path := range fallbackFontPaths(family) {
			_, err := os.Stat(path)
			if err == nil {
				return resolvedFontFace{
					path:            path,
					syntheticItalic: italic,
				}
			}
		}

		return resolvedFontFace{}
	}

	if resolved := resolveFontWithFCMatchExact(family, family, bold, italic); resolved.path != "" {
		return resolved
	}

	fallbackFamily := classifyFamilyFallback(family)

	for _, name := range candidateFontFamilies(fallbackFamily) {
		if resolved := resolveFontWithFCMatchExact(name, name, bold, italic); resolved.path != "" {
			return resolved
		}
	}

	for _, path := range fallbackFontPaths(fallbackFamily) {
		_, err := os.Stat(path)
		if err == nil {
			return resolvedFontFace{
				path:            path,
				syntheticItalic: italic,
			}
		}
	}

	return resolvedFontFace{}
}

// classifyFamilyFallback picks a Java logical font family ("Serif",
// "Monospaced", or "SansSerif") to substitute for an unknown requested family.
// Mirrors what OpenJDK's AWT does on Linux: rather than always falling back to
// sans-serif the way fontconfig does, pick a family consistent with the
// requested name so baselines rendered by Java and Go agree in character.
func classifyFamilyFallback(family string) string {
	normalized := strings.ToLower(strings.TrimSpace(family))
	if normalized == "" {
		return fontFamilySansSerif
	}

	monoTokens := []string{
		"mono", "courier", "consolas", "consola", "menlo", "typewriter",
		"fixedsys", "terminal", "inconsolata", "source code",
	}
	for _, token := range monoTokens {
		if strings.Contains(normalized, token) {
			return "Monospaced"
		}
	}

	serifTokens := []string{
		"serif", "roman", "times", "georgia", "bookman", "palatino",
		"garamond", "cambria", "caslon", "baskerville", "didot", "elephant",
		"minion", "century", "wst_",
	}
	for _, token := range serifTokens {
		if strings.Contains(normalized, token) {
			return "Serif"
		}
	}

	return fontFamilySansSerif
}

func isJavaGenericFontFamily(family string) bool {
	switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(family), " ", "")) {
	case fontFamilySansSerifKey, fontFamilyDialog, fontFamilySerif, fontFamilyMonospaced, fontFamilyDialogInput, fontFamilyMonospace:
		return true
	default:
		return false
	}
}

func candidateFontFamilies(family string) []string {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(family), " ", ""))
	switch normalized {
	case fontFamilySansSerifKey, fontFamilyDialog:
		return []string{"Noto Sans", "Helvetica", "Arial", "DejaVu Sans", "Liberation Sans"}
	case fontFamilySerif:
		return []string{"Nimbus Roman", "Nimbus Roman No9 L", "Noto Serif", "Times New Roman", "Times", "DejaVu Serif", "Liberation Serif"}
	case fontFamilyMonospaced, fontFamilyDialogInput, fontFamilyMonospace:
		return []string{"Noto Sans Mono", "Courier New", "Courier", "DejaVu Sans Mono", "Liberation Mono"}
	default:
		return []string{family}
	}
}

func resolveFontWithFCMatchExact(patternFamily, requestedFamily string, bold, italic bool) resolvedFontFace {
	_, err := exec.LookPath("fc-match")
	if err != nil {
		return resolvedFontFace{}
	}

	for _, pattern := range fontconfigPatterns(patternFamily, bold, italic) {
		out, err := exec.Command("fc-match", "-f", "%{family}\n%{file}\n", pattern.value).Output()
		if err != nil {
			continue
		}

		families, path := parseFCMatchOutput(string(out))
		if path == "" || !familyListMatchesRequested(families, requestedFamily) {
			continue
		}

		if path == "" {
			continue
		}

		_, err = os.Stat(path)
		if err == nil {
			return resolvedFontFace{
				path:            path,
				syntheticItalic: italic && !pattern.realItalic,
			}
		}
	}

	return resolvedFontFace{}
}

func parseFCMatchOutput(out string) ([]string, string) {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 0 {
		return nil, ""
	}

	path := strings.TrimSpace(lines[len(lines)-1])
	if len(lines) == 1 {
		return nil, path
	}

	var families []string

	for family := range strings.SplitSeq(lines[0], ",") {
		name := strings.TrimSpace(family)
		if name != "" {
			families = append(families, name)
		}
	}

	return families, path
}

func familyListMatchesRequested(families []string, requested string) bool {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return false
	}

	for _, family := range families {
		if strings.EqualFold(strings.TrimSpace(family), requested) {
			return true
		}
	}

	return false
}

type fontconfigPattern struct {
	value      string
	realItalic bool
}

func fontconfigPatterns(family string, bold, italic bool) []fontconfigPattern {
	family = strings.TrimSpace(family)
	if family == "" {
		return nil
	}

	patterns := make([]fontconfigPattern, 0, 5)

	switch {
	case bold && italic:
		patterns = append(patterns,
			fontconfigPattern{value: family + ":style=Bold Italic", realItalic: true},
			fontconfigPattern{value: family + ":style=Bold Oblique", realItalic: true},
			fontconfigPattern{value: family + ":weight=bold:slant=italic", realItalic: true},
			fontconfigPattern{value: family + ":weight=bold:slant=oblique", realItalic: true},
		)
	case bold:
		patterns = append(patterns,
			fontconfigPattern{value: family + ":style=Bold"},
			fontconfigPattern{value: family + ":style=Semibold"},
			fontconfigPattern{value: family + ":weight=bold"},
		)
	case italic:
		patterns = append(patterns,
			fontconfigPattern{value: family + ":style=Italic", realItalic: true},
			fontconfigPattern{value: family + ":style=Oblique", realItalic: true},
			fontconfigPattern{value: family + ":slant=italic", realItalic: true},
			fontconfigPattern{value: family + ":slant=oblique", realItalic: true},
		)
	default:
		patterns = append(patterns, fontconfigPattern{value: family})
		patterns = append(patterns,
			fontconfigPattern{value: family + ":style=Regular"},
			fontconfigPattern{value: family + ":style=Book"},
			fontconfigPattern{value: family + ":style=Roman"},
		)
	}

	if bold || italic {
		patterns = append(patterns, fontconfigPattern{value: family})
	}

	return patterns
}

func fallbackFontPaths(family string) []string {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(family), " ", ""))

	switch runtime.GOOS {
	case "darwin":
		switch normalized {
		case fontFamilySerif:
			return []string{
				"/System/Library/Fonts/Supplemental/Times New Roman.ttf",
				"/System/Library/Fonts/NewYork.ttf",
			}
		case fontFamilyMonospaced, fontFamilyDialogInput, fontFamilyMonospace:
			return []string{
				"/System/Library/Fonts/Menlo.ttc",
				"/System/Library/Fonts/Courier.dfont",
			}
		default:
			return []string{
				"/System/Library/Fonts/Supplemental/Arial.ttf",
				"/System/Library/Fonts/Helvetica.ttc",
			}
		}
	case "windows":
		switch normalized {
		case fontFamilySerif:
			return []string{`C:\Windows\Fonts\times.ttf`, `C:\Windows\Fonts\timesbd.ttf`}
		case fontFamilyMonospaced, fontFamilyDialogInput, fontFamilyMonospace:
			return []string{`C:\Windows\Fonts\consola.ttf`, `C:\Windows\Fonts\cour.ttf`}
		default:
			return []string{`C:\Windows\Fonts\arial.ttf`, `C:\Windows\Fonts\segoeui.ttf`}
		}
	default:
		switch normalized {
		case fontFamilySerif:
			return []string{
				"/usr/share/fonts/truetype/noto/NotoSerif-Regular.ttf",
				"/usr/share/fonts/truetype/dejavu/DejaVuSerif.ttf",
				"/usr/share/fonts/truetype/liberation2/LiberationSerif-Regular.ttf",
			}
		case fontFamilyMonospaced, fontFamilyDialogInput, fontFamilyMonospace:
			return []string{
				"/usr/share/fonts/truetype/noto/NotoSansMono-Regular.ttf",
				"/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",
				"/usr/share/fonts/truetype/liberation2/LiberationMono-Regular.ttf",
			}
		default:
			return []string{
				"/usr/share/fonts/truetype/noto/NotoSans-Regular.ttf",
				"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
				"/usr/share/fonts/truetype/liberation2/LiberationSans-Regular.ttf",
			}
		}
	}
}
