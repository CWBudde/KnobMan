package render

import (
	"image"
	"image/color"

	agg "github.com/cwbudde/agg_go"
)

// AggImageForPixBuf exposes a PixBuf as an agg_go image without copying.
func AggImageForPixBuf(buf *PixBuf) *agg.Image {
	if buf == nil || buf.Width <= 0 || buf.Height <= 0 || len(buf.Data) == 0 {
		return nil
	}

	return agg.NewImage(buf.Data, buf.Width, buf.Height, buf.Stride)
}

// AggContextForPixBuf creates an agg_go context backed by the PixBuf memory.
func AggContextForPixBuf(buf *PixBuf) *agg.Context {
	img := AggImageForPixBuf(buf)
	if img == nil {
		return nil
	}

	return agg.NewContextForImage(img)
}

// Agg2DForPixBuf creates a low-level agg_go renderer backed by the PixBuf memory.
func Agg2DForPixBuf(buf *PixBuf) *agg.Agg2D {
	if buf == nil || buf.Width <= 0 || buf.Height <= 0 || len(buf.Data) == 0 {
		return nil
	}

	a := agg.NewAgg2D()
	a.Attach(buf.Data, buf.Width, buf.Height, buf.Stride)
	return a
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
	for y := 0; y < buf.Height; y++ {
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

				if a == 0 {
					out.Pix[dstOffset+0] = 0
					out.Pix[dstOffset+1] = 0
					out.Pix[dstOffset+2] = 0
					out.Pix[dstOffset+3] = 0
					continue
				}

				out.Pix[dstOffset+0] = uint8((r*255 + a/2) / a)
				out.Pix[dstOffset+1] = uint8((g*255 + a/2) / a)
				out.Pix[dstOffset+2] = uint8((bl*255 + a/2) / a)
				out.Pix[dstOffset+3] = uint8(a)
			}
		}
		return out
	}

	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			n := color.NRGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
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
	for y := 0; y < h; y++ {
		srcOff := y * img.Stride
		dstOff := y * pb.Stride
		copy(pb.Data[dstOff:dstOff+w*4], img.Pix[srcOff:srcOff+w*4])
	}

	return pb
}
