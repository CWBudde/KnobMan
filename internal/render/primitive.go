package render

import (
	"bytes"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"strconv"
	"strings"

	"knobman/internal/model"
)

// RenderPrimitive draws a primitive into dst.
// This is the Phase 2 baseline implementation; complex Java parity details
// (advanced lighting/masks/anti-aliasing) are layered in later phases.
func RenderPrimitive(dst *PixBuf, p *model.Primitive, textures []*Texture, frame, totalFrames int) {
	if dst == nil || p == nil || dst.Width == 0 || dst.Height == 0 {
		return
	}

	switch model.PrimitiveType(p.Type.Val) {
	case model.PrimNone:
		return
	case model.PrimImage:
		renderImage(dst, p, frame, totalFrames)
	case model.PrimCircle:
		renderCircle(dst, p, true, textures)
	case model.PrimCircleFill:
		renderCircle(dst, p, false, textures)
	case model.PrimMetalCircle:
		renderMetalCircle(dst, p, textures)
	case model.PrimWaveCircle:
		renderWaveCircle(dst, p)
	case model.PrimSphere:
		renderSphere(dst, p, textures)
	case model.PrimRect:
		renderRect(dst, p, true)
	case model.PrimRectFill:
		renderRect(dst, p, false)
	case model.PrimTriangle:
		renderTriangle(dst, p)
	case model.PrimLine:
		renderLine(dst, p)
	case model.PrimRadiateLine:
		renderRadiateLines(dst, p)
	case model.PrimHLines:
		renderParallelLines(dst, p, true)
	case model.PrimVLines:
		renderParallelLines(dst, p, false)
	case model.PrimText:
		renderText(dst, p, frame, totalFrames)
	case model.PrimShape:
		renderShape(dst, p)
	}
}

func primitiveColor(p *model.Primitive) color.RGBA {
	c := p.Color.Val
	if c.A == 0 {
		c.A = 255
	}
	return c
}

func renderImage(dst *PixBuf, p *model.Primitive, frame, totalFrames int) {
	if len(p.EmbeddedImage) == 0 {
		return
	}
	img, _, err := image.Decode(bytes.NewReader(p.EmbeddedImage))
	if err != nil {
		return
	}
	src := pixBufFromImage(img)
	if src == nil || src.Width <= 0 || src.Height <= 0 {
		return
	}
	nf := p.NumFrame.Val
	if nf < 1 {
		nf = 1
	}
	frameBuf := ExtractFrameAligned(src, nf, frame, totalFrames, p.FrameAlign.Val)
	if frameBuf == nil || frameBuf.Width <= 0 || frameBuf.Height <= 0 {
		return
	}
	def := frameBuf.At(0, 0)
	if p.AutoFit.Val != 0 {
		drawPixBufToRect(dst, frameBuf, 0, 0, dst.Width, dst.Height, p, def)
		return
	}
	// Java behavior: when AutoFit is off, the source frame is copied at native size
	// into the top-left corner (clipped by canvas bounds).
	drawPixBufToRect(dst, frameBuf, 0, 0, frameBuf.Width, frameBuf.Height, p, def)
}

func pixBufFromImage(img image.Image) *PixBuf {
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
		for x := 0; x < w; x++ {
			pb.Set(x, y, rgbaFromImage(img, b.Min.X+x, b.Min.Y+y))
		}
	}
	return pb
}

func drawPixBufToRect(dst, src *PixBuf, dx0, dy0, dw, dh int, p *model.Primitive, def color.RGBA) {
	if src == nil || src.Width <= 0 || src.Height <= 0 || dw <= 0 || dh <= 0 {
		return
	}
	maxX := min(dst.Width, dx0+dw)
	maxY := min(dst.Height, dy0+dh)
	for y := max(0, dy0); y < maxY; y++ {
		sy := (y - dy0) * src.Height / dh
		for x := max(0, dx0); x < maxX; x++ {
			sx := (x - dx0) * src.Width / dw
			px := src.At(sx, sy)
			px = applyImageTransparency(px, def, p.Transparent.Val, p.IntelliAlpha.Val)
			if px.A == 0 {
				continue
			}
			dst.BlendOver(x, y, px)
		}
	}
}

