// Package fileio implements loading and saving of .knob project files.
//
// File format (v1.3.1+):
//   - 6-byte binary header: 'K' 'M' followed by 4-byte little-endian uint32
//     giving the byte offset of the text body (typically 6 for files without
//     embedded thumbnails, or len(thumbnail)+6 for files with a thumbnail).
//   - UTF-8 text body: INI-style key=value pairs grouped in [Section] blocks.
//     Lines are terminated with \r\n.  Floats use '.' as decimal separator.
//
// Embedded binary data (textures, images) is stored as hex-encoded strings
// split across multiple lines: key0=hex..., then continuation via ReadNext.
package fileio

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"knobman/internal/model"
)

// ── Low-level INI parser ─────────────────────────────────────────────────────

type iniFile struct {
	lines []string
}

func parseINI(text []byte) *iniFile {
	// Strip UTF-8 BOM (EF BB BF) or UTF-16LE BOM (FF FE) if present.
	if len(text) >= 3 && text[0] == 0xEF && text[1] == 0xBB && text[2] == 0xBF {
		text = text[3:]
	} else if len(text) >= 2 && text[0] == 0xFF && text[1] == 0xFE {
		text = text[2:]
	}

	raw := strings.ReplaceAll(string(text), "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")

	return &iniFile{lines: strings.Split(raw, "\n")}
}

// sectionStart returns the line index right after the [section] header,
// or -1 if not found.
func (f *iniFile) sectionStart(name string) int {
	header := "[" + name + "]"
	for i, l := range f.lines {
		if strings.HasPrefix(l, header) {
			return i + 1
		}
	}

	return -1
}

// readInt reads the first occurrence of key= in the given section.
func (f *iniFile) readInt(sec int, key string, def int) int {
	prefix := key + "="

	for i := sec; i < len(f.lines); i++ {
		l := f.lines[i]
		if strings.HasPrefix(l, "[") && i != sec {
			break
		}

		if strings.HasPrefix(l, prefix) {
			v, err := strconv.Atoi(strings.TrimSpace(l[len(prefix):]))
			if err != nil {
				return def
			}

			return v
		}
	}

	return def
}

// readFloat reads the first occurrence of key= as a float64.
// Handles comma-as-decimal-separator (some locales write "3,14").
func (f *iniFile) readFloat(sec int, key string, def float64) float64 {
	prefix := key + "="

	for i := sec; i < len(f.lines); i++ {
		l := f.lines[i]
		if strings.HasPrefix(l, "[") && i != sec {
			break
		}

		if strings.HasPrefix(l, prefix) {
			s := strings.ReplaceAll(l[len(prefix):], ",", ".")

			v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
			if err != nil {
				return def
			}

			return v
		}
	}

	return def
}

// readString reads the first occurrence of key= as a string value.
func (f *iniFile) readString(sec int, key, def string) string {
	prefix := key + "="

	for i := sec; i < len(f.lines); i++ {
		l := f.lines[i]
		if strings.HasPrefix(l, "[") && i != sec {
			break
		}

		if strings.HasPrefix(l, prefix) {
			return l[len(prefix):]
		}
	}

	return def
}

// extractBinary reassembles hex-encoded binary data stored as key0=hex, key1=hex….
func (f *iniFile) extractBinary(sec int, keyBase string) []byte {
	prefix0 := keyBase + "0="
	start := -1

	for i := sec; i < len(f.lines); i++ {
		if strings.HasPrefix(f.lines[i], "[") && i != sec {
			break
		}

		if strings.HasPrefix(f.lines[i], prefix0) {
			start = i
			break
		}
	}

	if start < 0 {
		return nil
	}
	var buf []byte

	for i := start; i < len(f.lines); i++ {
		l := f.lines[i]

		_, after, ok := strings.Cut(l, "=")
		if !ok {
			break
		}

		hexData := after
		if strings.HasPrefix(hexData, "[") || hexData == "" {
			break
		}

		b, err := hex.DecodeString(hexData)
		if err != nil {
			break
		}

		buf = append(buf, b...)
		// Java writes 256 hex bytes per line; fewer means it was the last line.
		if len(hexData) < 512 {
			break
		}
	}

	return buf
}

// readAnim replicates Control.ReadAnim: returns 0 if the animate flag is 0,
// otherwise returns curve index = curve key value + 1 (1-based).
func (f *iniFile) readAnim(sec int, animKey, curveKey string) int {
	if f.readInt(sec, animKey, 0) == 0 {
		return 0
	}

	return f.readInt(sec, curveKey, 1) + 1
}

// ── Public API ───────────────────────────────────────────────────────────────

// Load parses a .knob file from raw bytes and returns a Document.
func Load(data []byte) (*model.Document, error) {
	body, err := stripHeader(data)
	if err != nil {
		return nil, err
	}

	ini := parseINI(body)

	return loadDocument(ini)
}

// Save serialises a Document to .knob bytes (6-byte KM header + UTF-8 body).
func Save(doc *model.Document) ([]byte, error) {
	var sb strings.Builder
	saveDocument(&sb, doc)
	text := sb.String()

	body := []byte(text)
	out := make([]byte, 6+len(body))
	out[0] = 'K'
	out[1] = 'M'
	binary.LittleEndian.PutUint32(out[2:6], 6) // text body starts at offset 6
	copy(out[6:], body)

	return out, nil
}

// ── Header parsing ───────────────────────────────────────────────────────────

