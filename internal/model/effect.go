package model

// Effect holds the full effect stack for one layer.
// All animatable parameters exist as From/To float pairs plus an anim curve
// selector (0 = off/use From, 1 = linear, 2..9 = global curve 1..8).
// Field names and defaults match the Java Eff class exactly.
type Effect struct {
	// ── Transform ───────────────────────────────────────────────────────────
	AntiAlias  BoolParam   // enable antialiasing in transforms (default 1)
	Unfold     BoolParam   // unfold/mirror mode (default 0)
	AnimStep   IntParam    // independent anim step count (0 = use document frames)
	ZoomXYSepa BoolParam   // separate X/Y zoom (default 0)
	ZoomXF     FloatParam  // zoom X from % (default 100)
	ZoomXT     FloatParam  // zoom X to   %
	ZoomXAnim  SelectParam // anim curve selector
	ZoomYF     FloatParam  // zoom Y from % (default 100)
	ZoomYT     FloatParam
	ZoomYAnim  SelectParam
	OffXF      FloatParam // offset X from (default 0)
	OffXT      FloatParam
	OffXAnim   SelectParam
	OffYF      FloatParam
	OffYT      FloatParam
	OffYAnim   SelectParam
	KeepDir    BoolParam  // keep direction fixed during rotation (default 0)
	CenterX    FloatParam // rotation centre X (default 0)
	CenterY    FloatParam // rotation centre Y
	AngleF     FloatParam // rotation angle from degrees (default 0)
	AngleT     FloatParam
	AngleAnim  SelectParam

	// ── Color adjustments ───────────────────────────────────────────────────
	AlphaF         FloatParam // alpha from % (default 100)
	AlphaT         FloatParam
	AlphaAnim      SelectParam
	BrightF        FloatParam // brightness from (default 0)
	BrightT        FloatParam
	BrightAnim     SelectParam
	ContrastF      FloatParam // contrast from (default 0)
	ContrastT      FloatParam
	ContrastAnim   SelectParam
	SaturationF    FloatParam
	SaturationT    FloatParam
	SaturationAnim SelectParam
	HueF           FloatParam
	HueT           FloatParam
	HueAnim        SelectParam

	// ── Mask 1 ──────────────────────────────────────────────────────────────
	Mask1Ena       BoolParam   // enable mask 1 (default 0)
	Mask1Type      SelectParam // 0=Rotation 1=Radial 2=Horizontal 3=Vertical
	Mask1Grad      FloatParam  // gradient softness (default 0)
	Mask1GradDir   SelectParam // gradient direction
	Mask1StartF    FloatParam  // mask start from (default −140)
	Mask1StartT    FloatParam
	Mask1StartAnim SelectParam
	Mask1StopF     FloatParam // mask stop from (default 140)
	Mask1StopT     FloatParam
	Mask1StopAnim  SelectParam

	// ── Mask 2 ──────────────────────────────────────────────────────────────
	Mask2Ena       BoolParam
	Mask2Op        SelectParam // combine operation 0=AND 1=OR
	Mask2Type      SelectParam
	Mask2Grad      FloatParam
	Mask2GradDir   SelectParam
	Mask2StartF    FloatParam
	Mask2StartT    FloatParam
	Mask2StartAnim SelectParam
	Mask2StopF     FloatParam
	Mask2StopT     FloatParam
	Mask2StopAnim  SelectParam

	// ── Frame mask ──────────────────────────────────────────────────────────
	FMaskEna   SelectParam // 0=off 1=range 2=bitmask
	FMaskStart FloatParam  // frame range start % (default 0)
	FMaskStop  FloatParam  // frame range stop  % (default 100)
	FMaskBits  StringParam // bitmask string (default "11111111")

	// ── Specular highlight ───────────────────────────────────────────────────
	SLightDirF    FloatParam // specular light dir from (default −45)
	SLightDirT    FloatParam
	SLightDirAnim SelectParam
	SDensityF     FloatParam // specular density from (default 0)
	SDensityT     FloatParam
	SDensityAnim  SelectParam

	// ── Drop shadow ─────────────────────────────────────────────────────────
	DLightDirEna  BoolParam  // enable drop-shadow light direction (default 0)
	DLightDirF    FloatParam // (default −45)
	DLightDirT    FloatParam
	DLightDirAnim SelectParam
	DOffsetF      FloatParam // shadow offset from (default 5)
	DOffsetT      FloatParam
	DOffsetAnim   SelectParam
	DDensityF     FloatParam // shadow density from (default 0)
	DDensityT     FloatParam
	DDensityAnim  SelectParam
	DDiffuseF     FloatParam // shadow diffuse/blur from (default 0)
	DDiffuseT     FloatParam
	DDiffuseAnim  SelectParam
	DSType        SelectParam // shadow type 0=soft 1=hard
	DSGrad        FloatParam  // shadow gradient (default 100)

	// ── Inner shadow ────────────────────────────────────────────────────────
	ILightDirEna  BoolParam
	ILightDirF    FloatParam // (default −45)
	ILightDirT    FloatParam
	ILightDirAnim SelectParam
	IOffsetF      FloatParam // (default 5)
	IOffsetT      FloatParam
	IOffsetAnim   SelectParam
	IDensityF     FloatParam // (default 0)
	IDensityT     FloatParam
	IDensityAnim  SelectParam
	IDiffuseF     FloatParam // (default 20)
	IDiffuseT     FloatParam
	IDiffuseAnim  SelectParam

	// ── Emboss (highlight) ───────────────────────────────────────────────────
	ELightDirEna  BoolParam
	ELightDirF    FloatParam // (default −45)
	ELightDirT    FloatParam
	ELightDirAnim SelectParam
	EOffsetF      FloatParam // (default 0)
	EOffsetT      FloatParam
	EOffsetAnim   SelectParam
	EDensityF     FloatParam // (default 0)
	EDensityT     FloatParam
	EDensityAnim  SelectParam
}

// NewEffect returns an Effect with Java-matching default values.
func NewEffect() Effect {
	return Effect{
		AntiAlias: BoolParam{1},
		ZoomXF:    FloatParam{100}, ZoomXT: FloatParam{100},
		ZoomYF: FloatParam{100}, ZoomYT: FloatParam{100},
		AlphaF: FloatParam{100}, AlphaT: FloatParam{100},
		Mask1StartF: FloatParam{-140}, Mask1StartT: FloatParam{-140},
		Mask1StopF: FloatParam{140}, Mask1StopT: FloatParam{140},
		Mask2StartF: FloatParam{-140}, Mask2StartT: FloatParam{-140},
		Mask2StopF: FloatParam{140}, Mask2StopT: FloatParam{140},
		FMaskStop:  FloatParam{100},
		FMaskBits:  StringParam{"11111111"},
		SLightDirF: FloatParam{-45}, SLightDirT: FloatParam{-45},
		DLightDirF: FloatParam{-45}, DLightDirT: FloatParam{-45},
		DOffsetF: FloatParam{5}, DOffsetT: FloatParam{5},
		DDiffuseF: FloatParam{0}, DDiffuseT: FloatParam{0},
		DSGrad:     FloatParam{100},
		ILightDirF: FloatParam{-45}, ILightDirT: FloatParam{-45},
		IOffsetF: FloatParam{5}, IOffsetT: FloatParam{5},
		IDiffuseF: FloatParam{20}, IDiffuseT: FloatParam{20},
		ELightDirF: FloatParam{-45}, ELightDirT: FloatParam{-45},
	}
}
