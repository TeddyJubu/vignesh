package receptionist

import "testing"

func TestSanitizeReply_AllowsKnowledgePricing(t *testing.T) {
	cases := []string{
		"Visibility is $349/mo plus $99 per extra outlet — for 3 outlets that's $547/mo.",
		"Results within 30 days — guaranteed or full refund within 24 hours.",
		"WordPress website build is $1,500 one-time.",
		"We don't offer a free trial; we have a $1 Visibility Audit instead.",
	}
	for _, in := range cases {
		got := SanitizeReply(in)
		if got != in {
			t.Fatalf("SanitizeReply changed allowed reply:\nin: %q\ngot: %q", in, got)
		}
	}
}

func TestSanitizeReply_BlocksFirmSalesGuarantees(t *testing.T) {
	cases := []string{
		"I guarantee you will rank #1 on Google Maps.",
		"The fixed price is locked in forever.",
		"This will cost exactly $200 for you.",
	}
	for _, in := range cases {
		got := SanitizeReply(in)
		if got != deferReply {
			t.Fatalf("expected defer for %q, got %q", in, got)
		}
	}
}
