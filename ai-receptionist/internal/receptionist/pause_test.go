package receptionist

import (
	"testing"

	"ai-receptionist/internal/whatsapp"
)

func TestIsPauseKeyword(t *testing.T) {
	for _, kw := range []string{"PAUSE", "human", "Human", " stop bot "} {
		if !IsPauseKeyword(kw) {
			t.Fatalf("expected pause keyword: %q", kw)
		}
	}
	for _, kw := range []string{"hello", "stop", "STOP", "please stop"} {
		if IsPauseKeyword(kw) {
			t.Fatalf("should not pause: %q", kw)
		}
	}
}

func TestCanPauseSender(t *testing.T) {
	owner := "8801000000000"
	in := whatsapp.InboundContext{Sender: "8801000000000"}
	if !canPauseSender(in, owner) {
		t.Fatal("owner should pause")
	}
	in.Sender = "8801999999999"
	if canPauseSender(in, owner) {
		t.Fatal("lead should not pause")
	}
}
