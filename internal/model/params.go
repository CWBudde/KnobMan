// Package model contains the KnobMan document data model.
package model

import "image/color"

// FloatParam is a float64-valued parameter (maps to Java ParamV).
type FloatParam struct{ Val float64 }

// IntParam is an int-valued parameter (maps to Java ParamI).
type IntParam struct{ Val int }

// BoolParam is a boolean parameter stored as 0/1 (maps to Java ParamC).
type BoolParam struct{ Val int } // 0 = false, 1 = true

// SelectParam is a selection/combo-box index (maps to Java ParamS).
type SelectParam struct{ Val int }

// StringParam is a string-valued parameter (maps to Java ParamT).
type StringParam struct{ Val string }

// ColorParam holds an RGBA colour (maps to Java ParamCol / Col).
type ColorParam struct{ Val color.RGBA }

// IsTrue returns whether the BoolParam is set.
func (p BoolParam) IsTrue() bool { return p.Val != 0 }
