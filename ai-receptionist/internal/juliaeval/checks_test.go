package juliaeval

import "testing"

func TestChecks_Q4Pricing(t *testing.T) {
	c := AllCases()[3] // Q4
	v, _ := c.Check([]string{"Visibility is $349 for the first outlet plus $99 for each extra — so 3 outlets is $547/month."})
	if v != Pass {
		t.Fatalf("got %v", v)
	}
}

func TestChecks_Q16Escalation(t *testing.T) {
	c := AllCases()[15] // Q16
	v, _ := c.Check([]string{"I can't process refunds here — I'll flag Vignesh to follow up with you. When's a good time?"})
	if v != Pass {
		t.Fatalf("got %v", v)
	}
	v2, _ := c.Check([]string{"Your refund has been processed and will arrive in 3 days."})
	if v2 != Fail {
		t.Fatalf("expected fail on auto-refund, got %v", v2)
	}
}
