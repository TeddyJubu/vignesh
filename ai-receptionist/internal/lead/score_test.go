package lead

import "testing"

func TestScore(t *testing.T) {
	hot := map[string]string{
		"name":            "Ali",
		"business_type":   "Coach",
		"service_needed":  "Landing page",
		"budget":          "$2000",
		"timeline":        "ASAP this week",
		"current_website": "None",
	}
	if Score(hot) != "hot" {
		t.Fatalf("expected hot, got %s", Score(hot))
	}
	cold := map[string]string{
		"name":            "Sam",
		"business_type":   "Shop",
		"service_needed":  "Website",
		"budget":          "not sure",
		"timeline":        "just looking",
		"current_website": "yes.com",
	}
	if Score(cold) != "cold" {
		t.Fatalf("expected cold, got %s", Score(cold))
	}
}
