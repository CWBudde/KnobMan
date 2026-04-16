package render

import (
	"image"
	"image/color"
	"math"

	agg "github.com/cwbudde/agg_go"
)

func blendPremultipliedAggImageOverPixBuf(dst *PixBuf, src *agg.Image) {
	blendPremultipliedAggImageRectOverPixBuf(dst, src, 0, 0, dst.Width, dst.Height)
}

func blendPremultipliedAggImageRectOverPixBuf(dst *PixBuf, src *agg.Image, x0, y0, x1, y1 int) {
	if dst == nil || src == nil || dst.Width <= 0 || dst.Height <= 0 {
		return
	}

	if x0 < 0 {
		x0 = 0
	}

	if y0 < 0 {
		y0 = 0
	}

	if x1 > dst.Width {
		x1 = dst.Width
	}

	if y1 > dst.Height {
		y1 = dst.Height
	}

	if x1 > src.Width() {
		x1 = src.Width()
	}

	if y1 > src.Height() {
		y1 = src.Height()
	}

	if x0 >= x1 || y0 >= y1 {
		return
	}

	stride := src.Stride()
	for y := y0; y < y1; y++ {
		srcOff := y * stride
		for x := x0; x < x1; x++ {
			si := srcOff + x*4

			a := uint32(src.Data[si+3])
			if a <= 1 {
				continue
			}

			dst.BlendOver(x, y, color.RGBA{
				R: demultiplyRGBAComponent(uint32(src.Data[si+0]), a),
				G: demultiplyRGBAComponent(uint32(src.Data[si+1]), a),
				B: demultiplyRGBAComponent(uint32(src.Data[si+2]), a),
				A: uint8(a),
			})
		}
	}
}

func samplePixBufNearest(src *PixBuf, x, y int) color.RGBA {
	if src == nil || x < 0 || y < 0 || x >= src.Width || y >= src.Height {
		return color.RGBA{}
	}

	return src.At(x, y)
}

func samplePixBufBilinear(src *PixBuf, fx, fy float64) color.RGBA {
	if src == nil || src.Width <= 0 || src.Height <= 0 {
		return color.RGBA{}
	}

	x0 := int(math.Floor(fx))
	y0 := int(math.Floor(fy))
	tx := fx - float64(x0)
	ty := fy - float64(y0)

	c00 := samplePixBufNearest(src, x0, y0)
	c10 := samplePixBufNearest(src, x0+1, y0)
	c01 := samplePixBufNearest(src, x0, y0+1)
	c11 := samplePixBufNearest(src, x0+1, y0+1)

	w00 := (1.0 - tx) * (1.0 - ty)
	w10 := tx * (1.0 - ty)
	w01 := (1.0 - tx) * ty
	w11 := tx * ty

	pr := float64(premultiplyRGBAComponent(uint32(c00.R), uint32(c00.A)))*w00 +
		float64(premultiplyRGBAComponent(uint32(c10.R), uint32(c10.A)))*w10 +
		float64(premultiplyRGBAComponent(uint32(c01.R), uint32(c01.A)))*w01 +
		float64(premultiplyRGBAComponent(uint32(c11.R), uint32(c11.A)))*w11
	pg := float64(premultiplyRGBAComponent(uint32(c00.G), uint32(c00.A)))*w00 +
		float64(premultiplyRGBAComponent(uint32(c10.G), uint32(c10.A)))*w10 +
		float64(premultiplyRGBAComponent(uint32(c01.G), uint32(c01.A)))*w01 +
		float64(premultiplyRGBAComponent(uint32(c11.G), uint32(c11.A)))*w11
	pb := float64(premultiplyRGBAComponent(uint32(c00.B), uint32(c00.A)))*w00 +
		float64(premultiplyRGBAComponent(uint32(c10.B), uint32(c10.A)))*w10 +
		float64(premultiplyRGBAComponent(uint32(c01.B), uint32(c01.A)))*w01 +
		float64(premultiplyRGBAComponent(uint32(c11.B), uint32(c11.A)))*w11
	pa := float64(c00.A)*w00 + float64(c10.A)*w10 + float64(c01.A)*w01 + float64(c11.A)*w11

	a := clampInt(int(math.Round(pa)), 0, 255)
	if a == 0 {
		return color.RGBA{}
	}

	return color.RGBA{
		R: demultiplyRGBAComponent(uint32(clampInt(int(math.Round(pr)), 0, 255)), uint32(a)),
		G: demultiplyRGBAComponent(uint32(clampInt(int(math.Round(pg)), 0, 255)), uint32(a)),
		B: demultiplyRGBAComponent(uint32(clampInt(int(math.Round(pb)), 0, 255)), uint32(a)),
		A: uint8(a),
	}
}