func rgbaFromImage(img image.Image, x, y int) color.RGBA {
	r, g, b, a := img.At(x, y).RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

func applyImageTransparency(px, def color.RGBA, transparentMode, intelliAlpha int) color.RGBA {
	switch transparentMode {
	case 0: // file alpha
		return px
	case 1: // force opaque
		px.A = 255
		return px
	default: // first-pixel key
		if intelliAlpha > 0 {
			return intelliAlphaPix(def, px, intelliAlpha)
		}
		if px.R == def.R && px.G == def.G && px.B == def.B {
			return color.RGBA{}
		}
		px.A = 255
		return px
	}
}

func intelliAlphaPix(def, target color.RGBA, intelliAlpha int) color.RGBA {
	if def.R == target.R && def.G == target.G && def.B == target.B {
		return color.RGBA{}
	}
	alphaStep := 255 - intelliAlpha*254/100
	if alphaStep < 16 {
		alphaStep = 16
	}
	piAlpha := 0
	var r, g, b int
	for {
		piAlpha += alphaStep
		a := 1.0
		if piAlpha >= 255 {
			piAlpha = 255
		} else {
			a = float64(piAlpha) / 255.0
		}
		r = int((float64(target.R) - float64(def.R)*(1.0-a)) / a)
		g = int((float64(target.G) - float64(def.G)*(1.0-a)) / a)
		b = int((float64(target.B) - float64(def.B)*(1.0-a)) / a)
		if a >= 1.0 || (r >= 0 && r <= 255 && g >= 0 && g <= 255 && b >= 0 && b <= 255) {
			break
		}
	}
	return color.RGBA{R: uint8(clampInt(r, 0, 255)), G: uint8(clampInt(g, 0, 255)), B: uint8(clampInt(b, 0, 255)), A: uint8(piAlpha)}
}

func renderCircle(dst *PixBuf, p *model.Primitive, outline bool, textures []*Texture) {
	c := primitiveColor(p)
	cx := float64(dst.Width-1) * 0.5
	cy := float64(dst.Height-1) * 0.5
	rx := float64(dst.Width) * 0.45
	ry := float64(dst.Height) * 0.45
	asp := p.Aspect.Val / 100.0
	if asp > 0 {
		rx *= 1.0 + asp
	} else if asp < 0 {
		ry *= 1.0 - asp
	}

	stroke := math.Max(1, p.Width.Val*0.01*float64(min(dst.Width, dst.Height)))
	for y := 0; y < dst.Height; y++ {
		fy := (float64(y) - cy) / ry
		for x := 0; x < dst.Width; x++ {
			fx := (float64(x) - cx) / rx
			d := math.Sqrt(fx*fx + fy*fy)
			inside := d <= 1.0
			if !inside {
				continue
			}
			if outline {
				edgeDist := (1.0 - d) * math.Min(rx, ry)
				if edgeDist > stroke {
					continue
				}
			}
			pix := c
			pix = applyTextureOverlay(pix, textures, p, x, y)
			dst.BlendOver(x, y, pix)
		}
	}
}

func renderMetalCircle(dst *PixBuf, p *model.Primitive, textures []*Texture) {
	base := primitiveColor(p)
	cx := float64(dst.Width-1) * 0.5
	cy := float64(dst.Height-1) * 0.5
	r := float64(min(dst.Width, dst.Height)) * 0.46
	for y := 0; y < dst.Height; y++ {
		for x := 0; x < dst.Width; x++ {
			dx := (float64(x) - cx) / r
			dy := (float64(y) - cy) / r
			d2 := dx*dx + dy*dy
			if d2 > 1 {
				continue
			}
			band := 0.5 + 0.5*math.Cos((dx+0.3)*math.Pi*2.0)
			shade := 0.35 + 0.65*band
			pix := color.RGBA{
				R: uint8(clamp01(float64(base.R)/255.0*shade) * 255),
				G: uint8(clamp01(float64(base.G)/255.0*shade) * 255),
				B: uint8(clamp01(float64(base.B)/255.0*shade) * 255),
				A: base.A,
			}
			pix = applyTextureOverlay(pix, textures, p, x, y)
			dst.BlendOver(x, y, pix)
		}
	}
}

func renderWaveCircle(dst *PixBuf, p *model.Primitive) {
	c := primitiveColor(p)
	cx := float64(dst.Width-1) * 0.5
	cy := float64(dst.Height-1) * 0.5
	baseR := float64(min(dst.Width, dst.Height)) * 0.42
	amp := math.Max(1, p.Length.Val*0.01*float64(min(dst.Width, dst.Height))*0.2)
	freq := math.Max(2, p.Step.Val*0.1)
	stroke := math.Max(1, p.Width.Val*0.01*float64(min(dst.Width, dst.Height))*0.25)
	for y := 0; y < dst.Height; y++ {
		for x := 0; x < dst.Width; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			ang := math.Atan2(dy, dx)
			r := math.Hypot(dx, dy)
			want := baseR + amp*math.Sin(freq*ang)
			if math.Abs(r-want) <= stroke {
				dst.BlendOver(x, y, c)
			}
		}
	}
}

