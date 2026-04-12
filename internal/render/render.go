package render

import (
	"image/color"

	"knobman/internal/model"
)

// RenderFrame renders one document frame to dst.
func RenderFrame(dst *PixBuf, doc *model.Document, frame int, textures []*Texture) {
	if dst == nil || doc == nil {
		return
	}

	w, h := docSize(doc)
	if w <= 0 || h <= 0 {
		return
	}

	out := dst
	if dst.Width != w || dst.Height != h {
		out = NewPixBuf(w, h)
	}

	scale := 1

	if doc.Prefs.Oversampling.Val > 0 {
		s := 1 << doc.Prefs.Oversampling.Val
		if s > 0 {
			scale = s
		}
	}

	if scale <= 1 {
		renderLayers(out, doc, frame, textures)
	} else {
		hi := NewPixBuf(w*scale, h*scale)
		renderLayers(hi, doc, frame, textures)
		downsampleBox(out, hi, scale)
	}

	if out != dst {
		dst.CopyFrom(out)
	}
	flattenOverBackground(dst, doc.Prefs.BkColor.Val)
}

// RenderAll renders all export frames.
func RenderAll(doc *model.Document, textures []*Texture) []*PixBuf {
	if doc == nil {
		return nil
	}

	w, h := docSize(doc)
	if w <= 0 || h <= 0 {
		return nil
	}

	n := doc.Prefs.RenderFrames.Val
	if n < 1 {
		n = 1
	}

	frames := make([]*PixBuf, n)
	for i := range n {
		b := NewPixBuf(w, h)
		RenderFrame(b, doc, i, textures)
		frames[i] = b
	}

	return frames
}

func renderLayers(dst *PixBuf, doc *model.Document, frame int, textures []*Texture) {
	dst.Clear(color.RGBA{})

	totalFrames := doc.Prefs.RenderFrames.Val
	if totalFrames < 1 {
		totalFrames = 1
	}

	hasSolo := false

	for i := range doc.Layers {
		if doc.Layers[i].Visible.Val != 0 && doc.Layers[i].Solo.Val != 0 {
			hasSolo = true
			break
		}
	}

	for i := range doc.Layers {
		ly := &doc.Layers[i]
		if ly.Visible.Val == 0 {
			continue
		}

		if hasSolo && ly.Solo.Val == 0 {
			continue
		}

		prim := NewPixBuf(dst.Width, dst.Height)
		RenderPrimitive(prim, &ly.Prim, textures, frame, totalFrames)
		ApplyEffect(dst, prim, &ly.Eff, &doc.Curves, frame, totalFrames, textures)
	}
}

func flattenOverBackground(dst *PixBuf, bg color.RGBA) {
	if dst == nil {
		return
	}

	// Preserve fully transparent output if background alpha is also zero.
	if bg.A == 0 {
		return
	}

	for y := range dst.Height {
		off := y * dst.Stride
		for x := 0; x < dst.Width; x++ {
			i := off + x*4
			srcR := int(dst.Data[i+0])
			srcG := int(dst.Data[i+1])
			srcB := int(dst.Data[i+2])
			srcA := int(dst.Data[i+3])

			// Alpha compositing: color = src over bg.
			ia := 255 - srcA
			outA := srcA + (255-srcA)*int(bg.A)/255
			bgWeightedAlpha := int(bg.A) * ia / 255
			outR := (srcR*srcA + int(bg.R)*bgWeightedAlpha) / outA
			outG := (srcG*srcA + int(bg.G)*bgWeightedAlpha) / outA
			outB := (srcB*srcA + int(bg.B)*bgWeightedAlpha) / outA
			dst.Data[i+0] = uint8(clampInt(outR, 0, 255))
			dst.Data[i+1] = uint8(clampInt(outG, 0, 255))
			dst.Data[i+2] = uint8(clampInt(outB, 0, 255))
			dst.Data[i+3] = uint8(clampInt(outA, 0, 255))
		}
	}
}

func docSize(doc *model.Document) (int, int) {
	w := doc.Prefs.PWidth.Val

	h := doc.Prefs.PHeight.Val
	if w <= 0 {
		w = doc.Prefs.Width
	}

	if h <= 0 {
		h = doc.Prefs.Height
	}

	return w, h
}

func downsampleBox(dst, src *PixBuf, scale int) {
	if dst == nil || src == nil || scale <= 1 {
		if dst != nil && src != nil {
			dst.CopyFrom(src)
		}

		return
	}

	for y := range dst.Height {
		for x := range dst.Width {
			var sr, sg, sb, sa int

			for oy := range scale {
				for ox := range scale {
					c := src.At(x*scale+ox, y*scale+oy)
					sr += int(c.R)
					sg += int(c.G)
					sb += int(c.B)
					sa += int(c.A)
				}
			}

			n := scale * scale
			dst.Set(x, y, colorRGBA(sr/n, sg/n, sb/n, sa/n))
		}
	}
}

func colorRGBA(r, g, b, a int) color.RGBA {
	return color.RGBA{
		R: uint8(clampInt(r, 0, 255)),
		G: uint8(clampInt(g, 0, 255)),
		B: uint8(clampInt(b, 0, 255)),
		A: uint8(clampInt(a, 0, 255)),
	}
}
