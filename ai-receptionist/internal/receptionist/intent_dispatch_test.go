package receptionist

import "testing"

func TestParseLeadScrapeRequest(t *testing.T) {
	q, n, v := parseLeadScrapeRequest("Scrape 10 F&B consultants in Singapore with email")
	if n != 10 {
		t.Fatalf("count=%d", n)
	}
	if q == "" || v == "" {
		t.Fatal("empty query/vertical")
	}
}

func TestParseOutboundBookRequest(t *testing.T) {
	name, phone, purpose := parseOutboundBookRequest("Book a meeting with John Tan, +6598765432, about Epicware partnership")
	if phone != "6598765432" {
		t.Fatalf("phone=%q", phone)
	}
	if name == "" {
		t.Fatal("expected name")
	}
	if purpose == "" {
		t.Fatal("expected purpose")
	}
}

func TestParseGroupNLCreate(t *testing.T) {
	a, ok := parseGroupNL(`Create a WhatsApp group called Epicware VIP and add +6591234567`)
	if !ok || a.Kind != "create" {
		t.Fatalf("parse failed: %+v ok=%v", a, ok)
	}
	if a.Name != "Epicware VIP" {
		t.Fatalf("name=%q", a.Name)
	}
	if len(a.Phones) != 1 {
		t.Fatalf("phones=%v", a.Phones)
	}
}

func TestParseGroupNLCreateUnquoted(t *testing.T) {
	a, ok := parseGroupNL(`Create a WhatsApp group Epicware VIP and add +6591234567`)
	if !ok || a.Kind != "create" {
		t.Fatalf("parse failed: %+v ok=%v", a, ok)
	}
	if a.Name != "Epicware VIP" {
		t.Fatalf("name=%q want Epicware VIP", a.Name)
	}
}

func TestParseGuestSlotChoice(t *testing.T) {
	slots := []string{"Mon 3pm SGT", "Tue 10am SGT", "Wed 2pm SGT"}
	if chosen, ok := parseGuestSlotChoice("2", "2", slots); !ok || chosen != slots[1] {
		t.Fatalf("pick 2: chosen=%q ok=%v", chosen, ok)
	}
	if _, ok := parseGuestSlotChoice("who is this?", "who is this?", slots); ok {
		t.Fatal("expected no match for unrelated reply")
	}
}