// ImageToPixBuf converts a Go image into straight-alpha PixBuf storage.
func ImageToPixBuf(img image.Image) *PixBuf {
	rgba := ImageToRGBA(img)
	if rgba == nil {
		return nil
	}

	return pixBufFromRGBA(rgba)
}

// PixBufToNRGBA copies a PixBuf into a standard-library straight-alpha image.
func PixBufToNRGBA(buf *PixBuf) *image.NRGBA {
	if buf == nil || buf.Width <= 0 || buf.Height <= 0 || len(buf.Data) == 0 {
		return nil
	}

	out := image.NewNRGBA(image.Rect(0, 0, buf.Width, buf.Height))
	for y := range buf.Height {
		srcOff := y * buf.Stride
		dstOff := y * out.Stride
		copy(out.Pix[dstOff:dstOff+buf.Width*4], buf.Data[srcOff:srcOff+buf.Width*4])
	}

	return out
}

// ImageToRGBA normalizes arbitrary decoded images to straight-alpha RGBA.
func ImageToRGBA(img image.Image) *image.RGBA {
	if img == nil {
		return nil
	}

	b := img.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))

	switch src := img.(type) {
	case *image.NRGBA:
		for y := 0; y < b.Dy(); y++ {
			srcY := b.Min.Y + y
			srcOffset := srcY*src.Stride + b.Min.X*4
			dstOffset := y * out.Stride
			copy(out.Pix[dstOffset:dstOffset+b.Dx()*4], src.Pix[srcOffset:srcOffset+b.Dx()*4])
		}

		return out
	case *image.Gray:
		for y := 0; y < b.Dy(); y++ {
			srcY := b.Min.Y + y
			srcOffset := srcY*src.Stride + b.Min.X
			dstOffset := y * out.Stride

			for x := 0; x < b.Dx(); x++ {
				v := src.Pix[srcOffset+x]
				dst := dstOffset + x*4
				out.Pix[dst+0] = v
				out.Pix[dst+1] = v
				out.Pix[dst+2] = v
				out.Pix[dst+3] = 255
			}
		}

		return out
	case *image.RGBA:
		for y := 0; y < b.Dy(); y++ {
			srcY := b.Min.Y + y
			for x := 0; x < b.Dx(); x++ {
				srcX := b.Min.X + x
				srcOffset := srcY*src.Stride + srcX*4
				r := uint32(src.Pix[srcOffset+0])
				g := uint32(src.Pix[srcOffset+1])
				bl := uint32(src.Pix[srcOffset+2])
				a := uint32(src.Pix[srcOffset+3])
				dstOffset := y*out.Stride + x*4

				out.Pix[dstOffset+0] = demultiplyRGBAComponent(r, a)
				out.Pix[dstOffset+1] = demultiplyRGBAComponent(g, a)
				out.Pix[dstOffset+2] = demultiplyRGBAComponent(bl, a)
				out.Pix[dstOffset+3] = uint8(a)
			}
		}

		return out
	}

	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			n, ok := color.NRGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			if !ok {
				continue
			}

			dstOffset := y*out.Stride + x*4
			out.Pix[dstOffset+0] = n.R
			out.Pix[dstOffset+1] = n.G
			out.Pix[dstOffset+2] = n.B
			out.Pix[dstOffset+3] = n.A
		}
	}

	return out
}

func pixBufFromRGBA(img *image.RGBA) *PixBuf {
	if img == nil {
		return nil
	}

	b := img.Bounds()

	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return nil
	}

	pb := NewPixBuf(w, h)
	for y := range h {
		srcOff := y * img.Stride
		dstOff := y * pb.Stride
		copy(pb.Data[dstOff:dstOff+w*4], img.Pix[srcOff:srcOff+w*4])
	}

	return pb
}

func demultiplyRGBAComponent(c, a uint32) uint8 {
	if a == 0 {
		return 0
	}

	if a == 255 {
		return uint8(c)
	}

	// AGG's image path operates on premultiplied buffers, but filtered samples
	// can still produce RGB > A. Java's BufferedImage export path effectively
	// clamps those back into straight-alpha range rather than wrapping.
	if c >= a {
		return 255
	}

	return uint8((c*255 + a/2) / a)
}

func premultiplyRGBAComponent(c, a uint32) uint8 {
	if a == 0 {
		return 0
	}

	if a == 255 {
		return uint8(c)
	}

	t := c*a + 128

	return uint8((t + (t >> 8)) >> 8)
}
