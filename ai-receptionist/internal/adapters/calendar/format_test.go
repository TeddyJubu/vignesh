package calendar

import (
	"strings"
	"testing"
	"time"
)

func TestFormatHumanSlot(t *testing.T) {
	loc := time.FixedZone("SGT", 8*3600)
	cases := []struct {
		t    time.Time
		want string
	}{
		{time.Date(2026, 5, 29, 10, 0, 0, 0, loc), "Fri 10am"},
		{time.Date(2026, 5, 29, 15, 30, 0, 0, loc), "Fri 3:30pm"},
	}
	for _, tc := range cases {
		got := formatHumanSlot(tc.t)
		if got != tc.want {
			t.Fatalf("formatHumanSlot(%v) = %q, want %q", tc.t, got, tc.want)
		}
		if strings.Contains(got, "37") || strings.Contains(got, "55") {
			t.Fatalf("implausible slot label: %q", got)
		}
	}
}