// stripHeader extracts the UTF-8 INI text body from a .knob file, handling
// three on-disk formats:
//
//  1. KM format (new, v1.3.1+): 'K','M' + 4-byte LE offset to text body.
//     Text body is encoded as UTF-16LE.
//
//  2. PNG format (old): a valid PNG file that embeds the profile in a
//     "tEXt" chunk with keyword "Comment". Text is plain UTF-8.
//
//  3. Plain INI: raw UTF-8 text, possibly with BOM.
func stripHeader(data []byte) ([]byte, error) {
	if len(data) < 2 {
		return nil, errors.New("knob: file too short")
	}

	switch {
	case data[0] == 'K' && data[1] == 'M':
		// KM header: KM + 4-byte LE offset pointing to text body.
		// The body encoding is detected by its second byte (matching Java's
		// InputStreamReader selection logic):
		//   second byte == 0 → UTF-16LE  (old Java saves)
		//   second byte != 0 → UTF-8     (our Go saves, and some Java saves)
		if len(data) < 6 {
			return nil, errors.New("knob: KM header too short")
		}

		offset := binary.LittleEndian.Uint32(data[2:6])
		if int(offset) > len(data) {
			return nil, fmt.Errorf("knob: header offset %d exceeds file length %d", offset, len(data))
		}

		body := data[offset:]
		if len(body) >= 2 && body[1] == 0x00 {
			return decodeUTF16LE(body)
		}

		return body, nil

	case data[0] == 0x89 && data[1] == 0x50:
		// PNG file — profile is in a tEXt chunk with keyword "Comment"
		return extractPNGProfile(data)

	case data[0] == 0xFF && data[1] == 0xFE:
		// Bare UTF-16LE BOM
		return decodeUTF16LE(data[2:])

	case data[0] == 0xEF && data[1] == 0xBB && len(data) >= 3 && data[2] == 0xBF:
		// UTF-8 BOM
		return data[3:], nil

	default:
		return data, nil
	}
}

// decodeUTF16LE converts a UTF-16LE byte slice to UTF-8.
// It skips a leading BOM (FF FE) if present.
func decodeUTF16LE(b []byte) ([]byte, error) {
	// Strip optional BOM
	if len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE {
		b = b[2:]
	}

	if len(b)%2 != 0 {
		b = b[:len(b)-1] // trim odd trailing byte
	}

	runes := make([]rune, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		r := rune(b[i]) | rune(b[i+1])<<8
		runes = append(runes, r)
	}

	return []byte(string(runes)), nil
}

// extractPNGProfile scans PNG chunks for a tEXt chunk with keyword "Comment"
// and returns its text value as UTF-8 bytes.
func extractPNGProfile(data []byte) ([]byte, error) {
	if len(data) < 8 {
		return nil, errors.New("knob: PNG too short")
	}
	// PNG signature is 8 bytes; first chunk starts at offset 8.
	pos := 8
	for pos+12 <= len(data) {
		chunkLen := int(binary.BigEndian.Uint32(data[pos:]))
		chunkType := data[pos+4 : pos+8]
		chunkData := data[pos+8 : pos+8+chunkLen]

		if string(chunkType) == "tEXt" {
			// tEXt: keyword\0text
			_, after, ok := bytes.Cut(chunkData, []byte{0})
			if !ok {
				return nil, errors.New("knob: malformed tEXt chunk")
			}
			// We don't check the keyword name — accept whatever is there
			return after, nil
		}

		if string(chunkType) == "IEND" {
			break
		}

		pos += 12 + chunkLen
	}

	return nil, errors.New("knob: no tEXt profile chunk found in PNG")
}

// ── Document load ────────────────────────────────────────────────────────────

func loadDocument(ini *iniFile) (*model.Document, error) {
	doc := model.NewDocument()

	sec := ini.sectionStart("Prefs")
	if sec < 0 {
		return nil, errors.New("knob: missing [Prefs] section")
	}

	// Canvas size
	doc.Prefs.PWidth.Val = ini.readInt(sec, "OutputSizeX", 64)
	doc.Prefs.PHeight.Val = ini.readInt(sec, "OutputSizeY", 64)
	doc.Prefs.Oversampling.Val = ini.readInt(sec, "OverSampling", 0)

	// Frames — support old single-field format (NumOfImage) and new split format
	numOfImage := ini.readInt(sec, "NumOfImage", 0)
	if numOfImage > 0 {
		doc.Prefs.RenderFrames.Val = numOfImage
		doc.Prefs.PreviewFrames.Val = numOfImage
	} else {
		doc.Prefs.RenderFrames.Val = ini.readInt(sec, "RenderFrames", 31)
		doc.Prefs.PreviewFrames.Val = ini.readInt(sec, "PreviewFrames", 5)
	}

	doc.Prefs.AlignHorz.Val = ini.readInt(sec, "AlignHorizontal", 0)

	// Background colour
	r := ini.readInt(sec, "BkColorR", 255)
	g := ini.readInt(sec, "BkColorG", 255)
	b := ini.readInt(sec, "BkColorB", 255)
	doc.Prefs.BkColor.Val = color.RGBA{uint8(r), uint8(g), uint8(b), 255}

	// Export / animation settings
	doc.Prefs.ExportOption.Val = ini.readInt(sec, "ExportOption", 0)
	doc.Prefs.Duration.Val = ini.readInt(sec, "Duration", 100)
	doc.Prefs.Loop.Val = ini.readInt(sec, "Loop", 0)
	doc.Prefs.BiDir.Val = ini.readInt(sec, "BiDir", 0)

	// AnimCurves — the Java index mapping for points 1..10 uses hex chars
	// for points 5–10: a,b,c,d,e,f (see Control.java SaveExec)
	pointKeys := []string{"1", "2", "3", "4", "a", "b", "c", "d", "e", "f"}

	for ci := range 8 {
		n := ci + 1
		prefix := fmt.Sprintf("Curve%d", n)

		doc.Curves[ci].Lv[0] = ini.readInt(sec, prefix+"L0", 0)
		for j, pk := range pointKeys {
			doc.Curves[ci].Tm[j+1] = ini.readInt(sec, prefix+"T"+pk, -1)
			doc.Curves[ci].Lv[j+1] = ini.readInt(sec, prefix+"L"+pk, -1)
		}

		doc.Curves[ci].Lv[11] = ini.readInt(sec, prefix+"L5", 100)
		doc.Curves[ci].Tm[0] = 0
		doc.Curves[ci].Tm[11] = 100
		doc.Curves[ci].StepReso.Val = ini.readInt(sec, prefix+"StepReso", 0)
	}

	// Layer count and per-layer visibility pre-read
	nLayers := ini.readInt(sec, "Layers", 1)
	doc.Layers = make([]model.Layer, nLayers)

	visFlags := make([]int, nLayers)
	for i := range visFlags {
		visFlags[i] = ini.readInt(sec, fmt.Sprintf("Visible1_%d", i), -1)
	}

	for i := range doc.Layers {
		doc.Layers[i] = model.NewLayer()

		doc.Layers[i].Visible.Val = -1
		if visFlags[i] >= 0 {
			doc.Layers[i].Visible.Val = visFlags[i]
		}
	}

	// Load each layer
	for i := range doc.Layers {
		err := loadLayer(ini, &doc.Layers[i], i)
		if err != nil {
			return nil, fmt.Errorf("knob: layer %d: %w", i+1, err)
		}
	}

	linkSharedEmbeddedAssets(doc)

	return doc, nil
}