func renderSphere(dst *PixBuf, p *model.Primitive, textures []*Texture) {
	base := primitiveColor(p)
	cx := float64(dst.Width-1) * 0.5
	cy := float64(dst.Height-1) * 0.5
	rx := float64(dst.Width) * 0.45
	ry := float64(dst.Height) * 0.45
	for y := 0; y < dst.Height; y++ {
		for x := 0; x < dst.Width; x++ {
			nx, ny, nz, ok := SphereNormal(float64(x), float64(y), cx, cy, rx, ry)
			if !ok {
				continue
			}
			pix := PhongLighting(
				[3]float64{nx, ny, nz},
				p.LightDir.Val,
				p.Ambient.Val,
				p.Diffuse.Val+60,
				p.Specular.Val,
				p.SpecularWidth.Val,
				base,
			)
			pix = applyTextureOverlay(pix, textures, p, x, y)
			dst.BlendOver(x, y, pix)
		}
	}
}

func renderRect(dst *PixBuf, p *model.Primitive, outline bool) {
	c := primitiveColor(p)
	w := int(math.Max(1, p.Length.Val*0.01*float64(dst.Width)))
	h := int(math.Max(1, p.Aspect.Val*0.01*float64(dst.Height)))
	if h <= 1 {
		h = int(0.6 * float64(dst.Height))
	}
	x0 := (dst.Width - w) / 2
	y0 := (dst.Height - h) / 2
	x1 := x0 + w
	y1 := y0 + h
	if !outline {
		dst.FillRect(x0, y0, x1, y1, c)
		return
	}
	stroke := int(math.Max(1, p.Width.Val*0.02*float64(min(dst.Width, dst.Height))))
	dst.FillRect(x0, y0, x1, y0+stroke, c)
	dst.FillRect(x0, y1-stroke, x1, y1, c)
	dst.FillRect(x0, y0, x0+stroke, y1, c)
	dst.FillRect(x1-stroke, y0, x1, y1, c)
}

func renderTriangle(dst *PixBuf, p *model.Primitive) {
	c := primitiveColor(p)
	cx := dst.Width / 2
	cy := dst.Height / 2
	h := int(math.Max(2, p.Length.Val*0.01*float64(dst.Height)))
	w := int(math.Max(2, p.Width.Val*0.02*float64(dst.Width)))

	a := point{cx, cy - h/2}
	b := point{cx - w, cy + h/2}
	cc := point{cx + w, cy + h/2}

	if p.Fill.Val != 0 {
		fillTriangle(dst, a, b, cc, c)
		return
	}
	drawLine(dst, a.x, a.y, b.x, b.y, c, 1)
	drawLine(dst, b.x, b.y, cc.x, cc.y, c, 1)
	drawLine(dst, cc.x, cc.y, a.x, a.y, c, 1)
}

func renderLine(dst *PixBuf, p *model.Primitive) {
	c := primitiveColor(p)
	ang := p.LightDir.Val * math.Pi / 180.0
	length := math.Max(1, p.Length.Val*0.01*float64(min(dst.Width, dst.Height)))
	cx := float64(dst.Width-1) * 0.5
	cy := float64(dst.Height-1) * 0.5
	dx := math.Cos(ang) * length * 0.5
	dy := math.Sin(ang) * length * 0.5
	w := int(math.Max(1, p.Width.Val*0.02*float64(min(dst.Width, dst.Height))))
	drawLine(dst, int(cx-dx), int(cy-dy), int(cx+dx), int(cy+dy), c, w)
}

