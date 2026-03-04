package model

import "image/color"

// Prefs holds the document-wide settings.
type Prefs struct {
	Width        int        // canvas width  (pwidth  << oversampling)
	Height       int        // canvas height (pheight << oversampling)
	PWidth       IntParam   // logical canvas width  (default 64)
	PHeight      IntParam   // logical canvas height (default 64)
	Oversampling SelectParam // 0=1× 1=2× 2=4× 3=8×
	PreviewFrames IntParam   // number of preview frames (default 5)
	RenderFrames  IntParam   // number of export frames  (default 31)
	BkColor      ColorParam  // background colour (default white)
	AlignHorz    SelectParam // strip orientation: 0=vertical 1=horizontal
	ExportOption SelectParam // 0=strip-V 1=strip-H 2=frames 3=GIF 4=APNG
	Duration     IntParam    // ms per frame for animated exports (default 100)
	Loop         IntParam    // loop count (0=infinite)
	BiDir        BoolParam   // ping-pong animation
}

// NewPrefs returns a Prefs with default values matching the Java original.
func NewPrefs() Prefs {
	return Prefs{
		Width:         64,
		Height:        64,
		PWidth:        IntParam{64},
		PHeight:       IntParam{64},
		Oversampling:  SelectParam{0},
		PreviewFrames: IntParam{5},
		RenderFrames:  IntParam{31},
		BkColor:       ColorParam{color.RGBA{255, 255, 255, 255}},
		AlignHorz:     SelectParam{0},
		ExportOption:  SelectParam{0},
		Duration:      IntParam{100},
		Loop:          IntParam{0},
		BiDir:         BoolParam{0},
	}
}

// EffectiveWidth returns the internal render width (logical × oversample factor).
func (p *Prefs) EffectiveWidth() int { return p.PWidth.Val * (1 << p.Oversampling.Val) }

// EffectiveHeight returns the internal render height.
func (p *Prefs) EffectiveHeight() int { return p.PHeight.Val * (1 << p.Oversampling.Val) }
