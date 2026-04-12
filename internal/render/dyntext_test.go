package render

import "testing"

func TestResolveDynamicTextStaticList(t *testing.T) {
	cases := []struct {
		frame int
		want  string
	}{
		{0, "Sin"},
		{7, "Tri"},
		{15, "Sqr"},
		{23, "Saw"},
		{30, "Noise"},
	}

	for _, tc := range cases {
		got := ResolveDynamicText("Sin,Tri,Sqr,Saw,Noise", tc.frame, 31)
		if got != tc.want {
			t.Fatalf("frame %d: got %q want %q", tc.frame, got, tc.want)
		}
	}
}

func TestResolveDynamicTextZeroPaddedRange(t *testing.T) {
	if got := ResolveDynamicText("(0:99:%02d)", 0, 31); got != "00" {
		t.Fatalf("frame 0: got %q want %q", got, "00")
	}

	if got := ResolveDynamicText("(0:99:%02d)", 30, 31); got != "99" {
		t.Fatalf("last frame: got %q want %q", got, "99")
	}
}

func TestResolveDynamicTextRangeWithSuffix(t *testing.T) {
	if got := ResolveDynamicText("(1:3:%d)k", 0, 3); got != "1k" {
		t.Fatalf("frame 0: got %q want %q", got, "1k")
	}

	if got := ResolveDynamicText("(1:3:%d)k", 2, 3); got != "3k" {
		t.Fatalf("frame 2: got %q want %q", got, "3k")
	}
}

func TestSubstituteFrameCountersCompatibility(t *testing.T) {
	if got := SubstituteFrameCounters("A(1:9)", 0, 9); got != "A1" {
		t.Fatalf("unexpected start substitution: %q", got)
	}

	if got := SubstituteFrameCounters("N(01:99)", 0, 10); got != "N1" {
		t.Fatalf("unexpected dynamic range substitution: %q", got)
	}
}
