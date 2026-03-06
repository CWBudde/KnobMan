package render

import "testing"

func TestSubstituteFrameCountersBasic(t *testing.T) {
	if got := SubstituteFrameCounters("A(1:9)", 0, 9); got != "A1" {
		t.Fatalf("unexpected start substitution: %q", got)
	}

	if got := SubstituteFrameCounters("A(1:9)", 8, 9); got != "A9" {
		t.Fatalf("unexpected end substitution: %q", got)
	}
}

func TestSubstituteFrameCountersPadding(t *testing.T) {
	if got := SubstituteFrameCounters("N(01:99)", 0, 10); got != "N01" {
		t.Fatalf("expected zero-padding to width 2, got %q", got)
	}
}
