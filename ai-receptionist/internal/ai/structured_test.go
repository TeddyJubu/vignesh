package ai

import "testing"

func TestDecodeStructured_StripsFences(t *testing.T) {
	raw := "```json\n{\"reply\":\"hi\",\"lead_updates\":{},\"qualified\":false,\"summary\":\"\"}\n```"
	r, err := DecodeStructured(raw)
	if err != nil {
		t.Fatal(err)
	}
	if r.Reply != "hi" {
		t.Fatalf("reply=%q", r.Reply)
	}
}

func TestDecodeStructured_coercesLeadUpdateTypes(t *testing.T) {
	raw := `{"reply":"noted","lead_updates":{"owner_verified":true,"count":3},"qualified":false,"summary":""}`
	r, err := DecodeStructured(raw)
	if err != nil {
		t.Fatal(err)
	}
	if r.LeadUpdates["owner_verified"] != "true" {
		t.Fatalf("bool lead update=%q", r.LeadUpdates["owner_verified"])
	}
	if r.LeadUpdates["count"] != "3" {
		t.Fatalf("number lead update=%q", r.LeadUpdates["count"])
	}
}

func TestStripCodeFences(t *testing.T) {
	if got := StripCodeFences("```\n{}\n```"); got != "{}" {
		t.Fatalf("got %q", got)
	}
}
