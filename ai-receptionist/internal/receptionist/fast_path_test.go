package receptionist

import "testing"

func TestNeedsPlannerPath(t *testing.T) {
	if needsPlannerPath(modeSales, "hello", false) {
		t.Fatal("expected fast path for simple greeting")
	}
	if !needsPlannerPath(modeSales, "hello", true) {
		t.Fatal("pending plan should use planner path")
	}
	if !needsPlannerPath(modeBooking, "hi", false) {
		t.Fatal("booking mode should use planner path")
	}
	if !needsPlannerPath(modeSales, "can I book tomorrow?", false) {
		t.Fatal("booking keyword should use planner path")
	}
}

func TestLooksLikeSchedulingUpdate(t *testing.T) {
	cases := []struct {
		text string
		want bool
	}{
		{"Tuesday 3pm Singapore", true},
		{"how about fri 11am?", true},
		{"2:30 works for me", true},
		{"sgt please", true},
		{"hello", false},
		{"I am Teddy", false},
		{"what are your prices?", false},
	}
	for _, tc := range cases {
		if got := looksLikeSchedulingUpdate(tc.text); got != tc.want {
			t.Fatalf("looksLikeSchedulingUpdate(%q) = %v, want %v", tc.text, got, tc.want)
		}
	}
}

func TestShouldRecallMemory(t *testing.T) {
	if shouldRecallMemory(modeSales, "hello") {
		t.Fatal("simple chat should skip recall")
	}
	if !shouldRecallMemory(modeBooking, "hi") {
		t.Fatal("booking should allow recall")
	}
}
