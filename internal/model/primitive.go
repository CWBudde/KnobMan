package model

// PrimitiveType identifies which shape/content renderer to use.
type PrimitiveType int

const (
	PrimNone           PrimitiveType = 0
	PrimImage          PrimitiveType = 1
	PrimCircle         PrimitiveType = 2
	PrimCircleFill     PrimitiveType = 3
	PrimMetalCircle    PrimitiveType = 4
	PrimWaveCircle     PrimitiveType = 5
	PrimSphere         PrimitiveType = 6 // "Sphere" / Gradient Circle in file
	PrimRect           PrimitiveType = 7
	PrimRectFill       PrimitiveType = 8
	PrimTriangle       PrimitiveType = 9
	PrimLine           PrimitiveType = 10
	PrimRadiateLine    PrimitiveType = 11
	PrimHLines         PrimitiveType = 12
	PrimVLines         PrimitiveType = 13
	PrimText           PrimitiveType = 14
	PrimShape          PrimitiveType = 15
)

// primTypeNames maps PrimitiveType → the string stored in .knob files.
var primTypeNames = [16]string{
	"None", "Image", "Circle", "CircleFill", "MetalCircle", "WaveCircle",
	"Sphere", "Rect", "RectFill", "Triangle", "Line", "RadiateLine",
	"H-Lines", "V-Lines", "Text", "Shape",
}

// PrimTypeName returns the .knob file string for a PrimitiveType.
func PrimTypeName(t PrimitiveType) string {
	if t >= 0 && int(t) < len(primTypeNames) {
		return primTypeNames[t]
	}
	return "None"
}

// PrimTypeFromName parses the primitive type string from a .knob file.
func PrimTypeFromName(s string) PrimitiveType {
	for i, name := range primTypeNames {
		if name == s {
			return PrimitiveType(i)
		}
	}
	return PrimNone
}

// Primitive holds all parameters for a single layer's shape/content.
// All parameters are present regardless of PrimitiveType; unused ones are ignored
// during rendering. This mirrors the Java Primitive class.
type Primitive struct {
	Type         SelectParam // PrimitiveType cast to int
	Color        ColorParam
	TextureFile  SelectParam // index into texture list (0 = none)
	TextureName  string      // texture file name (used for matching/embedding)
	Transparent  SelectParam // transparency mode
	Font         SelectParam // font index
	TextAlign    SelectParam // text alignment (0=L 1=C 2=R)
	FrameAlign   SelectParam // image frame source (0=strip 1=horizontal 2=files)
	Aspect       FloatParam  // X/Y aspect distortion (default 0 = no distortion)
	Round        FloatParam  // corner rounding %
	Width        FloatParam  // element width % (default 10)
	Length       FloatParam  // element length % (default 50)
	Step         FloatParam  // spacing (default 20)
	AngleStep    FloatParam  // angular step degrees (default 45)
	Emboss       FloatParam  // emboss strength (+ raised / − inset, default 0)
	EmbossDiffuse FloatParam // emboss edge softness (default 0)
	Ambient      FloatParam  // ambient light level 0–100 (default 50)
	LightDir     FloatParam  // light direction degrees (default −50)
	Specular     FloatParam  // specular highlight strength (default 0)
	SpecularWidth FloatParam // specular highlight spread (default 50)
	TextureDepth FloatParam  // texture blend depth (default 0)
	TextureZoom  FloatParam  // texture zoom % (default 100)
	Diffuse      FloatParam  // edge feathering (default 0)
	FontSize     FloatParam  // text font size % (default 50)
	File         StringParam // image file path for PrimImage
	Text         StringParam // text content (may contain frame counter patterns)
	Shape        StringParam // SVG-like path data for PrimShape
	AutoFit      BoolParam   // auto-fit image to canvas (default 1)
	Bold         BoolParam
	Italic       BoolParam
	Fill         BoolParam   // fill shape (default 1)
	IntelliAlpha IntParam    // smart alpha mode (0=off)
	NumFrame     IntParam    // number of frames in image strip (default 1)

	// EmbeddedImage holds the raw PNG bytes of an embedded image extracted
	// from the .knob file (ImgBmp lines). Nil if loaded from an external path.
	EmbeddedImage []byte
	// EmbeddedTexture holds the raw PNG bytes of an embedded texture (TexBmp).
	EmbeddedTexture []byte
}

// NewPrimitive returns a Primitive with Java-matching default values.
func NewPrimitive() Primitive {
	return Primitive{
		Type:          SelectParam{int(PrimNone)},
		Color:         ColorParam{},
		Aspect:        FloatParam{0},
		Round:         FloatParam{0},
		Width:         FloatParam{10},
		Length:        FloatParam{50},
		Step:          FloatParam{20},
		AngleStep:     FloatParam{45},
		Emboss:        FloatParam{0},
		EmbossDiffuse: FloatParam{0},
		Ambient:       FloatParam{50},
		LightDir:      FloatParam{-50},
		Specular:      FloatParam{0},
		SpecularWidth: FloatParam{50},
		TextureDepth:  FloatParam{0},
		TextureZoom:   FloatParam{100},
		Diffuse:       FloatParam{0},
		FontSize:      FloatParam{50},
		AutoFit:       BoolParam{1},
		Fill:          BoolParam{1},
		NumFrame:      IntParam{1},
	}
}
