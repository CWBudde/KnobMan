package render

import agg "github.com/cwbudde/agg_go"

func measureLocalGSVTextWidth(text string, size float64) float64 {
	gsv := agg.NewGSVText()
	gsv.SetFlip(true)
	gsv.SetSize(size, 0)

	return gsv.MeasureText(text)
}

func appendLocalGSVText(ctx *agg.Context, x, y, size float64, text string) bool {
	if ctx == nil || text == "" {
		return false
	}

	gsv := agg.NewGSVText()
	gsv.SetFlip(true)
	gsv.SetSize(size, 0)
	gsv.SetText(text)
	gsv.SetStartPoint(float64(int(x)), float64(int(y)))

	ctx.BeginPath()
	gsv.Rewind(0)

	hasVertices := false

	for {
		vx, vy, cmd := gsv.Vertex()
		switch cmd {
		case agg.GSVPathCmdStop:
			return hasVertices
		case agg.GSVPathCmdMoveTo:
			ctx.MoveTo(vx, vy)

			hasVertices = true
		case agg.GSVPathCmdLineTo:
			ctx.LineTo(vx, vy)

			hasVertices = true
		}
	}
}

func appendFreeTypeOutlineText(ctx *agg.Context, text *agg.FreeTypeOutlineText) bool {
	if ctx == nil || text == nil {
		return false
	}

	ctx.BeginPath()
	text.Rewind(0)

	hasVertices := false

	for {
		x1, y1, cmd := text.Vertex()
		switch {
		case cmd == agg.PathCmdStop:
			return hasVertices
		case cmd == agg.PathCmdMoveTo:
			ctx.MoveTo(x1, y1)

			hasVertices = true
		case cmd == agg.PathCmdLineTo:
			ctx.LineTo(x1, y1)

			hasVertices = true
		case agg.IsPathCurve3(cmd):
			x2, y2, cmd2 := text.Vertex()
			if !agg.IsPathCurve3(cmd2) {
				return hasVertices
			}

			ctx.QuadricCurveTo(x1, y1, x2, y2)

			hasVertices = true
		case agg.IsPathCurve4(cmd):
			x2, y2, cmd2 := text.Vertex()

			x3, y3, cmd3 := text.Vertex()
			if !agg.IsPathCurve4(cmd2) || !agg.IsPathCurve4(cmd3) {
				return hasVertices
			}

			ctx.CubicCurveTo(x1, y1, x2, y2, x3, y3)

			hasVertices = true
		case agg.IsPathEndPoly(cmd):
			if agg.IsPathClose(cmd) {
				ctx.ClosePath()
			}
		}
	}
}
