package render

// CompatibilityMode toggles targeted legacy rendering behavior at render time.
type CompatibilityMode string

const (
	CompatibilityDefault            CompatibilityMode = ""
	CompatibilityJavaTriangleRaster CompatibilityMode = "java-triangle-raster"
)

// RenderOptions controls non-persistent renderer compatibility switches.
type RenderOptions struct {
	Compatibility CompatibilityMode
}

func DefaultRenderOptions() RenderOptions {
	return RenderOptions{}
}

func (o RenderOptions) useJavaTriangleRaster() bool {
	return o.Compatibility == CompatibilityJavaTriangleRaster
}
