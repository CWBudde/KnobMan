package render

import (
	"image/color"
	"math"
)

// SphereNormal returns a normalized sphere normal at x,y.
func SphereNormal(x, y, cx, cy, rx, ry float64) (nx, ny, nz float64, ok bool) {
	if rx <= 0 || ry <= 0 {
		return 0, 0, 0, false
	}

	dx := (x - cx) / rx
	dy := (y - cy) / ry

	r2 := dx*dx + dy*dy
	if r2 > 1 {
		return 0, 0, 0, false
	}

	dz := math.Sqrt(1 - r2)

	n := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if n == 0 {
		return 0, 0, 1, true
	}

	return dx / n, dy / n, dz / n, true
}

// PhongLighting evaluates a basic ambient+diffuse+specular light model.
func PhongLighting(normal [3]float64, lightDir, ambient, diffuse, specular, specWidth float64, base color.RGBA) color.RGBA {
	lx := math.Cos(lightDir * math.Pi / 180.0)
	ly := -math.Sin(lightDir * math.Pi / 180.0)
	lz := 0.7

	ln := math.Sqrt(lx*lx + ly*ly + lz*lz)
	if ln == 0 {
		ln = 1
	}

	lx, ly, lz = lx/ln, ly/ln, lz/ln

	nx, ny, nz := normal[0], normal[1], normal[2]

	ndotl := nx*lx + ny*ly + nz*lz
	if ndotl < 0 {
		ndotl = 0
	}

	// Reflection vs fixed view vector (0,0,1)
	rz := 2*ndotl*nz - lz

	vdotr := rz
	if vdotr < 0 {
		vdotr = 0
	}

	shininess := 8.0 + specWidth*0.8
	spec := math.Pow(vdotr, shininess) * (specular / 100.0)

	amb := ambient / 100.0
	dif := ndotl * (diffuse / 100.0)
	scale := amb + dif

	r := clamp01(float64(base.R)/255.0*scale + spec)
	g := clamp01(float64(base.G)/255.0*scale + spec)
	b := clamp01(float64(base.B)/255.0*scale + spec)

	return color.RGBA{R: uint8(r*255 + 0.5), G: uint8(g*255 + 0.5), B: uint8(b*255 + 0.5), A: base.A}
}

// TextureBlend linearly blends base with tex by depth in [0,100].
func TextureBlend(base, tex color.RGBA, depth float64) color.RGBA {
	t := clamp01(depth / 100.0)
	blend := func(a, b uint8) uint8 {
		return uint8(float64(a) + (float64(b)-float64(a))*t + 0.5)
	}

	return color.RGBA{
		R: blend(base.R, tex.R),
		G: blend(base.G, tex.G),
		B: blend(base.B, tex.B),
		A: base.A,
	}
}