func renderRadiateLines(dst *PixBuf, p *model.Primitive) {
	c := primitiveColor(p)
	step := p.AngleStep.Val
	if step <= 0 {
		step = 30
	}
	length := math.Max(1, p.Length.Val*0.01*float64(min(dst.Width, dst.Height)))
	cx := float64(dst.Width-1) * 0.5
	cy := float64(dst.Height-1) * 0.5
	w := int(math.Max(1, p.Width.Val*0.02*float64(min(dst.Width, dst.Height))))
	for a := 0.0; a < 360.0; a += step {
		rad := a * math.Pi / 180.0
		x1 := int(cx + math.Cos(rad)*length)
		y1 := int(cy + math.Sin(rad)*length)
		drawLine(dst, int(cx), int(cy), x1, y1, c, w)
	}
}

func renderParallelLines(dst *PixBuf, p *model.Primitive, horizontal bool) {
	c := primitiveColor(p)
	step := int(math.Max(1, p.Step.Val*0.01*float64(min(dst.Width, dst.Height))))
	stroke := int(math.Max(1, p.Width.Val*0.02*float64(min(dst.Width, dst.Height))))
	if horizontal {
		for y := 0; y < dst.Height; y += step {
			dst.FillRect(0, y, dst.Width, y+stroke, c)
		}
		return
	}
	for x := 0; x < dst.Width; x += step {
		dst.FillRect(x, 0, x+stroke, dst.Height, c)
	}
}

func renderText(dst *PixBuf, p *model.Primitive, frame, total int) {
	c := primitiveColor(p)
	txt := strings.TrimSpace(SubstituteFrameCounters(p.Text.Val, frame, total))
	if txt == "" {
		txt = "TEXT"
	}
	size := p.FontSize.Val * 0.01
	if size <= 0 {
		size = 0.5
	}
	charH := int(math.Max(6, float64(min(dst.Width, dst.Height))*size))
	charW := int(math.Max(4, float64(charH)*0.62))
	spacing := int(math.Max(1, float64(charW)/4.0))
	runes := []rune(txt)
	totalW := len(runes) * charW
	if len(runes) > 1 {
		totalW += (len(runes) - 1) * spacing
	}
	x := (dst.Width - totalW) / 2
	switch p.TextAlign.Val {
	case 1: // left
		x = spacing / 2
	case 2: // right
		x = dst.Width - totalW - spacing/2
	}
	y := (dst.Height - charH) / 2
	for _, r := range runes {
		if r != ' ' {
			dst.FillRect(x, y, x+charW, y+charH, c)
			dst.FillRect(x+1, y+1, x+charW-1, y+charH-1, color.RGBA{0, 0, 0, 0})
		}
		x += charW + spacing
	}
}

func renderShape(dst *PixBuf, p *model.Primitive) {
	c := primitiveColor(p)
	s := strings.TrimSpace(p.Shape.Val)
	if s == "" {
		// Fallback: diamond marker to visualize missing path data.
		cx := dst.Width / 2
		cy := dst.Height / 2
		r := min(dst.Width, dst.Height) / 4
		fillTriangle(dst, point{cx, cy - r}, point{cx - r, cy}, point{cx + r, cy}, c)
		fillTriangle(dst, point{cx, cy + r}, point{cx - r, cy}, point{cx + r, cy}, c)
		return
	}
	polys := parseKnobShapePolylines(s, dst.Width, dst.Height)
	if len(polys) == 0 {
		// Fallback parser for simple SVG-style M/L commands.
		pts := parseSimpleShapePoints(s, dst.Width, dst.Height)
		if len(pts) < 2 {
			return
		}
		if p.Fill.Val != 0 && len(pts) >= 3 {
			for i := 1; i+1 < len(pts); i++ {
				fillTriangle(dst, pts[0], pts[i], pts[i+1], c)
			}
			return
		}
		stroke := int(math.Max(1, p.Width.Val*0.02*float64(min(dst.Width, dst.Height))))
		for i := 0; i < len(pts)-1; i++ {
			drawLine(dst, pts[i].x, pts[i].y, pts[i+1].x, pts[i+1].y, c, stroke)
		}
		return
	}

	if p.Fill.Val != 0 {
		for y := 0; y < dst.Height; y++ {
			py := float64(y) + 0.5
			for x := 0; x < dst.Width; x++ {
				px := float64(x) + 0.5
				if pointInPolysEvenOdd(px, py, polys) {
					dst.BlendOver(x, y, c)
				}
			}
		}
		return
	}

	stroke := int(math.Max(1, p.Width.Val*0.02*float64(min(dst.Width, dst.Height))))
	for _, poly := range polys {
		for i := 0; i+1 < len(poly); i++ {
			drawLine(dst, int(poly[i].x+0.5), int(poly[i].y+0.5), int(poly[i+1].x+0.5), int(poly[i+1].y+0.5), c, stroke)
		}
		if len(poly) > 2 {
			a := poly[len(poly)-1]
			b := poly[0]
			drawLine(dst, int(a.x+0.5), int(a.y+0.5), int(b.x+0.5), int(b.y+0.5), c, stroke)
		}
	}
}