// linkSharedEmbeddedAssets mirrors legacy Java loading behavior:
// if a layer references the same texture/file name as a previous layer and
// does not carry embedded bytes itself, inherit the previous embedded data.
func linkSharedEmbeddedAssets(doc *model.Document) {
	if doc == nil {
		return
	}

	texByName := make(map[string][]byte)
	imgByFile := make(map[string][]byte)

	for i := range doc.Layers {
		ly := &doc.Layers[i]

		texName := strings.TrimSpace(ly.Prim.TextureName)
		if texName != "" {
			if len(ly.Prim.EmbeddedTexture) > 0 {
				texByName[texName] = append([]byte(nil), ly.Prim.EmbeddedTexture...)
			} else if prev, ok := texByName[texName]; ok {
				ly.Prim.EmbeddedTexture = append([]byte(nil), prev...)
			}
		}

		fileName := strings.TrimSpace(ly.Prim.File.Val)
		if fileName != "" {
			if len(ly.Prim.EmbeddedImage) > 0 {
				imgByFile[fileName] = append([]byte(nil), ly.Prim.EmbeddedImage...)
			} else if prev, ok := imgByFile[fileName]; ok {
				ly.Prim.EmbeddedImage = append([]byte(nil), prev...)
			}
		}
	}
}

func loadLayer(ini *iniFile, ly *model.Layer, idx int) error {
	sec := ini.sectionStart(fmt.Sprintf("Layer%d", idx+1))
	if sec < 0 {
		return fmt.Errorf("missing [Layer%d] section", idx+1)
	}

	ly.Name = ini.readString(sec, "Name", "")
	if ly.Visible.Val < 0 {
		ly.Visible.Val = ini.readInt(sec, "Visible", 1)
	}

	ly.Solo.Val = ini.readInt(sec, "VisibleSolo", 0)

	// Primitive type
	typeName := ini.readString(sec, "Primitive", "None")
	ly.Prim.Type.Val = int(model.PrimTypeFromName(typeName))

	// Primitive colour
	cr := ini.readInt(sec, "ColR", 255)
	cg := ini.readInt(sec, "ColG", 0)
	cb := ini.readInt(sec, "ColB", 0)
	ly.Prim.Color.Val = color.RGBA{uint8(cr), uint8(cg), uint8(cb), 255}

	// Primitive shape parameters
	ly.Prim.Aspect.Val = ini.readFloat(sec, "PrimAspect", 0)
	ly.Prim.Round.Val = ini.readFloat(sec, "PrimRound", 0)
	ly.Prim.Width.Val = ini.readFloat(sec, "PrimWidth", 10)
	ly.Prim.Length.Val = ini.readFloat(sec, "PrimLength", 50)
	ly.Prim.Step.Val = ini.readFloat(sec, "PrimStep", 20)
	ly.Prim.AngleStep.Val = ini.readFloat(sec, "PrimAngleStep", 45)
	ly.Prim.Emboss.Val = ini.readFloat(sec, "PrimEmboss", 0)
	ly.Prim.EmbossDiffuse.Val = ini.readFloat(sec, "PrimEmbossDiffuse", 0)
	ly.Prim.Ambient.Val = ini.readFloat(sec, "PrimAmbient", 50)
	ly.Prim.LightDir.Val = ini.readFloat(sec, "PrimLightDir", -50)
	ly.Prim.Specular.Val = ini.readFloat(sec, "PrimSpecular", 0)
	ly.Prim.SpecularWidth.Val = ini.readFloat(sec, "PrimSpecWidth", 50)
	ly.Prim.TextureName = ini.readString(sec, "PrimTextureFile", "")
	ly.Prim.TextureDepth.Val = ini.readFloat(sec, "PrimTexture", 0)
	ly.Prim.TextureZoom.Val = ini.readFloat(sec, "PrimTexZoom", 100)
	ly.Prim.Diffuse.Val = ini.readFloat(sec, "PrimDiffuse", 0)
	ly.Prim.FontName = ini.readString(sec, "PrimFont", "SansSerif")
	ly.Prim.FontSize.Val = ini.readFloat(sec, "PrimSize", 50)
	ly.Prim.Bold.Val = ini.readInt(sec, "PrimBold", 0)
	ly.Prim.Italic.Val = ini.readInt(sec, "PrimItalic", 0)
	ly.Prim.TextAlign.Val = ini.readInt(sec, "PrimTextAlign", 0)
	ly.Prim.Transparent.Val = ini.readInt(sec, "Transparent", 0)
	ly.Prim.AutoFit.Val = ini.readInt(sec, "AutoFit", 1)
	ly.Prim.IntelliAlpha.Val = ini.readInt(sec, "IntelliAlpha", 0)
	ly.Prim.NumFrame.Val = ini.readInt(sec, "PrimFrames", 1)
	ly.Prim.FrameAlign.Val = ini.readInt(sec, "PrimAlign", 0)
	ly.Prim.Fill.Val = ini.readInt(sec, "PrimFill", 1)
	ly.Prim.File.Val = ini.readString(sec, "PrimFile", "")
	ly.Prim.Text.Val = ini.readString(sec, "PrimText", "")
	ly.Prim.Shape.Val = ini.readString(sec, "PrimShape", "")

	// Embedded binary data
	ly.Prim.EmbeddedTexture = ini.extractBinary(sec, "TexBmp")
	ly.Prim.EmbeddedImage = ini.extractBinary(sec, "ImgBmp")

	// Effect stack
	e := &ly.Eff
	e.AntiAlias.Val = ini.readInt(sec, "Antialias", 1)
	e.Unfold.Val = ini.readInt(sec, "Unfold", 0)
	e.AnimStep.Val = ini.readInt(sec, "AnimStep", 0)
	e.ZoomXYSepa.Val = ini.readInt(sec, "ZoomXYSepa", 0)
	e.ZoomXF.Val = ini.readFloat(sec, "Zoom1", 100)
	e.ZoomXT.Val = ini.readFloat(sec, "Zoom2", 100)
	e.ZoomXAnim.Val = ini.readAnim(sec, "AnimateZoom", "ZoomCurve")
	e.ZoomYF.Val = ini.readFloat(sec, "ZoomY1", 100)
	e.ZoomYT.Val = ini.readFloat(sec, "ZoomY2", 100)
	e.ZoomYAnim.Val = ini.readAnim(sec, "AnimateZoomY", "ZoomYCurve")
	e.OffXF.Val = ini.readFloat(sec, "LayerOffsetX1", 0)
	e.OffXT.Val = ini.readFloat(sec, "LayerOffsetX2", 0)
	e.OffXAnim.Val = ini.readAnim(sec, "AnimateLayerOffsetX", "LayerOffsetXCurve")
	e.OffYF.Val = ini.readFloat(sec, "LayerOffsetY1", 0)
	e.OffYT.Val = ini.readFloat(sec, "LayerOffsetY2", 0)
	e.OffYAnim.Val = ini.readAnim(sec, "AnimateLayerOffsetY", "LayerOffsetYCurve")
	e.KeepDir.Val = ini.readInt(sec, "KeepDir", 0)
	e.CenterX.Val = ini.readFloat(sec, "RotCenterX1", 0)
	e.CenterY.Val = ini.readFloat(sec, "RotCenterY1", 0)
	e.AngleF.Val = ini.readFloat(sec, "Angle1", 0)
	e.AngleT.Val = ini.readFloat(sec, "Angle2", 0)
	e.AngleAnim.Val = ini.readAnim(sec, "AnimateAngle", "AngleCurve")
	e.AlphaF.Val = ini.readFloat(sec, "Alpha1", 100)
	e.AlphaT.Val = ini.readFloat(sec, "Alpha2", 100)
	e.AlphaAnim.Val = ini.readAnim(sec, "AnimateAlpha", "AlphaCurve")
	e.BrightF.Val = ini.readFloat(sec, "Brightness1", 0)
	e.BrightT.Val = ini.readFloat(sec, "Brightness2", 0)
	e.BrightAnim.Val = ini.readAnim(sec, "AnimateBrightness", "BrightnessCurve")
	e.ContrastF.Val = ini.readFloat(sec, "Contrast1", 0)
	e.ContrastT.Val = ini.readFloat(sec, "Contrast2", 0)
	e.ContrastAnim.Val = ini.readAnim(sec, "AnimateContrast", "ContrastCurve")
	e.SaturationF.Val = ini.readFloat(sec, "Saturation1", 0)
	e.SaturationT.Val = ini.readFloat(sec, "Saturation2", 0)
	e.SaturationAnim.Val = ini.readAnim(sec, "AnimateSaturation", "SaturationCurve")
	e.HueF.Val = ini.readFloat(sec, "Hue1", 0)
	e.HueT.Val = ini.readFloat(sec, "Hue2", 0)
	e.HueAnim.Val = ini.readAnim(sec, "AnimateHue", "HueCurve")
	e.Mask1Ena.Val = ini.readInt(sec, "UseMask", 0)
	e.Mask1Type.Val = ini.readInt(sec, "MaskType", 0)
	e.Mask1Grad.Val = ini.readFloat(sec, "MaskGradation", 0)
	e.Mask1GradDir.Val = ini.readInt(sec, "MaskGradDir", 0)
	e.Mask1StartF.Val = ini.readFloat(sec, "MaskStart1", -140)
	e.Mask1StartT.Val = ini.readFloat(sec, "MaskStart2", -140)
	e.Mask1StartAnim.Val = ini.readAnim(sec, "AnimateMaskStart", "MaskStartCurve")
	e.Mask1StopF.Val = ini.readFloat(sec, "MaskStop1", 140)
	e.Mask1StopT.Val = ini.readFloat(sec, "MaskStop2", 140)
	e.Mask1StopAnim.Val = ini.readAnim(sec, "AnimateMaskStop", "MaskStopCurve")
	e.Mask2Ena.Val = ini.readInt(sec, "UseMask2", 0)
	e.Mask2Op.Val = ini.readInt(sec, "Mask2Operation", 0)
	e.Mask2Type.Val = ini.readInt(sec, "Mask2Type", 0)
	e.Mask2Grad.Val = ini.readFloat(sec, "Mask2Gradation", 0)
	e.Mask2GradDir.Val = ini.readInt(sec, "Mask2GradDir", 0)
	e.Mask2StartF.Val = ini.readFloat(sec, "Mask2Start1", -140)
	e.Mask2StartT.Val = ini.readFloat(sec, "Mask2Start2", -140)
	e.Mask2StartAnim.Val = ini.readAnim(sec, "AnimateMask2Start", "Mask2StartCurve")
	e.Mask2StopF.Val = ini.readFloat(sec, "Mask2Stop1", 140)
	e.Mask2StopT.Val = ini.readFloat(sec, "Mask2Stop2", 140)
	e.Mask2StopAnim.Val = ini.readAnim(sec, "AnimateMask2Stop", "Mask2StopCurve")
	e.FMaskEna.Val = ini.readInt(sec, "UseFMask", 0)
	e.FMaskStart.Val = ini.readFloat(sec, "FMaskStart", 0)
	e.FMaskStop.Val = ini.readFloat(sec, "FMaskStop", 0)
	e.FMaskBits.Val = ini.readString(sec, "FMaskBits", "")
	e.SLightDirF.Val = ini.readFloat(sec, "LightDir", -45)
	e.SLightDirT.Val = ini.readFloat(sec, "LightDir2", -45)
	e.SLightDirAnim.Val = ini.readAnim(sec, "AnimateLightDir", "LightDirCurve")
	e.SDensityF.Val = ini.readFloat(sec, "Lighting", 0)
	e.SDensityT.Val = ini.readFloat(sec, "Lighting2", 0)
	e.SDensityAnim.Val = ini.readAnim(sec, "AnimateLighting", "LightingCurve")
	e.DLightDirEna.Val = ini.readInt(sec, "LightDirDEn", 0)
	e.DLightDirF.Val = ini.readFloat(sec, "LightDirD", -45)
	e.DLightDirT.Val = ini.readFloat(sec, "LightDirD2", -45)
	e.DLightDirAnim.Val = ini.readAnim(sec, "AnimateLightDirD", "LightDirDCurve")
	e.DOffsetF.Val = ini.readFloat(sec, "ShadowOffset1", 5)
	e.DOffsetT.Val = ini.readFloat(sec, "ShadowOffset2", 5)
	e.DOffsetAnim.Val = ini.readAnim(sec, "AnimateShadowOffset", "ShadowOffsetCurve")
	e.DDensityF.Val = ini.readFloat(sec, "ShadowDensity1", 0)
	e.DDensityT.Val = ini.readFloat(sec, "ShadowDensity2", 0)
	e.DDensityAnim.Val = ini.readAnim(sec, "AnimateShadowDensity", "ShadowDensityCurve")
	e.DDiffuseF.Val = ini.readFloat(sec, "ShadowDiffuse1", 0)
	e.DDiffuseT.Val = ini.readFloat(sec, "ShadowDiffuse2", 0)
	e.DDiffuseAnim.Val = ini.readAnim(sec, "AnimateShadowDiffuse", "ShadowDiffuseCurve")
	e.DSType.Val = ini.readInt(sec, "ShadowType", 0)
	e.DSGrad.Val = ini.readFloat(sec, "ShadowGradate", 100)
	e.ILightDirEna.Val = ini.readInt(sec, "LightDirIEn", 0)
	e.ILightDirF.Val = ini.readFloat(sec, "LightDirI", -45)
	e.ILightDirT.Val = ini.readFloat(sec, "LightDirI2", -45)
	e.ILightDirAnim.Val = ini.readAnim(sec, "AnimateLightDirI", "LightDirICurve")
	e.IOffsetF.Val = ini.readFloat(sec, "IShadowOffset1", 5)
	e.IOffsetT.Val = ini.readFloat(sec, "IShadowOffset2", 5)
	e.IOffsetAnim.Val = ini.readAnim(sec, "AnimateIShadowOffset", "IShadowOffsetCurve")
	e.IDensityF.Val = ini.readFloat(sec, "IShadowDensity1", 0)
	e.IDensityT.Val = ini.readFloat(sec, "IShadowDensity2", 0)
	e.IDensityAnim.Val = ini.readAnim(sec, "AnimateIShadowDensity", "IShadowDensityCurve")
	e.IDiffuseF.Val = ini.readFloat(sec, "IShadowDiffuse1", 20)
	e.IDiffuseT.Val = ini.readFloat(sec, "IShadowDiffuse2", 20)
	e.IDiffuseAnim.Val = ini.readAnim(sec, "AnimateIShadowDiffuse", "IShadowDiffuseCurve")
	e.ELightDirEna.Val = ini.readInt(sec, "LightDirEEn", 0)
	e.ELightDirF.Val = ini.readFloat(sec, "LightDirE", -45)
	e.ELightDirT.Val = ini.readFloat(sec, "LightDirE2", -45)
	e.ELightDirAnim.Val = ini.readAnim(sec, "AnimateLightDirE", "LightDirECurve")
	e.EOffsetF.Val = ini.readFloat(sec, "HilightOffset1", 0)
	e.EOffsetT.Val = ini.readFloat(sec, "HilightOffset2", 0)
	e.EOffsetAnim.Val = ini.readAnim(sec, "AnimateHilightOffset", "HilightOffsetCurve")
	e.EDensityF.Val = ini.readFloat(sec, "HilightDensity1", 0)
	e.EDensityT.Val = ini.readFloat(sec, "HilightDensity2", 0)
	e.EDensityAnim.Val = ini.readAnim(sec, "AnimateHilightDensity", "HilightDensityCurve")

	return nil
}

