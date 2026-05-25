package receptionist

import (
	"testing"
	"time"

	"ai-receptionist/internal/config"
)

func TestMergeLeadStatus(t *testing.T) {
	if got := mergeLeadStatus("notified", true, true); got != "notified" {
		t.Fatalf("notified + qualified = %q, want notified", got)
	}
	if got := mergeLeadStatus("new", false, true); got != "collecting" {
		t.Fatalf("new = %q, want collecting", got)
	}
	if got := mergeLeadStatus("collecting", true, true); got != "qualified" {
		t.Fatalf("collecting + qualified = %q", got)
	}
}

func TestShouldSendQuietHoursReply(t *testing.T) {
	q := config.QuietHours{Enabled: true, TZ: "UTC", Start: "22:00", End: "08:00"}
	// 23:00 UTC — in quiet hours
	now := time.Date(2026, 5, 26, 23, 0, 0, 0, time.UTC)
	if !shouldSendQuietHoursReply(q, nil, now) {
		t.Fatal("first quiet-hours message should auto-reply")
	}
	last := time.Date(2026, 5, 26, 23, 30, 0, 0, time.UTC)
	if shouldSendQuietHoursReply(q, &last, now) {
		t.Fatal("second message in same quiet window should not auto-reply")
	}
	// 10:00 UTC — outside quiet hours; prior reply was overnight
	morning := time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC)
	if shouldSendQuietHoursReply(q, &last, morning) {
		t.Fatal("morning user message should not trigger quiet auto-reply")
	}
}
