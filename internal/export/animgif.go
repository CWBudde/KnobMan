package export

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"math"

	"knobman/internal/model"
	"knobman/internal/render"
)

// ExportGIF renders the document as an animated GIF.
func ExportGIF(doc *model.Document, textures []*render.Texture) ([]byte, error) {
	if doc == nil {
		return nil, errors.New("export: nil document")
	}

	frames := render.RenderAll(doc, textures)
	if len(frames) == 0 {
		return nil, errors.New("export: no frames rendered")
	}

	sequence := gifFrameSequence(len(frames), doc.Prefs.BiDir.Val != 0)
	if len(sequence) == 0 {
		return nil, errors.New("export: empty frame sequence")
	}

	delay := max(int(math.Round(float64(maxInt(1, doc.Prefs.Duration.Val))/10.0)), 1)

	out := &gif.GIF{
		Image:     make([]*image.Paletted, 0, len(sequence)),
		Delay:     make([]int, 0, len(sequence)),
		LoopCount: loopCountForGIF(doc.Prefs.Loop.Val),
	}

	for _, idx := range sequence {
		fr := frames[idx]
		if fr == nil || fr.Width <= 0 || fr.Height <= 0 {
			return nil, fmt.Errorf("export: invalid frame at %d", idx)
		}

		src := image.NewNRGBA(image.Rect(0, 0, fr.Width, fr.Height))
		for y := range fr.Height {
			srcOff := y * fr.Stride
			dstOff := y * src.Stride
			copy(src.Pix[dstOff:dstOff+fr.Width*4], fr.Data[srcOff:srcOff+fr.Stride])
		}

		pal := image.NewPaletted(src.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(pal, src.Bounds(), src, image.Point{})
		out.Image = append(out.Image, pal)
		out.Delay = append(out.Delay, delay)
	}

	var buf bytes.Buffer

	err := gif.EncodeAll(&buf, out)
	if err != nil {
		return nil, fmt.Errorf("export: encode gif: %w", err)
	}

	return buf.Bytes(), nil
}

func gifFrameSequence(frameCount int, bidir bool) []int {
	if frameCount <= 0 {
		return nil
	}

	seq := make([]int, 0, frameCount*2)
	for i := range frameCount {
		seq = append(seq, i)
	}

	if !bidir || frameCount <= 2 {
		return seq
	}

	for i := frameCount - 2; i >= 1; i-- {
		seq = append(seq, i)
	}

	return seq
}

func loopCountForGIF(loop int) int {
	if loop <= 0 {
		return 0 // infinite
	}

	return loop
}

func maxInt(a, b int) int {
	if a >= b {
		return a
	}

	return b
}
