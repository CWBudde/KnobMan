package export

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"

	"knobman/internal/model"
	"knobman/internal/render"
)

// ExportPNGFrames renders all frames and returns one PNG blob per frame.
func ExportPNGFrames(doc *model.Document, textures []*render.Texture) ([][]byte, error) {
	if doc == nil {
		return nil, errors.New("export: nil document")
	}

	frames := render.RenderAll(doc, textures)
	if len(frames) == 0 {
		return nil, errors.New("export: no frames rendered")
	}

	out := make([][]byte, 0, len(frames))
	for i, fr := range frames {
		if fr == nil || fr.Width <= 0 || fr.Height <= 0 {
			return nil, fmt.Errorf("export: invalid frame at %d", i)
		}

		img := image.NewNRGBA(image.Rect(0, 0, fr.Width, fr.Height))
		for y := range fr.Height {
			srcOff := y * fr.Stride
			dstOff := y * img.Stride
			copy(img.Pix[dstOff:dstOff+fr.Width*4], fr.Data[srcOff:srcOff+fr.Stride])
		}

		var buf bytes.Buffer

		err := png.Encode(&buf, img)
		if err != nil {
			return nil, fmt.Errorf("export: encode frame %d png: %w", i, err)
		}

		out = append(out, buf.Bytes())
	}

	return out, nil
}
