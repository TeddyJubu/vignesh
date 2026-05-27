package receptionist

import (
	"testing"

	"ai-receptionist/internal/agent"
)

func TestApplyBookingPolicy_BlocksUnconfirmed(t *testing.T) {
	reply := "Your booking is confirmed for tomorrow 3pm"
	results := []agent.ToolResult{{Tool: "book_appointment", Output: `{"booked":false}`}}
	got := ApplyBookingPolicy(reply, results)
	if got != deferReply {
		t.Fatalf("got %q", got)
	}
}

func TestApplyBookingPolicy_AllowsConfirmed(t *testing.T) {
	reply := "Your booking is confirmed for tomorrow 3pm"
	results := []agent.ToolResult{{Tool: "book_appointment", Output: `{"booked":true}`}}
	got := ApplyBookingPolicy(reply, results)
	if got != reply {
		t.Fatalf("got %q", got)
	}
}
