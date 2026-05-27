package receptionist

import (
	"strings"
	"testing"
)

func TestFinalizeCustomerReply_personaAndModel(t *testing.T) {
	out := FinalizeCustomerReply(
		"I am a large language model trained by Google.",
		"What model are you?",
		"Teddy",
		"We build websites.",
		nil,
	)
	if strings.Contains(strings.ToLower(out), "google") || strings.Contains(strings.ToLower(out), "language model") {
		t.Fatalf("model leak: %q", out)
	}
	if !strings.Contains(out, "Julia") {
		t.Fatalf("expected Julia persona: %q", out)
	}
}

func TestFinalizeCustomerReply_serviceEcho(t *testing.T) {
	out := FinalizeCustomerReply(
		"What services do you offer?",
		"What services do you offer?",
		"Teddy",
		"We build fast websites for coaches.",
		nil,
	)
	if out == "What services do you offer?" {
		t.Fatal("echo not fixed")
	}
	if !strings.Contains(out, "Julia") {
		t.Fatalf("expected services answer: %q", out)
	}
}

func TestFinalizeCustomerReply_stripsQualifiedLead(t *testing.T) {
	out := FinalizeCustomerReply(
		"🔥 New qualified lead — Teddy [HOT]\nName: Raj",
		"thanks",
		"Teddy",
		"websites",
		nil,
	)
	if strings.Contains(strings.ToLower(out), "qualified lead") {
		t.Fatalf("internal alert leaked: %q", out)
	}
}

func TestFinalizeCustomerReply_singleQuestion(t *testing.T) {
	out := FinalizeCustomerReply(
		"What's your name? And your budget? And timeline?",
		"hi",
		"Teddy",
		"websites",
		nil,
	)
	if strings.Count(out, "?") != 1 {
		t.Fatalf("expected one question, got %q", out)
	}
}

func TestFinalizeCustomerReply_invalidSlots(t *testing.T) {
	out := FinalizeCustomerReply(
		"I can do Fri 37am or Fri 55am.",
		"book",
		"Teddy",
		"websites",
		nil,
	)
	if strings.Contains(out, "37am") {
		t.Fatalf("invalid slots not stripped: %q", out)
	}
}

func TestFinalizeCustomerReply_validTwoDigitSlots(t *testing.T) {
	out := FinalizeCustomerReply(
		"Fri 10am and Fri 11am work, or Fri 12pm if you prefer.",
		"book",
		"Teddy",
		"websites",
		nil,
	)
	if !strings.Contains(out, "10am") || !strings.Contains(out, "12pm") {
		t.Fatalf("valid two-digit slots were stripped: %q", out)
	}
}

func TestCustomerSafeToolOutput_keepsValidTwoDigitSlots(t *testing.T) {
	out := CustomerSafeToolOutput("check_calendar_availability", `{"slots":["Fri 10am","Fri 37am","Fri 12pm"]}`)
	if !strings.Contains(out, "Fri 10am") || !strings.Contains(out, "Fri 12pm") {
		t.Fatalf("valid slots removed: %s", out)
	}
	if strings.Contains(out, "37am") {
		t.Fatalf("invalid slot kept: %s", out)
	}
}
