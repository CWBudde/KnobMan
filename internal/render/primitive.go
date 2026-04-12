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

	agg "github.com/cwbudde/agg_go"
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
		renderCircle(dst, p, false, textures)
	case model.PrimWaveCircle:
		renderWaveCircle(dst, p, textures)
	case model.PrimSphere:
		renderCircle(dst, p, false, textures)
	case model.PrimRect:
		renderRect(dst, p, true, textures)
	case model.PrimRectFill:
		renderRect(dst, p, false, textures)
	case model.PrimTriangle:
		renderTriangle(dst, p, textures)
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
		renderShape(dst, p, textures)
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

	src := ImageToPixBuf(img)
	if src == nil || src.Width <= 0 || src.Height <= 0 {
		return
	}

	nf := max(p.NumFrame.Val, 1)

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

func drawPixBufToRect(dst, src *PixBuf, dx0, dy0, dw, dh int, p *model.Primitive, def color.RGBA) {
	if src == nil || src.Width <= 0 || src.Height <= 0 || dw <= 0 || dh <= 0 {
		return
	}

	if p != nil && p.Transparent.Val == 0 && pixBufRegionTransparent(dst, dx0, dy0, dw, dh) && drawPixBufToRectAgg(dst, src, dx0, dy0, dw, dh) {
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

func drawPixBufToRectAgg(dst, src *PixBuf, dx0, dy0, dw, dh int) bool {
	if dst == nil || src == nil || dw <= 0 || dh <= 0 {
		return false
	}

	a := Agg2DForPixBuf(dst)

	srcImg := AggImageForPixBuf(src)
	if a == nil || srcImg == nil {
		return false
	}

	a.ImageFilter(agg.FilterNoFilter)
	a.ImageResample(agg.NoResample)

	err := a.TransformImageSimple(srcImg, float64(dx0), float64(dy0), float64(dx0+dw), float64(dy0+dh))
	if err != nil {
		return false
	}

	return true
}

func pixBufRegionTransparent(buf *PixBuf, dx0, dy0, dw, dh int) bool {
	if buf == nil || dw <= 0 || dh <= 0 {
		return false
	}

	maxX := min(buf.Width, dx0+dw)

	maxY := min(buf.Height, dy0+dh)
	for y := max(0, dy0); y < maxY; y++ {
		row := y * buf.Stride
		for x := max(0, dx0); x < maxX; x++ {
			if buf.Data[row+x*4+3] != 0 {
				return false
			}
		}
	}

	return true
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

	alphaStep := max(255-intelliAlpha*254/100, 16)

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

func renderText(dst *PixBuf, p *model.Primitive, frame, total int) {
	if dst == nil || p == nil {
		return
	}

	txt := ResolveDynamicText(p.Text.Val, frame, total)
	if txt == "" {
		txt = "TEXT"
	}

	size := p.FontSize.Val * 0.01 * float64(dst.Height)
	size = math.Floor(size)
	if size < 6 {
		size = 6
	}

	ctx := AggContextForPixBuf(dst)
	if ctx == nil {
		return
	}

	ctx.SetColor(agg.Color{R: primitiveColor(p).R, G: primitiveColor(p).G, B: primitiveColor(p).B, A: primitiveColor(p).A})
	ctx.TextHints(true)
	backend, textSize := configureAggTextFont(ctx, p, size)

	a := ctx.GetAgg2D()
	textWidth := a.TextWidth(txt)

	spaceWidth := a.TextWidth(" ")
	if spaceWidth <= 0 {
		spaceWidth = math.Max(1, textSize*0.25)
	}

	anchorX := (float64(dst.Width) - textWidth) * 0.5
	switch p.TextAlign.Val {
	case 1:
		anchorX = spaceWidth * 0.5
	case 2:
		anchorX = float64(dst.Width) - textWidth - spaceWidth*0.5
	}

	ascent := ctx.GetAscender()
	descent := -ctx.GetDescender()
	if backend == aggTextBackendGSV && ascent <= 0 && descent <= 0 {
		// GSV has no font metrics; approximate Java's baseline formula.
		ascent = textSize * 0.8
		descent = textSize * 0.2
	}
	if ascent <= 0 && descent <= 0 {
		ascent = size
	}

	anchorY := (float64(dst.Height) + ascent - descent) * 0.5

	a.TextAlignment(agg.AlignLeft, agg.AlignBottom)

	if backend == aggTextBackendGSV {
		renderTextGSVStyled(ctx, a, p, anchorX, anchorY, txt)
		return
	}

	a.Text(anchorX, anchorY, txt, true, 0, 0)
}

func renderTextGSVStyled(ctx *agg.Context, a *agg.Agg2D, p *model.Primitive, x, y float64, txt string) {
	if ctx == nil || a == nil {
		return
	}

	if p.Italic.Val != 0 {
		ctx.PushTransform()
		ctx.Translate(x, y)
		ctx.Skew(-12*math.Pi/180, 0)

		ctx.Translate(-x, -y)
		defer ctx.PopTransform()
	}

	a.Text(x, y, txt, true, 0, 0)

	if p.Bold.Val != 0 {
		a.Text(x, y, txt, true, 0.7, 0)
		a.Text(x, y, txt, true, 1.4, 0)
	}
}

func renderShape(dst *PixBuf, p *model.Primitive, textures []*Texture) {
	base := primitiveColor(p)

	s := strings.TrimSpace(p.Shape.Val)
	if s == "" {
		return
	}

	mask := NewPixBuf(dst.Width, dst.Height)
	mask.Clear(color.RGBA{A: 255})

	maskCtx := AggContextForPixBuf(mask)
	if maskCtx == nil {
		return
	}

	maskCtx.Clear(agg.Color{A: 255})
	a := maskCtx.GetAgg2D()
	a.FillEvenOdd(true)
	maskCtx.BeginPath()

	if !appendShapePath(maskCtx, s, dst.Width, dst.Height, p.Fill.Val != 0) {
		return
	}

	blue := agg.Color{B: 255, A: 255}
	if p.Fill.Val != 0 {
		maskCtx.SetColor(blue)
		a.NoLine()
		maskCtx.Fill()
	} else {
		renderShapeOutlineMask(mask, s, dst.Width, dst.Height, p.Width.Val*float64(dst.Width)*0.004)
	}

	rCX := float64(dst.Width) * 0.5

	rCY := float64(dst.Height) * 0.5
	for y := range dst.Height {
		rY := -((float64(y) + 0.5) - rCY)
		rYN := rY / rCY

		rPY := float64(y) - rCY + 0.5
		for x := range dst.Width {
			i := y*mask.Stride + x*4

			coverage := int(mask.Data[i+2])
			if coverage == 0 {
				continue
			}

			lumi := 0
			alpha := 255
			rPX := float64(x) - rCX + 0.5
			rX := -((float64(x) + 0.5) - rCX)
			rXN := rX / rCX
			lumi, alpha = sampleTextureLumiAlpha(textures, p, rPX, rPY)

			col := base
			col = changeBrightnessRGBA(col, int((rXN+rYN)*128.0*p.Specular.Val*0.01)+lumi)
			col.A = uint8(clampInt(coverage*alpha/255, 0, 255))
			dst.Set(x, y, col)
		}
	}
}

func renderShapeOutlineMask(mask *PixBuf, s string, w, h int, strokeWidth float64) {
	if mask == nil || strokeWidth <= 0 {
		return
	}

	polys := parseKnobShapeAnchorPolylines(s, w, h)
	if len(polys) == 0 {
		pts := parseSimpleShapePoints(s, w, h)
		if len(pts) < 2 {
			return
		}

		poly := make([]fpoint, len(pts))
		for i := range pts {
			poly[i] = fpoint{x: float64(pts[i].x), y: float64(pts[i].y)}
		}

		polys = [][]fpoint{poly}
	}

	const samples = 16
	half := strokeWidth * 0.5

	for y := range h {
		for x := range w {
			hits := 0

			for sy := range samples {
				py := float64(y) + (float64(sy)+0.5)/samples
				for sx := range samples {
					px := float64(x) + (float64(sx)+0.5)/samples
					if pointOnStrokePolys(px, py, polys, half) {
						hits++
					}
				}
			}

			if hits == 0 {
				continue
			}

			cov := uint8(clampInt(int(float64(hits)*255.0/float64(samples*samples)+0.5), 0, 255))
			mask.Set(x, y, color.RGBA{B: cov, A: 255})
		}
	}
}

func pointOnStrokePolys(px, py float64, polys [][]fpoint, halfWidth float64) bool {
	for _, poly := range polys {
		for i := 0; i+1 < len(poly); i++ {
			if pointInSquareCappedSegment(px, py, poly[i], poly[i+1], halfWidth) {
				return true
			}
		}
	}

	return false
}

func pointInSquareCappedSegment(px, py float64, a, b fpoint, halfWidth float64) bool {
	dx := b.x - a.x
	dy := b.y - a.y

	l := math.Hypot(dx, dy)
	if l == 0 {
		return math.Hypot(px-a.x, py-a.y) <= halfWidth
	}

	ux := dx / l
	uy := dy / l
	vx := px - a.x
	vy := py - a.y
	t := vx*ux + vy*uy
	n := -vx*uy + vy*ux

	return t >= -halfWidth && t <= l+halfWidth && math.Abs(n) <= halfWidth
}

func appendShapePath(ctx *agg.Context, s string, w, h int, closePath bool) bool {
	if polys := parseKnobShapeKnots(s); len(polys) > 0 {
		for _, knots := range polys {
			if len(knots) < 2 {
				continue
			}

			ctx.MoveTo(shapeScaleX(knots[0].pX, w), shapeScaleY(knots[0].pY, h))

			for i := 1; i < len(knots); i++ {
				prev := knots[i-1]
				cur := knots[i]
				ctx.GetAgg2D().CubicCurveTo(
					shapeScaleX(prev.outX, w), shapeScaleY(prev.outY, h),
					shapeScaleX(cur.inX, w), shapeScaleY(cur.inY, h),
					shapeScaleX(cur.pX, w), shapeScaleY(cur.pY, h),
				)
			}

			if closePath {
				last := knots[len(knots)-1]
				first := knots[0]
				ctx.GetAgg2D().CubicCurveTo(
					shapeScaleX(last.outX, w), shapeScaleY(last.outY, h),
					shapeScaleX(first.inX, w), shapeScaleY(first.inY, h),
					shapeScaleX(first.pX, w), shapeScaleY(first.pY, h),
				)
			}
		}

		return true
	}

	pts := parseSimpleShapePoints(s, w, h)
	if len(pts) < 2 {
		return false
	}

	ctx.MoveTo(float64(pts[0].x), float64(pts[0].y))

	for i := 1; i < len(pts); i++ {
		ctx.LineTo(float64(pts[i].x), float64(pts[i].y))
	}

	if closePath {
		ctx.ClosePath()
	}

	return true
}

type svgPathCmd struct {
	op   byte
	vals []float64
}

func parseSimpleShapePoints(s string, w, h int) []point {
	cmds := parseSVGPathCommands(s)
	if len(cmds) == 0 {
		return parseLooseShapePairs(s, w, h)
	}

	toPix := func(x, y float64) point {
		px := int(clamp01(x/100.0)*float64(w-1) + 0.5)
		py := int(clamp01(y/100.0)*float64(h-1) + 0.5)

		return point{px, py}
	}

	pts := make([]point, 0, 64)
	var curX, curY float64
	var startX, startY float64
	hasCur := false

	for _, c := range cmds {
		switch c.op {
		case 'M':
			curX, curY = c.vals[0], c.vals[1]
			startX, startY = curX, curY
			hasCur = true

			pts = append(pts, toPix(curX, curY))
		case 'L':
			if !hasCur {
				continue
			}

			curX, curY = c.vals[0], c.vals[1]
			pts = append(pts, toPix(curX, curY))
		case 'Q':
			if !hasCur {
				continue
			}

			poly := flattenQuadratic(
				fpoint{x: curX, y: curY},
				fpoint{x: c.vals[0], y: c.vals[1]},
				fpoint{x: c.vals[2], y: c.vals[3]},
				12,
			)
			for i := 1; i < len(poly); i++ {
				pts = append(pts, toPix(poly[i].x, poly[i].y))
			}

			curX, curY = c.vals[2], c.vals[3]
		case 'C':
			if !hasCur {
				continue
			}

			poly := flattenCubic(
				fpoint{x: curX, y: curY},
				fpoint{x: c.vals[0], y: c.vals[1]},
				fpoint{x: c.vals[2], y: c.vals[3]},
				fpoint{x: c.vals[4], y: c.vals[5]},
				12,
			)
			for i := 1; i < len(poly); i++ {
				pts = append(pts, toPix(poly[i].x, poly[i].y))
			}

			curX, curY = c.vals[4], c.vals[5]
		case 'Z':
			if !hasCur {
				continue
			}

			curX, curY = startX, startY
			pts = append(pts, toPix(curX, curY))
		}
	}

	return pts
}

func parseLooseShapePairs(s string, w, h int) []point {
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

func parseSVGPathCommands(s string) []svgPathCmd {
	tokens := tokenizeSVGPath(s)
	out := make([]svgPathCmd, 0, 32)
	var cur byte

	for i := 0; i < len(tokens); {
		tk := tokens[i]
		if len(tk) == 1 && isAlphaASCII(tk[0]) {
			cur = byte(strings.ToUpper(tk)[0])
			i++

			if cur == 'Z' {
				out = append(out, svgPathCmd{op: 'Z'})
			}

			continue
		}

		if cur == 0 {
			i++
			continue
		}

		need := svgPathCmdArity(cur)
		if need <= 0 {
			i++
			continue
		}

		vals := make([]float64, 0, need)
		for i < len(tokens) && len(vals) < need {
			if len(tokens[i]) == 1 && isAlphaASCII(tokens[i][0]) {
				break
			}

			v, err := strconv.ParseFloat(tokens[i], 64)
			if err != nil {
				break
			}

			vals = append(vals, v)
			i++
		}

		if len(vals) != need {
			break
		}

		out = append(out, svgPathCmd{op: cur, vals: vals})
		if cur == 'M' {
			cur = 'L'
		}
	}

	return out
}

func tokenizeSVGPath(s string) []string {
	tokens := make([]string, 0, len(s)/2)
	for i := 0; i < len(s); {
		c := s[i]
		if isAlphaASCII(c) {
			tokens = append(tokens, s[i:i+1])
			i++

			continue
		}

		if c == ',' || c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			i++
			continue
		}

		j := i
		if s[j] == '+' || s[j] == '-' {
			j++
		}

		sawDigit := false
		sawDot := false

		for j < len(s) {
			ch := s[j]
			if ch >= '0' && ch <= '9' {
				sawDigit = true
				j++

				continue
			}

			if ch == '.' && !sawDot {
				sawDot = true
				j++

				continue
			}

			break
		}

		if !sawDigit {
			i++
			continue
		}

		tokens = append(tokens, s[i:j])
		i = j
	}

	return tokens
}

func svgPathCmdArity(op byte) int {
	switch op {
	case 'M', 'L':
		return 2
	case 'Q':
		return 4
	case 'C':
		return 6
	case 'Z':
		return 0
	default:
		return -1
	}
}

func isAlphaASCII(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

type fpoint struct{ x, y float64 }

type shapeKnot struct {
	inX, inY, pX, pY, outX, outY float64
}

func parseKnobShapeKnots(s string) [][]shapeKnot {
	if !strings.Contains(s, "/") {
		return nil
	}

	parts := strings.Split(s, "/")
	var polys [][]shapeKnot

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		chunks := strings.Split(part, ":")

		knots := make([]shapeKnot, 0, len(chunks))
		for _, ch := range chunks {
			vals := strings.Split(ch, ",")
			if len(vals) != 6 {
				continue
			}
			var k shapeKnot
			ok := true
			nums := []*float64{&k.inX, &k.inY, &k.pX, &k.pY, &k.outX, &k.outY}

			for i := range 6 {
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

		if len(knots) >= 2 {
			polys = append(polys, knots)
		}
	}

	return polys
}

func parseKnobShapePolylines(s string, w, h int) [][]fpoint {
	return parseKnobShapePolylinesWithClosure(s, w, h, true)
}

func parseKnobShapePolylinesOpen(s string, w, h int) [][]fpoint {
	return parseKnobShapePolylinesWithClosure(s, w, h, false)
}

func parseKnobShapeAnchorPolylines(s string, w, h int) [][]fpoint {
	knotPolys := parseKnobShapeKnots(s)
	if len(knotPolys) == 0 {
		return nil
	}

	polys := make([][]fpoint, 0, len(knotPolys))
	for _, knots := range knotPolys {
		poly := make([]fpoint, 0, len(knots))
		for _, k := range knots {
			poly = append(poly, fpoint{x: shapeScaleX(k.pX, w), y: shapeScaleY(k.pY, h)})
		}

		if len(poly) >= 2 {
			polys = append(polys, poly)
		}
	}

	return polys
}

func parseKnobShapePolylinesWithClosure(s string, w, h int, closePath bool) [][]fpoint {
	knotPolys := parseKnobShapeKnots(s)
	if len(knotPolys) == 0 {
		return nil
	}

	polys := make([][]fpoint, 0, len(knotPolys))
	for _, knots := range knotPolys {
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

		if closePath {
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
		}

		polys = append(polys, poly)
	}

	return polys
}

func shapeScaleX(v float64, w int) float64 { return (v - 128.0) / 256.0 * float64(w) }
func shapeScaleY(v float64, h int) float64 { return (v - 128.0) / 256.0 * float64(h) }

func flattenQuadratic(p0, c, p1 fpoint, steps int) []fpoint {
	if steps < 2 {
		steps = 2
	}

	out := make([]fpoint, 0, steps+1)
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		u := 1.0 - t
		x := u*u*p0.x + 2*u*t*c.x + t*t*p1.x
		y := u*u*p0.y + 2*u*t*c.y + t*t*p1.y
		out = append(out, fpoint{x: x, y: y})
	}

	return out
}

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
		for i := range poly {
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

func linePointDistance(x0, y0, x1, y1, px, py float64) float64 {
	dx := x1 - x0
	dy := y1 - y0

	a := dx*dx + dy*dy
	if a == 0.0 {
		return math.Hypot(x0-px, y0-py)
	}

	b := dx*(x0-px) + dy*(y0-py)

	t := -(b / a)
	if t < 0.0 {
		t = 0.0
	}

	if t >= 1.0 {
		t = 1.0
	}

	x := t*dx + x0
	y := t*dy + y0

	return math.Hypot(x-px, y-py)
}

func rotatePoint(x, y, sinT, cosT float64) (float64, float64) {
	return x*cosT - y*sinT, x*sinT + y*cosT
}

func changeBrightnessRGBA(c color.RGBA, delta int) color.RGBA {
	c.R = uint8(clampInt(int(c.R)+delta, 0, 255))
	c.G = uint8(clampInt(int(c.G)+delta, 0, 255))
	c.B = uint8(clampInt(int(c.B)+delta, 0, 255))

	return c
}

func brightRGBA(c color.RGBA, scale int) color.RGBA {
	c.R = uint8(clampInt(int(c.R)*scale/255, 0, 255))
	c.G = uint8(clampInt(int(c.G)*scale/255, 0, 255))
	c.B = uint8(clampInt(int(c.B)*scale/255, 0, 255))

	return c
}

func blendToRGBA(base, target color.RGBA, alpha int) color.RGBA {
	base.R = uint8((int(base.R)*(256-alpha) + int(target.R)*alpha) / 256)
	base.G = uint8((int(base.G)*(256-alpha) + int(target.G)*alpha) / 256)
	base.B = uint8((int(base.B)*(256-alpha) + int(target.B)*alpha) / 256)

	return base
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

func sampleTextureLumiAlpha(textures []*Texture, p *model.Primitive, x, y float64) (lumi, alpha int) {
	if p == nil || p.TextureDepth.Val == 0 {
		return 0, 255
	}

	idx := p.TextureFile.Val
	if idx <= 0 || idx > len(textures) {
		return 0, 255
	}

	tex := textures[idx-1]
	if tex == nil {
		return 0, 255
	}

	luma, texAlpha := tex.SampleHeightAlpha(x, y, p.TextureZoom.Val)
	txd := p.TextureDepth.Val * 0.01
	lumi = int(float64(luma-128) * txd)
	alpha = 255 - int(float64(255-texAlpha)*txd)

	return lumi, alpha
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
