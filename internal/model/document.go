package model

// Document is the top-level KnobMan project.
type Document struct {
	Prefs  Prefs
	Curves [8]AnimCurve
	Layers []Layer
}

// NewDocument returns a document with 3 empty layers matching Java's New().
func NewDocument() *Document {
	doc := &Document{
		Prefs: NewPrefs(),
	}
	for i := range doc.Curves {
		doc.Curves[i] = NewAnimCurve()
	}
	doc.Layers = []Layer{NewLayer(), NewLayer(), NewLayer()}
	return doc
}

// Clone returns a deep copy of the document (used by the undo system).
func (d *Document) Clone() *Document {
	c := &Document{
		Prefs:  d.Prefs,
		Curves: d.Curves,
	}
	c.Layers = make([]Layer, len(d.Layers))
	for i, l := range d.Layers {
		c.Layers[i] = l.Clone()
	}
	return c
}
