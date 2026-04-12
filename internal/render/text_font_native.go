//go:build !js || !wasm

package render

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	agg "github.com/cwbudde/agg_go"
	"knobman/internal/model"
)

var fontPathCache sync.Map

func loadAggTrueTypeFont(ctx *agg.Context, p *model.Primitive, size float64) bool {
	if ctx == nil || p == nil || size <= 0 {
		return false
	}

	fontPath := resolveFontPath(primitiveFontFamily(p), p.Bold.Val != 0, p.Italic.Val != 0)
	if fontPath == "" {
		return false
	}

	// We already resolve the concrete face file (for example Georgia_Bold_Italic.ttf),
	// so keep agg_go in plain mode here instead of asking it to synthesize styles.
	return ctx.Font(fontPath, size, false, false, agg.RasterFontCache, 0) == nil
}

func resolveFontPath(family string, bold, italic bool) string {
	key := strings.ToLower(strings.TrimSpace(family))

	key += "|"
	if bold {
		key += "b"
	}

	if italic {
		key += "i"
	}

	if cached, ok := fontPathCache.Load(key); ok {
		return cached.(string)
	}

	path := findFontPath(family, bold, italic)
	fontPathCache.Store(key, path)

	return path
}

func findFontPath(family string, bold, italic bool) string {
	family = strings.TrimSpace(family)
	if family == "" {
		family = "SansSerif"
	}

	if filepath.IsAbs(family) {
		if _, err := os.Stat(family); err == nil {
			return family
		}
	}

	if isJavaGenericFontFamily(family) {
		for _, name := range candidateFontFamilies(family) {
			if path := resolveFontWithFCMatchExact(name, name, bold, italic); path != "" {
				return path
			}
		}

		for _, path := range fallbackFontPaths(family) {
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}

		return ""
	}

	if path := resolveFontWithFCMatchExact(family, family, bold, italic); path != "" {
		return path
	}

	for _, name := range candidateFontFamilies("SansSerif") {
		if path := resolveFontWithFCMatchExact(name, name, bold, italic); path != "" {
			return path
		}
	}

	for _, path := range fallbackFontPaths("SansSerif") {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func isJavaGenericFontFamily(family string) bool {
	switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(family), " ", "")) {
	case "sansserif", "dialog", "serif", "monospaced", "dialoginput", "monospace":
		return true
	default:
		return false
	}
}

func candidateFontFamilies(family string) []string {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(family), " ", ""))
	switch normalized {
	case "sansserif", "dialog":
		return []string{"Noto Sans", "Helvetica", "Arial", "DejaVu Sans", "Liberation Sans"}
	case "serif":
		return []string{"Noto Serif", "Times New Roman", "Times", "DejaVu Serif", "Liberation Serif"}
	case "monospaced", "dialoginput", "monospace":
		return []string{"Noto Sans Mono", "Courier New", "Courier", "DejaVu Sans Mono", "Liberation Mono"}
	default:
		return []string{family}
	}
}

func resolveFontWithFCMatchExact(patternFamily, requestedFamily string, bold, italic bool) string {
	if _, err := exec.LookPath("fc-match"); err != nil {
		return ""
	}

	for _, pattern := range fontconfigPatterns(patternFamily, bold, italic) {
		out, err := exec.Command("fc-match", "-f", "%{family}\n%{file}\n", pattern).Output()
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

		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
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

func fontconfigPatterns(family string, bold, italic bool) []string {
	family = strings.TrimSpace(family)
	if family == "" {
		return nil
	}

	patterns := []string{family}

	switch {
	case bold && italic:
		patterns = append(patterns, family+":style=Bold Italic", family+":style=Bold Oblique")
	case bold:
		patterns = append(patterns, family+":style=Bold", family+":style=Semibold")
	case italic:
		patterns = append(patterns, family+":style=Italic", family+":style=Oblique")
	default:
		patterns = append(patterns, family+":style=Regular", family+":style=Book", family+":style=Roman")
	}

	return patterns
}

func fallbackFontPaths(family string) []string {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(family), " ", ""))

	switch runtime.GOOS {
	case "darwin":
		switch normalized {
		case "serif":
			return []string{
				"/System/Library/Fonts/Supplemental/Times New Roman.ttf",
				"/System/Library/Fonts/NewYork.ttf",
			}
		case "monospaced", "dialoginput", "monospace":
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
		case "serif":
			return []string{`C:\Windows\Fonts\times.ttf`, `C:\Windows\Fonts\timesbd.ttf`}
		case "monospaced", "dialoginput", "monospace":
			return []string{`C:\Windows\Fonts\consola.ttf`, `C:\Windows\Fonts\cour.ttf`}
		default:
			return []string{`C:\Windows\Fonts\arial.ttf`, `C:\Windows\Fonts\segoeui.ttf`}
		}
	default:
		switch normalized {
		case "serif":
			return []string{
				"/usr/share/fonts/truetype/noto/NotoSerif-Regular.ttf",
				"/usr/share/fonts/truetype/dejavu/DejaVuSerif.ttf",
				"/usr/share/fonts/truetype/liberation2/LiberationSerif-Regular.ttf",
			}
		case "monospaced", "dialoginput", "monospace":
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
