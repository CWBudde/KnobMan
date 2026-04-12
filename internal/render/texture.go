package render

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
)

// Texture is a tiled RGBA texture map.
type Texture struct {
	Data []uint8
	W, H int
}

// TextureSet stores built-in and user-provided textures.
type TextureSet struct {
	Items []*Texture
}

// NewTextureSet allocates an empty texture list.
func NewTextureSet() *TextureSet {
	return &TextureSet{Items: make([]*Texture, 0, 32)}
}

// Add appends t and returns its 1-based index (0 means "none").
func (s *TextureSet) Add(t *Texture) int {
	if s == nil || t == nil {
		return 0
	}

	s.Items = append(s.Items, t)

	return len(s.Items)
}

// Get returns texture by 1-based index. Returns nil for invalid indices.
func (s *TextureSet) Get(idx int) *Texture {
	if s == nil || idx <= 0 || idx > len(s.Items) {
		return nil
	}

	return s.Items[idx-1]
}

// DecodeTexture decodes PNG/JPEG/GIF bytes into a texture.
func DecodeTexture(data []byte) (*Texture, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err == nil {
		return NewTextureFromImage(img), nil
	}

	// Standard library does not include BMP decoding in this toolchain build.
	// Add a narrow Windows BMP compatibility path used by bundled KnobMan textures.
	if tex, ok := decodeBMPAsRGBA(data); ok {
		return tex, nil
	}

	return nil, err
}

func decodeBMPAsRGBA(data []byte) (*Texture, bool) {
	if len(data) < 54 || string(data[0:2]) != "BM" {
		return nil, false
	}

	pixelOffset := int(binary.LittleEndian.Uint32(data[10:14]))
	headerSize := int(binary.LittleEndian.Uint32(data[14:18]))
	if headerSize < 40 || pixelOffset <= 0 || pixelOffset >= len(data) {
		return nil, false
	}

	width := int(int32(binary.LittleEndian.Uint32(data[18:22])))
	heightSigned := int(int32(binary.LittleEndian.Uint32(data[22:26])))
	if width <= 0 || heightSigned == 0 {
		return nil, false
	}

	bpp := int(binary.LittleEndian.Uint16(data[28:30]))
	if bpp != 24 && bpp != 32 {
		return nil, false
	}

	compression := binary.LittleEndian.Uint32(data[30:34])
	if compression != 0 {
		return nil, false
	}

	flipY := true
	height := heightSigned
	if height < 0 {
		flipY = false
		height = -height
	}
	if height == 0 {
		return nil, false
	}

	rowBytes := ((width*bpp + 31) / 32) * 4
	if pixelOffset+rowBytes*height > len(data) {
		return nil, false
	}

	out := make([]uint8, width*height*4)
	outIdx := 0
	for y := 0; y < height; y++ {
		srcY := y
		if flipY {
			srcY = height - 1 - y
		}
		row := pixelOffset + srcY*rowBytes

		for x := 0; x < width; x++ {
			p := row + x*(bpp/8)
			if p+2 >= len(data) {
				return nil, false
			}

			b := data[p]
			g := data[p+1]
			r := data[p+2]
			a := uint8(255)
			if bpp == 32 {
				if p+3 >= len(data) {
					return nil, false
				}
				a = data[p+3]
			}

			out[outIdx+0] = r
			out[outIdx+1] = g
			out[outIdx+2] = b
			out[outIdx+3] = a
			outIdx += 4
		}
	}

	return &Texture{Data: out, W: width, H: height}, true
}

// NewTextureFromImage converts an image to RGBA texture storage.
func NewTextureFromImage(img image.Image) *Texture {
	if img == nil {
		return nil
	}

	b := img.Bounds()
	w := b.Dx()

	h := b.Dy()
	if w <= 0 || h <= 0 {
		return nil
	}

	data := make([]uint8, w*h*4)
	i := 0

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			data[i+0] = uint8(r >> 8)
			data[i+1] = uint8(g >> 8)
			data[i+2] = uint8(bl >> 8)
			data[i+3] = uint8(a >> 8)
			i += 4
		}
	}

	return &Texture{Data: data, W: w, H: h}
}

// Sample returns a bilinear filtered texel with tiled wrapping.
// u and v are unbounded texture coordinates in texel units.
func (t *Texture) Sample(u, v, zoom float64) color.RGBA {
	if t == nil || t.W <= 0 || t.H <= 0 || len(t.Data) < t.W*t.H*4 {
		return color.RGBA{}
	}

	if zoom == 0 {
		zoom = 1
	}

	u /= zoom
	v /= zoom

	x := wrapFloat(u, float64(t.W))
	y := wrapFloat(v, float64(t.H))

	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	x1 := (x0 + 1) % t.W
	y1 := (y0 + 1) % t.H
	fx := x - float64(x0)
	fy := y - float64(y0)

	c00 := t.at(x0, y0)
	c10 := t.at(x1, y0)
	c01 := t.at(x0, y1)
	c11 := t.at(x1, y1)

	return color.RGBA{
		R: lerpByte(lerpByte(c00.R, c10.R, fx), lerpByte(c01.R, c11.R, fx), fy),
		G: lerpByte(lerpByte(c00.G, c10.G, fx), lerpByte(c01.G, c11.G, fx), fy),
		B: lerpByte(lerpByte(c00.B, c10.B, fx), lerpByte(c01.B, c11.B, fx), fy),
		A: lerpByte(lerpByte(c00.A, c10.A, fx), lerpByte(c01.A, c11.A, fx), fy),
	}
}

func (t *Texture) at(x, y int) color.RGBA {
	i := (y*t.W + x) * 4
	return color.RGBA{R: t.Data[i], G: t.Data[i+1], B: t.Data[i+2], A: t.Data[i+3]}
}

func wrapFloat(v, n float64) float64 {
	if n <= 0 {
		return 0
	}

	v = math.Mod(v, n)
	if v < 0 {
		v += n
	}

	return v
}

func lerpByte(a, b uint8, t float64) uint8 {
	if t <= 0 {
		return a
	}

	if t >= 1 {
		return b
	}

	av := float64(a)

	return uint8(av + (float64(b)-av)*t + 0.5)
}
