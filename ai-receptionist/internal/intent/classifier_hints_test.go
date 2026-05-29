package intent

import "testing"

func TestApplyIntentHintsThirdPartyBooking(t *testing.T) {
	msg := "Book a meeting with John Tan, +6598765432, about Epicware partnership"
	r := Result{Intent: "sales_qualify", Confidence: 0.9, Summary: "book meeting"}
	got := applyIntentHints(msg, normalizeResult(r))
	if got.Intent != "outbound_book" {
		t.Fatalf("expected outbound_book, got %q", got.Intent)
	}
}

func TestApplyIntentHintsSelfBooking(t *testing.T) {
	msg := "I want to book a meeting with you"
	r := Result{Intent: "outbound_book", Confidence: 0.8, Summary: "book"}
	got := applyIntentHints(msg, normalizeResult(r))
	if got.Intent != "sales_qualify" {
		t.Fatalf("expected sales_qualify, got %q", got.Intent)
	}
}
