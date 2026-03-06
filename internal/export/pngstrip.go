package export

import (
	"bytes"
	"fmt"
	"image"
	"image/png"

	"knobman/internal/model"
	"knobman/internal/render"
)

// ExportPNGStrip renders all frames and stitches them into a PNG strip.
// horizontal=false -> vertical strip, horizontal=true -> horizontal strip.
func ExportPNGStrip(doc *model.Document, textures []*render.Texture, horizontal bool) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("export: nil document")
	}
	frames := render.RenderAll(doc, textures)
	if len(frames) == 0 {
		return nil, fmt.Errorf("export: no frames rendered")
	}
	fw := frames[0].Width
	fh := frames[0].Height
	if fw <= 0 || fh <= 0 {
		return nil, fmt.Errorf("export: invalid frame size %dx%d", fw, fh)
	}

	count := len(frames)
	outW, outH := fw, fh*count
	if horizontal {
		outW, outH = fw*count, fh
	}
	out := image.NewNRGBA(image.Rect(0, 0, outW, outH))

	for i, fr := range frames {
		if fr == nil || fr.Width != fw || fr.Height != fh {
			return nil, fmt.Errorf("export: inconsistent frame at %d", i)
		}
		dx, dy := 0, i*fh
		if horizontal {
			dx, dy = i*fw, 0
		}
		for y := 0; y < fh; y++ {
			srcOff := y * fr.Stride
			dstOff := (dy+y)*out.Stride + dx*4
			copy(out.Pix[dstOff:dstOff+fw*4], fr.Data[srcOff:srcOff+fr.Stride])
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil, fmt.Errorf("export: encode png: %w", err)
	}
	return buf.Bytes(), nil
}