func parseSimpleShapePoints(s string, w, h int) []point {
	fields := strings.Fields(strings.NewReplacer(",", " ", "M", " ", "L", " ", "m", " ", "l", " ", "Z", " ", "z", " ").Replace(s))
	pts := make([]point, 0, len(fields)/2)
	for i := 0; i+1 < len(fields); i += 2 {
		x, okx := parseFloat(fields[i])
		y, oky := parseFloat(fields[i+1])
		if !okx || !oky {
			continue
		}
		px := int(clamp01(x/100.0)*float64(w-1) + 0.5)
		py := int(clamp01(y/100.0)*float64(h-1) + 0.5)
		pts = append(pts, point{px, py})
	}
	return pts
}

type fpoint struct{ x, y float64 }

func parseKnobShapePolylines(s string, w, h int) [][]fpoint {
	if !strings.Contains(s, "/") {
		return nil
	}
	parts := strings.Split(s, "/")
	var polys [][]fpoint
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		chunks := strings.Split(part, ":")
		type knot struct {
			inX, inY, pX, pY, outX, outY float64
		}
		knots := make([]knot, 0, len(chunks))
		for _, ch := range chunks {
			vals := strings.Split(ch, ",")
			if len(vals) != 6 {
				continue
			}
			var k knot
			ok := true
			nums := []*float64{&k.inX, &k.inY, &k.pX, &k.pY, &k.outX, &k.outY}
			for i := 0; i < 6; i++ {
				v, err := strconv.ParseFloat(strings.TrimSpace(vals[i]), 64)
				if err != nil {
					ok = false
					break
				}
				*nums[i] = v
			}
			if ok {
				knots = append(knots, k)
			}
		}
		if len(knots) < 2 {
			continue
		}
		poly := make([]fpoint, 0, len(knots)*12)
		start := fpoint{x: shapeScaleX(knots[0].pX, w), y: shapeScaleY(knots[0].pY, h)}
		poly = append(poly, start)
		for i := 1; i < len(knots); i++ {
			prev := knots[i-1]
			cur := knots[i]
			cubic := flattenCubic(
				fpoint{x: shapeScaleX(prev.pX, w), y: shapeScaleY(prev.pY, h)},
				fpoint{x: shapeScaleX(prev.outX, w), y: shapeScaleY(prev.outY, h)},
				fpoint{x: shapeScaleX(cur.inX, w), y: shapeScaleY(cur.inY, h)},
				fpoint{x: shapeScaleX(cur.pX, w), y: shapeScaleY(cur.pY, h)},
				12,
			)
			poly = append(poly, cubic[1:]...)
		}
		// Close with last->first cubic like the Java MakePath(fill!=0) path.
		last := knots[len(knots)-1]
		first := knots[0]
		cubic := flattenCubic(
			fpoint{x: shapeScaleX(last.pX, w), y: shapeScaleY(last.pY, h)},
			fpoint{x: shapeScaleX(last.outX, w), y: shapeScaleY(last.outY, h)},
			fpoint{x: shapeScaleX(first.inX, w), y: shapeScaleY(first.inY, h)},
			fpoint{x: shapeScaleX(first.pX, w), y: shapeScaleY(first.pY, h)},
			12,
		)
		poly = append(poly, cubic[1:]...)
		polys = append(polys, poly)
	}
	return polys
}

