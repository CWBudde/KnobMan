package model

// Layer is one compositing layer in the document.
type Layer struct {
	Name    string
	Visible BoolParam // 1 = visible (default)
	Solo    BoolParam // 1 = solo (isolates this layer in preview)
	Prim    Primitive
	Eff     Effect
}

// NewLayer returns a layer with sensible defaults and no primitive.
func NewLayer() Layer {
	return Layer{
		Visible: BoolParam{1},
		Prim:    NewPrimitive(),
		Eff:     NewEffect(),
	}
}

// Clone returns a deep copy of the layer.
func (l Layer) Clone() Layer {
	c := l
	// Slices in Primitive that need deep copying
	if l.Prim.EmbeddedImage != nil {
		c.Prim.EmbeddedImage = append([]byte(nil), l.Prim.EmbeddedImage...)
	}
	if l.Prim.EmbeddedTexture != nil {
		c.Prim.EmbeddedTexture = append([]byte(nil), l.Prim.EmbeddedTexture...)
	}
	return c
}
