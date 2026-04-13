package render

import agg "github.com/cwbudde/agg_go"

func measureLocalGSVTextWidth(text string, size float64) float64 {
	gsv := agg.NewGSVText()
	gsv.SetFlip(true)
	gsv.SetSize(size, 0)
	return gsv.MeasureText(text)
}

func drawLocalGSVText(a *agg.Agg2D, x, y, size float64, text string) bool {
	if a == nil || text == "" {
		return false
	}

	gsv := agg.NewGSVText()
	gsv.SetFlip(true)
	gsv.SetSize(size, 0)
	gsv.SetText(text)
	gsv.SetStartPoint(float64(int(x)), float64(int(y)))

	a.ResetPath()
	gsv.Rewind(0)

	hasVertices := false
	for {
		vx, vy, cmd := gsv.Vertex()
		switch cmd {
		case agg.GSVPathCmdStop:
			return hasVertices
		case agg.GSVPathCmdMoveTo:
			a.MoveTo(vx, vy)
			hasVertices = true
		case agg.GSVPathCmdLineTo:
			a.LineTo(vx, vy)
			hasVertices = true
		}
	}
}