func shapeScaleX(v float64, w int) float64 { return (v - 128.0) / 256.0 * float64(w) }
func shapeScaleY(v float64, h int) float64 { return (v - 128.0) / 256.0 * float64(h) }

func flattenCubic(p0, c1, c2, p1 fpoint, steps int) []fpoint {
	if steps < 2 {
		steps = 2
	}
	out := make([]fpoint, 0, steps+1)
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		u := 1.0 - t
		x := u*u*u*p0.x + 3*u*u*t*c1.x + 3*u*t*t*c2.x + t*t*t*p1.x
		y := u*u*u*p0.y + 3*u*u*t*c1.y + 3*u*t*t*c2.y + t*t*t*p1.y
		out = append(out, fpoint{x: x, y: y})
	}
	return out
}

func pointInPolysEvenOdd(x, y float64, polys [][]fpoint) bool {
	inside := false
	for _, poly := range polys {
		if len(poly) < 3 {
			continue
		}
		j := len(poly) - 1
		for i := 0; i < len(poly); i++ {
			xi, yi := poly[i].x, poly[i].y
			xj, yj := poly[j].x, poly[j].y
			intersect := ((yi > y) != (yj > y)) && (x < (xj-xi)*(y-yi)/(yj-yi+1e-12)+xi)
			if intersect {
				inside = !inside
			}
			j = i
		}
	}
	return inside
}

func parseFloat(s string) (float64, bool) {
	v, err := strconvParseFloat(s)
	if err != nil {
		return 0, false
	}
	return v, true
}

// strconvParseFloat is split for testability and to keep parse helpers close.
var strconvParseFloat = func(s string) (float64, error) { return strconv.ParseFloat(s, 64) }

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func applyTextureOverlay(base color.RGBA, textures []*Texture, p *model.Primitive, x, y int) color.RGBA {
	idx := p.TextureFile.Val
	if idx <= 0 || idx > len(textures) {
		return base
	}
	tex := textures[idx-1]
	if tex == nil {
		return base
	}
	zoom := p.TextureZoom.Val / 100.0
	if zoom <= 0 {
		zoom = 1
	}
	tx := float64(x)
	ty := float64(y)
	tc := tex.Sample(tx, ty, zoom)
	return TextureBlend(base, tc, p.TextureDepth.Val)
}

type point struct{ x, y int }

func fillTriangle(dst *PixBuf, p0, p1, p2 point, c color.RGBA) {
	minX := min(p0.x, min(p1.x, p2.x))
	maxX := max(p0.x, max(p1.x, p2.x))
	minY := min(p0.y, min(p1.y, p2.y))
	maxY := max(p0.y, max(p1.y, p2.y))
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= dst.Width {
		maxX = dst.Width - 1
	}
	if maxY >= dst.Height {
		maxY = dst.Height - 1
	}
	area := edge(p0, p1, p2)
	if area == 0 {
		return
	}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			p := point{x, y}
			w0 := edge(p1, p2, p)
			w1 := edge(p2, p0, p)
			w2 := edge(p0, p1, p)
			if (w0 >= 0 && w1 >= 0 && w2 >= 0) || (w0 <= 0 && w1 <= 0 && w2 <= 0) {
				dst.BlendOver(x, y, c)
			}
		}
	}
}

func edge(a, b, c point) int {
	return (c.x-a.x)*(b.y-a.y) - (c.y-a.y)*(b.x-a.x)
}

func drawLine(dst *PixBuf, x0, y0, x1, y1 int, c color.RGBA, width int) {
	if width < 1 {
		width = 1
	}
	dx := int(math.Abs(float64(x1 - x0)))
	dy := -int(math.Abs(float64(y1 - y0)))
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		hw := width / 2
		for oy := -hw; oy <= hw; oy++ {
			for ox := -hw; ox <= hw; ox++ {
				dst.BlendOver(x0+ox, y0+oy, c)
			}
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
