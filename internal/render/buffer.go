package render

import "image/color"

// PixBuf stores an RGBA image in row-major order.
type PixBuf struct {
	Data          []uint8
	Width, Height int
	Stride        int
}

// NewPixBuf allocates a new zero-initialized RGBA buffer.
func NewPixBuf(w, h int) *PixBuf {
	if w <= 0 || h <= 0 {
		return &PixBuf{}
	}
	return &PixBuf{
		Data:   make([]uint8, w*h*4),
		Width:  w,
		Height: h,
		Stride: w * 4,
	}
}

// Clone returns a deep copy of b.
func (b *PixBuf) Clone() *PixBuf {
	if b == nil {
		return nil
	}
	c := &PixBuf{Width: b.Width, Height: b.Height, Stride: b.Stride}
	if len(b.Data) > 0 {
		c.Data = append([]uint8(nil), b.Data...)
	}
	return c
}

// Clear fills the entire image with c.
func (b *PixBuf) Clear(c color.RGBA) {
	if b == nil || b.Width <= 0 || b.Height <= 0 {
		return
	}
	for y := 0; y < b.Height; y++ {
		off := y * b.Stride
		for x := 0; x < b.Width; x++ {
			i := off + x*4
			b.Data[i+0] = c.R
			b.Data[i+1] = c.G
			b.Data[i+2] = c.B
			b.Data[i+3] = c.A
		}
	}
}

// At returns the pixel at x,y. Out-of-bounds reads return zero.
func (b *PixBuf) At(x, y int) color.RGBA {
	if b == nil || x < 0 || y < 0 || x >= b.Width || y >= b.Height {
		return color.RGBA{}
	}
	i := y*b.Stride + x*4
	return color.RGBA{R: b.Data[i], G: b.Data[i+1], B: b.Data[i+2], A: b.Data[i+3]}
}

// Set writes c at x,y. Out-of-bounds writes are ignored.
func (b *PixBuf) Set(x, y int, c color.RGBA) {
	if b == nil || x < 0 || y < 0 || x >= b.Width || y >= b.Height {
		return
	}
	i := y*b.Stride + x*4
	b.Data[i+0] = c.R
	b.Data[i+1] = c.G
	b.Data[i+2] = c.B
	b.Data[i+3] = c.A
}

// BlendOver alpha-blends src over the destination pixel at x,y.
func (b *PixBuf) BlendOver(x, y int, src color.RGBA) {
	if b == nil || x < 0 || y < 0 || x >= b.Width || y >= b.Height {
		return
	}
	i := y*b.Stride + x*4
	dr := float64(b.Data[i+0]) / 255.0
	dg := float64(b.Data[i+1]) / 255.0
	db := float64(b.Data[i+2]) / 255.0
	da := float64(b.Data[i+3]) / 255.0

	sr := float64(src.R) / 255.0
	sg := float64(src.G) / 255.0
	sb := float64(src.B) / 255.0
	sa := float64(src.A) / 255.0

	outA := sa + da*(1.0-sa)
	if outA <= 0 {
		b.Data[i+0], b.Data[i+1], b.Data[i+2], b.Data[i+3] = 0, 0, 0, 0
		return
	}
	outR := (sr*sa + dr*da*(1.0-sa)) / outA
	outG := (sg*sa + dg*da*(1.0-sa)) / outA
	outB := (sb*sa + db*da*(1.0-sa)) / outA

	b.Data[i+0] = uint8(clamp01(outR)*255.0 + 0.5)
	b.Data[i+1] = uint8(clamp01(outG)*255.0 + 0.5)
	b.Data[i+2] = uint8(clamp01(outB)*255.0 + 0.5)
	b.Data[i+3] = uint8(clamp01(outA)*255.0 + 0.5)
}

// CopyFrom copies src into b (overlapping extent only).
func (b *PixBuf) CopyFrom(src *PixBuf) {
	if b == nil || src == nil || b.Width == 0 || b.Height == 0 || src.Width == 0 || src.Height == 0 {
		return
	}
	w := b.Width
	if src.Width < w {
		w = src.Width
	}
	h := b.Height
	if src.Height < h {
		h = src.Height
	}
	for y := 0; y < h; y++ {
		dOff := y * b.Stride
		sOff := y * src.Stride
		copy(b.Data[dOff:dOff+w*4], src.Data[sOff:sOff+w*4])
	}
}

// FillRect paints an axis-aligned rectangle clipped to bounds.
func (b *PixBuf) FillRect(x0, y0, x1, y1 int, c color.RGBA) {
	if b == nil {
		return
	}
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	if x1 <= 0 || y1 <= 0 || x0 >= b.Width || y0 >= b.Height {
		return
	}
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > b.Width {
		x1 = b.Width
	}
	if y1 > b.Height {
		y1 = b.Height
	}
	for y := y0; y < y1; y++ {
		off := y * b.Stride
		for x := x0; x < x1; x++ {
			i := off + x*4
			b.Data[i+0] = c.R
			b.Data[i+1] = c.G
			b.Data[i+2] = c.B
			b.Data[i+3] = c.A
		}
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