// ── Document save ────────────────────────────────────────────────────────────

func writeStr(sb *strings.Builder, key, val string)   { sb.WriteString(key + "=" + val + "\r\n") }
func writeInt(sb *strings.Builder, key string, v int) { writeStr(sb, key, strconv.Itoa(v)) }
func writeFloat(sb *strings.Builder, key string, v float64) {
	writeStr(sb, key, strconv.FormatFloat(v, 'f', -1, 64))
}
func section(sb *strings.Builder, name string) { sb.WriteString("[" + name + "]\r\n") }
func writeAnim(sb *strings.Builder, k1, k2, kAnimate, kCurve string, from, to float64, anim int) {
	writeFloat(sb, k1, from)
	writeFloat(sb, k2, to)

	if anim != 0 {
		writeInt(sb, kAnimate, 1)
		writeInt(sb, kCurve, anim-1)
	} else {
		writeInt(sb, kAnimate, 0)
		writeInt(sb, kCurve, 0)
	}
}

func saveDocument(sb *strings.Builder, doc *model.Document) {
	section(sb, "Prefs")
	writeStr(sb, "Version", "1490")
	writeInt(sb, "Layers", len(doc.Layers))
	writeInt(sb, "CurrentLayer", 1)
	writeInt(sb, "OutputSizeX", doc.Prefs.PWidth.Val)
	writeInt(sb, "OutputSizeY", doc.Prefs.PHeight.Val)
	writeInt(sb, "OverSampling", doc.Prefs.Oversampling.Val)
	writeInt(sb, "RenderFrames", doc.Prefs.RenderFrames.Val)
	writeInt(sb, "PreviewFrames", doc.Prefs.PreviewFrames.Val)
	writeInt(sb, "AlignHorizontal", doc.Prefs.AlignHorz.Val)
	writeInt(sb, "BkColorR", int(doc.Prefs.BkColor.Val.R))
	writeInt(sb, "BkColorG", int(doc.Prefs.BkColor.Val.G))
	writeInt(sb, "BkColorB", int(doc.Prefs.BkColor.Val.B))
	writeInt(sb, "ExportOption", doc.Prefs.ExportOption.Val)
	writeInt(sb, "Duration", doc.Prefs.Duration.Val)
	writeInt(sb, "Loop", doc.Prefs.Loop.Val)
	writeInt(sb, "BiDir", doc.Prefs.BiDir.Val)

	for i, ly := range doc.Layers {
		writeInt(sb, fmt.Sprintf("Visible1_%d", i), ly.Visible.Val)
	}

	pointKeys := []string{"1", "2", "3", "4", "a", "b", "c", "d", "e", "f"}

	for ci := range 8 {
		n := ci + 1
		c := &doc.Curves[ci]
		prefix := fmt.Sprintf("Curve%d", n)
		writeInt(sb, prefix+"L0", c.Lv[0])

		for j, pk := range pointKeys {
			writeInt(sb, prefix+"T"+pk, c.Tm[j+1])
			writeInt(sb, prefix+"L"+pk, c.Lv[j+1])
		}

		writeInt(sb, prefix+"L5", c.Lv[11])
		writeInt(sb, prefix+"StepReso", c.StepReso.Val)
	}

	for i := range doc.Layers {
		saveLayer(sb, &doc.Layers[i], i)
	}

	sb.WriteString("[End]\r\n")
}

