package model

import (
	"math"
	"testing"
)

func TestAnimCurveLinear(t *testing.T) {
	c := NewAnimCurve() // default: 0→100 linear ramp

	cases := [][2]float64{
		{0.0, 0.0},
		{0.25, 0.25},
		{0.5, 0.5},
		{0.75, 0.75},
		{1.0, 1.0},
	}
	for _, tc := range cases {
		got := c.Eval(tc[0])
		if math.Abs(got-tc[1]) > 1e-9 {
			t.Errorf("Eval(%v) = %v, want %v", tc[0], got, tc[1])
		}
	}
}

func TestAnimCurveStep(t *testing.T) {
	c := NewAnimCurve()
	c.StepReso.Val = 4 // quantize to 4 steps: 0, 1/3, 2/3, 1

	cases := [][2]float64{
		{0.0, 0.0},
		{0.1, 0.0},
		{0.3, 1.0 / 3.0},
		{0.6, 2.0 / 3.0},
		{0.9, 1.0},
	}
	for _, tc := range cases {
		got := c.Eval(tc[0])
		if math.Abs(got-tc[1]) > 1e-9 {
			t.Errorf("Eval(%v) with steps=4: got %v, want %v", tc[0], got, tc[1])
		}
	}
}

func TestAnimCurveFreeze(t *testing.T) {
	c := NewAnimCurve()

	c.StepReso.Val = 1 // freeze: always returns 0
	for _, r := range []float64{0, 0.5, 1.0} {
		if got := c.Eval(r); got != 0 {
			t.Errorf("Eval(%v) with StepReso=1: got %v, want 0", r, got)
		}
	}
}

func TestEvalParam(t *testing.T) {
	var curves [8]AnimCurve
	for i := range curves {
		curves[i] = NewAnimCurve()
	}
	// Disabled animation: returns From.
	got := EvalParam(10, 20, 0, &curves, 0.5)
	if math.Abs(got-10.0) > 1e-9 {
		t.Errorf("EvalParam off: got %v want 10", got)
	}

	// Linear interpolation.
	got = EvalParam(10, 20, 1, &curves, 0.5)
	if math.Abs(got-15.0) > 1e-9 {
		t.Errorf("EvalParam linear: got %v want 15", got)
	}

	// Curve 1 (legacy selector value 2) uses curves[0].
	got = EvalParam(10, 20, 2, &curves, 0.5)
	if math.Abs(got-15.0) > 1e-9 {
		t.Errorf("EvalParam curve1: got %v want 15", got)
	}
}
