package render

import (
	"bytes"
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
	if err != nil {
		return nil, err
	}
	return NewTextureFromImage(img), nil
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