func saveLayer(sb *strings.Builder, ly *model.Layer, idx int) {
	section(sb, fmt.Sprintf("Layer%d", idx+1))
	writeStr(sb, "Name", ly.Name)
	writeInt(sb, "Visible", ly.Visible.Val)
	writeInt(sb, "VisibleSolo", ly.Solo.Val)
	writeInt(sb, "ColR", int(ly.Prim.Color.Val.R))
	writeInt(sb, "ColG", int(ly.Prim.Color.Val.G))
	writeInt(sb, "ColB", int(ly.Prim.Color.Val.B))
	writeStr(sb, "Primitive", model.PrimTypeName(model.PrimitiveType(ly.Prim.Type.Val)))
	writeInt(sb, "PrimFill", ly.Prim.Fill.Val)
	writeFloat(sb, "PrimAspect", ly.Prim.Aspect.Val)
	writeFloat(sb, "PrimWidth", ly.Prim.Width.Val)
	writeFloat(sb, "PrimRound", ly.Prim.Round.Val)
	writeFloat(sb, "PrimLength", ly.Prim.Length.Val)
	writeFloat(sb, "PrimStep", ly.Prim.Step.Val)
	writeFloat(sb, "PrimAngleStep", ly.Prim.AngleStep.Val)
	writeFloat(sb, "PrimEmboss", ly.Prim.Emboss.Val)
	writeFloat(sb, "PrimEmbossDiffuse", ly.Prim.EmbossDiffuse.Val)
	writeStr(sb, "PrimTextureFile", ly.Prim.TextureName)
	writeFloat(sb, "PrimTexture", ly.Prim.TextureDepth.Val)
	writeFloat(sb, "PrimTexZoom", ly.Prim.TextureZoom.Val)
	writeFloat(sb, "PrimDiffuse", ly.Prim.Diffuse.Val)

	fontName := strings.TrimSpace(ly.Prim.FontName)
	if fontName == "" {
		fontName = "SansSerif"
	}

	writeStr(sb, "PrimFont", fontName)
	writeFloat(sb, "PrimAmbient", ly.Prim.Ambient.Val)
	writeFloat(sb, "PrimSpecWidth", ly.Prim.SpecularWidth.Val)
	writeFloat(sb, "PrimSpecular", ly.Prim.Specular.Val)
	writeFloat(sb, "PrimLightDir", ly.Prim.LightDir.Val)
	writeFloat(sb, "PrimSize", ly.Prim.FontSize.Val)
	writeInt(sb, "PrimBold", ly.Prim.Bold.Val)
	writeInt(sb, "PrimItalic", ly.Prim.Italic.Val)
	writeInt(sb, "PrimTextAlign", ly.Prim.TextAlign.Val)
	writeStr(sb, "PrimText", ly.Prim.Text.Val)
	writeStr(sb, "PrimFile", ly.Prim.File.Val)
	writeStr(sb, "PrimShape", ly.Prim.Shape.Val)
	writeInt(sb, "Transparent", ly.Prim.Transparent.Val)
	writeInt(sb, "AutoFit", ly.Prim.AutoFit.Val)
	writeInt(sb, "IntelliAlpha", ly.Prim.IntelliAlpha.Val)
	writeInt(sb, "PrimFrames", ly.Prim.NumFrame.Val)
	writeInt(sb, "PrimAlign", ly.Prim.FrameAlign.Val)

	e := &ly.Eff
	writeInt(sb, "Antialias", e.AntiAlias.Val)
	writeInt(sb, "Unfold", e.Unfold.Val)
	writeInt(sb, "KeepDir", e.KeepDir.Val)
	writeInt(sb, "AnimStep", e.AnimStep.Val)
	writeAnim(sb, "Angle1", "Angle2", "AnimateAngle", "AngleCurve", e.AngleF.Val, e.AngleT.Val, e.AngleAnim.Val)
	writeInt(sb, "ZoomXYSepa", e.ZoomXYSepa.Val)
	writeAnim(sb, "Zoom1", "Zoom2", "AnimateZoom", "ZoomCurve", e.ZoomXF.Val, e.ZoomXT.Val, e.ZoomXAnim.Val)
	writeAnim(sb, "ZoomY1", "ZoomY2", "AnimateZoomY", "ZoomYCurve", e.ZoomYF.Val, e.ZoomYT.Val, e.ZoomYAnim.Val)
	writeFloat(sb, "RotCenterX1", e.CenterX.Val)
	writeFloat(sb, "RotCenterY1", e.CenterY.Val)
	writeAnim(sb, "LayerOffsetX1", "LayerOffsetX2", "AnimateLayerOffsetX", "LayerOffsetXCurve", e.OffXF.Val, e.OffXT.Val, e.OffXAnim.Val)
	writeAnim(sb, "LayerOffsetY1", "LayerOffsetY2", "AnimateLayerOffsetY", "LayerOffsetYCurve", e.OffYF.Val, e.OffYT.Val, e.OffYAnim.Val)
	writeAnim(sb, "Alpha1", "Alpha2", "AnimateAlpha", "AlphaCurve", e.AlphaF.Val, e.AlphaT.Val, e.AlphaAnim.Val)
	writeAnim(sb, "Brightness1", "Brightness2", "AnimateBrightness", "BrightnessCurve", e.BrightF.Val, e.BrightT.Val, e.BrightAnim.Val)
	writeAnim(sb, "Contrast1", "Contrast2", "AnimateContrast", "ContrastCurve", e.ContrastF.Val, e.ContrastT.Val, e.ContrastAnim.Val)
	writeAnim(sb, "Saturation1", "Saturation2", "AnimateSaturation", "SaturationCurve", e.SaturationF.Val, e.SaturationT.Val, e.SaturationAnim.Val)
	writeAnim(sb, "Hue1", "Hue2", "AnimateHue", "HueCurve", e.HueF.Val, e.HueT.Val, e.HueAnim.Val)
	writeInt(sb, "UseMask", e.Mask1Ena.Val)
	writeInt(sb, "MaskType", e.Mask1Type.Val)
	writeFloat(sb, "MaskGradation", e.Mask1Grad.Val)
	writeInt(sb, "MaskGradDir", e.Mask1GradDir.Val)
	writeAnim(sb, "MaskStart1", "MaskStart2", "AnimateMaskStart", "MaskStartCurve", e.Mask1StartF.Val, e.Mask1StartT.Val, e.Mask1StartAnim.Val)
	writeAnim(sb, "MaskStop1", "MaskStop2", "AnimateMaskStop", "MaskStopCurve", e.Mask1StopF.Val, e.Mask1StopT.Val, e.Mask1StopAnim.Val)
	writeInt(sb, "UseMask2", e.Mask2Ena.Val)
	writeInt(sb, "Mask2Operation", e.Mask2Op.Val)
	writeInt(sb, "Mask2Type", e.Mask2Type.Val)
	writeFloat(sb, "Mask2Gradation", e.Mask2Grad.Val)
	writeInt(sb, "Mask2GradDir", e.Mask2GradDir.Val)
	writeAnim(sb, "Mask2Start1", "Mask2Start2", "AnimateMask2Start", "Mask2StartCurve", e.Mask2StartF.Val, e.Mask2StartT.Val, e.Mask2StartAnim.Val)
	writeAnim(sb, "Mask2Stop1", "Mask2Stop2", "AnimateMask2Stop", "Mask2StopCurve", e.Mask2StopF.Val, e.Mask2StopT.Val, e.Mask2StopAnim.Val)
	writeInt(sb, "UseFMask", e.FMaskEna.Val)
	writeFloat(sb, "FMaskStart", e.FMaskStart.Val)
	writeFloat(sb, "FMaskStop", e.FMaskStop.Val)
	writeStr(sb, "FMaskBits", e.FMaskBits.Val)
	writeAnim(sb, "LightDir", "LightDir2", "AnimateLightDir", "LightDirCurve", e.SLightDirF.Val, e.SLightDirT.Val, e.SLightDirAnim.Val)
	writeAnim(sb, "Lighting", "Lighting2", "AnimateLighting", "LightingCurve", e.SDensityF.Val, e.SDensityT.Val, e.SDensityAnim.Val)
	writeInt(sb, "LightDirDEn", e.DLightDirEna.Val)
	writeAnim(sb, "LightDirD", "LightDirD2", "AnimateLightDirD", "LightDirDCurve", e.DLightDirF.Val, e.DLightDirT.Val, e.DLightDirAnim.Val)
	writeAnim(sb, "ShadowOffset1", "ShadowOffset2", "AnimateShadowOffset", "ShadowOffsetCurve", e.DOffsetF.Val, e.DOffsetT.Val, e.DOffsetAnim.Val)
	writeAnim(sb, "ShadowDensity1", "ShadowDensity2", "AnimateShadowDensity", "ShadowDensityCurve", e.DDensityF.Val, e.DDensityT.Val, e.DDensityAnim.Val)
	writeAnim(sb, "ShadowDiffuse1", "ShadowDiffuse2", "AnimateShadowDiffuse", "ShadowDiffuseCurve", e.DDiffuseF.Val, e.DDiffuseT.Val, e.DDiffuseAnim.Val)
	writeInt(sb, "ShadowType", e.DSType.Val)
	writeFloat(sb, "ShadowGradate", e.DSGrad.Val)
	writeInt(sb, "LightDirIEn", e.ILightDirEna.Val)
	writeAnim(sb, "LightDirI", "LightDirI2", "AnimateLightDirI", "LightDirICurve", e.ILightDirF.Val, e.ILightDirT.Val, e.ILightDirAnim.Val)
	writeAnim(sb, "IShadowOffset1", "IShadowOffset2", "AnimateIShadowOffset", "IShadowOffsetCurve", e.IOffsetF.Val, e.IOffsetT.Val, e.IOffsetAnim.Val)
	writeAnim(sb, "IShadowDensity1", "IShadowDensity2", "AnimateIShadowDensity", "IShadowDensityCurve", e.IDensityF.Val, e.IDensityT.Val, e.IDensityAnim.Val)
	writeAnim(sb, "IShadowDiffuse1", "IShadowDiffuse2", "AnimateIShadowDiffuse", "IShadowDiffuseCurve", e.IDiffuseF.Val, e.IDiffuseT.Val, e.IDiffuseAnim.Val)
	writeInt(sb, "LightDirEEn", e.ELightDirEna.Val)
	writeAnim(sb, "LightDirE", "LightDirE2", "AnimateLightDirE", "LightDirECurve", e.ELightDirF.Val, e.ELightDirT.Val, e.ELightDirAnim.Val)
	writeAnim(sb, "HilightOffset1", "HilightOffset2", "AnimateHilightOffset", "HilightOffsetCurve", e.EOffsetF.Val, e.EOffsetT.Val, e.EOffsetAnim.Val)
	writeAnim(sb, "HilightDensity1", "HilightDensity2", "AnimateHilightDensity", "HilightDensityCurve", e.EDensityF.Val, e.EDensityT.Val, e.EDensityAnim.Val)

	// Embedded binary data
	writeBinary(sb, "TexBmp", ly.Prim.EmbeddedTexture)
	writeBinary(sb, "ImgBmp", ly.Prim.EmbeddedImage)
}

// writeBinary writes hex-encoded binary data in 256-bytes-per-line format.
func writeBinary(sb *strings.Builder, keyBase string, data []byte) {
	if len(data) == 0 {
		return
	}

	lineNum := 0

	for i := 0; i < len(data); i += 256 {
		end := min(i+256, len(data))

		fmt.Fprintf(sb, "%s%d=%s\r\n", keyBase, lineNum, hex.EncodeToString(data[i:end]))
		lineNum++
	}
}

// ── Round-trip helper (for tests) ────────────────────────────────────────────

// RoundTrip loads a .knob file, saves it, and returns the re-parsed document.
// Used to verify that load→save→load produces equivalent documents.
func RoundTrip(data []byte) (*model.Document, []byte, error) {
	doc, err := Load(data)
	if err != nil {
		return nil, nil, err
	}

	out, err := Save(doc)
	if err != nil {
		return nil, nil, err
	}

	doc2, err := Load(out)
	if err != nil {
		return nil, nil, err
	}

	_ = bytes.Equal // ensure bytes import used

	return doc2, out, nil
}
